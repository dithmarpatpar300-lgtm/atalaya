package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atalaya/internal/pruebas"
)

// Un subagente con: el encargo, un mensaje de Claude repartido en dos líneas con el mismo id
// (el usage no se cuenta dos veces), una orden con su resultado, una herramienta aún sin resultado.
const transcrito = `{"isSidechain":true,"type":"user","timestamp":"2026-10-30T22:00:00Z","cwd":"C:\\Proyectos\\Tienda Demo","message":{"role":"user","content":"**Asignación: nivel enfocado · clase E2 enfocada · rol auditor · opus/high · presupuesto 20 min · base ` + "`main` en `a8b0b0d`" + `**\nLee AGENTS.md. Tu bitácora: ` + "`docs/trazabilidad/F5/avance/auditor-p3max.md`" + `. Deja el informe en docs/trazabilidad/F5/10-p3-auditoria.md"}}
{"isSidechain":true,"type":"user","isMeta":true,"timestamp":"2026-10-30T22:00:01Z","message":{"role":"user","content":"<system-reminder>x</system-reminder>"}}
{"isSidechain":true,"type":"assistant","timestamp":"2026-10-30T22:00:05Z","message":{"id":"msg_1","model":"claude-opus-5-5","role":"assistant","content":[{"type":"text","text":"Empiezo por el ADR."}],"usage":{"input_tokens":3,"cache_creation_input_tokens":1000,"cache_read_input_tokens":5000,"output_tokens":10}}}
{"isSidechain":true,"type":"assistant","timestamp":"2026-10-30T22:00:06Z","message":{"id":"msg_1","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_1","name":"Bash","input":{"command":"cargo test -p pedidos p3_ --token sk-ant-api03-abcdefghijklmnop","description":"Corre las pruebas de P3"}}],"usage":{"input_tokens":3,"cache_creation_input_tokens":1000,"cache_read_input_tokens":5000,"output_tokens":40}}}
{"isSidechain":true,"type":"user","timestamp":"2026-10-30T22:01:00Z","message":{"role":"user","content":[{"tool_use_id":"tu_1","type":"tool_result","content":"running 2 tests\ntest result: ok. 2 passed","is_error":false}]}}
{"isSidechain":true,"type":"assistant","timestamp":"2026-10-30T22:01:05Z","message":{"id":"msg_2","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"tu_2","name":"Read","input":{"file_path":"C:\\Proyectos\\Tienda Demo\\crates\\pedidos\\src\\pedidos.rs","offset":2218,"limit":48}}],"usage":{"input_tokens":1,"cache_creation_input_tokens":200,"cache_read_input_tokens":6000,"output_tokens":20}}}
`

