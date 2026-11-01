package servidor

import (
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"atalaya/internal/avance"
	"atalaya/internal/claude"
	"atalaya/internal/estadisticas"
)

// La portada «Tus proyectos»: una tarjeta por proyecto registrado, armada con sus archivos.
// Cada dato sale de un lector que ya existe: el contrato de trazabilidad (siete archivos), el avance de la fase en
// curso (internal/avance), la salud de hoy (informeProceso), los agentes vivos (todos) y las consultas a Atalaya de
// hoy (los pasos «Atalaya» de los transcritos y de la traza de Codex). Donde falta un archivo, lo dice: no se supone.
//
// Barato: el contrato son siete stat por proyecto y el avance, un AVANCE.md. Lo único que recorre carpetas es la
// salud (las temporales de cada proyecto), que se guarda mientras no cambien las temporales, las bitácoras ni los
// agentes del proyecto, y 30 s como mucho.

// archivoContrato: uno de los siete del contrato de trazabilidad.
type archivoContrato struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"` // como lo nombra el kit
	Ruta   string `json:"ruta"`   // relativa a la raíz: la que se encontró o, si falta, la que se espera
	Esta   bool   `json:"esta"`
}

// avancePortafolio: lo de la fase en curso que dibuja el hilo de la tarjeta.
type avancePortafolio struct {
	Fase      string  `json:"fase"`
	Ruta      string  `json:"ruta"`
	Ahora     string  `json:"ahora"`
	Calculado float64 `json:"calculado"`
	Horas     float64 `json:"horas"`
	Hechas    float64 `json:"horas_hechas"`
	// EnCurso: lo avanzado de los paquetes en curso (horas × %); Avanzadas: Σ horas × % de todos los paquetes.
	EnCurso   float64 `json:"horas_en_curso"`
	Avanzadas float64 `json:"horas_avanzadas"`
}

type saludPortafolio struct {
	EnRegla int `json:"en_regla"`
	Total   int `json:"total"`
}

type proyectoPortafolio struct {
	Nombre string `json:"nombre"`
	Sigla  string `json:"sigla"`
	Ruta   string `json:"ruta"`
	Nivel  int    `json:"nivel"`
	// Existe: la carpeta del proyecto está en el disco.
	Existe bool `json:"existe"`
	// Fase: la del AVANCE.md o, sin él, la carpeta docs/trazabilidad/F<n> más alta; vacía si no hay ninguna.
	Fase      string            `json:"fase,omitempty"`
	Contrato  []archivoContrato `json:"contrato"`
	Avance    *avancePortafolio `json:"avance"`
	Salud     saludPortafolio   `json:"salud"`
	Vivos     int               `json:"vivos"`
	Consultas int               `json:"consultas"`
	// Decisiones: las preguntas de sus sesiones que te esperan (de menos de un día; decisiones.go).
	Decisiones int `json:"decisiones"`
}

func (s *Servidor) verPortafolio(w http.ResponseWriter, _ *http.Request) {
	escribir(w, s.portafolio())
}

