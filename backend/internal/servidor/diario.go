package servidor

import (
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"atalaya/internal/alertas"
	"atalaya/internal/diario"
)

// El diario: cada 5 minutos y al arrancar, lo que Atalaya tiene en memoria de cada día se funde con
// su archivo (internal/diario). Las filas de los agentes son las mismas que usa Estadísticas (filasAgentes): el diario
// guarda exactamente lo que después se mide.

// cadaDiario: cada cuántas vueltas se escribe (150 vueltas de 2 s, 5 minutos).
const cadaDiario = 150

// proyectoDeRaiz: el proyecto del registro con esa raíz; si no está registrado, el nombre de su carpeta.
func (s *Servidor) proyectoDeRaiz(raiz string) string {
	for _, p := range s.listaProyectos() {
		if strings.EqualFold(filepath.Clean(p.Raiz), filepath.Clean(raiz)) {
			return p.Nombre
		}
	}
	return filepath.Base(raiz)
}

// filasAgentes: cada sesión, subagente y llamada a Codex en vivo que Atalaya tiene en memoria, con sus cifras. Las
// llamadas que solo están en REGISTRO.md no van: ese archivo ya las guarda para siempre.
func (s *Servidor) filasAgentes(ahora time.Time) []diario.Agente {
	var out []diario.Agente
	raices := map[string]string{}
	raizDe := func(p string) string {
		if r, ok := raices[p]; ok {
			return r
		}
		raices[p] = s.raizDe(p)
		return raices[p]
	}
	for _, a := range s.Claude.Agentes() {
		raiz := raizDe(a.Proyecto)
		fam := familia(a.Modelo)
		if fam == "" {
			fam = familia(a.ModeloAsignado)
		}
		ef := a.Effort
		if ef == "" {
			ef = effortAsignado(a.ModeloAsignado)
		}
		v := s.vistaClaude(a, false)
		out = append(out, diario.Agente{ID: a.ID, Nombre: v.Nombre, Proyecto: a.Proyecto, Proveedor: "Claude",
			Tipo: a.Tipo, Clase: a.Clase, Modelo: fam, Effort: ef, Fase: faseDeAgente(raiz, a.Inicio, a.Bitacora, a.Informe, a.Encargo),
			Inicio: a.Inicio, Fin: a.Ultimo, Activo: activo(EstadoClaude(a, ahora)), Minutos: a.Ultimo.Sub(a.Inicio).Minutes(),
			Salida: a.Tokens.Salida, Nuevos: a.Tokens.Entrada + a.Tokens.CacheEscrita, Leida: a.Tokens.CacheLeida,
			Presupuesto: minutosDe(a.Presupuesto), Padre: a.Padre, Corto: v.ModeloCorto})
	}
	if s.CodexVivo != nil {
		ls := s.lanzamientosRecientes(ahora)
		for _, c := range s.CodexVivo.Llamadas() {
			proyecto := s.proyectoDeRaiz(c.Meta.Raiz)
			min := c.Ultimo.Sub(c.Inicio).Minutes()
			if c.Meta.Segundos > 0 {
				min = float64(c.Meta.Segundos) / 60
			}
			// Una sesión interactiva cuenta como sesión (Estadísticas mide agentes con encargo, no sesiones).
			tipo, vivo := "codex", !c.Terminada
			if c.Interactiva {
				tipo, vivo = "sesion", c.Activa(ahora) || c.EnEspera(ahora)
			}
			out = append(out, diario.Agente{ID: c.ID, Nombre: "Codex · " + c.Meta.Tema, Proyecto: proyecto, Proveedor: "Codex", Tipo: tipo,
				Clase: c.Meta.Clase, Modelo: c.Meta.Modelo, Effort: c.Meta.Effort, Fase: faseDeAgente(raizDe(proyecto), c.Inicio, rutaFase(c.Meta.Fase)),
				Inicio: c.Inicio, Fin: c.Ultimo, Activo: vivo, Minutos: min, Salida: c.Tokens.Salida, Nuevos: c.Tokens.Entrada,
				Leida: c.Tokens.CacheLeida, Presupuesto: float64(c.Meta.MinutosMax), Padre: padreCodexVivo(ls, c),
				Corto: modeloCorto(c.Meta.Modelo, c.Meta.Effort), Registro: claveRegistro(c)})
		}
	}
	return out
}

// maxDiasDiario: cuántos días hacia atrás se leen del diario para Estadísticas (con «fase») y la línea de tiempo.
const maxDiasDiario = 120

// filasConDiario: las de la memoria y, de los días del diario desde `desde` (AAAA-MM-DD; vacío, todos), las que la
// memoria ya no tiene (el transcrito o la traza se borraron). Esas van como terminadas: si siguieran vivas, estarían en
// la memoria. Devuelve también cuántas vinieron del diario.
func (s *Servidor) filasConDiario(ahora time.Time, desde string) ([]diario.Agente, int) {
	filas := s.filasAgentes(ahora)
	en := map[string]bool{}
	for _, f := range filas {
		en[f.ID] = true
	}
	solo := s.soloDelDiario(desde, en)
	return append(filas, solo...), len(solo)
}