func preparar(t *testing.T) (string, string) {
	t.Helper()
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "C--Proyectos-Tienda-Demo", "ses1", "subagents")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(dir, "agent-abc.jsonl")
	if err := os.WriteFile(ruta, []byte(pruebas.NativaJSON(transcrito)), 0o644); err != nil {
		t.Fatal(err)
	}
	meta := `{"agentType":"auditor","description":"P3: auditoría E2","model":"opus","effort":"high"}`
	if err := os.WriteFile(filepath.Join(dir, "agent-abc.meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	return raiz, ruta
}

// Un transcrito borrado del disco sale del índice y su id se avisa como cambiado; los demás siguen.
func TestSoltarBorrados(t *testing.T) {
	raiz, ruta := preparar(t)
	ix := NuevoIndice(raiz)
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := ix.Agente("ses1~abc", 0); !ok {
		t.Fatal("el agente debía estar")
	}
	if err := os.Remove(ruta); err != nil {
		t.Fatal(err)
	}
	ids, err := ix.Actualizar()
	if err != nil || len(ids) != 1 || ids[0] != "ses1~abc" {
		t.Fatalf("ids %v, err %v", ids, err)
	}
	if _, _, ok := ix.Agente("ses1~abc", 0); ok {
		t.Error("el agente de un transcrito borrado sigue en el índice")
	}
}

func TestSubagente(t *testing.T) {
	raiz, ruta := preparar(t)
	ix := NuevoIndice(raiz)
	ids, err := ix.Actualizar()
	if err != nil || len(ids) != 1 || ids[0] != "ses1~abc" {
		t.Fatalf("ids %v, err %v", ids, err)
	}
	a, pasos, ok := ix.Agente("ses1~abc", 0)
	if !ok {
		t.Fatal("no está el agente")
	}
	if a.Rol != "Auditor" || a.Tarea != "P3: auditoría E2" || a.Effort != "high" || a.Proyecto != "Tienda Demo" {
		t.Errorf("datos: %+v", a)
	}
	if a.Modelo != "claude-opus-5-5" {
		t.Errorf("modelo %q", a.Modelo)
	}
	if a.Clase != "E2" || a.Presupuesto != "20 min" || a.Base != "main en a8b0b0d" {
		t.Errorf("asignación: clase %q presupuesto %q base %q", a.Clase, a.Presupuesto, a.Base)
	}
	if a.Bitacora != "docs/trazabilidad/F5/avance/auditor-p3max.md" || a.Informe != "docs/trazabilidad/F5/10-p3-auditoria.md" {
		t.Errorf("bitácora %q informe %q", a.Bitacora, a.Informe)
	}
	// msg_1 aparece dos veces: cuenta una vez, con el último usage. msg_2 suma aparte.
	want := Tokens{Entrada: 4, CacheEscrita: 1200, CacheLeida: 11000, Salida: 60}
	if a.Tokens != want {
		t.Errorf("tokens %+v, quería %+v", a.Tokens, want)
	}
	tipos := []string{}
	for _, p := range pasos {
		tipos = append(tipos, p.Tipo)
	}
	if strings.Join(tipos, ",") != "Pide,Responde,Ejecuta,Lee" {
		t.Errorf("pasos %v", tipos)
	}
	// Una página vieja: los dos pasos de antes del paso 3 (1 y 2); los últimos dos, sin «antes».
	if _, viejos, _ := ix.AgenteAntes("ses1~abc", 3, 2); len(viejos) != 2 || viejos[0].N != 1 || viejos[1].N != 2 {
		t.Errorf("antes de 3: %+v", viejos)
	}
	if _, ultimos, _ := ix.Agente("ses1~abc", 2); len(ultimos) != 2 || ultimos[0].N != 2 {
		t.Errorf("los últimos dos: %+v", ultimos)
	}
	if !a.Pendiente || pasos[3].Hecho {
		t.Error("el Read sin resultado debería estar pendiente")
	}
	if pasos[2].Resumen != "Corre las pruebas de P3" || !pasos[2].Hecho || pasos[2].Meta != "salida 0 · 2 líneas" {
		t.Errorf("paso Ejecuta: %+v", pasos[2])
	}
	if pasos[3].Resumen != "crates/pedidos/src/pedidos.rs, desde la línea 2218" {
		t.Errorf("resumen de Read: %q", pasos[3].Resumen)
	}

	d, err := ix.Detalle("ses1~abc", 2)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d.Orden, "cargo test -p pedidos p3_") || strings.Contains(d.Orden, "abcdefghijklmnop") {
		t.Errorf("la orden no está o no tapó el secreto: %q", d.Orden)
	}
	if !strings.Contains(d.Salida, "2 passed") {
		t.Errorf("salida %q", d.Salida)
	}

	// Lo que se agrega al archivo se lee desde donde quedó.
	f, _ := os.OpenFile(ruta, os.O_APPEND|os.O_WRONLY, 0)
	_, _ = f.WriteString(`{"isSidechain":true,"type":"user","timestamp":"2026-10-30T22:02:00Z","message":{"role":"user","content":[{"tool_use_id":"tu_2","type":"tool_result","content":"a\nb\nc"}]}}` + "\n" + `{"isSidechain":true,"type":"assistant","timestamp":"2026-10-30T22:02`)
	f.Close()
	if ids, _ := ix.Actualizar(); len(ids) != 1 {
		t.Fatalf("no vio el cambio: %v", ids)
	}
	a, pasos, _ = ix.Agente("ses1~abc", 0)
	if a.Pendiente || !pasos[3].Hecho || pasos[3].Meta != "3 líneas" || a.Pasos != 4 {
		t.Errorf("tras crecer: pendiente %v, paso %+v, pasos %d", a.Pendiente, pasos[3], a.Pasos)
	}
	d, _ = ix.Detalle("ses1~abc", 3)
	if !strings.HasPrefix(d.Orden, pruebas.Nativa("Read C:\\Proyectos")) || d.Salida != "a\nb\nc" {
		t.Errorf("detalle del Read: %+v", d)
	}
}

