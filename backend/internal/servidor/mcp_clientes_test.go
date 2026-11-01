package servidor

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/cuotas"
)

func TestHerramientaDeCliente(t *testing.T) {
	casos := map[string]string{
		"claude-code/2.1.316 (claude-desktop, agent-sdk/0.3.309)": "claude",
		"codex_cli_rs/0.166.0 (Windows 10.0.26100; x86_64)":       "codex",
		"curl/8.4.0": "",
		"":           "",
	}
	for ua, want := range casos {
		if got := herramientaDeCliente(ua); got != want {
			t.Errorf("herramientaDeCliente(%q) = %q, quiero %q", ua, got, want)
		}
	}
}

// La conexión y la consulta de Claude Code por el MCP quedan anotadas, se guardan y otro servidor las lee al arrancar.
func TestClientesMCP(t *testing.T) {
	datos := t.TempDir()
	ahora := time.Date(2026, 11, 1, 9, 0, 0, 0, time.Local)
	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(t.TempDir()), Datos: datos, Ahora: func() time.Time { return ahora }}
	t.Cleanup(s.esperarEscrituras)
	h := s.Handler()
	pedir := func(cuerpo string) {
		r := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(cuerpo))
		r.Host = "127.0.0.1:7420"
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("User-Agent", "claude-code/2.1.316 (cli)")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", cuerpo, w.Code, w.Body.String())
		}
	}
	pedir(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}`)
	c := s.clientesVistos()["claude"]
	if !c.Conexion.Equal(ahora) || !c.Consulta.IsZero() {
		t.Fatalf("tras initialize = %+v", c)
	}
	ahora = ahora.Add(time.Minute)
	pedir(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"salud","arguments":{}}}`)
	if c := s.clientesVistos()["claude"]; !c.Consulta.Equal(ahora) {
		t.Fatalf("tras tools/call = %+v", c)
	}
	if _, ok := s.clientesVistos()["codex"]; ok {
		t.Fatal("Codex no habló")
	}
	otro := &Servidor{Datos: datos}
	if c := otro.clientesVistos()["claude"]; !c.Consulta.Equal(ahora) {
		t.Fatalf("lo guardado = %+v", c)
	}
}
