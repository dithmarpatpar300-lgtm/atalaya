package servidor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"atalaya/internal/codexvivo"
)

func TestRaizDeCarpeta(t *testing.T) {
	s := nuevo(t)
	s.Proyectos = []Proyecto{{Nombre: "Tienda Demo", Raiz: `C:\Proyectos\Tienda Demo`}, {Nombre: "Kit", Raiz: `C:\Proyectos\Tienda Demo\kit`}}
	casos := map[string]string{
		`C:\Proyectos\Tienda Demo`:             `C:\Proyectos\Tienda Demo`,
		`c:\proyectos\tienda demo\backend\cmd`: `C:\Proyectos\Tienda Demo`,
		`C:\Proyectos\Tienda Demo\kit\x`:       `C:\Proyectos\Tienda Demo\kit`,
		`C:\Proyectos\Tienda Demo 2`:           "",
		"":                                     "",
	}
	for in, want := range casos {
		if got := s.RaizDeCarpeta(in); got != want {
			t.Errorf("%q: %q, quería %q", in, got, want)
		}
	}
}

// Una sesión de la TUI abierta en una subcarpeta sale como agente de tipo codex del proyecto, sin sesión de Claude que
// la lance, y pasa de trabajando a en espera y a inactiva.
func TestSesionCodexEnElPanel(t *testing.T) {
	casa := t.TempDir()
	dir := filepath.Join(casa, "sessions", "2026", "10", "31")
	_ = os.MkdirAll(dir, 0o755)
	const id = "019a2b3c-4d5e-7f60-8a9b-0c1d2e3f4a5b"
	ruta := filepath.Join(dir, "rollout-2026-10-31T21-00-00-"+id+".jsonl")
	rollout := `{"timestamp":"2026-10-31T21:00:00Z","type":"session_meta","payload":{"id":"` + id + `","timestamp":"2026-10-31T21:00:00Z","cwd":"C:\\Proyectos\\Tienda Demo\\backend","cli_version":"0.166.0","source":"cli"}}
{"timestamp":"2026-10-31T21:00:01Z","type":"turn_context","payload":{"cwd":"C:\\Proyectos\\Tienda Demo\\backend","model":"gpt-6.1-terra","effort":"medium","summary":"auto"}}
{"timestamp":"2026-10-31T21:00:01Z","type":"event_msg","payload":{"type":"task_started"}}
{"timestamp":"2026-10-31T21:00:02Z","type":"event_msg","payload":{"type":"user_message","message":"Revisa el README"}}
`
	if err := os.WriteFile(ruta, []byte(rollout), 0o644); err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 31, 21, 1, 0, 0, time.UTC)
	s := nuevo(t)
	s.Ahora = func() time.Time { return ahora }
	s.Proyectos = []Proyecto{{Nombre: "Tienda Demo", Raiz: `C:\Proyectos\Tienda Demo`}}
	s.CodexVivo = codexvivo.NuevoLector(t.TempDir())
	s.CodexVivo.Sesiones, s.CodexVivo.RaizDe, s.CodexVivo.Ahora = casa, s.RaizDeCarpeta, s.Ahora
	if _, err := s.CodexVivo.Actualizar(); err != nil {
		t.Fatal(err)
	}
	buscar := func() AgenteVista {
		for _, a := range s.todos() {
			if a.ID == codexvivo.Prefijo+codexvivo.PrefijoSesion+id {
				return a
			}
		}
		t.Fatal("no está la sesión")
		return AgenteVista{}
	}
	a := buscar()
	if a.Proyecto != "Tienda Demo" || a.Tipo != "codex" || !a.Interactiva || a.Rol != "Codex · interactiva" || a.Padre != "" ||
		a.Estado != Trabajando || a.Tarea != "Revisa el README" || a.ModeloCorto == "" {
		t.Fatalf("la sesión: %+v", a)
	}
	f, _ := os.OpenFile(ruta, os.O_APPEND|os.O_WRONLY, 0)
	_, _ = f.WriteString(`{"timestamp":"2026-10-31T21:00:30Z","type":"event_msg","payload":{"type":"task_complete"}}` + "\n")
	f.Close()
	_, _ = s.CodexVivo.Actualizar()
	ahora = ahora.Add(5 * time.Minute)
	if a := buscar(); a.Estado != EnEspera {
		t.Errorf("tras el turno: %s", a.Estado)
	}
	ahora = ahora.Add(time.Hour)
	if a := buscar(); a.Estado != Inactiva {
		t.Errorf("una hora después: %s", a.Estado)
	}
}
