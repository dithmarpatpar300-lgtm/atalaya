package servidor

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/codexvivo"
	"atalaya/internal/cuotas"
	"atalaya/internal/trazabilidad"
)

func nuevo(t *testing.T) *Servidor {
	return &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(t.TempDir())}
}

func TestSoloLocal(t *testing.T) {
	h := nuevo(t).Handler()
	for host, codigo := range map[string]int{"127.0.0.1:7420": 200, "localhost:7420": 200, "evil.example:7420": 403, "192.168.1.5:7420": 403} {
		r := httptest.NewRequest("GET", "/api/salud", nil)
		r.Host = host
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != codigo {
			t.Errorf("Host %s: %d, quería %d", host, w.Code, codigo)
		}
	}
}

func TestDatapediaSinIndice(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/datapedia?q=x", nil)
	r.Host = "localhost"
	w := httptest.NewRecorder()
	nuevo(t).Handler().ServeHTTP(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("sin índice: %d", w.Code)
	}
}

func TestCodexEnVivoReemplazaSuFilaDelRegistro(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "2026-10-31-e2-x")
	_ = os.WriteFile(base+".meta.json", []byte(`{"tema":"e2-x","clase":"E2","modelo":"gpt-6.1-sol","fase":"F5","raiz":"C:\\Proyectos\\Tienda Demo","inicio":"2026-10-31T10:00:00Z","fin":"2026-10-31T10:02:00Z","salida":0,"segundos":120}`), 0o644)
	_ = os.WriteFile(base+".jsonl", []byte(`{"type":"item.completed","item":{"id":"a","type":"agent_message","text":"Listo"}}`+"\n"), 0o644)
	s := nuevo(t)
	s.CodexVivo = codexvivo.NuevoLector(dir)
	if _, err := s.CodexVivo.Actualizar(); err != nil {
		t.Fatal(err)
	}
	s.codex = map[string][]trazabilidad.LlamadaCodex{"Tienda Demo": {
		{ID: "codex~F5~2026-10-31-e2-x", Proyecto: "Tienda Demo", Tema: "e2-x", Contraste: "correcto", Entrada: 10, EnCache: 90, TokSalida: 5},
		{ID: "codex~F5~2026-10-30-r1-viejo", Proyecto: "Tienda Demo", Tema: "r1-viejo"},
	}}
	todos := s.todos()
	if len(todos) != 2 {
		t.Fatalf("debían quedar dos (la viva y la vieja del registro): %+v", todos)
	}
	var vivo AgenteVista
	for _, a := range todos {
		if strings.HasPrefix(a.ID, codexvivo.Prefijo) {
			vivo = a
		}
	}
	if vivo.Proyecto != "Tienda Demo" || vivo.Rol != "Segunda opinión" || vivo.Estado != Entregado || vivo.Pasos != 1 {
		t.Errorf("vista: %+v", vivo)
	}
	if !strings.Contains(vivo.Avance.Texto, "contraste: correcto") || vivo.Tokens.CacheLeida != 90 {
		t.Errorf("lo del registro: %+v %+v", vivo.Avance, vivo.Tokens)
	}
}

