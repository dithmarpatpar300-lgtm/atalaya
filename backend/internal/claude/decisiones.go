package claude

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	"atalaya/internal/secretos"
)

// Las decisiones que una sesión te pide: la pregunta con botones del final de un turno
// (mcp__ccd_turn__wrap_up con «ask») y las preguntas de varias opciones (AskUserQuestion). Se leen del transcrito
// como los pasos: Atalaya no responde nada, solo muestra qué espera y, una vez respondida, qué elegiste.

// Opcion de una pregunta; Recomendada es la primera de los botones o la que lleva «(Recomendado)».
type Opcion struct {
	Etiqueta    string `json:"etiqueta"`
	Nota        string `json:"nota,omitempty"`
	Recomendada bool   `json:"recomendada,omitempty"`
}

// Pregunta con sus opciones y, si ya se respondió, la respuesta (una etiqueta o lo que escribiste).
type Pregunta struct {
	Texto      string   `json:"texto"`
	Encabezado string   `json:"encabezado,omitempty"`
	Multiple   bool     `json:"multiple,omitempty"`
	Opciones   []Opcion `json:"opciones"`
	Respuesta  string   `json:"respuesta,omitempty"`
}

// agregar suma una opción salvo que su etiqueta ya esté: la etiqueta es la identidad de la opción (la clave en el
// panel y lo que se elige), así que dos iguales serían la misma.
func (q *Pregunta) agregar(o Opcion) {
	for _, v := range q.Opciones {
		if v.Etiqueta == o.Etiqueta {
			return
		}
	}
	q.Opciones = append(q.Opciones, o)
}

// Decision: una pregunta con botones o un grupo de preguntas de una sesión.
type Decision struct {
	ID        string     `json:"id"`     // agente~paso
	Agente    string     `json:"agente"` // la sesión
	Proyecto  string     `json:"proyecto"`
	Sesion    string     `json:"sesion"` // el nombre de la sesión
	Tipo      string     `json:"tipo"`   // botones | preguntas
	Hora      time.Time  `json:"hora"`
	Paso      int        `json:"paso"`
	Contexto  string     `json:"contexto,omitempty"` // el resumen que acompaña a los botones
	Preguntas []Pregunta `json:"preguntas"`
	// Estado: pendiente, respondida o reemplazada (otra pregunta llegó antes de que respondieras).
	Estado     string    `json:"estado"`
	Respondida time.Time `json:"respondida,omitzero"`
}

// Estados de una decisión.
const (
	DecPendiente   = "pendiente"
	DecRespondida  = "respondida"
	DecReemplazada = "reemplazada"
)

const herramientaBotones = "mcp__ccd_turn__wrap_up"

// anotarDecision mira si un tool_use es una pregunta para ti y la guarda. paso es el número del paso que la lleva.
func (a *archivo) anotarDecision(nombre string, input json.RawMessage, idUso string, hora time.Time, paso int) {
	switch {
	case strings.HasSuffix(nombre, "__wrap_up") && strings.HasPrefix(nombre, "mcp__ccd_turn"):
		var in struct {
			Recap   string `json:"recap"`
			Ask     string `json:"ask"`
			Actions []struct {
				Label  string `json:"label"`
				Action string `json:"action"`
			} `json:"actions"`
		}
		if json.Unmarshal(input, &in) != nil || strings.TrimSpace(in.Ask) == "" {
			return
		}
		q := Pregunta{Texto: tapar(in.Ask)}
		for i, ac := range in.Actions {
			et := ac.Label
			if et == "" {
				et = ac.Action
			}
			q.agregar(Opcion{Etiqueta: tapar(et), Recomendada: i == 0})
		}
		if a.botones != nil && a.botones.Estado == DecPendiente {
			a.botones.Estado = DecReemplazada
		}
		d := &Decision{Tipo: "botones", Hora: hora, Paso: paso, Contexto: tapar(in.Recap), Preguntas: []Pregunta{q}, Estado: DecPendiente}
		a.decisiones = append(a.decisiones, d)
		a.botones = d
	case nombre == "AskUserQuestion":
		var in struct {
			Questions []struct {
				Question    string `json:"question"`
				Header      string `json:"header"`
				MultiSelect bool   `json:"multiSelect"`
				Options     []struct {
					Label       string `json:"label"`
					Description string `json:"description"`
				} `json:"options"`
			} `json:"questions"`
		}
		if json.Unmarshal(input, &in) != nil || len(in.Questions) == 0 {
			return
		}
		d := &Decision{Tipo: "preguntas", Hora: hora, Paso: paso, Estado: DecPendiente}
		for _, iq := range in.Questions {
			q := Pregunta{Texto: tapar(iq.Question), Encabezado: iq.Header, Multiple: iq.MultiSelect}
			for _, o := range iq.Options {
				q.agregar(Opcion{Etiqueta: tapar(o.Label), Nota: tapar(o.Description), Recomendada: esRecomendada(o.Label)})
			}
			d.Preguntas = append(d.Preguntas, q)
		}
		a.decisiones = append(a.decisiones, d)
		if a.preguntas == nil {
			a.preguntas = map[string]*Decision{}
		}
		a.preguntas[idUso] = d
	}
}

