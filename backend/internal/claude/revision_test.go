package claude

import (
	"strings"
	"testing"
	"time"
)

func TestLineaAsignacion(t *testing.T) {
	casos := map[string]int{
		"**Asignación: nivel enfocado · clase I2**\n\nTarea…":                    1,
		"\n\nAsignación: nivel ligero":                                           1,
		"Lee `.claude/agents/codificador.md`.\n\n**Asignación: nivel enfocado**": 2,
		"Sin asignación aquí.\nNi aquí.":                                         0,
		"Read AGENTS.md.\n\n**Assignment: level focused · class I1**":            2,
	}
	for enc, want := range casos {
		if got := lineaAsignacion(enc); got != want {
			t.Errorf("%q: %d, quería %d", enc, got, want)
		}
	}
}

func TestMiradaYTemporales(t *testing.T) {
	h := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	pasos := []*Paso{
		{N: 0, Hora: h, Tipo: "Ejecuta", orden: `& "$env:LOCALAPPDATA\Cuotas\cuotas.exe" una-vez --json`},
		{N: 1, Hora: h.Add(time.Minute), Tipo: "Ejecuta", orden: `New-Item -ItemType Directory D:\TiendaDemo-tmp\2026-10-31-medidor-w6\datos`},
		{N: 2, Hora: h.Add(2 * time.Minute), Tipo: "Escribe", Hilo: `D:/TiendaDemo-tmp/2026-10-31-medidor-w6/notas.md`},
		{N: 3, Hora: h.Add(3 * time.Minute), Tipo: "Delega"},
		{N: 4, Hora: h.Add(4 * time.Minute), Tipo: "Usa", Herramienta: "mcp__ccd_session_mgmt__get_usage"},
		{N: 5, Hora: h.Add(5 * time.Minute), Tipo: "Ejecuta", orden: `Get-ChildItem C:\Users\x\AppData\Local\Temp\TiendaDemo-tmp\2026-10-30-codex-e1.`},
	}
	if m := miradaAntes(pasos, 3); m.Paso != 0 || !m.Hora.Equal(h) {
		t.Errorf("mirada antes del paso 3: %+v", m)
	}
	if m := miradaAntes(pasos, 6); m.Paso != 4 {
		t.Errorf("get_usage también cuenta: %+v", m)
	}
	if m := miradaAntes(pasos, 0); !m.Hora.IsZero() {
		t.Errorf("sin mirada: %+v", m)
	}
	if got := strings.Join(temporalesDe(pasos), ","); got != "2026-10-31-medidor-w6,2026-10-30-codex-e1" {
		t.Errorf("temporales: %s", got)
	}
}

// LanzamientosCodex y Revisiones calculan la mirada en una sola pasada: tienen que dar lo mismo que miradaAntes.
func TestMiradaEnUnaPasada(t *testing.T) {
	h := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	pasos := []*Paso{
		{N: 0, Hora: h, Tipo: "Ejecuta", orden: `cuotas.exe una-vez`},
		{N: 1, Hora: h.Add(time.Minute), Tipo: "Ejecuta", orden: `.\scripts\codex.ps1 -Tema a`},
		{N: 2, Hora: h.Add(2 * time.Minute), Tipo: "Usa", Herramienta: "mcp__ccd_session_mgmt__get_usage"},
		{N: 3, Hora: h.Add(3 * time.Minute), Tipo: "Delega", idUso: "t1"},
		{N: 4, Hora: h.Add(4 * time.Minute), Tipo: "Ejecuta", orden: `.\scripts\codex.ps1 -Tema b`},
	}
	ses := &archivo{ruta: "ses.jsonl", agente: &Agente{ID: "ses", Tipo: "sesion", Proyecto: "Tienda", Ultimo: h.Add(time.Hour)}, pasos: pasos}
	sub := &archivo{ruta: "sub.jsonl", agente: &Agente{ID: "ses~a", Tipo: "subagente", Padre: "ses", ToolUseID: "t1", Pasos: 1, Ultimo: h.Add(time.Hour)}}
	ix := &Indice{arch: map[string]*archivo{"ses": ses, "sub": sub}}

	ls := ix.LanzamientosCodex(time.Time{})
	if len(ls) != 2 {
		t.Fatalf("lanzamientos: %+v", ls)
	}
	for _, l := range ls {
		if want := miradaAntes(pasos, l.Paso); l.Mirada != want {
			t.Errorf("lanzamiento del paso %d: mirada %+v, quiero %+v", l.Paso, l.Mirada, want)
		}
		if !strings.Contains(l.OrdenNorm, "codex.ps1") {
			t.Errorf("sin la orden normalizada: %+v", l)
		}
	}
	r := ix.Revisiones(time.Time{})["ses~a"]
	if r.PasoLanzado != 3 || r.Mirada != miradaAntes(pasos, 3) || r.Mirada.Paso != 2 {
		t.Errorf("revisión: %+v", r)
	}
}

// Las marcas que agregar guarda en el paso dicen lo mismo que la regla calculada en el momento, y un paso sin marcar
// (que no pasó por agregar) las calcula igual.
func TestMarcasDelPaso(t *testing.T) {
	casos := []struct {
		p    Paso
		want uint8
	}{
		{Paso{Tipo: "Ejecuta", orden: `cuotas.exe una-vez`}, marcaCuota},
		{Paso{Tipo: "Usa", Herramienta: "mcp__ccd_session_mgmt__get_usage"}, marcaCuota},
		{Paso{Tipo: "Ejecuta", orden: `git -C 'D:\x' commit -F msg.txt`}, marcaCommit},
		{Paso{Tipo: "Ejecuta", orden: `.\scripts\codex.ps1 -Tema a`}, marcaCodex},
		{Paso{Tipo: "Edita", editado: "d:/p/docs/trazabilidad/estado-para-retomar.md"}, marcaDocEstado},
		{Paso{Tipo: "Edita", editado: "d:/p/backend/main.go"}, 0},
		{Paso{Tipo: "Lee", orden: "git commit"}, 0}, // solo un «Ejecuta» comitea
	}
	a := &archivo{agente: &Agente{Tipo: "sesion"}}
	for i, c := range casos {
		suelto := c.p
		agregado := c.p
		a.agregar(&agregado)
		for _, m := range []uint8{marcaCuota, marcaCommit, marcaCodex, marcaDocEstado} {
			want := c.want&m != 0
			if got := suelto.tiene(m); got != want {
				t.Errorf("caso %d sin marcar, marca %d: %v", i, m, got)
			}
			if got := agregado.tiene(m); got != want {
				t.Errorf("caso %d agregado, marca %d: %v", i, m, got)
			}
		}
		if agregado.marcas&marcado == 0 {
			t.Errorf("caso %d: agregar no lo marcó", i)
		}
	}
}
