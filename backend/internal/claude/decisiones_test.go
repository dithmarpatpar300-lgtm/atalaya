package claude

import (
	"os"
	"path/filepath"
	"testing"
)

// Una sesión que pide permiso con botones, recibe primero el informe de un agente (no es tu respuesta) y luego tu
// clic; hace dos preguntas de opciones que respondes; y deja dos botones seguidos, el primero reemplazado.
const sesionConDecisiones = `{"type":"user","timestamp":"2026-10-31T15:00:00Z","cwd":"C:\\Proyectos\\Atalaya","message":{"role":"user","content":"Construye la bandeja"}}
{"type":"assistant","timestamp":"2026-10-31T15:10:00Z","message":{"id":"m1","role":"assistant","content":[{"type":"tool_use","id":"w1","name":"mcp__ccd_turn__wrap_up","input":{"recap":"Listo el kit.","ask":"¿Hago el commit y push?","actions":[{"label":"Sí, commit y push"},{"label":"Todavía no"}]}}]}}
{"type":"user","timestamp":"2026-10-31T15:11:00Z","message":{"role":"user","content":"Another Claude session sent a message: informe del agente"}}
{"type":"user","timestamp":"2026-10-31T15:12:00Z","message":{"role":"user","content":"<artifact-view-context artifact=\"x\">\n{\"mode\":\"focused\"}\n</artifact-view-context>\n<artifact-content-authored-by-others/>\n\nSí, commit y push"}}
{"type":"assistant","timestamp":"2026-10-31T15:20:00Z","message":{"id":"m2","role":"assistant","content":[{"type":"tool_use","id":"q1","name":"AskUserQuestion","input":{"questions":[{"question":"¿Mido ahora?","header":"Medición","multiSelect":false,"options":[{"label":"Ahora (Recomendado)","description":"Cierra H3 hoy."},{"label":"Después","description":"Mañana."}]},{"question":"¿Abro D2?","header":"D2","options":[{"label":"Sí"},{"label":"No"}]}]}}]}}
{"type":"user","timestamp":"2026-10-31T15:25:00Z","message":{"role":"user","content":[{"tool_use_id":"q1","type":"tool_result","content":"The user answered"}]},"toolUseResult":{"questions":[],"answers":{"¿Mido ahora?":"Ahora (Recomendado)","¿Abro D2?":"No, mejor mañana"}}}
{"type":"assistant","timestamp":"2026-10-31T15:30:00Z","message":{"id":"m3","role":"assistant","content":[{"type":"tool_use","id":"w2","name":"mcp__ccd_turn__wrap_up","input":{"recap":"Primero.","ask":"¿Sigo con A?","actions":[{"label":"Sí"}]}}]}}
{"type":"assistant","timestamp":"2026-10-31T15:31:00Z","message":{"id":"m4","role":"assistant","content":[{"type":"tool_use","id":"w3","name":"mcp__ccd_turn__wrap_up","input":{"recap":"Sin pregunta."}}]}}
{"type":"assistant","timestamp":"2026-10-31T15:32:00Z","message":{"id":"m5","role":"assistant","content":[{"type":"tool_use","id":"w4","name":"mcp__ccd_turn__wrap_up","input":{"recap":"Segundo.","ask":"¿Sigo con B?","actions":[{"action":"commit"},{"label":"Otra cosa"}]}}]}}
`

func TestDecisiones(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "C--Proyectos-Atalaya")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ses.jsonl"), []byte(sesionConDecisiones), 0o644); err != nil {
		t.Fatal(err)
	}
	ix := NuevoIndice(raiz)
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	ds := ix.Decisiones("")
	if len(ds) != 4 {
		t.Fatalf("decisiones: %d, quiere 4 (el wrap_up sin pregunta no cuenta): %+v", len(ds), ds)
	}
	// De la más nueva a la más vieja.
	b, a, q, c := ds[0], ds[1], ds[2], ds[3]
	if c.Tipo != "botones" || c.Estado != DecRespondida || c.Preguntas[0].Respuesta != "Sí, commit y push" || c.Contexto != "Listo el kit." {
		t.Errorf("los primeros botones: %+v", c)
	}
	if !c.Preguntas[0].Opciones[0].Recomendada || c.Preguntas[0].Opciones[1].Recomendada {
		t.Errorf("la primera opción de los botones es la recomendada: %+v", c.Preguntas[0].Opciones)
	}
	if q.Tipo != "preguntas" || q.Estado != DecRespondida || len(q.Preguntas) != 2 ||
		q.Preguntas[0].Respuesta != "Ahora (Recomendado)" || q.Preguntas[1].Respuesta != "No, mejor mañana" {
		t.Errorf("las preguntas: %+v", q)
	}
	if !q.Preguntas[0].Opciones[0].Recomendada || q.Preguntas[0].Encabezado != "Medición" || q.Preguntas[0].Opciones[0].Nota != "Cierra H3 hoy." {
		t.Errorf("las opciones: %+v", q.Preguntas[0])
	}
	if a.Estado != DecReemplazada || b.Estado != DecPendiente {
		t.Errorf("reemplazada %q y pendiente %q", a.Estado, b.Estado)
	}
	if b.Preguntas[0].Opciones[0].Etiqueta != "commit" || b.Proyecto != "Atalaya" || b.Agente != "ses" {
		t.Errorf("la pendiente: %+v", b)
	}
	if len(ix.Decisiones("Tienda Demo")) != 0 {
		t.Error("el filtro por proyecto")
	}
}
