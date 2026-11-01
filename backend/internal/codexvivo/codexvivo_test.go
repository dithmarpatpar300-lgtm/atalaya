package codexvivo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const eventos = `{"type":"thread.started","thread_id":"t1"}
{"type":"turn.started"}
{"type":"item.completed","item":{"id":"i0","type":"reasoning","text":"**Leer el cambio**\nPrimero el diff."}}
{"type":"item.started","item":{"id":"i1","type":"command_execution","command":"rg -n stock crates","aggregated_output":"","status":"in_progress"}}
`

const resto = `{"type":"item.completed","item":{"id":"i1","type":"command_execution","command":"rg -n stock crates","aggregated_output":"pedidos.rs:2230\npedidos.rs:2246\n","exit_code":0,"status":"completed"}}
{"type":"item.completed","item":{"id":"i2","type":"command_execution","command":"cat no-existe","aggregated_output":"cat: no-existe: No such file","exit_code":1,"status":"failed"}}
{"type":"item.completed","item":{"id":"i3","type":"todo_list","items":[{"text":"Leer","completed":true},{"text":"Contrastar","completed":false}]}}
{"type":"item.completed","item":{"id":"i4","type":"agent_message","text":"**Veredicto: aceptar.** Token sk-ant-api03-abcdefghijklmnop"}}
{"type":"turn.completed","usage":{"input_tokens":1000,"cached_input_tokens":800,"output_tokens":50}}
`

func TestLlamadaEnVivo(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "2026-10-31-e2-prueba")
	_ = os.WriteFile(base+".meta.json", []byte("\ufeff"+`{"tema":"e2-prueba","clase":"E2","modelo":"gpt-6.1-sol","effort":"high","fase":"F5","raiz":"C:\\Proyectos\\Tienda Demo","inicio":"2026-10-31T10:00:00-05:00","base":"commit a8b0b0d"}`), 0o644)
	_ = os.WriteFile(base+".jsonl", []byte(eventos), 0o644)

	ahora := time.Date(2026, 10, 31, 15, 1, 0, 0, time.UTC)
	l := NuevoLector(dir)
	l.Ahora = func() time.Time { return ahora }
	if ids, err := l.Actualizar(); err != nil || len(ids) != 1 || ids[0] != "codexvivo~2026-10-31-e2-prueba" {
		t.Fatalf("%v %v", ids, err)
	}
	c, _ := l.Llamada("codexvivo~2026-10-31-e2-prueba")
	if c.Terminada || len(c.Pasos) != 2 || c.Pasos[1].Tipo != "Ejecuta" || c.Pasos[1].Hecho || c.Meta.Clase != "E2" {
		t.Fatalf("en curso: %+v", c)
	}
	if c.Pasos[0].Tipo != "Razona" || c.Pasos[0].Resumen != "Leer el cambio" {
		t.Errorf("razonamiento: %+v", c.Pasos[0])
	}

	// Llega el resto y la ficha de fin: la orden se completa en su mismo paso, no en uno nuevo.
	f, _ := os.OpenFile(base+".jsonl", os.O_APPEND|os.O_WRONLY, 0)
	_, _ = f.WriteString(resto)
	f.Close()
	_ = os.WriteFile(base+".meta.json", []byte(`{"tema":"e2-prueba","clase":"E2","inicio":"2026-10-31T10:00:00-05:00","fin":"2026-10-31T10:02:00-05:00","salida":0,"segundos":120,"informe":"docs/trazabilidad/F5/codex/2026-10-31-e2-prueba.md"}`), 0o644)
	if ids, _ := l.Actualizar(); len(ids) != 1 {
		t.Fatalf("no vio el cambio")
	}
	c, _ = l.Llamada("codexvivo~2026-10-31-e2-prueba")
	tipos := []string{}
	for _, p := range c.Pasos {
		tipos = append(tipos, p.Tipo)
	}
	if strings.Join(tipos, ",") != "Razona,Ejecuta,Ejecuta,Planifica,Responde" {
		t.Errorf("pasos: %v", tipos)
	}
	if !c.Terminada || !c.Pasos[1].Hecho || c.Pasos[1].Meta != "salida 0 · 2 líneas" || !c.Pasos[2].Error {
		t.Errorf("órdenes: %+v / %+v", c.Pasos[1], c.Pasos[2])
	}
	// La orden que se completó conserva su lugar (turno, lote) y el resumen va sin negrita.
	if c.Pasos[1].Turno != 1 || c.Pasos[1].Lote != 2 || c.Pasos[1].Paralelo != 1 || c.Pasos[1].Continua != -1 {
		t.Errorf("lugar de la orden: %+v", c.Pasos[1])
	}
	if c.Pasos[4].Resumen != "Veredicto: aceptar. Token sk-ant-[tapado]" {
		t.Errorf("resumen: %q", c.Pasos[4].Resumen)
	}
	// Una ficha dañada no borra la buena.
	_ = os.WriteFile(base+".meta.json", []byte(`{"tema":"e2-prueba","raiz":"C:\Proyectos"`), 0o644)
	_, _ = l.Actualizar()
	if c, _ = l.Llamada(c.ID); !c.Terminada || c.Meta.Clase != "E2" {
		t.Errorf("la ficha dañada borró la buena: %+v", c.Meta)
	}
	if c.Pasos[3].Resumen != "Contrastar" {
		t.Errorf("plan: %+v", c.Pasos[3])
	}
	if c.Tokens.Entrada != 200 || c.Tokens.CacheLeida != 800 || c.Tokens.Salida != 50 {
		t.Errorf("tokens: %+v", c.Tokens)
	}
	d, err := l.Detalle(c.ID, 4)
	if err != nil || strings.Contains(d.Salida, "abcdefghijklmnop") || !strings.Contains(d.Salida, "Veredicto") {
		t.Errorf("detalle: %+v %v", d, err)
	}
	if d, _ := l.Detalle(c.ID, 1); d.Orden != "rg -n stock crates" || !strings.Contains(d.Stdout, "2246") {
		t.Errorf("orden: %+v", d)
	}
	if c.Activa(ahora) {
		t.Error("terminada no debe estar activa")
	}
}

func TestLlamadaTerminadaSeRepartePorElTiempo(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "2026-10-31-r1")
	_ = os.WriteFile(base+".meta.json", []byte(`{"tema":"r1","inicio":"2026-10-31T10:00:00Z","fin":"2026-10-31T10:04:00Z","salida":0}`), 0o644)
	_ = os.WriteFile(base+".jsonl", []byte(eventos+resto), 0o644)
	l := NuevoLector(dir)
	_, _ = l.Actualizar()
	c, _ := l.Llamada(Prefijo + "2026-10-31-r1")
	inicio := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	for i, p := range c.Pasos {
		if !p.Hora.After(inicio) || (i > 0 && !p.Hora.After(c.Pasos[i-1].Hora)) {
			t.Errorf("hora del paso %d: %v", i, p.Hora)
		}
	}
}