// Una llamada a Codex cuelga de la sesión que corrió codex.ps1 con su tema; la que no encuentra su sesión queda
// como raíz. La sesión lleva su título de Claude Code.
func TestCodexCuelgaDeSuSesion(t *testing.T) {
	transcritos := t.TempDir()
	dirSes := filepath.Join(transcritos, "C--x-Tienda-Demo")
	_ = os.MkdirAll(dirSes, 0o755)
	sesion := `{"type":"user","timestamp":"2026-10-31T09:59:00Z","cwd":"C:\\x\\Tienda Demo","message":{"role":"user","content":"Pide la segunda opinión"}}
{"type":"custom-title","customTitle":"Auditar T1","sessionId":"ses1"}
{"type":"assistant","timestamp":"2026-10-31T09:59:50Z","message":{"id":"m1","role":"assistant","content":[{"type":"tool_use","id":"t1","name":"PowerShell","input":{"command":"& scripts/ops/codex.ps1 -Tema E2-X -Clase E2"}}]}}
{"type":"assistant","timestamp":"2026-10-31T10:05:00Z","message":{"id":"m2","role":"assistant","content":[{"type":"tool_use","id":"t2","name":"PowerShell","input":{"command":"$tareas = @('i1-lote', 'i1-otro')\n# ` + strings.Repeat("relleno ", 60) + `\nforeach ($t in $tareas) { Start-Job { & scripts/ops/codex.ps1 -Tema $using:t -Clase I1 } }"}}]}}
`
	_ = os.WriteFile(filepath.Join(dirSes, "ses1.jsonl"), []byte(sesion), 0o644)

	dir := t.TempDir()
	llamada := func(nombre, meta string) {
		base := filepath.Join(dir, nombre)
		_ = os.WriteFile(base+".meta.json", []byte(meta), 0o644)
		_ = os.WriteFile(base+".jsonl", []byte(`{"type":"item.completed","item":{"id":"a","type":"agent_message","text":"Listo"}}`+"\n"), 0o644)
	}
	llamada("2026-10-31-e2-x", `{"tema":"e2-x","clase":"E2","modelo":"gpt-6.1-sol","fase":"F5","raiz":"C:\\Proyectos\\Tienda Demo","inicio":"2026-10-31T10:00:30Z","fin":"2026-10-31T10:02:00Z","salida":0,"segundos":90}`)
	// Lanzada en un lote: «codex.ps1» queda pasados los 400 caracteres de la orden.
	llamada("2026-10-31-i1-lote", `{"tema":"i1-lote","clase":"I1","fase":"F5","raiz":"C:\\Proyectos\\Tienda Demo","inicio":"2026-10-31T10:05:10Z","fin":"2026-10-31T10:06:00Z","salida":0,"segundos":50}`)
	// Con su sesión anotada en la meta: manda la meta, aunque no esté en los transcritos.
	llamada("2026-10-31-i1-anotada", `{"tema":"i1-anotada","clase":"I1","fase":"F5","raiz":"C:\\Proyectos\\Tienda Demo","sesion":"ses9","inicio":"2026-10-31T10:07:00Z","fin":"2026-10-31T10:08:00Z","salida":0,"segundos":60}`)
	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(transcritos), CodexVivo: codexvivo.NuevoLector(dir)}
	if _, err := s.Claude.Actualizar(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CodexVivo.Actualizar(); err != nil {
		t.Fatal(err)
	}
	s.codex = map[string][]trazabilidad.LlamadaCodex{"Tienda Demo": {
		{ID: "codex~F5~2026-10-30-r1-viejo", Proyecto: "Tienda Demo", Fecha: "2026-10-30", Tema: "r1-viejo", Terminada: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)},
	}}
	por := map[string]AgenteVista{}
	for _, a := range s.todos() {
		por[a.Tarea] = a
	}
	if ses := por["Pide la segunda opinión"]; ses.ID != "ses1" || ses.Titulo != "Auditar T1" || ses.Proyecto != "Tienda Demo" {
		t.Errorf("la sesión: %+v", ses)
	}
	if vivo := por["e2-x"]; vivo.Padre != "ses1" {
		t.Errorf("la llamada en vivo debía colgar de ses1: %q", vivo.Padre)
	}
	if lote := por["i1-lote"]; lote.Padre != "ses1" {
		t.Errorf("la llamada lanzada en lote debía colgar de ses1: %q", lote.Padre)
	}
	if anotada := por["i1-anotada"]; anotada.Padre != "ses9" {
		t.Errorf("la llamada con su sesión en la meta debía colgar de ses9: %q", anotada.Padre)
	}
	if viejo := por["r1-viejo"]; viejo.ID == "" || viejo.Padre != "" {
		t.Errorf("la llamada sin sesión debía quedar como raíz: %+v", viejo)
	}
	// El detalle dice lo mismo que la lista.
	r := httptest.NewRequest("GET", "/api/agentes/"+por["e2-x"].ID, nil)
	r.Host = "localhost"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if !strings.Contains(w.Body.String(), `"padre":"ses1"`) {
		t.Errorf("el detalle no trae el padre: %s", w.Body.String())
	}
}

func TestInicioRegistro(t *testing.T) {
	fin := time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)
	if got := inicioRegistro(trazabilidad.LlamadaCodex{Terminada: fin, Segundos: 60}); !got.Equal(fin.Add(-time.Minute)) {
		t.Errorf("con informe: %v", got)
	}
	if got := inicioRegistro(trazabilidad.LlamadaCodex{Fecha: "2026-10-31"}); got.Day() != 31 || got.Hour() != 23 {
		t.Errorf("sin informe, el fin de su fecha: %v", got)
	}
}

func TestEstadoClaude(t *testing.T) {
	ahora := time.Unix(10_000, 0)
	casos := []struct {
		a    claude.Agente
		want string
	}{
		{claude.Agente{Tipo: "subagente", Mod: ahora.Add(-30 * time.Second)}, Trabajando},
		{claude.Agente{Tipo: "subagente", Mod: ahora.Add(-30 * time.Second), Cerrado: true}, Entregado},
		{claude.Agente{Tipo: "subagente", Mod: ahora.Add(-10 * time.Minute), Pendiente: true}, Trabajando},
		{claude.Agente{Tipo: "subagente", Mod: ahora.Add(-20 * time.Minute), Pendiente: true}, Entregado},
		{claude.Agente{Tipo: "sesion", Mod: ahora.Add(-time.Minute)}, Trabajando},
		{claude.Agente{Tipo: "sesion", Mod: ahora.Add(-time.Minute), Cerrado: true}, EnEspera},
		{claude.Agente{Tipo: "sesion", Mod: ahora.Add(-2 * time.Hour)}, Inactiva},
	}
	for i, c := range casos {
		if got := EstadoClaude(c.a, ahora); got != c.want {
			t.Errorf("caso %d: %s, quería %s", i, got, c.want)
		}
	}
}

func TestEventos(t *testing.T) {
	s := nuevo(t)
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", strings.Replace(srv.URL, "127.0.0.1", "localhost", 1)+"/api/eventos", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type %q", ct)
	}
	lector := bufio.NewReader(resp.Body)
	for {
		l, err := lector.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(l, "event: hola") {
			break
		}
	}
	s.difusion.enviar(Evento{Tipo: "agentes", IDs: []string{"a"}})
	for {
		l, err := lector.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if strings.HasPrefix(l, "data: {\"tipo\":\"agentes\"") {
			return
		}
	}
}

func TestConCache(t *testing.T) {
	h := conCache(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	for ruta, quiere := range map[string]string{
		"/":                         "no-cache",
		"/index.html":               "no-cache",
		"/_next/static/chunks/a.js": "public, max-age=31536000, immutable",
	} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", ruta, nil))
		if got := w.Header().Get("Cache-Control"); got != quiere {
			t.Errorf("%s: %q, quiere %q", ruta, got, quiere)
		}
	}
}
