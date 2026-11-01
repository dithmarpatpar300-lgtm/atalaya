package servidor

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"atalaya/internal/claude"
)

// `preguntar` del MCP y la puerta del panel para responder (preguntas.go).

var herramientaPreguntar = herramienta{Name: "preguntar", Title: "Preguntarle al dueño del proyecto en Atalaya",
	Annotations: map[string]any{"readOnlyHint": false, "openWorldHint": false},
	Description: "Hace una o varias preguntas al dueño del proyecto en el panel de Atalaya (sección Decisiones) y espera su respuesta. " +
		"Úsala para decisiones de varios pasos, o que pueden necesitar texto o una captura: cada pregunta trae siempre " +
		"«Escribir otra respuesta», y el dueño del proyecto puede adjuntar imágenes, que llegan como imagen. Una opción puede abrir " +
		"preguntas posteriores (`abre`: sus ids). Espera hasta 4 minutos; si no hay respuesta, devuelve el id: vuelve a " +
		"llamar con {\"id\": …} para seguir esperando, o sigue con otra cosa y llama después. La respuesta queda guardada.",
	InputSchema: esquema(map[string]any{
		"id":       map[string]any{"type": "string", "description": "Para seguir esperando una pregunta ya hecha. Con id, lo demás se ignora."},
		"proyecto": propProyecto,
		"titulo":   map[string]any{"type": "string", "description": "Una línea con lo que se decide."},
		"contexto": map[string]any{"type": "string", "description": "Lo que el dueño del proyecto necesita saber para decidir, en pocas líneas."},
		"preguntas": map[string]any{"type": "array", "maxItems": maxPreguntas, "items": map[string]any{
			"type": "object", "required": []string{"texto"}, "additionalProperties": false,
			"properties": map[string]any{
				"id":         map[string]any{"type": "string", "description": "Corto y único («medir», «d2»); sin él, p1, p2…"},
				"texto":      map[string]any{"type": "string"},
				"encabezado": map[string]any{"type": "string", "description": "Dos o tres palabras para el paso."},
				"multiple":   map[string]any{"type": "boolean", "description": "Si se puede elegir más de una opción."},
				"opciones": map[string]any{"type": "array", "maxItems": maxOpciones, "items": map[string]any{
					"type": "object", "required": []string{"etiqueta"}, "additionalProperties": false,
					"properties": map[string]any{
						"etiqueta":    map[string]any{"type": "string"},
						"nota":        map[string]any{"type": "string", "description": "Qué pasa si se elige."},
						"recomendada": map[string]any{"type": "boolean"},
						"abre":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Ids de preguntas posteriores que esta opción abre."},
					}}},
			}}},
	})}

// esPreguntar: un pedido suelo de tools/call a `preguntar`, que se atiende con su propio flujo (no cabe en un lote).
func esPreguntar(p rpcPedido) bool {
	if p.Method != "tools/call" {
		return false
	}
	var q struct {
		Name string `json:"name"`
	}
	return json.Unmarshal(p.Params, &q) == nil && q.Name == "preguntar"
}

