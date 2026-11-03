package servidor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"atalaya/internal/claude"
)

func TestRutaMapa(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "C--proyecto")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	linea := `{"type":"user","timestamp":"2026-10-31T17:20:24Z","message":{"role":"user","content":"Hola"}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "ses1.jsonl"), []byte(linea), 0o644); err != nil {
		t.Fatal(err)
	}
	s := nuevo(t)
	s.Claude = claude.NuevoIndice(raiz)
	if _, err := s.Claude.Actualizar(); err != nil {
		t.Fatal(err)
	}
	for _, caso := range []struct {
		id     string
		codigo int
	}{{"ses1", http.StatusOK}, {"codex~sesion", http.StatusNotFound}} {
		r := httptest.NewRequest("GET", "/api/agentes/"+caso.id+"/mapa", nil)
		r.Host = "localhost"
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		if w.Code != caso.codigo {
			t.Fatalf("%s: código %d, cuerpo %s", caso.id, w.Code, w.Body.String())
		}
		if caso.codigo == http.StatusOK {
			var m claude.Mapa
			if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
				t.Fatal(err)
			}
			if m.Total != 1 || m.PorCelda != 1 || len(m.Celdas) != 1 || m.Celdas[0].N != 0 || m.Celdas[0].Pasos != 1 || m.Celdas[0].Pedidos != 1 || m.Celdas[0].Hora.Format("2006-01-02T15:04:05Z") != "2026-10-31T17:20:24Z" {
				t.Errorf("mapa: %+v", m)
			}
		}
	}
}
