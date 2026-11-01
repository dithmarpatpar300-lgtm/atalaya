package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnlazarDesvioYVuelta(t *testing.T) {
	// Edita A, se desvía a leer B y C (con una orden sin hilo en medio), vuelve a A; luego D sin vuelta.
	hilos := []string{"", "a.rs", "b.rs", "", "c.rs", "a.rs", "d.rs"}
	pasos := make([]Paso, len(hilos))
	for i, h := range hilos {
		pasos[i] = Paso{N: i, Hilo: h, Turno: 1, Lote: i, Continua: -1, idUso: "x"}
	}
	pasos[0].Tipo, pasos[0].idUso = "Pide", ""
	enlazar(pasos)
	prof := []int{}
	for _, p := range pasos {
		prof = append(prof, p.Profundidad)
	}
	if got := []int{0, 0, 1, 1, 1, 0, 0}; !igual(prof, got) {
		t.Errorf("profundidades %v, quería %v", prof, got)
	}
	if !pasos[2].Desvio || pasos[3].Desvio || pasos[5].Continua != 1 || pasos[6].Continua != -1 {
		t.Errorf("desvío o continuidad: %+v", pasos)
	}
	// Mismo hilo con solo órdenes en medio: es el mismo camino, no un desvío.
	p2 := []Paso{{N: 0, Hilo: "a", Lote: 1}, {N: 1, Lote: 2}, {N: 2, Hilo: "a", Lote: 3}}
	enlazar(p2)
	if p2[1].Profundidad != 0 || p2[2].Continua != 0 && p2[2].Continua != -1 {
		t.Errorf("sin otro hilo: %+v", p2)
	}
}

func TestEnlazarParaleloYTurnos(t *testing.T) {
	pasos := []Paso{
		{N: 0, Turno: 1, Lote: 1, Hilo: "a", idUso: "1"},
		{N: 1, Turno: 1, Lote: 2, Hilo: "b", idUso: "2"},
		{N: 2, Turno: 1, Lote: 2, Hilo: "c", idUso: "3"},
		{N: 3, Turno: 2, Lote: 3, Hilo: "a", idUso: "4"}, // otro turno: no es una vuelta
	}
	for i := range pasos {
		pasos[i].Continua = -1
	}
	enlazar(pasos)
	if pasos[0].Paralelo != 1 || pasos[1].Paralelo != 2 || pasos[2].Paralelo != 2 {
		t.Errorf("paralelo: %+v", pasos)
	}
	if pasos[3].Continua != -1 || pasos[1].Profundidad != 0 {
		t.Errorf("entre turnos no hay desvío: %+v", pasos)
	}
}

func igual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestDiffLineas(t *testing.T) {
	d := DiffLineas("uno\ndos\ntres", "uno\nDOS\ntres\ncuatro")
	if got := strings.Join(d.Lineas, "|"); got != " uno|-dos|+DOS| tres|+cuatro" {
		t.Errorf("%q", got)
	}
}

