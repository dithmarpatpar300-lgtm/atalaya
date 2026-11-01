package servidor

import (
	"testing"

	"atalaya/internal/claude"
)

// Las dos fuentes de decisiones se juntan sin ids repetidos y en su orden: se queda la primera.
func TestJuntarDecisiones(t *testing.T) {
	transcritos := []claude.Decision{{ID: "s~4", Tipo: "botones"}, {ID: "s~9"}}
	centro := []claude.Decision{{ID: "s~4", Tipo: "atalaya"}, {ID: "dec-1"}}
	got := juntarDecisiones(transcritos, centro)
	if len(got) != 3 || got[0].ID != "s~4" || got[0].Tipo != "botones" || got[1].ID != "s~9" || got[2].ID != "dec-1" {
		t.Errorf("juntas: %+v", got)
	}
	if len(juntarDecisiones(nil, nil)) != 0 {
		t.Error("sin decisiones")
	}
}
