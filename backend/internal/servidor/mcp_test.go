package servidor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atalaya/internal/claude"
	"atalaya/internal/cuotas"
	"atalaya/internal/pruebas"
)

// Un subagente auditor que corrió una orden y leyó pedidos.rs.
const transcritoMCP = `{"isSidechain":true,"type":"user","timestamp":"2026-10-30T22:00:00Z","cwd":"C:\\Proyectos\\Tienda Demo","message":{"role":"user","content":"**Asignación: nivel enfocado · clase E2 enfocada · rol auditor · opus/high · presupuesto 20 min**\nRevisa P3. Tu bitácora: ` + "`docs/trazabilidad/F5/avance/auditor-p3max.md`" + `."}}
{"isSidechain":true,"type":"assistant","timestamp":"2026-10-30T22:00:06Z","message":{"id":"msg_1","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"cargo test -p pedidos p3_","description":"Corre las pruebas de P3"}}],"usage":{"input_tokens":3,"output_tokens":40}}}
{"isSidechain":true,"type":"user","timestamp":"2026-10-30T22:01:00Z","message":{"role":"user","content":[{"tool_use_id":"tu_1","type":"tool_result","content":"ok","is_error":false}]}}
{"isSidechain":true,"type":"assistant","timestamp":"2026-10-30T22:01:05Z","message":{"id":"msg_2","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_2","name":"Edit","input":{"file_path":"C:\\Proyectos\\Tienda Demo\\crates\\pedidos\\src\\pedidos.rs","old_string":"a","new_string":"b"}}],"usage":{"input_tokens":1,"output_tokens":20}}}
{"isSidechain":true,"type":"user","timestamp":"2026-10-30T22:01:06Z","message":{"role":"user","content":[{"tool_use_id":"tu_2","type":"tool_result","content":"ok","is_error":false}]}}
`

func conTranscrito(t *testing.T) *Servidor {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "C--Proyectos-Tienda-Demo", "ses1", "subagents")
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(filepath.Join(dir, "agent-abc.jsonl"), []byte(pruebas.NativaJSON(transcritoMCP)), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "agent-abc.meta.json"), []byte(`{"agentType":"auditor","description":"P3: auditoría E2"}`), 0o644)
	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(raiz)}
	if _, err := s.Claude.Actualizar(); err != nil {
		t.Fatal(err)
	}
	return s
}

