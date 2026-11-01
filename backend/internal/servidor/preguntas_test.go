package servidor

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Un PNG de 1×1 píxel.
var png1x1, _ = base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==")

const preguntaDePrueba = `{"titulo":"Antes del commit de T1b","contexto":"Medidor de memoria entregado.","preguntas":[
 {"id":"medir","texto":"¿Mido ahora?","encabezado":"Medición","opciones":[{"etiqueta":"Ahora (Recomendado)"},{"etiqueta":"Después"}]},
 {"id":"d2","texto":"¿Abro D2?","opciones":[{"etiqueta":"Sí, en paralelo","abre":["effort"]},{"etiqueta":"No"}]},
 {"id":"effort","texto":"¿Effort?","opciones":[{"etiqueta":"Medio"},{"etiqueta":"Alto"}]}]}`

func TestArmarPregunta(t *testing.T) {
	var a argsPreguntar
	if err := json.Unmarshal([]byte(preguntaDePrueba), &a); err != nil {
		t.Fatal(err)
	}
	p, err := armarPregunta(a)
	if err != nil {
		t.Fatal(err)
	}
	if d := p.Preguntas[2].Depende; d == nil || d.Pregunta != "d2" || d.Opcion != "Sí, en paralelo" {
		t.Errorf("«abre» se vuelve «depende»: %+v", p.Preguntas[2].Depende)
	}
	if !p.Preguntas[0].Opciones[0].Recomendada {
		t.Error("«(Recomendado)» en la etiqueta la marca recomendada")
	}
	if got := len(visibles(p.Preguntas, map[string][]string{"d2": {"No"}})); got != 2 {
		t.Errorf("con «No», effort no se pregunta: %d visibles", got)
	}
	// Una opción que abre una pregunta anterior (un ciclo) no se acepta.
	a.Preguntas[2].Opciones[0].Abre = []string{"medir"}
	if _, err := armarPregunta(a); err == nil {
		t.Error("abrir una pregunta anterior debe fallar")
	}
}

func TestValidarRespuesta(t *testing.T) {
	var a argsPreguntar
	_ = json.Unmarshal([]byte(preguntaDePrueba), &a)
	p, _ := armarPregunta(a)
	ahora := time.Now()
	casos := []struct {
		nombre string
		c      cuerpoRespuesta
		falla  string
	}{
		{"falta una", cuerpoRespuesta{Elecciones: map[string][]string{"medir": {"Ahora (Recomendado)"}}}, "falta responder"},
		{"opción ajena", cuerpoRespuesta{Elecciones: map[string][]string{"medir": {"Nunca"}, "d2": {"No"}}}, "no es una opción"},
		{"dos en una única", cuerpoRespuesta{Elecciones: map[string][]string{"medir": {"Ahora (Recomendado)", "Después"}, "d2": {"No"}}}, "una sola"},
		{"falta la abierta", cuerpoRespuesta{Elecciones: map[string][]string{"medir": {"Después"}, "d2": {"Sí, en paralelo"}}}, "falta responder «¿Effort?»"},
	}
	for _, c := range casos {
		if _, err := validarRespuesta(p, c.c, ahora); err == nil || !strings.Contains(err.Error(), c.falla) {
			t.Errorf("%s: %v", c.nombre, err)
		}
	}
	// Texto propio en vez de opción, y lo de una pregunta cerrada se descarta.
	r, err := validarRespuesta(p, cuerpoRespuesta{Elecciones: map[string][]string{"d2": {"No"}, "effort": {"Alto"}},
		Textos: map[string]string{"medir": " mañana, con la máquina libre "}}, ahora)
	if err != nil || r.Textos["medir"] != "mañana, con la máquina libre" || r.Elecciones["effort"] != nil {
		t.Errorf("texto propio: %+v, %v", r, err)
	}
	// Una «imagen» que no lo es.
	_, err = validarRespuesta(p, cuerpoRespuesta{Elecciones: map[string][]string{"medir": {"Después"}, "d2": {"No"}},
		Imagenes: []struct {
			Nombre string `json:"nombre"`
			Base64 string `json:"base64"`
		}{{Nombre: "x.png", Base64: base64.StdEncoding.EncodeToString([]byte("no soy una imagen"))}}}, ahora)
	if err == nil || !strings.Contains(err.Error(), "PNG") {
		t.Errorf("tipo de imagen: %v", err)
	}
}