func TestPartirAsignacionReal(t *testing.T) {
	encargo := "**Asignación: nivel enfocado · clase V · rol medidor (general-purpose) · sonnet/medium · presupuesto 30 min · base `main` en `d349c68` (sin cambios de código) · cuota: Codex normal 0 %/18 % · Claude normal 4 %/11 %.** Si vas a pasarte, para y escribe lo que tienes.\nToca `crates/pedidos/src/pedidos.rs` y `download_tests.rs`; lee `docs/adr/0013-x.md` y otra vez `pedidos.rs`."
	ag := &Agente{}
	leerAsignacion(ag, encargo)
	if ag.Nivel != "enfocado" || ag.ClaseDetalle != "V" || ag.Clase != "V" || ag.RolAsignado != "medidor (general-purpose)" || ag.ModeloAsignado != "sonnet/medium" {
		t.Errorf("campos: %+v", ag)
	}
	if ag.Presupuesto != "30 min" || ag.Cuota != "Codex normal 0 %/18 % · Claude normal 4 %/11 %" {
		t.Errorf("presupuesto %q, cuota %q", ag.Presupuesto, ag.Cuota)
	}
	if strings.Contains(ag.Asignacion, "Si vas a pasarte") {
		t.Errorf("la asignación se llevó el resto de la línea: %q", ag.Asignacion)
	}
	if strings.Join(ag.Nombrados, ",") != "pedidos.rs,download_tests.rs,0013-x.md" {
		t.Errorf("nombrados: %v", ag.Nombrados)
	}
}

// El kit en inglés escribe la línea y sus campos en inglés; el coordinador puede nombrar el modelo («modelo opus/high»).
func TestAsignacionEnInglesYModeloNombrado(t *testing.T) {
	ag := &Agente{}
	leerAsignacion(ag, "**Assignment: level focused · class I2 · role designer · model opus/high · budget 30 min · base `main` at `abc123` · quota: Claude normal**\nBuild the view.")
	if ag.Nivel != "focused" || ag.ClaseDetalle != "I2" || ag.Clase != "I2" || ag.RolAsignado != "designer" || ag.ModeloAsignado != "opus/high" {
		t.Errorf("campos: %+v", ag)
	}
	if ag.Presupuesto != "30 min" || ag.Cuota != "Claude normal" {
		t.Errorf("presupuesto %q, cuota %q", ag.Presupuesto, ag.Cuota)
	}
	es := &Agente{}
	partirAsignacion(es, "nivel enfocado · clase I2 · rol diseñador · modelo opus/high · presupuesto 30 min")
	if es.ModeloAsignado != "opus/high" || es.RolAsignado != "diseñador" {
		t.Errorf("modelo nombrado: %+v", es)
	}
	// «rolling» no es el campo «rol».
	otro := &Agente{}
	partirAsignacion(otro, "rolling update · clase V")
	if otro.RolAsignado != "" || otro.ClaseDetalle != "V" {
		t.Errorf("prefijo suelto: %+v", otro)
	}
}

// Tienda Demo deja el encargo en un archivo y el prompt solo apunta a él: la asignación sale del Read de ese archivo.
func TestEncargoEnArchivo(t *testing.T) {
	puntero := "Lee tu encargo completo en `C:\\Users\\usuario\\AppData\\Local\\Temp\\scratchpad\\encargo-medidor-t1b-mem.md` y síguelo. Es el encargo del coordinador; empieza por su línea de asignación."
	m := reEncargoArchivo.FindStringSubmatch(puntero)
	if m == nil || m[1] != "encargo-medidor-t1b-mem.md" {
		t.Fatalf("archivo del encargo: %v", m)
	}
	a := &archivo{agente: &Agente{Tarea: lineaDeTarea(puntero, 140)}}
	a.encargoLeido("     1→**Asignación: nivel enfocado · clase T1 · rol medidor · sonnet/high · presupuesto 45 min · base `main` en `abc123`**\n     2→\n     3→Eres el **medidor de T1b** del proyecto.")
	ag := a.agente
	if ag.ClaseDetalle != "T1" || ag.ModeloAsignado != "sonnet/high" || ag.Presupuesto != "45 min" {
		t.Errorf("campos: %+v", ag)
	}
	sinDosPuntos := &Agente{}
	partirAsignacion(sinDosPuntos, "nivel enfocado · clase V · presupuesto 30 min · cuota Codex normal · Claude normal (5 h al 19 %, semana al 21 %) · sin Internet.")
	if sinDosPuntos.Cuota != "Codex normal · Claude normal (5 h al 19 %, semana al 21 %) · sin Internet" {
		t.Errorf("cuota sin dos puntos: %q", sinDosPuntos.Cuota)
	}
	if lineaAsignacion(ag.Encargo) != 1 || ag.Tarea != "Eres el **medidor de T1b** del proyecto." && ag.Tarea != "Eres el medidor de T1b del proyecto." {
		t.Errorf("línea %d, tarea %q", lineaAsignacion(ag.Encargo), ag.Tarea)
	}
}

