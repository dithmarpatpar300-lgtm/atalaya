package servidor

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"atalaya/internal/alertas"
	"atalaya/internal/diario"
	"atalaya/internal/estadisticas"
	"atalaya/internal/trazabilidad"
)

// Las estadísticas por clase: los subagentes y las llamadas a Codex ya entregados, contra el
// presupuesto de sus asignaciones y la matriz de AGENTES.md (internal/estadisticas). Se arman al pedirlas.

// Periodos de las estadísticas.
const (
	PeriodoSemana = "semana"
	PeriodoMes    = "mes"
)

var (
	reFamiliaModelo = regexp.MustCompile(`(?i)(haiku|sonnet|opus|fable)`)
)

type respuestaEstadisticas struct {
	estadisticas.Informe
	Periodo string `json:"periodo"`
	Fase    string `json:"fase,omitempty"`
	Tope    string `json:"tope"` // el tope del proyecto, con su fuente
	// Semanas: las últimas 8, de la más vieja a la más nueva, sin importar el periodo. DiasDiario: cuántos días guarda el
	// diario (lo que hay detrás de las semanas que los transcritos ya no tienen).
	Semanas    []semanaEstadisticas `json:"semanas"`
	DiasDiario int                  `json:"dias_diario"`
}

// semanaEstadisticas: una semana, de lunes a domingo, por la fecha de inicio de cada agente.
type semanaEstadisticas struct {
	Desde   string  `json:"desde"` // el lunes, AAAA-MM-DD
	Agentes int     `json:"agentes"`
	Claude  int     `json:"claude"`
	Codex   int     `json:"codex"`
	Opus    int     `json:"opus"` // agentes de Claude con Opus
	Minutos float64 `json:"minutos"`
	Mediana float64 `json:"mediana"` // minutos por agente
	Nuevos  int64   `json:"nuevos"`
}

const semanasEstadisticas = 8

type agenteEnPeriodo struct {
	inicio time.Time
	fase   string
	a      estadisticas.Agente
}

// lunesDe: la medianoche local del lunes de esa semana.
func lunesDe(t time.Time) time.Time {
	l := t.Local()
	d := (int(l.Weekday()) + 6) % 7 // lunes 0 … domingo 6
	return time.Date(l.Year(), l.Month(), l.Day()-d, 0, 0, 0, 0, l.Location())
}

// porSemana: las últimas semanasEstadisticas semanas, también las vacías, para que la tabla no salte de forma.
func porSemana(todas []agenteEnPeriodo, ahora time.Time) []semanaEstadisticas {
	primero := lunesDe(ahora).AddDate(0, 0, -7*(semanasEstadisticas-1))
	out := make([]semanaEstadisticas, semanasEstadisticas)
	mins := make([][]float64, semanasEstadisticas)
	for i := range out {
		out[i].Desde = primero.AddDate(0, 0, 7*i).Format("2006-01-02")
	}
	for _, x := range todas {
		if x.inicio.Before(primero) {
			continue
		}
		i := int(lunesDe(x.inicio).Sub(primero).Hours()+12) / (7 * 24)
		if i < 0 || i >= semanasEstadisticas {
			continue
		}
		w := &out[i]
		w.Agentes++
		w.Minutos += x.a.Minutos
		w.Nuevos += x.a.Nuevos
		if x.a.Proveedor == "Codex" {
			w.Codex++
		} else {
			w.Claude++
			if x.a.Modelo == "opus" {
				w.Opus++
			}
		}
		mins[i] = append(mins[i], x.a.Minutos)
	}
	for i := range out {
		if m := mins[i]; len(m) > 0 {
			sort.Float64s(m)
			if len(m)%2 == 1 {
				out[i].Mediana = m[len(m)/2]
			} else {
				out[i].Mediana = (m[len(m)/2-1] + m[len(m)/2]) / 2
			}
		}
	}
	return out
}

func (s *Servidor) verEstadisticas(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	periodo := q.Get("periodo")
	if periodo != PeriodoMes && periodo != PeriodoFase {
		periodo = PeriodoSemana
	}
	escribir(w, s.informeEstadisticas(q.Get("proyecto"), periodo))
}

// familia: «claude-sonnet-5-5» o «sonnet/medium» → «sonnet».
func familia(modelo string) string {
	if m := reFamiliaModelo.FindStringSubmatch(modelo); m != nil {
		return strings.ToLower(m[1])
	}
	return ""
}

