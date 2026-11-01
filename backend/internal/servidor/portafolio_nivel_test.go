package servidor

import "testing"

// El contrato pide lo que instala el kit de cada nivel: el 1 no lleva documentos de operación ni roles.
func TestContratoPorNivel(t *testing.T) {
	raiz := t.TempDir()
	ids := func(nivel int) []string {
		var out []string
		for _, c := range contrato(raiz, true, nivel, nil) {
			out = append(out, c.ID)
		}
		return out
	}
	if got := ids(1); len(got) != 5 || got[4] != "registro" {
		t.Errorf("nivel 1: %v", got)
	}
	for _, n := range []int{2, 3} {
		if got := ids(n); len(got) != 7 || got[4] != "operacion" || got[5] != "subagentes" {
			t.Errorf("nivel %d: %v", n, got)
		}
	}
}
