package servidor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"atalaya/internal/usoclaude"
)

func TestUsoClaudeHistorialReinicios(t *testing.T) {
	ahora := time.Now().Truncate(time.Second)
	dir := t.TempDir()
	datos := fmt.Sprintf(`{"t":%d,"ventanas":{"session":13},"reinicios":{"session":%d}}`+"\n", ahora.Unix(), ahora.Add(time.Hour).Unix())
	if err := os.WriteFile(filepath.Join(dir, "uso-claude.jsonl"), []byte(datos), 0o600); err != nil {
		t.Fatal(err)
	}
	l := usoclaude.NuevoLector(dir, false)
	l.Exe = "falso"
	l.Ahora = func() time.Time { return ahora }
	s := &Servidor{UsoClaude: l}
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest("GET", "http://localhost/api/uso-claude", nil))
	var e struct {
		Historial []struct {
			R map[string]int64 `json:"r"`
		} `json:"historial"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil || w.Code != http.StatusOK || len(e.Historial) != 1 || e.Historial[0].R["session"] != ahora.Add(time.Hour).Unix() {
		t.Fatalf("%d %s: %v", w.Code, w.Body.String(), err)
	}
}

func TestUsoClaudeAPI(t *testing.T) {
	ahora := time.Date(2026, 11, 1, 14, 0, 0, 0, time.UTC)
	l := usoclaude.NuevoLector(t.TempDir(), true)
	l.Demo = false
	l.Exe = "falso"
	l.Ahora = func() time.Time { return ahora }
	l.Correr = func(context.Context, string, string, []string) ([]byte, error) {
		return []byte(`{"type":"result","result":"rate limited until 3:10pm"}`), nil
	}
	s := &Servidor{UsoClaude: l}
	s.ConectarUsoClaude()
	post := func(panel bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://localhost/api/uso-claude/leer", nil)
		if panel {
			r.Header.Set(cabPanel, "1")
			r.Header.Set("Origin", "http://localhost")
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w
	}
	if w := post(false); w.Code != http.StatusForbidden {
		t.Fatal(w.Code)
	}
	if w := post(true); w.Code != http.StatusOK {
		t.Fatal(w.Code, w.Body.String())
	}
	w := post(true)
	var e usoclaude.Estado
	if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil || w.Code != 429 || e.Estado != "limitado" || e.ProximaEn != time.Date(2026, 11, 1, 15, 10, 0, 0, time.UTC).Unix() {
		t.Fatal(w.Code, w.Body.String(), err)
	}
	s.Demo = true
	w = post(true)
	if err := json.Unmarshal(w.Body.Bytes(), &e); err != nil || e.Estado != "sin-claude" {
		t.Fatal(w.Body.String(), err)
	}
}