func (s *Servidor) portafolio() []proyectoPortafolio {
	ahora := s.ahora()
	hoy := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, ahora.Location())

	vivos := map[string]int{}
	estados := map[string][]string{} // por proyecto: «id=estado» de cada agente, para la clave de la salud
	for _, a := range s.todos() {
		if activo(a.Estado) {
			vivos[a.Proyecto]++
		}
		if a.Tipo != "sesion" {
			estados[a.Proyecto] = append(estados[a.Proyecto], a.ID+"="+a.Estado)
		}
	}
	consultas := s.consultasDeHoy(hoy)
	decisiones := map[string]int{}
	for _, d := range s.decisiones("").Decisiones {
		if d.Estado == claude.DecPendiente && !d.Vieja {
			decisiones[d.Proyecto]++
		}
	}

	out := make([]proyectoPortafolio, 0, len(s.listaProyectos()))
	for _, p := range s.listaProyectos() {
		fi, err := os.Stat(p.Raiz)
		existe := err == nil && fi.IsDir()
		f := avance.Actual(p.Raiz)
		sigla := p.Sigla
		if sigla == "" {
			sigla = strings.ToUpper(string([]rune(p.Nombre)[:min(2, len([]rune(p.Nombre)))]))
		}
		pp := proyectoPortafolio{Nombre: p.Nombre, Sigla: sigla, Ruta: p.Raiz, Nivel: p.Nivel, Existe: existe,
			Contrato: contrato(p.Raiz, p.Registrado, p.Nivel, f), Vivos: vivos[p.Nombre], Consultas: consultas[p.Nombre], Decisiones: decisiones[p.Nombre]}
		if f != nil {
			pp.Fase, pp.Avance = f.Fase, avanceDe(f)
		} else if existe {
			pp.Fase = faseActual(p.Raiz)
		}
		if existe {
			pp.Salud = s.saludDeHoy(p, ahora, estados[p.Nombre])
		}
		out = append(out, pp)
	}
	return out
}

func avanceDe(f *avance.Fase) *avancePortafolio {
	a := &avancePortafolio{Fase: f.Fase, Ruta: f.Ruta, Ahora: f.Ahora, Calculado: f.Calculado, Horas: f.HorasTot, Hechas: f.HorasHech}
	for _, p := range f.Paquetes {
		if p.Horas <= 0 {
			continue // no pesa en el % (avance.calcular)
		}
		v := p.Horas * float64(p.Pct) / 100
		a.Avanzadas += v
		if p.Estado == avance.EnCurso {
			a.EnCurso += v
		}
	}
	return a
}

// contrato: los archivos del contrato de trazabilidad que pide el nivel del proyecto, cada uno con su presencia: cinco
// hasta el nivel 1 (el kit de nivel 1 no instala documentos de operación ni roles) y siete desde el 2. f es el avance
// ya leído.
func contrato(raiz string, registrado bool, nivel int, f *avance.Fase) []archivoContrato {
	hay := func(rel string, dir bool) bool {
		fi, err := os.Stat(filepath.Join(raiz, filepath.FromSlash(rel)))
		return err == nil && fi.IsDir() == dir
	}
	archivo := func(id, nombre, rel string) archivoContrato {
		return archivoContrato{ID: id, Nombre: nombre, Ruta: rel, Esta: hay(rel, false)}
	}

	estado := archivo("estado", "docs/trazabilidad/ESTADO-para-retomar.md", "docs/trazabilidad/ESTADO-para-retomar.md")
	if !estado.Esta && hay("docs/trazabilidad/00-ESTADO-para-retomar.md", false) {
		estado.Ruta, estado.Esta = "docs/trazabilidad/00-ESTADO-para-retomar.md", true
	}
	av := archivoContrato{ID: "avance", Nombre: "docs/trazabilidad/F<n>/AVANCE.md", Ruta: "docs/trazabilidad/F<n>/AVANCE.md"}
	if f != nil {
		av.Ruta, av.Esta = f.Ruta, true
	} else if fase := faseActual(raiz); fase != "" {
		av.Ruta = "docs/trazabilidad/" + fase + "/AVANCE.md"
	}
	// El documento de agentes: el de siempre o el que cita AGENTS.md (la skill lo instala en docs/operacion-neutro/
	// cuando el proyecto ya tenía su docs/operacion).
	op := archivo("operacion", "docs/operacion/AGENTES.md", "docs/operacion/AGENTES.md")
	if r := estadisticas.RutaAgentes(raiz); !op.Esta && r != "" && hay(r, false) {
		op.Ruta, op.Esta = r, true
	}
	out := []archivoContrato{archivo("agents", "AGENTS.md", "AGENTS.md"), archivo("mcp", ".mcp.json", ".mcp.json"), estado, av}
	if nivel >= 2 {
		out = append(out, op, archivoContrato{ID: "subagentes", Nombre: ".claude/agents/", Ruta: ".claude/agents/", Esta: hay(".claude/agents", true)})
	}
	return append(out, archivoContrato{ID: "registro", Nombre: "registrado en proyectos.json", Ruta: "proyectos.json", Esta: registrado})
}

