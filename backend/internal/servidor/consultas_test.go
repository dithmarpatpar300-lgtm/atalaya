package servidor

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/cuotas"
)

// Una sesión que consultó salud y buscar a Atalaya; el panel se reinició después de salud (no está en la memoria).
const transcritoConsultas = `{"type":"user","timestamp":"2026-10-31T17:00:00Z","cwd":"C:\\Proyectos\\Tienda Demo","message":{"role":"user","content":"Revisa T1."}}
{"type":"assistant","timestamp":"2026-10-31T17:20:35Z","message":{"id":"m1","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"mcp__atalaya__salud","input":{"proyecto":"Tienda Demo"}}],"usage":{"input_tokens":1,"output_tokens":10}}}
{"type":"user","timestamp":"2026-10-31T17:20:40Z","message":{"role":"user","content":[{"tool_use_id":"tu_1","type":"tool_result","content":[{"type":"text","text":"Ninguna regla rota ahora."}]}]}}
{"type":"assistant","timestamp":"2026-10-31T17:25:00Z","message":{"id":"m2","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_2","name":"mcp__atalaya__buscar","input":{"texto":"veredicto","horas":30}}],"usage":{"input_tokens":1,"output_tokens":10}}}
{"type":"user","timestamp":"2026-10-31T17:25:01Z","message":{"role":"user","content":[{"tool_use_id":"tu_2","type":"tool_result","content":[{"type":"text","text":"3 pasos coinciden"}]}]}}
`

func TestVerConsultas(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "C--Proyectos-Tienda-Demo")
	_ = os.MkdirAll(dir, 0o755)
	if err := os.WriteFile(filepath.Join(dir, "ses1.jsonl"), []byte(transcritoConsultas), 0o644); err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC)
	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(raiz), Ahora: func() time.Time { return ahora }}
	if _, err := s.Claude.Actualizar(); err != nil {
		t.Fatal(err)
	}
	h := func(hh, mm, ss int) time.Time { return time.Date(2026, 10, 31, hh, mm, ss, 0, time.UTC) }
	s.consultas = []Consulta{
		// La misma llamada de buscar, vista por el servidor un segundo después (las claves en otro orden).
		{Hora: h(17, 25, 1), Cliente: "claude-code/2.1.309", Herramienta: "buscar", Argumentos: `{"horas":30,"texto":"veredicto"}`, Bytes: 999, Ms: 12, Respuesta: "Atalaya · 3 pasos coinciden"},
		// Un curl de prueba que ningún transcrito registró.
		{Hora: h(17, 40, 0), Cliente: "curl/8.9", Herramienta: "salud", Bytes: 50, Ms: 3, Respuesta: "Ninguna regla rota"},
	}

	w := httptest.NewRecorder()
	s.verConsultas(w, httptest.NewRequest("GET", "/api/consultas", nil))
	var cs []Consulta
	if err := json.Unmarshal(w.Body.Bytes(), &cs); err != nil {
		t.Fatal(err)
	}
	if len(cs) != 3 {
		t.Fatalf("esperaba 3 consultas: %+v", cs)
	}
	curl, buscar, salud := cs[0], cs[1], cs[2]
	if curl.Origen != "memoria" || curl.Cliente != "curl/8.9" || curl.Agente != "" {
		t.Errorf("curl: %+v", curl)
	}
	if buscar.Origen != "transcrito+memoria" || buscar.Ms != 12 || buscar.Cliente != "claude-code/2.1.309" || buscar.Respuesta != "Atalaya · 3 pasos coinciden" ||
		buscar.Bytes != 999 || buscar.Agente != "ses1" || buscar.Nombre == "" || !buscar.Hora.Equal(h(17, 25, 0)) {
		t.Errorf("buscar: %+v", buscar)
	}
	if salud.Origen != "transcrito" || salud.Ms != 0 || salud.Respuesta != "Ninguna regla rota ahora." || salud.Bytes != len("Ninguna regla rota ahora.") ||
		salud.Argumentos != `{"proyecto":"Tienda Demo"}` || salud.Agente != "ses1" {
		t.Errorf("salud: %+v", salud)
	}

	// Fuera de la ventana de días, solo queda lo de la memoria.
	ahora = ahora.AddDate(0, 0, DiasConsultas+1)
	w = httptest.NewRecorder()
	s.verConsultas(w, httptest.NewRequest("GET", "/api/consultas", nil))
	cs = nil
	_ = json.Unmarshal(w.Body.Bytes(), &cs)
	if len(cs) != 2 || cs[0].Origen != "memoria" || cs[1].Origen != "memoria" {
		t.Errorf("ventana: %+v", cs)
	}
}

func TestMismaEntrada(t *testing.T) {
	for _, c := range []struct {
		a, b  string
		igual bool
	}{
		{`{"texto":"x","horas":3}`, `{"horas":3, "texto":"x"}`, true},
		{"", "{}", true},
		{`{"texto":"x"}`, `{"texto":"y"}`, false},
	} {
		if mismaEntrada(c.a, c.b) != c.igual {
			t.Errorf("%s ~ %s: esperaba %v", c.a, c.b, c.igual)
		}
	}
}