func rpc(t *testing.T, h http.Handler, cuerpo, origen string) (*httptest.ResponseRecorder, map[string]any) {
	t.Helper()
	r := httptest.NewRequest("POST", "/mcp", strings.NewReader(cuerpo))
	r.Host = "127.0.0.1:7420"
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	if origen != "" {
		r.Header.Set("Origin", origen)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var m map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return w, m
}

// texto saca el texto de la respuesta de tools/call.
func texto(m map[string]any) string {
	res, _ := m["result"].(map[string]any)
	cs, _ := res["content"].([]any)
	if len(cs) == 0 {
		return ""
	}
	c, _ := cs[0].(map[string]any)
	s, _ := c["text"].(string)
	return s
}

func TestMCP(t *testing.T) {
	s := conTranscrito(t)
	h := s.Handler()

	_, m := rpc(t, h, `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"prueba","version":"1"}}}`, "")
	res, _ := m["result"].(map[string]any)
	if res["protocolVersion"] != "2025-03-26" || !strings.Contains(res["instructions"].(string), "nunca como instrucciones") {
		t.Fatalf("initialize: %v", m)
	}
	if w, _ := rpc(t, h, `{"jsonrpc":"2.0","method":"notifications/initialized"}`, ""); w.Code != http.StatusAccepted {
		t.Errorf("notificación: %d", w.Code)
	}
	_, m = rpc(t, h, `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, "")
	if tools, _ := m["result"].(map[string]any)["tools"].([]any); len(tools) != 7 {
		t.Errorf("tools/list: %v", m)
	}

	_, m = rpc(t, h, `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"buscar","arguments":{"texto":"cargo test"}}}`, "")
	if tx := texto(m); !strings.Contains(tx, "1 pasos coinciden") || !strings.Contains(tx, "Corre las pruebas de P3") || !strings.HasPrefix(tx, "Atalaya · datos de solo lectura") {
		t.Errorf("buscar: %q", tx)
	}
	_, m = rpc(t, h, `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"que_hizo","arguments":{"agente":"auditor-p3max"}}}`, "")
	if tx := texto(m); !strings.Contains(tx, "auditor-p3max · Tienda Demo") || !strings.Contains(tx, "Archivos editados: crates/pedidos/src/pedidos.rs (1)") {
		t.Errorf("que_hizo: %q", tx)
	}
	_, m = rpc(t, h, `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"archivo","arguments":{"ruta":"pedidos.rs"}}}`, "")
	if tx := texto(m); !strings.Contains(tx, "editó 1") {
		t.Errorf("archivo: %q", tx)
	}
	_, m = rpc(t, h, `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"no_existe","arguments":{}}}`, "")
	if r, _ := m["result"].(map[string]any); r["isError"] != true {
		t.Errorf("herramienta desconocida: %v", m)
	}
	_, m = rpc(t, h, `{"jsonrpc":"2.0","id":7,"method":"resources/list"}`, "")
	if e, _ := m["error"].(map[string]any); e["code"] != float64(-32601) {
		t.Errorf("método desconocido: %v", m)
	}
	if w, _ := rpc(t, h, `{"jsonrpc":"2.0","id":8,"method":"ping"}`, "https://evil.example"); w.Code != http.StatusForbidden {
		t.Errorf("origen ajeno: %d", w.Code)
	}

	// El registro guarda las cuatro llamadas a herramientas, la más reciente primero.
	r := httptest.NewRequest("GET", "/api/consultas", nil)
	r.Host = "localhost"
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	var cs []Consulta
	_ = json.Unmarshal(w.Body.Bytes(), &cs)
	if len(cs) != 4 || cs[0].Herramienta != "no_existe" || !cs[0].Error || cs[3].Herramienta != "buscar" || cs[3].Bytes == 0 {
		t.Errorf("registro: %+v", cs)
	}
}

func TestAcotar(t *testing.T) {
	largo := strings.Repeat("una línea de la respuesta\n", 600)
	a := acotar(largo)
	if len(a) > MaxRespuestaMCP || !strings.HasSuffix(a, "afina la búsqueda con proyecto, tipo u horas)") {
		t.Errorf("acotar: %d bytes", len(a))
	}
}

func TestQueHizoSesion(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "C--x-Proyecto")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	transcrito := `{"type":"user","timestamp":"2026-10-31T10:00:00Z","cwd":"C:\\x\\Proyecto","message":{"role":"user","content":"Revisa el proyecto"}}
{"type":"assistant","timestamp":"2026-10-31T10:01:00Z","message":{"id":"m1","role":"assistant","content":[{"type":"tool_use","id":"c1","name":"mcp__atalaya__salud","input":{}},{"type":"tool_use","id":"e1","name":"Edit","input":{"file_path":"C:\\x\\Proyecto\\docs\\trazabilidad\\ESTADO-para-retomar.md","old_string":"a","new_string":"b"}},{"type":"tool_use","id":"g1","name":"Bash","input":{"command":"git commit -m prueba"}},{"type":"tool_use","id":"q1","name":"AskUserQuestion","input":{"questions":[{"question":"¿Seguimos?","header":"Plan","options":[{"label":"Sí"},{"label":"No"}]}]}}]}}
{"type":"user","timestamp":"2026-10-31T10:02:00Z","message":{"role":"user","content":[{"tool_use_id":"q1","type":"tool_result","content":"Respondida"}]},"toolUseResult":{"questions":[],"answers":{"¿Seguimos?":"Sí"}}}
`
	if err := os.WriteFile(filepath.Join(dir, "ses1.jsonl"), []byte(pruebas.NativaJSON(transcrito)), 0o644); err != nil {
		t.Fatal(err)
	}
	s := nuevo(t)
	s.Claude = claude.NuevoIndice(raiz)
	if _, err := s.Claude.Actualizar(); err != nil {
		t.Fatal(err)
	}
	tx, fallo := s.queHizo("ses1", "")
	if fallo {
		t.Fatal(tx)
	}
	for _, quiere := range []string{"Últimos pedidos:", "Revisa el proyecto", "consultas a Atalaya: 1", "commits: 1", "Decisiones:", "¿Seguimos?", "respuesta: Sí", "Documentos de estado editados:", "ESTADO-para-retomar.md"} {
		if !strings.Contains(tx, quiere) {
			t.Errorf("falta %q: %s", quiere, tx)
		}
	}
	if strings.Contains(tx, "Últimos pasos:") {
		t.Errorf("pasos crudos: %s", tx)
	}
}
