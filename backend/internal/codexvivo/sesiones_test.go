package codexvivo

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

const idSesion = "019a2b3c-4d5e-7f60-8a9b-0c1d2e3f4a5b"

// Un turno de la TUI: lo que pide el usuario, el razonamiento, una orden con su salida, un parche, el plan, una
// consulta a Atalaya y la respuesta, con sus tokens.
const rolloutTurno = `{"timestamp":"2026-10-31T21:00:00.000Z","type":"session_meta","payload":{"id":"` + idSesion + `","timestamp":"2026-10-31T21:00:00.000Z","cwd":"C:\\Proyectos\\Tienda Demo\\backend","originator":"codex_cli_rs","cli_version":"0.166.0","source":"cli","model_provider":"openai"}}
{"timestamp":"2026-10-31T21:00:01.000Z","type":"turn_context","payload":{"cwd":"C:\\Proyectos\\Tienda Demo\\backend","approval_policy":"on-request","model":"gpt-6.1-terra","effort":"medium","summary":"auto"}}
{"timestamp":"2026-10-31T21:00:01.100Z","type":"response_item","payload":{"type":"message","role":"user","content":[{"type":"input_text","text":"<environment_context>cwd</environment_context>"}]}}
{"timestamp":"2026-10-31T21:00:01.200Z","type":"event_msg","payload":{"type":"task_started","model_context_window":272000}}
{"timestamp":"2026-10-31T21:00:01.300Z","type":"event_msg","payload":{"type":"user_message","message":"Cuenta los archivos .go\ny dime el más largo","images":[]}}
{"timestamp":"2026-10-31T21:00:03.000Z","type":"response_item","payload":{"type":"reasoning","summary":[{"type":"summary_text","text":"**Contar archivos**\nCon rg."}],"encrypted_content":"x"}}
{"timestamp":"2026-10-31T21:00:03.500Z","type":"response_item","payload":{"type":"reasoning","summary":[],"encrypted_content":"y"}}
{"timestamp":"2026-10-31T21:00:04.000Z","type":"response_item","payload":{"type":"function_call","name":"shell","arguments":"{\"command\":[\"powershell.exe\",\"-Command\",\"rg --files -g *.go | Measure-Object\"],\"workdir\":\"C:\\\\Proyectos\"}","call_id":"call_1"}}
`

const rolloutResto = `{"timestamp":"2026-10-31T21:00:06.000Z","type":"response_item","payload":{"type":"function_call_output","call_id":"call_1","output":"{\"output\":\"Count : 42\\n\",\"metadata\":{\"exit_code\":0,\"duration_seconds\":0.4}}"}}
{"timestamp":"2026-10-31T21:00:07.000Z","type":"response_item","payload":{"type":"function_call","name":"shell_command","arguments":"{\"command\":\"cat no-existe\"}","call_id":"call_2"}}
{"timestamp":"2026-10-31T21:00:07.500Z","type":"event_msg","payload":{"type":"exec_command_end","call_id":"call_2","exit_code":1,"aggregated_output":"cat: no-existe: No such file\n"}}
{"timestamp":"2026-10-31T21:00:07.600Z","type":"response_item","payload":{"type":"function_call_output","call_id":"call_2","output":"Exit code: 1\nOutput:\ncat: no-existe: No such file\n"}}
{"timestamp":"2026-10-31T21:00:08.000Z","type":"response_item","payload":{"type":"custom_tool_call","status":"completed","call_id":"call_3","name":"apply_patch","input":"*** Begin Patch\n*** Update File: backend/main.go\n@@\n-a\n+b\n*** End Patch"}}
{"timestamp":"2026-10-31T21:00:08.200Z","type":"response_item","payload":{"type":"custom_tool_call_output","call_id":"call_3","output":"Success. Updated the following files:\nM backend/main.go\n"}}
{"timestamp":"2026-10-31T21:00:09.000Z","type":"response_item","payload":{"type":"function_call","name":"update_plan","arguments":"{\"plan\":[{\"step\":\"Contar\",\"status\":\"completed\"},{\"step\":\"Responder\",\"status\":\"in_progress\"}]}","call_id":"call_4"}}
{"timestamp":"2026-10-31T21:00:09.500Z","type":"response_item","payload":{"type":"function_call","name":"centro__estado","arguments":"{}","call_id":"call_5"}}
{"timestamp":"2026-10-31T21:00:10.000Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Hay **42** archivos. Token sk-ant-api03-abcdefghijklmnop"}]}}
{"timestamp":"2026-10-31T21:00:10.100Z","type":"event_msg","payload":{"type":"agent_message","message":"Hay 42 archivos."}}
{"timestamp":"2026-10-31T21:00:10.200Z","type":"event_msg","payload":{"type":"token_count","info":{"total_token_usage":{"input_tokens":5000,"cached_input_tokens":3000,"output_tokens":200,"reasoning_output_tokens":50,"total_tokens":5200},"last_token_usage":{"input_tokens":5000}}}}
{"timestamp":"2026-10-31T21:00:10.300Z","type":"event_msg","payload":{"type":"task_complete","last_agent_message":"Hay 42 archivos."}}
`