func esRecomendada(etiqueta string) bool {
	e := strings.ToLower(etiqueta)
	return strings.Contains(e, "(recomendado)") || strings.Contains(e, "(recomendada)") || strings.Contains(e, "(recommended)")
}

// responderPreguntas: el resultado de un AskUserQuestion trae tus respuestas en toolUseResult.answers, por el texto de
// cada pregunta. Solo se decodifica esta línea, no todas.
func (a *archivo) responderPreguntas(idUso string, crudo []byte, hora time.Time, error bool) {
	d := a.preguntas[idUso]
	if d == nil {
		return
	}
	delete(a.preguntas, idUso)
	var r struct {
		ToolUseResult struct {
			Answers map[string]string `json:"answers"`
		} `json:"toolUseResult"`
	}
	_ = json.Unmarshal(crudo, &r)
	for i := range d.Preguntas {
		if v, ok := r.ToolUseResult.Answers[d.Preguntas[i].Texto]; ok {
			d.Preguntas[i].Respuesta = tapar(recortar(v, 600))
		}
	}
	d.Estado, d.Respondida = DecRespondida, hora
	if error && len(r.ToolUseResult.Answers) == 0 {
		d.Estado = DecReemplazada // la cerraste sin responder
	}
}

// responderBotones: tu mensaje siguiente responde a los botones (un clic manda la etiqueta; también vale lo que
// escribas en su lugar).
func (a *archivo) responderBotones(texto string, hora time.Time) {
	d := a.botones
	// Un aviso de compactación o el informe de un agente que vuelve no es tu respuesta.
	if d == nil || d.Estado != DecPendiente || esDeSistema(texto) {
		return
	}
	d.Preguntas[0].Respuesta = tapar(primeraLinea(texto, 300))
	d.Estado, d.Respondida = DecRespondida, hora
	a.botones = nil
}

func tapar(s string) string { return secretos.Tapar(strings.TrimSpace(s)) }

// Decisiones: las de todas las sesiones (de un proyecto, si se pide), de la más nueva a la más vieja.
func (ix *Indice) Decisiones(proyecto string) []Decision {
	ix.mu.RLock()
	var out []Decision
	for _, a := range ix.arch {
		ag := a.agente
		if len(a.decisiones) == 0 || (proyecto != "" && ag.Proyecto != proyecto) {
			continue
		}
		nombre := ag.Titulo
		if nombre == "" {
			nombre = ag.Proyecto
		}
		vistos := map[string]bool{}
		for i, d := range a.decisiones {
			c := *d
			c.Preguntas = append([]Pregunta(nil), d.Preguntas...)
			c.ID = ag.ID + "~" + strconv.Itoa(d.Paso)
			// Dos decisiones del mismo paso no comparten id: la segunda lleva su posición, que no cambia (la lista
			// solo crece). El panel pagina y dibuja por el id.
			if vistos[c.ID] {
				c.ID += "~" + strconv.Itoa(i)
			}
			vistos[c.ID] = true
			c.Agente, c.Proyecto, c.Sesion = ag.ID, ag.Proyecto, nombre
			out = append(out, c)
		}
	}
	ix.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Hora.After(out[j].Hora) })
	return out
}