// soloDelDiario: las filas del diario desde `desde` (vacío, todas) cuyo id no está en `en`.
func (s *Servidor) soloDelDiario(desde string, en map[string]bool) []diario.Agente {
	if s.Diario == "" {
		return nil
	}
	var out []diario.Agente
	visto := map[string]bool{}
	for i, fecha := range diario.Fechas(s.Diario) {
		if i >= maxDiasDiario || (desde != "" && fecha < desde) {
			break // de la más nueva a la más vieja
		}
		d, err := diario.Leer(s.Diario, fecha)
		if err != nil {
			continue
		}
		for _, a := range d.Agentes {
			if !en[a.ID] && !visto[a.ID] {
				visto[a.ID], a.Activo = true, false
				out = append(out, a)
			}
		}
	}
	return out
}

// diaDeHoy y los demás: lo de la memoria repartido por fecha local (la del inicio de cada agente, la de cada pregunta
// y la de cada alerta).
func (s *Servidor) diasEnMemoria(ahora time.Time) map[string]*diario.Dia {
	dias := map[string]*diario.Dia{}
	dia := func(t time.Time) *diario.Dia {
		f := diario.Fecha(t)
		if dias[f] == nil {
			dias[f] = &diario.Dia{Fecha: f}
		}
		return dias[f]
	}
	for _, a := range s.filasAgentes(ahora) {
		if !a.Inicio.IsZero() {
			d := dia(a.Inicio)
			d.Agentes = append(d.Agentes, a)
		}
	}
	for _, d := range append(s.Claude.Decisiones(""), s.decisionesDelCentro("", time.Time{})...) {
		if !d.Hora.IsZero() {
			x := dia(d.Hora)
			x.Decisiones = append(x.Decisiones, diario.Decision{ID: d.ID, Proyecto: d.Proyecto, Sesion: d.Sesion, Hora: d.Hora, Estado: d.Estado})
		}
	}
	s.mu.RLock()
	for _, a := range s.alertas {
		if a.Nivel != alertas.Error {
			continue
		}
		h := a.Hora
		if h.IsZero() || h.After(ahora) {
			h = ahora
		}
		x := dia(h)
		x.Alertas = append(x.Alertas, diario.Alerta{ID: a.ID, Proyecto: a.Proyecto, Regla: a.Regla, Agente: a.Agente, Titulo: a.Titulo, Hora: h})
	}
	s.mu.RUnlock()
	return dias
}

// escribirDiario funde cada día de la memoria con su archivo. Va aparte de la vuelta y nunca dos a la vez.
func (s *Servidor) escribirDiario() {
	if s.Diario == "" || !s.diarioMu.TryLock() {
		return
	}
	go func() {
		defer s.diarioMu.Unlock()
		ahora := s.ahora()
		// Los días que el dueño eligió guardar (Configuración, «Privacidad»): lo más viejo no se vuelve a escribir desde
		// la memoria y, una vez al día, se borra del disco.
		dias := s.preferencias().DiarioDias
		limite := ""
		if dias > 0 {
			limite = diario.Fecha(ahora.AddDate(0, 0, -dias))
			if hoy := diario.Fecha(ahora); s.diarioRecortado != hoy {
				if n, err := diario.Recortar(s.Diario, dias, ahora); err != nil {
					log.Printf("diario: recortar a %d días: %v", dias, err)
				} else {
					s.diarioRecortado = hoy
					if n > 0 {
						log.Printf("diario: %d días de más de %d días borrados", n, dias)
					}
				}
			}
		}
		escritos := 0
		for _, d := range s.diasEnMemoria(ahora) {
			if limite != "" && d.Fecha < limite {
				continue
			}
			ok, err := diario.Guardar(s.Diario, *d)
			if err != nil {
				log.Printf("diario %s: %v", d.Fecha, err)
			}
			if ok {
				escritos++
			}
		}
		if escritos > 0 {
			s.difusion.enviar(Evento{Tipo: "diario"})
		}
	}()
}

type resumenDia struct {
	Fecha      string `json:"fecha"`
	Agentes    int    `json:"agentes"`
	Decisiones int    `json:"decisiones"`
	Alertas    int    `json:"alertas"`
}

// verDiario: las fechas con diario y cuánto guarda cada una; con ?dia=AAAA-MM-DD, el día entero.
func (s *Servidor) verDiario(w http.ResponseWriter, r *http.Request) {
	if s.Diario == "" {
		fallo(w, http.StatusNotFound, "el diario está apagado")
		return
	}
	if f := r.URL.Query().Get("dia"); f != "" {
		d, err := diario.Leer(s.Diario, f)
		if err != nil {
			fallo(w, http.StatusBadRequest, err.Error())
			return
		}
		escribir(w, d)
		return
	}
	out := []resumenDia{}
	for _, f := range diario.Fechas(s.Diario) {
		if d, err := diario.Leer(s.Diario, f); err == nil {
			out = append(out, resumenDia{Fecha: f, Agentes: len(d.Agentes), Decisiones: len(d.Decisiones), Alertas: len(d.Alertas)})
		}
	}
	escribir(w, out)
}
