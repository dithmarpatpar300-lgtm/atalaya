package servidor

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"atalaya/internal/codexvivo"
)

func TestRegistroConsultasReinicioYRecorte(t *testing.T) {
	ahora := time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC)
	datos := t.TempDir()
	s := &Servidor{Datos: datos, Ahora: func() time.Time { return ahora }}
	t.Cleanup(s.esperarEscrituras)
	s.guardarConsulta(Consulta{Hora: ahora.AddDate(0, 0, -91), Herramienta: "antigua"})
	s.anotarConsulta(Consulta{Hora: ahora, Herramienta: "salud", Cliente: "codex", Ms: 42, Bytes: 123, Respuesta: "respuesta"})
	ruta := filepath.Join(datos, "consultas.jsonl")
	limite := time.Now().Add(5 * time.Second)
	for {
		consultasArchivoMu.Lock()
		cs, _, err := leerRegistroConsultas(ruta, ahora)
		consultasArchivoMu.Unlock()
		if err == nil && len(cs) == 1 && cs[0].Herramienta == "salud" {
			break
		}
		if time.Now().After(limite) {
			t.Fatal("no se guardó la consulta")
		}
		time.Sleep(10 * time.Millisecond)
	}
	otro := &Servidor{Datos: datos, Ahora: s.Ahora}
	t.Cleanup(otro.esperarEscrituras)
	otro.mu.Lock()
	otro.cargarConsultas()
	cs := append([]Consulta(nil), otro.consultas...)
	otro.mu.Unlock()
	if len(cs) != 1 || cs[0].Cliente != "codex" || cs[0].Ms != 42 || cs[0].Bytes != 123 || cs[0].Respuesta != "respuesta" {
		t.Fatalf("tras reiniciar: %+v", cs)
	}
	consultasArchivoMu.Lock()
	defer consultasArchivoMu.Unlock()
	b, err := os.ReadFile(ruta)
	if err != nil || len(b) == 0 || string(b) == "" {
		t.Fatalf("archivo: %s %v", b, err)
	}
	var unica Consulta
	if err := json.Unmarshal(b, &unica); err != nil || unica.Herramienta != "salud" {
		t.Fatalf("recorte: %s %v", b, err)
	}
}

func TestConsultaCodexProyectoYBytes(t *testing.T) {
	ahora := time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC)
	repo, dir := t.TempDir(), t.TempDir()
	worktree := filepath.Join(repo, ".claude", "worktrees", "tarea")
	base := filepath.Join(dir, "2026-10-31-prueba")
	meta := `{"tema":"prueba","raiz":` + jsonCadena(worktree) + `,"inicio":"2026-10-31T18:00:00Z"}`
	traza := `{"type":"item.completed","item":{"id":"i1","type":"mcp_tool_call","server":"atalaya","tool":"salud","status":"completed"}}` + "\n"
	if err := os.WriteFile(base+".meta.json", []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base+".jsonl", []byte(traza), 0o644); err != nil {
		t.Fatal(err)
	}
	s := nuevo(t)
	s.Ahora = func() time.Time { return ahora }
	s.Proyectos = []Proyecto{{Nombre: "Centro", Raiz: repo}}
	s.CodexVivo = codexvivo.NuevoLector(dir)
	s.CodexVivo.Ahora = s.Ahora
	if _, err := s.CodexVivo.Actualizar(); err != nil {
		t.Fatal(err)
	}
	s.Datos = t.TempDir()
	s.guardarConsulta(Consulta{Hora: ahora, Herramienta: "salud", Cliente: "codex", Ms: 17, Bytes: 456})
	// Un servidor nuevo recupera el registro antes de juntar la traza.
	s = &Servidor{Datos: s.Datos, Ahora: s.Ahora, Proyectos: s.Proyectos, Claude: s.Claude, CodexVivo: s.CodexVivo}
	t.Cleanup(s.esperarEscrituras)
	w := httptest.NewRecorder()
	s.verConsultas(w, httptest.NewRequest("GET", "/api/consultas", nil))
	var cs []Consulta
	if err := json.Unmarshal(w.Body.Bytes(), &cs); err != nil {
		t.Fatal(err)
	}
	if len(cs) != 1 || cs[0].Origen != "transcrito+memoria" || cs[0].Bytes != 456 || cs[0].Ms != 17 || cs[0].Proyecto != "Centro" {
		t.Fatalf("consulta Codex: %+v", cs)
	}
}