// De punta a punta: la sesión pregunta por el MCP con SSE y espera; el panel responde con una imagen; la sesión recibe
// el texto y la imagen. Y sin la cabecera del panel, no se puede responder.
func TestPreguntarDePuntaAPunta(t *testing.T) {
	s := &Servidor{Proyectos: []Proyecto{{Nombre: "Tienda Demo"}}}
	h := s.Handler()
	pedido := `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"preguntar","arguments":` + preguntaDePrueba + `,"_meta":{"progressToken":"t1"}}}`

	salida := make(chan string, 1)
	go func() {
		r := httptest.NewRequest("POST", "http://127.0.0.1:7420/mcp?proyecto=Tienda%20Demo", strings.NewReader(pedido))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Accept", "application/json, text/event-stream")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		salida <- w.Body.String()
	}()

	var id string
	for i := 0; i < 200 && id == ""; i++ {
		if ps := s.preguntas.todas(); len(ps) == 1 {
			id = ps[0].ID
			if ps[0].Proyecto != "Tienda Demo" || ps[0].Estado != PregEsperando {
				t.Errorf("guardada: %+v", ps[0])
			}
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
	if id == "" {
		t.Fatal("la pregunta no llegó a Atalaya")
	}

	cuerpo := map[string]any{"elecciones": map[string][]string{"medir": {"Ahora (Recomendado)"}, "d2": {"Sí, en paralelo"}, "effort": {"Medio"}},
		"nota": "Corre el estrés antes", "imagenes": []map[string]string{{"nombre": "captura.png", "base64": base64.StdEncoding.EncodeToString(png1x1)}}}
	b, _ := json.Marshal(cuerpo)
	responder := func(conCabecera bool) int {
		r := httptest.NewRequest("POST", "http://127.0.0.1:7420/api/preguntas/"+id+"/responder", bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		if conCabecera {
			r.Header.Set(cabPanel, "1")
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Code
	}
	if c := responder(false); c != http.StatusForbidden {
		t.Errorf("sin la cabecera del panel: %d, quiere 403", c)
	}
	if c := responder(true); c != http.StatusOK {
		t.Fatalf("responder: %d", c)
	}

	var out string
	select {
	case out = <-salida:
	case <-time.After(5 * time.Second):
		t.Fatal("la sesión no recibió la respuesta")
	}
	if !strings.HasPrefix(out, ": esperando") {
		t.Errorf("el primer byte sale al instante: %.60q", out)
	}
	i := strings.LastIndex(out, "data: ")
	var resp struct {
		ID     int `json:"id"`
		Result struct {
			Content []struct {
				Type     string `json:"type"`
				Text     string `json:"text"`
				MimeType string `json:"mimeType"`
			} `json:"content"`
		} `json:"result"`
	}
	if i < 0 || json.Unmarshal([]byte(strings.TrimSpace(out[i+6:])), &resp) != nil {
		t.Fatalf("sin respuesta final en el flujo: %q", out)
	}
	if resp.ID != 7 || len(resp.Result.Content) != 2 || resp.Result.Content[1].Type != "image" || resp.Result.Content[1].MimeType != "image/png" {
		t.Fatalf("contenido: %+v", resp.Result.Content)
	}
	texto := resp.Result.Content[0].Text
	for _, quiere := range []string{"Medición: Ahora (Recomendado)", "¿Effort?: Medio", "Nota: Corre el estrés antes", "Adjuntó 1 imagen"} {
		if !strings.Contains(texto, quiere) {
			t.Errorf("falta %q en:\n%s", quiere, texto)
		}
	}
	if ps := s.preguntas.todas(); ps[0].Estado != PregEntregada || ps[0].Respuesta.Imagenes[0].datos != nil {
		t.Errorf("entregada y sin los bytes de la imagen: %+v", ps[0])
	}
	if c := responder(true); c != http.StatusUnprocessableEntity {
		t.Errorf("responder dos veces: %d", c)
	}
}
