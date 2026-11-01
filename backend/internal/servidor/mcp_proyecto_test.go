package servidor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atalaya/internal/codexvivo"
)

// Los tres fallos que destapó la medición del MCP (2026-10-31): el proyecto escrito como lo nombra el panel, una
// llamada a Codex pedida por su nombre y los archivos que Codex cambió con su parche.
func TestMCPLoQueEscribeUnAgente(t *testing.T) {
	raiz := t.TempDir()
	codexDir := filepath.Join(raiz, "docs", "trazabilidad", "F2", "codex")
	if err := os.MkdirAll(codexDir, 0o755); err != nil {
		t.Fatal(err)
	}
	parche := "diff --git a/backend/internal/servidor/registro.go b/backend/internal/servidor/registro.go\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a\n+b\n" +
		"diff --git a/backend/internal/datapedia/datapedia.go b/backend/internal/datapedia/datapedia.go\n"
	if err := os.WriteFile(filepath.Join(codexDir, "2026-10-31-p6-datapedia.patch"), []byte(parche), 0o644); err != nil {
		t.Fatal(err)
	}
	vivo := t.TempDir()
	base := filepath.Join(vivo, "2026-10-31-p6-datapedia")
	meta := `{"tema":"p6-datapedia","clase":"I1","modelo":"gpt-6.1-terra","fase":"F2","raiz":` + jsonCadena(raiz) +
		`,"inicio":"2026-10-31T17:58:00-05:00","fin":"2026-10-31T18:01:00-05:00","salida":0,"segundos":130}`
	if err := os.WriteFile(base+".meta.json", []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base+".jsonl", []byte(`{"type":"item.completed","item":{"id":"a","type":"agent_message","text":"Listo"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := nuevo(t)
	nombre := filepath.Base(raiz)
	s.Proyectos = []Proyecto{{Nombre: "Agenda-Clinica", Raiz: `C:\Proyectos\Agenda-Clinica`, Sigla: "AC"}, {Nombre: nombre, Raiz: raiz, Sigla: "PR"}}
	s.CodexVivo = codexvivo.NuevoLector(vivo)
	if _, err := s.CodexVivo.Actualizar(); err != nil {
		t.Fatal(err)
	}

	// Como lo escribe un agente: con palabras vacías, tildes, espacios, otra caja o la sigla.
	for entra, sale := range map[string]string{"Agenda de clínica": "Agenda-Clinica", "agenda clinica": "Agenda-Clinica", "ac": "Agenda-Clinica", "Agenda-Clinica": "Agenda-Clinica"} {
		if got, ok := s.proyectoMCP(entra); !ok || got != sale {
			t.Errorf("%q → %q, %v; quiere %q", entra, got, ok, sale)
		}
	}
	tx, fallo := s.llamar("archivo", []byte(`{"ruta":"registro.go","proyecto":"Tienda Demor"}`), "prueba", "")
	if !fallo || !strings.Contains(tx, "No conozco el proyecto «Tienda Demor»") || !strings.Contains(tx, "Agenda-Clinica") {
		t.Errorf("un proyecto que no existe debe decirse, no responder «ninguno»: %s", tx)
	}

	tx, fallo = s.llamar("que_hizo", []byte(`{"agente":"Codex · p6-datapedia","proyecto":"PR"}`), "prueba", "")
	if fallo || !strings.Contains(tx, "id: codexvivo~2026-10-31-p6-datapedia") || !strings.Contains(tx, "Archivos de su parche (2): backend/internal/servidor/registro.go") {
		t.Errorf("que_hizo de una llamada a Codex por su nombre: %s", tx)
	}

	tx, fallo = s.llamar("archivo", []byte(`{"ruta":"servidor/registro.go"}`), "prueba", "")
	if fallo || !strings.Contains(tx, "Codex · p6-datapedia") || !strings.Contains(tx, "editó 1") {
		t.Errorf("archivo debe ver lo que cambió el parche de Codex: %s", tx)
	}
}

func jsonCadena(s string) string { return `"` + strings.ReplaceAll(s, `\`, `\\`) + `"` }
