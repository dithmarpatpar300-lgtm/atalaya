package alertas

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// El contrato del kit: las reglas que Atalaya cita de AGENTS.md se encuentran igual en la plantilla de nivel 2 en
// inglés (la raíz del kit) y en español (es/). Si una traducción cambia una frase, esta prueba lo dice.
func TestPlantillaDelKitEnLasDosLenguas(t *testing.T) {
	kit := filepath.Join("..", "..", "..", "kit", "atalaya-proyecto")
	halladas := map[string]string{}
	for _, lengua := range []string{".", "es"} {
		// Como lo arma el instalador: la plantilla del nivel y, al final, la sección común de Atalaya.
		b, err := os.ReadFile(filepath.Join(kit, lengua, "nivel-2", "AGENTS.plantilla.md"))
		if err != nil {
			t.Fatalf("%s: %v", lengua, err)
		}
		comun, err := os.ReadFile(filepath.Join(kit, lengua, "comun", "AGENTS-atalaya.md"))
		if err != nil {
			t.Fatalf("%s: %v", lengua, err)
		}
		b = append(append(b, '\n'), comun...)
		// Lo que el instalador llena: un tope de verdad y una carpeta de temporales.
		texto := strings.NewReplacer("{{TOPE_MIN}}", "25", "{{TMP}}", `D:\tmp-agentes`).Replace(string(b))
		raiz := t.TempDir()
		if err := os.WriteFile(filepath.Join(raiz, "AGENTS.md"), []byte(texto), 0o644); err != nil {
			t.Fatal(err)
		}
		r := Leer(raiz)
		if r.Tope.Minutes() != 25 {
			t.Errorf("%s: tope %v", lengua, r.Tope)
		}
		var ids []string
		for _, regla := range r.Lista {
			if !regla.Omision {
				ids = append(ids, regla.ID)
			}
		}
		// Las del proceso se guardan aparte, con su fuente.
		for _, id := range []string{"asignacion", "effort", "cuota", "bitacora", "acta"} {
			if r.Fuente(id) != "" {
				ids = append(ids, id)
			}
		}
		if r.TmpRaiz != "" {
			ids = append(ids, "temporales")
		}
		halladas[lengua] = strings.Join(ids, ",")
	}
	// Las mismas reglas, y todas las del proceso que la plantilla escribe (contexto y quieto van siempre por omisión).
	if halladas["."] != halladas["es"] {
		t.Errorf("inglés %q, español %q", halladas["."], halladas["es"])
	}
	for _, id := range []string{"tiempo", "frontera", "asignacion", "effort", "cuota", "bitacora", "acta"} {
		if !strings.Contains(","+halladas["."]+",", ","+id+",") {
			t.Errorf("falta %q en la plantilla en inglés: %q", id, halladas["."])
		}
	}
}