// mcpPreguntar: guarda la pregunta (o retoma una por su id) y espera la respuesta. Con SSE, el primer byte sale al
// instante (Claude Code corta a los 60 s sin él) y cada 20 s va un aviso de progreso, o un comentario si la sesión no
// mandó progressToken. Sin SSE, espera menos de 60 s.
func (s *Servidor) mcpPreguntar(w http.ResponseWriter, r *http.Request, p rpcPedido, cliente, delProyecto string) {
	t0 := time.Now()
	var q struct {
		Arguments json.RawMessage `json:"arguments"`
		Meta      struct {
			ProgressToken any `json:"progressToken"`
		} `json:"_meta"`
	}
	_ = json.Unmarshal(p.Params, &q)
	var a argsPreguntar
	fallar := func(texto string) {
		escribirRPC(w, rpcRespuesta{JSONRPC: "2.0", ID: p.ID, Result: map[string]any{"content": []map[string]any{{"type": "text", "text": texto}}, "isError": true}})
		s.anotarConsulta(Consulta{Hora: t0, Paso: -1, Cliente: cliente, Herramienta: "preguntar", Argumentos: recortarTexto(string(q.Arguments), 300), Ms: time.Since(t0).Milliseconds(), Error: true, Respuesta: texto})
	}
	if len(q.Arguments) > 0 && json.Unmarshal(q.Arguments, &a) != nil {
		fallar("Los argumentos no son un objeto JSON válido.")
		return
	}
	var preg *PreguntaCentro
	if a.ID != "" {
		if preg = s.preguntas.buscar(a.ID); preg == nil {
			fallar("No existe la pregunta «" + a.ID + "»: Atalaya se reinició o ya la olvidó. Hazla de nuevo.")
			return
		}
	} else {
		var err error
		if preg, err = armarPregunta(a); err != nil {
			fallar("La pregunta no se pudo armar: " + err.Error() + ".")
			return
		}
		preg.Proyecto = s.proyectoOMayor(firstNonEmpty(a.Proyecto, delProyecto))
		preg.Cliente, preg.Creada = cliente, s.ahora()
		s.preguntas.guardar(preg, preg.Creada)
		s.difusion.enviar(Evento{Tipo: "decisiones"})
	}

	fl, _ := w.(http.Flusher)
	flujo := fl != nil && strings.Contains(r.Header.Get("Accept"), "text/event-stream")
	espera := esperaMCP
	if !flujo {
		espera = 50 * time.Second
	} else {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, ": esperando al dueño del proyecto en Atalaya\n\n")
		fl.Flush()
	}
	latido := time.NewTicker(latidoMCP)
	defer latido.Stop()
	limite := time.NewTimer(espera)
	defer limite.Stop()
	respondida := false
	for esperar, n := true, 0; esperar; {
		select {
		case <-preg.listo:
			respondida, esperar = true, false
		case <-limite.C:
			esperar = false
		case <-r.Context().Done():
			return // la sesión cortó; la pregunta sigue en el panel y se retoma con su id
		case <-latido.C:
			n++
			if !flujo {
				continue
			}
			if q.Meta.ProgressToken != nil {
				aviso, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "notifications/progress", "params": map[string]any{
					"progressToken": q.Meta.ProgressToken, "progress": n, "message": "Esperando la respuesta del dueño del proyecto en Atalaya"}})
				fmt.Fprintf(w, "event: message\ndata: %s\n\n", aviso)
			} else {
				fmt.Fprint(w, ": sigue esperando\n\n")
			}
			fl.Flush()
		}
	}

	var contenido []map[string]any
	if respondida {
		contenido = s.preguntas.entregar(preg)
		s.difusion.enviar(Evento{Tipo: "decisiones"})
	} else {
		contenido = []map[string]any{{"type": "text", "text": fmt.Sprintf("Atalaya · el dueño del proyecto todavía no respondió «%s» (pregunta %s). "+
			"La ve en Atalaya, sección Decisiones. Vuelve a llamar a preguntar con {\"id\": \"%s\"} para seguir esperando, o sigue con otra cosa y llama después: la respuesta queda guardada.",
			preg.Titulo, preg.ID, preg.ID)}}
	}
	resp := rpcRespuesta{JSONRPC: "2.0", ID: p.ID, Result: map[string]any{"content": contenido, "isError": false}}
	if flujo {
		b, _ := json.Marshal(resp)
		fmt.Fprintf(w, "event: message\ndata: %s\n\n", b)
		fl.Flush()
	} else {
		escribirRPC(w, resp)
	}
	texto, _ := contenido[0]["text"].(string)
	s.anotarConsulta(Consulta{Hora: t0, Paso: -1, Cliente: cliente, Herramienta: "preguntar", Argumentos: recortarTexto(string(q.Arguments), 300),
		Bytes: len(texto), Ms: time.Since(t0).Milliseconds(), Respuesta: recortarTexto(texto, 1200)})
}