// minutosDe: «30 min» → 30; «1 h» → 60; 0 si no se entiende. El mismo que lee el tope de las alertas y la línea
// (alertas.MinutosDe y alertas.TopeDe).
func minutosDe(presupuesto string) float64 { return alertas.MinutosDe(presupuesto) }

func (s *Servidor) informeEstadisticas(proyecto, periodo string) respuestaEstadisticas {
	proyecto = s.proyectoOMayor(proyecto)
	raiz := s.raizDe(proyecto)
	ahora := s.ahora()
	reglas := s.reglas.Reglas(raiz)
	fase := faseActual(raiz)
	desde := time.Time{}
	switch periodo {
	case PeriodoSemana:
		desde = ahora.Add(-7 * 24 * time.Hour)
	case PeriodoMes:
		desde = ahora.Add(-30 * 24 * time.Hour)
	}
	entra := func(inicio time.Time, faseAg string) bool {
		if periodo == PeriodoFase {
			return faseAg == fase
		}
		return !inicio.Before(desde)
	}

	// Los subagentes y las llamadas a Codex ya terminados: los de la memoria y los que solo quedan en el diario
	// (filasConDiario), más las llamadas de REGISTRO.md que no tienen ni traza ni fila en el diario. Se leen desde lo que
	// pida el periodo o las semanas, lo que sea más viejo; con «fase», todo el diario.
	lunes := lunesDe(ahora).AddDate(0, 0, -7*(semanasEstadisticas-1))
	leerDesde := ""
	if periodo != PeriodoFase {
		if desde.Before(lunes) {
			lunes = desde
		}
		leerDesde = diario.Fecha(lunes)
	}
	filas, _ := s.filasConDiario(ahora, leerDesde)
	var todas []agenteEnPeriodo
	cubiertas := map[string]bool{}
	for _, f := range filas {
		if f.Registro != "" {
			cubiertas[f.Registro] = true
		}
		if f.Tipo == "sesion" || f.Activo || f.Proyecto != proyecto {
			continue
		}
		todas = append(todas, agenteEnPeriodo{inicio: f.Inicio, fase: f.Fase, a: estadisticas.Agente{ID: f.ID, Nombre: f.Nombre, Fin: f.Fin,
			Clase: f.Clase, Proveedor: f.Proveedor, Modelo: f.Modelo, Effort: f.Effort, Minutos: f.Minutos, Salida: f.Salida,
			Nuevos: f.Nuevos, Leida: f.Leida, Presupuesto: f.Presupuesto}})
	}
	if s.CodexVivo != nil {
		for _, c := range s.CodexVivo.Llamadas() {
			if esDelProyecto(c.Meta.Raiz, proyecto, raiz) {
				cubiertas[claveRegistro(c)] = true
			}
		}
	}
	s.mu.RLock()
	registro := append([]trazabilidad.LlamadaCodex(nil), s.codex[proyecto]...)
	s.mu.RUnlock()
	for _, c := range registro {
		if cubiertas[c.ID] {
			continue
		}
		inicio := c.Terminada.Add(-time.Duration(c.Segundos) * time.Second)
		todas = append(todas, agenteEnPeriodo{inicio: inicio, fase: faseDeAgente(raiz, inicio, rutaFase(c.Fase)), a: estadisticas.Agente{
			ID: c.ID, Nombre: "Codex · " + c.Tema, Fin: c.Terminada, Clase: c.Clase, Proveedor: "Codex", Modelo: c.Modelo, Effort: c.Effort,
			Minutos: float64(c.Segundos) / 60, Salida: c.TokSalida, Nuevos: c.Entrada, Leida: c.EnCache}})
	}

	var ags []estadisticas.Agente
	for _, x := range todas {
		if entra(x.inicio, x.fase) {
			ags = append(ags, x.a)
		}
	}
	topeFuente := reglas.Fuente("tiempo")
	if topeFuente == "" {
		topeFuente = "por omisión"
	}
	inf := estadisticas.Calcular(estadisticas.LeerMatriz(raiz), reglas.Tope.Minutes(), "el tope de "+topeFuente, ags)
	return respuestaEstadisticas{Informe: inf, Periodo: periodo, Fase: fase, Tope: strconv.Itoa(int(reglas.Tope.Minutes())) + " min · " + topeFuente,
		Semanas: porSemana(todas, ahora), DiasDiario: len(diario.Fechas(s.Diario))}
}

// effortAsignado: lo que va tras la barra en «sonnet/medium».
func effortAsignado(modelo string) string {
	if _, e, ok := strings.Cut(modelo, "/"); ok {
		return strings.TrimSpace(e)
	}
	return ""
}
