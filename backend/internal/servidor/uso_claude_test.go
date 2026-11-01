package servidor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// El mod empuja five_hour y seven_day: ponen al día la ventana de 5 h y la semanal, la semanal por modelo se queda.
func TestUsoClaudeEmpujar(t *testing.T) {
	ahora := time.Date(2026, 11, 1, 14, 0, 0, 0, time.UTC)
	l := usoclaude.NuevoLector(t.TempDir(), false)
	l.Exe = "falso"
	l.Ahora = func() time.Time { return ahora }
	reinicio := ahora.Add(3 * time.Hour).Format(time.RFC3339)
	l.Correr = func(context.Context, string, string, []string) ([]byte, error) {
		return []byte(`{"type":"system","usage_report":{"rate_limits":{"limits":[` +
			`{"kind":"session","percent":40,"resets_at":"` + reinicio + `","severity":"normal"},` +
			`{"kind":"weekly_all","percent":50,"resets_at":"` + reinicio + `"},` +
			`{"kind":"weekly_scoped","percent":7,"resets_at":"` + reinicio + `","scope":{"model":{"display_name":"Fable"}}}]}}}`), nil
	}
	if !l.Leer(context.Background(), true) {
		t.Fatal("no leyó /usage")
	}
	s := &Servidor{UsoClaude: l}
	empujar := func(cuerpo string, cab bool, origen string) int {
		r := httptest.NewRequest("POST", "http://127.0.0.1:7420/api/uso-claude/empujar", strings.NewReader(cuerpo))
		r.Header.Set("Content-Type", "application/json")
		if cab {
			r.Header.Set(cabModUso, "uso-claude")
		}
		if origen != "" {
			r.Header.Set("Origin", origen)
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w.Code
	}
	cuerpo := `{"rate_limits":[{"kind":"five_hour","percent_used":43.5,"resets_at":"` + reinicio + `"},{"kind":"seven_day","percent_used":51,"resets_at":"` + reinicio + `"}]}`
	if c := empujar(cuerpo, false, ""); c != http.StatusForbidden {
		t.Errorf("sin la cabecera del mod: %d", c)
	}
	if c := empujar(cuerpo, true, "https://ajena.example"); c != http.StatusForbidden {
		t.Errorf("desde una página ajena: %d", c)
	}
	if c := empujar(`{"rate_limits":[{"kind":"five_hour","percent_used":140,"resets_at":"`+reinicio+`"}]}`, true, ""); c != http.StatusBadRequest {
		t.Errorf("un %% imposible: %d", c)
	}
	if c := empujar(cuerpo, true, ""); c != http.StatusNoContent {
		t.Fatalf("empujar: %d", c)
	}
	e := l.Estado(24)
	got := map[string]float64{}
	for _, v := range e.Ventanas {
		got[v.Clave] = v.Porcentaje
	}
	if got["session"] != 44 || got["weekly_all"] != 51 || got["weekly_scoped:Fable"] != 7 || e.Fuente != "app" || len(e.Historial) != 2 {
		t.Errorf("tras empujar: %v, fuente %q, %d puntos", got, e.Fuente, len(e.Historial))
	}
	if e.Ventanas[0].Severidad != "normal" {
		t.Errorf("la severidad de /usage se perdió: %+v", e.Ventanas[0])
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