func firstNonEmpty(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

// El panel: la lista, una pregunta y la respuesta.

func (s *Servidor) verPreguntas(w http.ResponseWriter, _ *http.Request) {
	escribir(w, s.preguntas.todas())
}

func (s *Servidor) verPregunta(w http.ResponseWriter, r *http.Request) {
	for _, p := range s.preguntas.todas() {
		if p.ID == r.PathValue("id") {
			escribir(w, p)
			return
		}
	}
	fallo(w, http.StatusNotFound, "no existe esa pregunta")
}

// responderPregunta: la única escritura del panel. Solo desde este equipo (soloLocal y origenLocal) y con la cabecera
// del panel, que una página ajena no puede mandar sin un permiso CORS que Atalaya no da.
func (s *Servidor) responderPregunta(w http.ResponseWriter, r *http.Request) {
	if !origenLocal(r) || r.Header.Get(cabPanel) != "1" {
		fallo(w, http.StatusForbidden, "solo el panel de Atalaya responde preguntas")
		return
	}
	var c cuerpoRespuesta
	if err := json.NewDecoder(io.LimitReader(r.Body, maxCuerpoRespuest)).Decode(&c); err != nil {
		fallo(w, http.StatusBadRequest, "la respuesta no es un JSON válido o pasa de 26 MB")
		return
	}
	p, err := s.preguntas.responder(r.PathValue("id"), c, s.ahora())
	if err != nil {
		fallo(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	s.difusion.enviar(Evento{Tipo: "decisiones"})
	escribir(w, map[string]any{"id": p.ID, "estado": p.Estado})
}

// decisionesDelCentro: las preguntas por el MCP, como decisiones de la bandeja; la sesión que preguntó se busca en
// los transcritos (el paso «preguntar» con el mismo título), si ya lo escribió.
func (s *Servidor) decisionesDelCentro(proyecto string, desde time.Time) []claude.Decision {
	var llamadas []claude.LlamadaCentro
	var out []claude.Decision
	for _, p := range s.preguntas.todas() {
		if proyecto != "" && p.Proyecto != proyecto {
			continue
		}
		d := claude.Decision{ID: prefijoDec + p.ID, Proyecto: p.Proyecto, Sesion: "Pregunta por Atalaya", Tipo: "atalaya", Hora: p.Creada,
			Paso: -1, Contexto: p.Contexto, Estado: claude.DecPendiente}
		if llamadas == nil {
			llamadas = s.Claude.ConsultasCentro(desde, MaxConsultas)
		}
		for _, l := range llamadas {
			if l.Herramienta == "preguntar" && strings.Contains(l.Argumentos, jsonTexto(p.Titulo)) && !l.Paso.Hora.Before(p.Creada.Add(-time.Minute)) {
				a := l.Agente
				nombre := s.nombreDe(a)
				d.Agente, d.Paso, d.Sesion = a.ID, l.Paso.N, nombre
				break
			}
		}
		if p.Respuesta != nil {
			d.Estado, d.Respondida = claude.DecRespondida, p.Respuesta.Hora
		}
		for _, q := range visiblesOTodas(p) {
			pq := claude.Pregunta{Texto: q.Texto, Encabezado: q.Encabezado, Multiple: q.Multiple}
			for _, o := range q.Opciones {
				pq.Opciones = append(pq.Opciones, claude.Opcion{Etiqueta: o.Etiqueta, Nota: o.Nota, Recomendada: o.Recomendada})
			}
			if p.Respuesta != nil {
				partes := append([]string(nil), p.Respuesta.Elecciones[q.ID]...)
				if t := p.Respuesta.Textos[q.ID]; t != "" {
					partes = append(partes, t)
				}
				pq.Respuesta = strings.Join(partes, ", ")
			}
			d.Preguntas = append(d.Preguntas, pq)
		}
		out = append(out, d)
	}
	return out
}

// visiblesOTodas: respondida, las que se vieron; esperando, todas (la bandeja muestra lo que se preguntará).
func visiblesOTodas(p PreguntaCentro) []preguntaP {
	if p.Respuesta != nil {
		return visibles(p.Preguntas, p.Respuesta.Elecciones)
	}
	return p.Preguntas
}

// jsonTexto: cómo queda un texto dentro de un JSON (para buscarlo en los argumentos guardados).
func jsonTexto(s string) string {
	b, _ := json.Marshal(s)
	return strings.Trim(string(b), `"`)
}

// `preguntar` va al final de la lista de herramientas.
func init() { herramientas = append(herramientas, herramientaPreguntar) }