// Un transcrito con plan, lote en paralelo, Edit con structuredPatch y una orden con stdout y stderr.
const conPlan = `{"type":"user","timestamp":"2026-10-31T10:00:00Z","cwd":"D:\\P","message":{"role":"user","content":"Arregla la prueba"}}
{"type":"assistant","timestamp":"2026-10-31T10:00:01Z","message":{"id":"m1","role":"assistant","content":[{"type":"tool_use","id":"p1","name":"mcp__ccd_turn__update_plan","input":{"items":[{"content":"Leer","status":"in_progress"},{"content":"Corregir","status":"pending"}]}}]}}
{"type":"assistant","timestamp":"2026-10-31T10:00:02Z","message":{"id":"m2","role":"assistant","content":[{"type":"tool_use","id":"r1","name":"Read","input":{"file_path":"D:\\P\\src\\a.rs"}}],"usage":{"input_tokens":1,"cache_creation_input_tokens":10,"cache_read_input_tokens":1100,"output_tokens":4}}}
{"type":"assistant","timestamp":"2026-10-31T10:00:02Z","message":{"id":"m2","role":"assistant","content":[{"type":"tool_use","id":"r2","name":"Read","input":{"file_path":"D:\\P\\src\\b.rs"}}],"usage":{"input_tokens":1,"cache_creation_input_tokens":10,"cache_read_input_tokens":1100,"output_tokens":9}}}
{"type":"user","timestamp":"2026-10-31T10:00:03Z","message":{"role":"user","content":[{"tool_use_id":"r1","type":"tool_result","content":"1\tfn a"},{"tool_use_id":"r2","type":"tool_result","content":"1\tfn b"}]}}
{"type":"system","subtype":"compact_boundary","timestamp":"2026-10-31T10:00:03Z","compactMetadata":{"trigger":"auto","preTokens":968199,"postTokens":22202}}
{"type":"assistant","timestamp":"2026-10-31T10:00:04Z","message":{"id":"m3","role":"assistant","content":[{"type":"tool_use","id":"p2","name":"mcp__ccd_turn__update_plan","input":{"advance":true}}],"usage":{"input_tokens":0,"cache_creation_input_tokens":100,"cache_read_input_tokens":1400,"output_tokens":3}}}
{"type":"assistant","timestamp":"2026-10-31T10:00:05Z","message":{"id":"m4","role":"assistant","content":[{"type":"tool_use","id":"e1","name":"Edit","input":{"file_path":"D:\\P\\src\\a.rs","old_string":"fn a","new_string":"fn a2"}}]}}
{"type":"user","timestamp":"2026-10-31T10:00:06Z","message":{"role":"user","content":[{"tool_use_id":"e1","type":"tool_result","content":"ok"}]},"toolUseResult":{"filePath":"D:\\P\\src\\a.rs","structuredPatch":[{"oldStart":7,"oldLines":1,"newStart":7,"newLines":1,"lines":["-fn a","+fn a2"]}]}}
{"type":"assistant","timestamp":"2026-10-31T10:00:07Z","message":{"id":"m5","role":"assistant","content":[{"type":"tool_use","id":"b1","name":"Bash","input":{"command":"cargo test","description":"Pruebas"}}],"usage":{"input_tokens":2,"cache_creation_input_tokens":0,"cache_read_input_tokens":1600,"output_tokens":5}}}
{"type":"user","timestamp":"2026-10-31T10:00:09Z","message":{"role":"user","content":[{"tool_use_id":"b1","type":"tool_result","content":"ok\nwarn"}]},"toolUseResult":{"stdout":"ok","stderr":"warn","interrupted":false}}
`