func TestSesionIgnoraLineasDeSubagente(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "P")
	_ = os.MkdirAll(dir, 0o755)
	body := `{"type":"user","timestamp":"2026-10-30T20:00:00Z","cwd":"C:\\Proyectos\\Atalaya","message":{"role":"user","content":"<system-reminder>aviso</system-reminder>\nTask (from the session that started this one):\nConstruye el panel"}}
{"type":"assistant","isSidechain":true,"timestamp":"2026-10-30T20:00:01Z","message":{"id":"m0","role":"assistant","content":[{"type":"text","text":"ajeno"}],"usage":{"input_tokens":99,"output_tokens":99}}}
{"type":"assistant","timestamp":"2026-10-30T20:00:02Z","message":{"id":"m1","model":"claude-opus-5-5","role":"assistant","stop_reason":"end_turn","content":[{"type":"text","text":"Listo."}],"usage":{"input_tokens":5,"output_tokens":7}}}
`
	_ = os.WriteFile(filepath.Join(dir, "s.jsonl"), []byte(pruebas.NativaJSON(body)), 0o644)
	ix := NuevoIndice(raiz)
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	a, pasos, ok := ix.Agente("s", 0)
	if !ok || a.Tipo != "sesion" || a.Proyecto != "Atalaya" || a.Tarea != "Construye el panel" {
		t.Fatalf("%+v", a)
	}
	if a.Tokens.Entrada != 5 || a.Tokens.Salida != 7 || len(pasos) != 2 || !a.Cerrado {
		t.Errorf("tokens %+v, pasos %d, cerrado %v", a.Tokens, len(pasos), a.Cerrado)
	}
}

func TestTituloDeLaSesion(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "P")
	_ = os.MkdirAll(filepath.Join(dir, "solo-archivo"), 0o755)
	pedido := pruebas.NativaJSON(`{"type":"user","timestamp":"2026-10-30T20:00:00Z","cwd":"C:\\Proyectos\\Atalaya","message":{"role":"user","content":"Construye el panel"}}`) + "\n"
	escribir := func(nombre, cuerpo string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, nombre), []byte(cuerpo), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// El resumen es el respaldo; ai-title le gana; custom-title gana a todo, y vale la última.
	escribir("resumen.jsonl", `{"type":"summary","summary":"Panel en Go","leafUuid":"x"}`+"\n"+pedido)
	escribir("ia.jsonl", `{"type":"summary","summary":"Panel en Go"}`+"\n"+pedido+`{"type":"ai-title","aiTitle":"Panel local","sessionId":"ia"}`+"\n")
	escribir("propio.jsonl", pedido+`{"type":"custom-title","customTitle":"Primero","sessionId":"propio"}`+"\n"+`{"type":"ai-title","aiTitle":"Panel local"}`+"\n"+`{"type":"custom-title","customTitle":"Atalaya","sessionId":"propio"}`+"\n")
	escribir("solo-archivo.jsonl", pedido)
	escribir(filepath.Join("solo-archivo", "custom-title.json"), `{"customTitle":"Del archivo"}`)
	escribir("sin.jsonl", pedido)
	ix := NuevoIndice(raiz)
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{"resumen": "Panel en Go", "ia": "Panel local", "propio": "Atalaya", "solo-archivo": "Del archivo", "sin": ""} {
		a, _, ok := ix.Agente(id, 0)
		if !ok || a.Titulo != want || a.Tarea != "Construye el panel" {
			t.Errorf("%s: título %q (quería %q), tarea %q", id, a.Titulo, want, a.Tarea)
		}
	}
	// Renombrar la sesión agrega otra línea: se lee desde donde quedó.
	f, _ := os.OpenFile(filepath.Join(dir, "propio.jsonl"), os.O_APPEND|os.O_WRONLY, 0)
	_, _ = f.WriteString(`{"type":"custom-title","customTitle":"Otro nombre","sessionId":"propio"}` + "\n")
	f.Close()
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	if a, _, _ := ix.Agente("propio", 0); a.Titulo != "Otro nombre" {
		t.Errorf("tras renombrar: %q", a.Titulo)
	}
}
