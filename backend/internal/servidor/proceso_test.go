package servidor

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"atalaya/internal/claude"
)

// palabraEntera dice lo mismo que la expresión que reemplaza.
func TestPalabraEnteraComoLaExpresion(t *testing.T) {
	casos := [][2]string{
		{`.\scripts\codex.ps1 -tema e2-x -modelo a`, "e2-x"},
		{`.\scripts\codex.ps1 -tema e2-x-sol`, "e2-x"},
		{`codex.ps1 -tema e2-x-sol; codex.ps1 -tema e2-x`, "e2-x"},
		{`codex.ps1 -tema 'e2-x'`, "e2-x"},
		{`e2-x`, "e2-x"},
		{`ae2-x`, "e2-x"},
		{`e2-x.md`, "e2-x"},
		{`e2-x_b`, "e2-x"},
		{`sin tema`, "e2-x"},
		{`-tema "r1-Ñandú"`, "r1-Ñandú"},
	}
	for _, c := range casos {
		re := regexp.MustCompile(`(^|[^\w.-])` + regexp.QuoteMeta(c[1]) + `($|[^\w.-])`)
		if got, want := palabraEntera(c[0], c[1]), re.MatchString(c[0]); got != want {
			t.Errorf("%q en %q: %v, la expresión dice %v", c[1], c[0], got, want)
		}
	}
}

func TestSinCortadas(t *testing.T) {
	got := strings.Join(sinCortadas([]string{"2026-10-31-medidor-w6r", "2026-10-31-med", "2026-10-30-codex-e1"}), ",")
	if got != "2026-10-31-medidor-w6r,2026-10-30-codex-e1" {
		t.Errorf("%s", got)
	}
}

func TestLanzamientoDeCodex(t *testing.T) {
	h := time.Date(2026, 10, 31, 10, 0, 0, 0, time.UTC)
	ls := []claude.LanzamientoCodex{
		{Proyecto: "RG", Hora: h.Add(-time.Hour), Orden: `scripts\ops\codex.ps1 -Tema e1-w6a-terra -Clase E1`},
		{Proyecto: "RG", Hora: h.Add(-time.Minute), Orden: `scripts\ops\codex.ps1 -Tema e1-w6a-terra -Clase E1`},
		{Proyecto: "RG", Hora: h.Add(time.Hour), Orden: `scripts\ops\codex.ps1 -Tema e1-w6a-terra`},
		{Proyecto: "Otro", Hora: h, Orden: `codex.ps1 -Tema e1-w6a-terra`},
	}
	l, ok := lanzamientoDe(ls, "RG", "e1-w6a-terra", h)
	if !ok || !l.Hora.Equal(h.Add(-time.Minute)) {
		t.Errorf("el más cercano antes del inicio: %+v %v", l, ok)
	}
	if _, ok := lanzamientoDe(ls, "RG", "r1-otro", h); ok {
		t.Error("otro tema no debe enlazar")
	}
	if _, ok := lanzamientoDe(ls, "RG", "e1-w6a", h); ok {
		t.Error("un tema que es el comienzo de otro no debe enlazar")
	}
	if _, ok := lanzamientoDe([]claude.LanzamientoCodex{{Proyecto: "RG", Hora: h, Orden: `codex.ps1 -Tema "e1-w6a"`}}, "RG", "e1-w6a", h); !ok {
		t.Error("entre comillas sí enlaza")
	}
}