func TestContinuidadDesdeElTranscrito(t *testing.T) {
	raiz := t.TempDir()
	_ = os.MkdirAll(filepath.Join(raiz, "P"), 0o755)
	_ = os.WriteFile(filepath.Join(raiz, "P", "s.jsonl"), []byte(conPlan), 0o644)
	ix := NuevoIndice(raiz)
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	_, pasos, ok := ix.Agente("s", 0)
	if !ok || len(pasos) != 7 {
		t.Fatalf("pasos: %d", len(pasos))
	}
	if pasos[1].Tipo != "Planifica" || pasos[1].Resumen != "Leer" || pasos[2].Etapa != "Leer" {
		t.Errorf("plan: %+v %+v", pasos[1], pasos[2])
	}
	if pasos[2].Paralelo != 2 || pasos[3].Paralelo != 2 || pasos[2].Lote != pasos[3].Lote {
		t.Errorf("lote en paralelo: %+v %+v", pasos[2], pasos[3])
	}
	if pasos[4].Resumen != "Corregir" || pasos[5].Etapa != "Corregir" {
		t.Errorf("advance: %+v %+v", pasos[4], pasos[5])
	}
	// Leyó a y b en paralelo, y editó a: b quedó entre las dos visitas de a.
	if pasos[5].Hilo != "src/a.rs" || pasos[5].Continua != 2 || pasos[3].Profundidad != 1 {
		t.Errorf("vuelta a a.rs: %+v / b: %+v", pasos[5], pasos[3])
	}
	// Tokens: el lote m2 (dos Read) comparte salida y contexto; cada Read mete su resultado; el contexto tras
	// los Read es el de la petición siguiente (m3).
	if pasos[2].TokSalida != 9 || pasos[3].TokSalida != 9 || pasos[2].TokContexto != 1111 || pasos[2].TokEntra != 2 {
		t.Errorf("tokens del lote: %+v", pasos[2])
	}
	if pasos[2].TokTras != 1500 || pasos[6].TokTras != pasos[6].TokContexto+pasos[6].TokSalida+pasos[6].TokEntra {
		t.Errorf("contexto tras el paso: %d, último %+v", pasos[2].TokTras, pasos[6])
	}
	d, err := ix.Detalle("s", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Diff) != 1 || d.Diff[0].ViejoDesde != 7 || strings.Join(d.Diff[0].Lineas, "|") != "-fn a|+fn a2" {
		t.Errorf("diff: %+v", d.Diff)
	}
	if !strings.Contains(d.EntradaCruda, `"old_string": "fn a"`) {
		t.Errorf("entrada cruda: %q", d.EntradaCruda)
	}
	d, _ = ix.Detalle("s", 6)
	if d.Stdout != "ok" || d.Stderr != "warn" || d.Interrumpido {
		t.Errorf("orden: %+v", d)
	}

	ctx, err := ix.Contexto("s")
	if err != nil {
		t.Fatal(err)
	}
	// Tres mensajes con usage (m2, m3, m5): m2 en dos líneas cuenta una vez, con la salida mayor.
	if len(ctx.Peticiones) != 3 || ctx.Peticiones[0].Contexto != 1111 || ctx.Peticiones[0].Salida != 9 || ctx.Peticiones[1].Contexto != 1500 {
		t.Errorf("peticiones: %+v", ctx.Peticiones)
	}
	// La compactación de 968 k dice que la ventana es de 1 M, y cae antes de la segunda petición.
	if len(ctx.Compactaciones) != 1 || ctx.Compactaciones[0].Peticion != 1 || ctx.Compactaciones[0].Despues != 22202 {
		t.Errorf("compactaciones: %+v", ctx.Compactaciones)
	}
	if ctx.Peticiones[0].Resumen != "Lee · src/a.rs" || ctx.Peticiones[0].Etapa != "Leer" || ctx.Ventana != 1_000_000 {
		t.Errorf("resumen %q, etapa %q, ventana %d", ctx.Peticiones[0].Resumen, ctx.Peticiones[0].Etapa, ctx.Ventana)
	}
	if len(ctx.Pesados) == 0 || ctx.Pesados[0].Tokens < ctx.Pesados[len(ctx.Pesados)-1].Tokens || len(ctx.Lleno) == 0 {
		t.Errorf("pesados %+v, lleno %+v", ctx.Pesados, ctx.Lleno)
	}

	trazas := ix.Trazas()
	if len(trazas) != 1 {
		t.Fatalf("trazas: %+v", trazas)
	}
	tr := trazas[0]
	if tr.Turno != 1 || !strings.HasPrefix(tr.Titulo, "P · turno 1 · Arregla la prueba") || RutaTraza(tr.Agente, tr.Turno) != "trazas/s/turno-1" {
		t.Errorf("traza: %+v", tr)
	}
	for _, want := range []string{"**Etapa: Leer**", "**Etapa: Corregir**", "∥ 2 en paralelo", "  - ", "↩ vuelve a `src/a.rs`", "- `src/a.rs`: leído 1, editado 1"} {
		if !strings.Contains(tr.Cuerpo, want) {
			t.Errorf("la traza no tiene %q:\n%s", want, tr.Cuerpo)
		}
	}
	if !strings.Contains(tr.Resumen, "etapas: Leer; Corregir") || !strings.Contains(tr.Resumen, "1 desvío, 1 vuelta") {
		t.Errorf("resumen: %q", tr.Resumen)
	}
}