func escribirRollout(t *testing.T, casa, dia, texto string) string {
	t.Helper()
	dir := filepath.Join(casa, "sessions", dia[:4], dia[5:7], dia[8:10])
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	r := filepath.Join(dir, "rollout-"+dia+"T21-00-00-"+idSesion+".jsonl")
	if err := os.WriteFile(r, []byte(texto), 0o644); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestSesionInteractiva(t *testing.T) {
	casa := t.TempDir()
	ruta := escribirRollout(t, casa, "2026-10-31", rolloutTurno)
	ahora := time.Date(2026, 10, 31, 21, 0, 30, 0, time.UTC)
	_ = os.Chtimes(ruta, ahora, ahora)
	l := NuevoLector(filepath.Join(casa, "trazas-que-no-existen"))
	l.Sesiones = casa
	l.Ahora = func() time.Time { return ahora }
	l.RaizDe = func(c string) string {
		if strings.HasPrefix(c, `C:\Proyectos\Tienda Demo`) {
			return `C:\Proyectos\Tienda Demo`
		}
		return ""
	}
	id := Prefijo + PrefijoSesion + idSesion
	if ids, err := l.Actualizar(); err != nil || len(ids) != 1 || ids[0] != id {
		t.Fatalf("%v %v", ids, err)
	}
	c, ok := l.Llamada(id)
	if !ok || !c.Interactiva || c.Origen != "cli" || c.Meta.Raiz != `C:\Proyectos\Tienda Demo` || c.Carpeta != `C:\Proyectos\Tienda Demo\backend` {
		t.Fatalf("sesión: %+v", c)
	}
	if c.Meta.Modelo != "gpt-6.1-terra" || c.Meta.Effort != "medium" || c.Meta.Version != "0.166.0" || c.Meta.Tema != "Cuenta los archivos .go" {
		t.Errorf("meta: %+v", c.Meta)
	}
	if !c.EnTurno || !c.Activa(ahora) || c.EnEspera(ahora) {
		t.Errorf("con el turno abierto debería trabajar: %+v", c)
	}
	// Lo del usuario sale una vez (de event_msg, no el contexto que agrega Codex) y el razonamiento cifrado no sale.
	if len(c.Pasos) != 3 || c.Pasos[0].Tipo != "Pide" || c.Pasos[1].Tipo != "Razona" || c.Pasos[1].Resumen != "Contar archivos" || c.Pasos[2].Tipo != "Ejecuta" || c.Pasos[2].Hecho {
		t.Fatalf("pasos: %+v", c.Pasos)
	}
	if c.Pasos[2].Resumen != "rg --files -g *.go | Measure-Object" || !c.Pasos[2].Hora.Equal(time.Date(2026, 10, 31, 21, 0, 4, 0, time.UTC)) {
		t.Errorf("orden: %+v", c.Pasos[2])
	}

	// Llega el resto del turno: la orden se completa en su mismo paso.
	f, _ := os.OpenFile(ruta, os.O_APPEND|os.O_WRONLY, 0)
	_, _ = f.WriteString(rolloutResto)
	f.Close()
	ahora = ahora.Add(10 * time.Minute)
	_ = os.Chtimes(ruta, ahora, ahora)
	if ids, _ := l.Actualizar(); len(ids) != 1 {
		t.Fatalf("no vio el cambio")
	}
	c, _ = l.Llamada(id)
	var tipos []string
	for _, p := range c.Pasos {
		tipos = append(tipos, p.Tipo)
	}
	if strings.Join(tipos, ",") != "Pide,Razona,Ejecuta,Ejecuta,Edita,Planifica,Atalaya,Responde" {
		t.Fatalf("pasos: %v", tipos)
	}
	if !c.Pasos[2].Hecho || c.Pasos[2].Meta != "salida 0 · 1 líneas" || c.Pasos[2].Error {
		t.Errorf("orden bien: %+v", c.Pasos[2])
	}
	if !c.Pasos[3].Error || c.Pasos[3].Meta != "salida 1 · 1 líneas" {
		t.Errorf("orden con error: %+v", c.Pasos[3])
	}
	if c.Pasos[4].Resumen != "update backend/main.go" || c.Pasos[4].Hilo != "backend/main.go" {
		t.Errorf("parche: %+v", c.Pasos[4])
	}
	if c.Pasos[5].Resumen != "Responder" || c.Pasos[6].Herramienta != "mcp__atalaya__estado" {
		t.Errorf("plan y Atalaya: %+v %+v", c.Pasos[5], c.Pasos[6])
	}
	if strings.Contains(c.Pasos[7].Resumen, "sk-ant-api03-abcdefghijklmnop") || strings.Contains(c.Pasos[7].Resumen, "**") {
		t.Errorf("respuesta sin tapar: %q", c.Pasos[7].Resumen)
	}
	if c.Tokens.Entrada != 2000 || c.Tokens.CacheLeida != 3000 || c.Tokens.Salida != 200 {
		t.Errorf("tokens: %+v", c.Tokens)
	}
	// Turno cerrado hace 10 min: en espera del usuario; pasada media hora, terminada.
	if c.EnTurno || c.Activa(ahora) || !c.EnEspera(ahora) || c.EnEspera(ahora.Add(31*time.Minute)) {
		t.Errorf("estado tras el turno: %+v", c)
	}
	d, err := l.Detalle(id, 2)
	if err != nil || d.Salida != "Count : 42\n" {
		t.Errorf("detalle: %+v %v", d, err)
	}
	if d, _ := l.Detalle(id, 3); !strings.Contains(d.Salida, "No such file") {
		t.Errorf("detalle del error: %+v", d)
	}
}

// Sin carpeta de Codex, o con una sin sessions/ (otra versión, Linux sin Codex): sin errores y sin sesiones.
func TestSinSesiones(t *testing.T) {
	for _, casa := range []string{"", filepath.Join(t.TempDir(), "no-existe"), t.TempDir()} {
		l := NuevoLector(t.TempDir())
		l.Sesiones = casa
		if ids, err := l.Actualizar(); err != nil || len(ids) != 0 || len(l.Llamadas()) != 0 {
			t.Errorf("%q: %v %v", casa, ids, err)
		}
	}
}

// Las viejas (fuera de la ventana) no se leen; una archivada sí, y se encuentra también por el índice de Codex.
func TestSesionesVentanaEIndice(t *testing.T) {
	casa := t.TempDir()
	ahora := time.Date(2026, 10, 31, 22, 0, 0, 0, time.UTC)
	vieja := escribirRollout(t, casa, "2026-08-01", rolloutTurno)
	_ = os.Chtimes(vieja, ahora.Add(-90*24*time.Hour), ahora.Add(-90*24*time.Hour))

	// Una sesión fuera de sessions/ que solo conoce state_5.sqlite.
	otra := filepath.Join(casa, "en-otro-lado")
	_ = os.MkdirAll(otra, 0o755)
	const id2 = "019a2b3c-0000-7f60-8a9b-0c1d2e3f4a5b"
	r2 := filepath.Join(otra, "rollout-2026-10-31T21-00-00-"+id2+".jsonl")
	_ = os.WriteFile(r2, []byte(strings.ReplaceAll(rolloutTurno, idSesion, id2)), 0o644)
	_ = os.Chtimes(r2, ahora, ahora)
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(filepath.Join(casa, "state_5.sqlite")))
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE threads (id TEXT PRIMARY KEY, rollout_path TEXT NOT NULL, updated_at INTEGER NOT NULL);
		INSERT INTO threads VALUES (?, ?, ?), ('viejo', 'C:\no\existe.jsonl', 1)`, id2, r2, ahora.Unix())
	db.Close()
	if err != nil {
		t.Fatal(err)
	}

	l := NuevoLector(t.TempDir())
	l.Sesiones = casa
	l.Ahora = func() time.Time { return ahora }
	ids, err := l.Actualizar()
	if err != nil || len(ids) != 1 || ids[0] != Prefijo+PrefijoSesion+id2 {
		t.Fatalf("%v %v", ids, err)
	}
	// La base de Codex no se tocó: abrirla solo lectura no la cambia.
	if _, err := os.Stat(filepath.Join(casa, "state_5.sqlite-journal")); err == nil {
		t.Errorf("quedó un journal")
	}
}

func TestOrdenDe(t *testing.T) {
	casos := map[string]string{
		`"git status"`:                        "git status",
		`["bash","-lc","ls -la"]`:             "ls -la",
		`["powershell.exe","-Command","dir"]`: "dir",
		`["rg","-n","x"]`:                     "rg -n x",
		`[]`:                                  "",
	}
	for in, want := range casos {
		if got := ordenDe([]byte(in)); got != want {
			t.Errorf("%s: %q, quería %q", in, got, want)
		}
	}
}