// consultasDeHoy: por proyecto, los pasos «Atalaya» (mcp__atalaya__… en Claude Code; en Codex, la misma herramienta en
// su traza) desde la medianoche.
// esConsultaAtalaya: un paso que es una llamada a Atalaya (también con la clave de antes, claude.HerramientaAtalaya).
func esConsultaAtalaya(herr string) bool {
	_, ok := claude.HerramientaAtalaya(herr)
	return ok
}

func (s *Servidor) consultasDeHoy(hoy time.Time) map[string]int {
	out := map[string]int{}
	hall, _ := s.Claude.BuscarPasos(claude.FiltroPasos{Tipo: "Atalaya", Desde: hoy})
	for _, h := range hall {
		if esConsultaAtalaya(h.Paso.Herramienta) {
			out[h.Agente.Proyecto]++
		}
	}
	if s.CodexVivo == nil {
		return out
	}
	for _, c := range s.CodexVivo.Llamadas() {
		if c.Ultimo.Before(hoy) {
			continue
		}
		for _, p := range s.listaProyectos() {
			if !esDelProyecto(c.Meta.Raiz, p.Nombre, p.Raiz) {
				continue
			}
			for _, paso := range c.Pasos {
				if esConsultaAtalaya(paso.Herramienta) && !paso.Hora.Before(hoy) {
					out[p.Nombre]++
				}
			}
			break
		}
	}
	return out
}

// La salud de hoy guardada por servidor y proyecto, con la clave de lo que la cambia.
type saludGuardada struct {
	clave string
	hora  time.Time
	salud saludPortafolio
}

var (
	muSaludes sync.Mutex
	saludes   = map[*Servidor]map[string]saludGuardada{}
)

const vigenciaSalud = 30 * time.Second

// saludDeHoy: cuántos subagentes y llamadas a Codex de hoy van en regla (sin faltas ni medias), de informeProceso.
// Se vuelve a calcular si cambian las temporales, las bitácoras o el estado de los agentes del proyecto, si cambia
// el día o a los 30 s.
func (s *Servidor) saludDeHoy(p Proyecto, ahora time.Time, estados []string) saludPortafolio {
	sort.Strings(estados)
	var b strings.Builder
	b.WriteString(ahora.Format("2006-01-02") + "|")
	s.mu.RLock()
	b.WriteString(s.firmaTmp + "|")
	var bitacoras []string
	pre := strings.ToLower(filepath.Clean(p.Raiz) + string(filepath.Separator))
	for k, v := range s.firma {
		if strings.HasPrefix(strings.ToLower(k), pre) {
			bitacoras = append(bitacoras, k+"="+v)
		}
	}
	s.mu.RUnlock()
	sort.Strings(bitacoras)
	b.WriteString(strings.Join(bitacoras, ";") + "|" + strings.Join(estados, ";"))
	clave := b.String()

	muSaludes.Lock()
	g, ok := saludes[s][p.Nombre]
	muSaludes.Unlock()
	if ok && g.clave == clave && ahora.Sub(g.hora) < vigenciaSalud && !ahora.Before(g.hora) {
		return g.salud
	}

	inf := s.informeProceso(p.Nombre, PeriodoHoy)
	sal := saludPortafolio{Total: len(inf.Filas)}
	for _, f := range inf.Filas {
		if f.Faltas == 0 && f.Medias == 0 {
			sal.EnRegla++
		}
	}
	muSaludes.Lock()
	if saludes[s] == nil {
		saludes[s] = map[string]saludGuardada{}
	}
	saludes[s][p.Nombre] = saludGuardada{clave: clave, hora: ahora, salud: sal}
	muSaludes.Unlock()
	return sal
}
