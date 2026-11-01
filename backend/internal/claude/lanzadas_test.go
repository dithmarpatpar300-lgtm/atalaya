package claude

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"atalaya/internal/pruebas"
)

// Una sesión lanza otra con start_session: la lanzada queda como sesión principal, con su origen, el título que le dio
// la que la lanzó y su tarea de verdad.
func TestSesionLanzadaPorOtra(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "P")
	_ = os.MkdirAll(dir, 0o755)
	encargo := "La persona pide una auditoría técnica del proyecto ajeno y una comparación honesta con el nuestro.\n\n**Antes de empezar:** lee `AGENTS.md`."
	entrada, _ := json.Marshal(map[string]string{"title": "Auditoría comparativa", "name": "auditoria", "context": "fresh", "prompt": encargo})
	padre := pruebas.NativaJSON(`{"type":"user","timestamp":"2026-11-01T20:50:00Z","cwd":"C:\\Proyectos\\Tienda","message":{"role":"user","content":"Crea una sesión nueva a partir de aquí"}}`) + "\n" +
		`{"type":"assistant","timestamp":"2026-11-01T20:56:22Z","message":{"id":"m1","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"mcp__ccd_session__start_session","input":` + string(entrada) + `}],"usage":{"input_tokens":1,"output_tokens":1}}}` + "\n"
	primero, _ := json.Marshal("Background (from the session that started this one):\nLa tienda es un proyecto en Rust.\n\nTask:\n" + encargo)
	hija := `{"type":"user","timestamp":"2026-11-01T20:56:28Z","cwd":"C:\\Proyectos\\Tienda","message":{"role":"user","content":` + string(primero) + `}}` + "\n"
	sola, _ := json.Marshal("Background (from the session that started this one):\nOtra cosa.\n\nTask:\nUn encargo que nadie lanzó.")
	huerfana := `{"type":"user","timestamp":"2026-11-01T20:57:00Z","cwd":"C:\\Proyectos\\Tienda","message":{"role":"user","content":` + string(sola) + `}}` + "\n"
	for nombre, cuerpo := range map[string]string{"padre.jsonl": padre, "hija.jsonl": hija, "huerfana.jsonl": huerfana} {
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(cuerpo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ix := NuevoIndice(raiz)
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	porID := map[string]Agente{}
	for _, a := range ix.Agentes() {
		porID[a.ID] = a
	}
	h := porID["hija"]
	if h.Tipo != "sesion" || h.Padre != "" || h.LanzadaPor != "padre" {
		t.Errorf("la lanzada: tipo %q, padre %q, lanzada por %q", h.Tipo, h.Padre, h.LanzadaPor)
	}
	if h.Titulo != "Auditoría comparativa" || h.Tarea != "La persona pide una auditoría técnica del proyecto ajeno y una comparación honesta con el nuestro." {
		t.Errorf("título %q, tarea %q", h.Titulo, h.Tarea)
	}
	if _, pasos, _ := ix.Agente("hija", 0); len(pasos) == 0 || pasos[0].Resumen != h.Tarea {
		t.Errorf("el pedido de la lanzada es su encargo: %+v", pasos)
	}
	if o := porID["huerfana"]; o.LanzadaPor != "" || o.Tarea != "Un encargo que nadie lanzó." {
		t.Errorf("sin quien la lanzó: %+v", o)
	}
}
