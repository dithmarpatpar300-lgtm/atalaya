package avance

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// El contrato del kit: el AVANCE.md de la fase 0, llenado como lo llena el instalador, da la misma fase en inglés (la
// raíz del kit) y en español (es/), con los paquetes en sus estados.
func TestAvanceDelKitEnLasDosLenguas(t *testing.T) {
	kit := filepath.Join("..", "..", "..", "kit", "atalaya-proyecto")
	paquetes := map[string]string{
		".":  "| K0 | Install the kit | 2 | done | 100 | AGENTS.md |\n| K1 | First package | 4 | in progress | 50 | |\n| K2 | Second | 2 | not started | 0 | |",
		"es": "| K0 | Instalar el kit | 2 | hecho | 100 | AGENTS.md |\n| K1 | Primer paquete | 4 | en curso | 50 | |\n| K2 | Segundo | 2 | sin empezar | 0 | |",
	}
	var fases []Fase
	for _, lengua := range []string{".", "es"} {
		b, err := os.ReadFile(filepath.Join(kit, lengua, "comun", "docs", "trazabilidad", "F0", "AVANCE.md"))
		if err != nil {
			t.Fatalf("%s: %v", lengua, err)
		}
		texto := strings.NewReplacer("{{AHORA}}", "x", "{{PLAN}}", "docs/plan.md", "{{QUE_ES_LA_FASE}}", "",
			"{{PAQUETES}}", paquetes[lengua], "{{HORA_DE_INSTALACION}}", "10:00", "{{NIVEL}}", "2").Replace(string(b))
		f := Parsear(texto, "F0", "x")
		// Los nombres de los paquetes cambian con la lengua; el resto no.
		for i := range f.Paquetes {
			f.Paquetes[i].Que, f.Paquetes[i].Evidencia = "", ""
		}
		f.Hitos, f.Ahora = nil, ""
		fases = append(fases, f)
	}
	if len(fases[0].Paquetes) != 3 || fases[0].Paquetes[1].Estado != EnCurso {
		t.Fatalf("inglés: %+v", fases[0])
	}
	if !reflect.DeepEqual(fases[0], fases[1]) {
		t.Errorf("inglés %+v\nespañol %+v", fases[0], fases[1])
	}
}
