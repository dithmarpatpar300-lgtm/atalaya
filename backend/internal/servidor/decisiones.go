package servidor

import (
	"net/http"
	"sort"
	"time"

	"atalaya/internal/claude"
)

// Las decisiones que las sesiones te piden: solo lectura. Atalaya muestra la pregunta, sus
// opciones y la recomendada; respondes en la app. Una pendiente de hace más de un día se separa como «sin respuesta»:
// la sesión probablemente siguió por otro lado.

type decisionVista struct {
	claude.Decision
	EstadoSesion string `json:"estado_sesion"`
	Vieja        bool   `json:"vieja,omitempty"`
}

type respuestaDecisiones struct {
	Pendientes int             `json:"pendientes"`
	Decisiones []decisionVista `json:"decisiones"`
}

const (
	maxDecisiones   = 200
	diasDecisiones  = 14
	vejezDePregunta = 24 * time.Hour
)

// juntarDecisiones une las de los transcritos con las de `preguntar`, sin ids repetidos (se queda la primera): el panel
// pagina y dibuja por el id, y uno repetido duplicaba filas.
func juntarDecisiones(fuentes ...[]claude.Decision) []claude.Decision {
	vistos := map[string]bool{}
	var out []claude.Decision
	for _, f := range fuentes {
		for _, d := range f {
			if vistos[d.ID] {
				continue
			}
			vistos[d.ID] = true
			out = append(out, d)
		}
	}
	return out
}

func (s *Servidor) verDecisiones(w http.ResponseWriter, r *http.Request) {
	escribir(w, s.decisiones(r.URL.Query().Get("proyecto")))
}

func (s *Servidor) decisiones(proyecto string) respuestaDecisiones {
	ahora := s.ahora()
	// El estado y el nombre de cada sesión, los mismos de la lista (estadoClaude, nombreVista).
	estado, nombre := map[string]string{}, map[string]string{}
	equipo := s.equipoTrabajando()
	for _, a := range s.Claude.Agentes() {
		if a.Tipo == "sesion" {
			estado[a.ID] = s.estadoClaude(a, equipo)
			nombre[a.ID] = s.nombreDe(a)
		}
	}
	out := respuestaDecisiones{Decisiones: []decisionVista{}}
	// Las de los transcritos y las que llegaron por `preguntar`, juntas y de la más nueva a la más vieja.
	todas := juntarDecisiones(s.Claude.Decisiones(proyecto), s.decisionesDelCentro(proyecto, ahora.AddDate(0, 0, -diasDecisiones)))
	sort.SliceStable(todas, func(i, j int) bool { return todas[i].Hora.After(todas[j].Hora) })
	for _, d := range todas {
		if ahora.Sub(d.Hora) > diasDecisiones*24*time.Hour || len(out.Decisiones) >= maxDecisiones {
			break // vienen de la más nueva a la más vieja
		}
		if n := nombre[d.Agente]; n != "" {
			d.Sesion = n
		}
		v := decisionVista{Decision: d, EstadoSesion: estado[d.Agente]}
		if d.Estado == claude.DecPendiente {
			v.Vieja = ahora.Sub(d.Hora) > vejezDePregunta
			if !v.Vieja {
				out.Pendientes++
			}
		}
		out.Decisiones = append(out.Decisiones, v)
	}
	return out
}
