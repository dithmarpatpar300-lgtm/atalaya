package servidor

import (
	"net/http"
	"sort"
	"time"

	"atalaya/internal/avance"
	"atalaya/internal/claude"
)

// La vista de la sesión principal: en vez del encargo y la bitácora, que una sesión no tiene,
// sus pedidos y lo que siguió a cada uno (claude.(*Indice).Sesion), el equipo que lanzó y el avance de la fase de su
// proyecto, que es la bitácora del coordinador.

// Miembro del equipo de una sesión: un subagente o una llamada a Codex que cuelga de ella.
type Miembro struct {
	ID          string    `json:"id"`
	Nombre      string    `json:"nombre"`
	Tarea       string    `json:"tarea"`
	Proveedor   string    `json:"proveedor"`
	Clase       string    `json:"clase,omitempty"`
	Modelo      string    `json:"modelo"`
	ModeloCorto string    `json:"modelo_corto"`
	Effort      string    `json:"effort,omitempty"`
	Estado      string    `json:"estado"`
	Inicio      time.Time `json:"inicio"`
	// Sesion: otra sesión principal que esta lanzó con start_session; tiene su propio equipo.
	Sesion bool `json:"sesion,omitempty"`
}

type respuestaSesion struct {
	claude.ResumenSesion
	Equipo     []Miembro         `json:"equipo"`
	Avance     *avance.Fase      `json:"avance"`
	Decisiones []claude.Decision `json:"decisiones"`
}

func (s *Servidor) verSesion(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	res, ok := s.Claude.Sesion(id)
	if !ok {
		fallo(w, http.StatusNotFound, "no existe esa sesión")
		return
	}
	out := respuestaSesion{ResumenSesion: res, Equipo: []Miembro{}, Decisiones: []claude.Decision{}}
	proyecto := ""
	for _, a := range s.todos() {
		if a.ID == id {
			proyecto = a.Proyecto
		}
		// Su equipo: lo que cuelga de ella y las sesiones que lanzó (estas, con su título: su nombre es el del proyecto).
		if a.Padre != id && a.LanzadaPor != id {
			continue
		}
		m := Miembro{ID: a.ID, Nombre: a.Nombre, Tarea: a.Tarea, Proveedor: a.Proveedor, Clase: a.Clase,
			Modelo: a.Modelo, ModeloCorto: a.ModeloCorto, Effort: a.Effort, Estado: a.Estado, Inicio: a.Inicio, Sesion: a.LanzadaPor == id}
		if m.Sesion && a.Titulo != "" {
			m.Nombre = a.Titulo
		}
		out.Equipo = append(out.Equipo, m)
	}
	sort.Slice(out.Equipo, func(i, j int) bool { return out.Equipo[i].Inicio.Before(out.Equipo[j].Inicio) })
	out.Avance = avance.Actual(s.raizDe(proyecto))
	// Las decisiones que pidió esta sesión, todas (sin el tope de 200 y 14 días de la bandeja: se filtra antes), de la
	// más nueva a la más vieja.
	for _, d := range juntarDecisiones(s.Claude.Decisiones(proyecto), s.decisionesDelCentro(proyecto, time.Time{})) {
		if d.Agente == id {
			out.Decisiones = append(out.Decisiones, d)
		}
	}
	sort.SliceStable(out.Decisiones, func(i, j int) bool { return out.Decisiones[i].Hora.After(out.Decisiones[j].Hora) })
	escribir(w, out)
}
