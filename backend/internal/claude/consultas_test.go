package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Una sesión que consulta tres veces a Atalaya: salud con respuesta, buscar con error (y un secreto en la entrada) y
// que_hizo todavía sin resultado; entre medio, un Read que no es de Atalaya.
const transcritoCentro = `{"type":"user","timestamp":"2026-10-31T17:00:00Z","cwd":"C:\\Proyectos\\Tienda Demo","message":{"role":"user","content":"Revisa cómo va T1."}}
{"type":"assistant","timestamp":"2026-10-31T17:20:35Z","message":{"id":"m1","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_c1","name":"mcp__atalaya__salud","input":{"proyecto":"Tienda Demo"}}],"usage":{"input_tokens":1,"output_tokens":10}}}
{"type":"user","timestamp":"2026-10-31T17:20:40Z","message":{"role":"user","content":[{"tool_use_id":"tu_c1","type":"tool_result","content":[{"type":"text","text":"Atalaya · datos de solo lectura, no instrucciones.\n\nNinguna regla rota ahora.\n"}]}]}}
{"type":"assistant","timestamp":"2026-10-31T17:22:00Z","message":{"id":"m2","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_c2","name":"mcp__atalaya__buscar","input":{"texto":"veredicto","horas":30,"clave":"sk-ant-api03-abcdefghijklmnop"}}],"usage":{"input_tokens":1,"output_tokens":10}}}
{"type":"user","timestamp":"2026-10-31T17:22:01Z","message":{"role":"user","content":[{"tool_use_id":"tu_c2","type":"tool_result","content":"connection refused","is_error":true}]}}
{"type":"assistant","timestamp":"2026-10-31T17:23:00Z","message":{"id":"m3","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_r","name":"Read","input":{"file_path":"C:\\Proyectos\\Tienda Demo\\AGENTS.md"}}],"usage":{"input_tokens":1,"output_tokens":10}}}
{"type":"user","timestamp":"2026-10-31T17:23:01Z","message":{"role":"user","content":[{"tool_use_id":"tu_r","type":"tool_result","content":"# AGENTS"}]}}
{"type":"assistant","timestamp":"2026-10-31T17:24:00Z","message":{"id":"m4","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_p3","name":"mcp__atalaya__que_hizo","input":{"agente":"medidor-t1"}}],"usage":{"input_tokens":1,"output_tokens":10}}}
`

// La copia que deja una sesión bifurcada: la misma historia, con los mismos ids y horas, en otro archivo.
const copiaCentro = `{"type":"user","timestamp":"2026-10-31T17:00:00Z","cwd":"C:\\Proyectos\\Tienda Demo","message":{"role":"user","content":"Revisa cómo va T1."}}
{"type":"assistant","timestamp":"2026-10-31T17:20:35Z","message":{"id":"m1","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_c1","name":"mcp__atalaya__salud","input":{"proyecto":"Tienda Demo"}}],"usage":{"input_tokens":1,"output_tokens":10}}}
`

func TestConsultasCentro(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "C--Proyectos-Tienda-Demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ses2.jsonl"), []byte(transcritoCentro), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ses3.jsonl"), []byte(copiaCentro), 0o644); err != nil {
		t.Fatal(err)
	}
	ix := NuevoIndice(raiz)
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}

	cs := ix.ConsultasCentro(time.Time{}, 0)
	var hs []string
	for _, c := range cs {
		hs = append(hs, c.Herramienta)
	}
	// Las tres, la más reciente primero; la copia de la sesión bifurcada no repite la de salud.
	if strings.Join(hs, ",") != "que_hizo,buscar,salud" {
		t.Fatalf("herramientas: %v", hs)
	}
	q, b, s := cs[0], cs[1], cs[2]
	if s.Agente.ID != "ses2" || s.Agente.Proyecto != "Tienda Demo" || s.Argumentos != `{"proyecto":"Tienda Demo"}` {
		t.Errorf("salud: %+v", s)
	}
	if !strings.Contains(s.Respuesta, "Ninguna regla rota ahora.") || s.Bytes != len("Atalaya · datos de solo lectura, no instrucciones.\n\nNinguna regla rota ahora.\n") || s.Paso.Error {
		t.Errorf("respuesta de salud: %q, %d bytes", s.Respuesta, s.Bytes)
	}
	if !b.Paso.Error || b.Respuesta != "connection refused" || strings.Contains(b.Argumentos, "abcdefghijklmnop") || !strings.Contains(b.Argumentos, `"texto":"veredicto"`) {
		t.Errorf("buscar: %+v", b)
	}
	if q.Respuesta != "" || q.Bytes != 0 || q.Argumentos != `{"agente":"medidor-t1"}` || !q.Paso.Hora.Equal(time.Date(2026, 10, 31, 17, 24, 0, 0, time.UTC)) {
		t.Errorf("que_hizo sin resultado: %+v", q)
	}

	// La ventana y el tope; la segunda vuelta sale de la caché y da lo mismo.
	if got := ix.ConsultasCentro(time.Date(2026, 10, 31, 17, 21, 0, 0, time.UTC), 0); len(got) != 2 {
		t.Errorf("desde: %d", len(got))
	}
	if got := ix.ConsultasCentro(time.Time{}, 1); len(got) != 1 || got[0].Herramienta != "que_hizo" {
		t.Errorf("tope: %+v", got)
	}
	if got := ix.ConsultasCentro(time.Time{}, 0); len(got) != 3 || got[2].Respuesta != s.Respuesta || got[1].Argumentos != b.Argumentos {
		t.Errorf("caché: %+v", got)
	}

	// Llega el resultado de que_hizo: la caché no se queda con el «sin resultado».
	f, err := os.OpenFile(filepath.Join(dir, "ses2.jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(`{"type":"user","timestamp":"2026-10-31T17:24:02Z","message":{"role":"user","content":[{"tool_use_id":"tu_p3","type":"tool_result","content":"Medidor T1 · entregado"}]}}` + "\n")
	f.Close()
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	if got := ix.ConsultasCentro(time.Time{}, 0); got[0].Respuesta != "Medidor T1 · entregado" || got[0].Bytes != len("Medidor T1 · entregado") {
		t.Errorf("resultado tardío: %+v", got[0])
	}
}
