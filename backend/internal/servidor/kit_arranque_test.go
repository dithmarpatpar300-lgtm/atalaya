package servidor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"atalaya/internal/configurar"
)

// Copiar la skill: la vista previa trae cada archivo del kit con su ruta en ~/.claude/skills, al aplicar queda «al día»,
// pedirla otra vez no tiene nada que hacer y deshacer la quita.
func TestSkillPorLaPuerta(t *testing.T) {
	s, _ := conRegistro(t, "")
	s.Casa, s.Kit = t.TempDir(), t.TempDir()
	_ = os.MkdirAll(filepath.Join(s.Kit, "nivel-2", "scripts"), 0o755)
	_ = os.WriteFile(filepath.Join(s.Kit, "SKILL.md"), []byte("---\nname: atalaya-proyecto\n---\n"), 0o644)
	_ = os.WriteFile(filepath.Join(s.Kit, "nivel-2", "scripts", "claude.ps1"), []byte("# script\n"), 0o644)
	h := s.Handler()
	if got := s.estadoSkill("claude"); got != "falta" {
		t.Fatalf("antes = %q", got)
	}
	w := pedir(t, h, "POST", "/api/configurar/skill", `{"herramienta":"claude"}`, true)
	if w.Code != 200 {
		t.Fatalf("proponer: %d %s", w.Code, w.Body)
	}
	var prop configurar.Propuesta
	_ = json.Unmarshal(w.Body.Bytes(), &prop)
	destino := filepath.Join(s.Casa, ".claude", "skills", NombreSkill)
	if len(prop.Cambios) != 2 || !strings.HasPrefix(prop.Cambios[0].Ruta, destino) {
		t.Fatalf("cambios = %+v", prop.Cambios)
	}
	if w = pedir(t, h, "POST", "/api/configurar/aplicar", `{"id":"`+prop.ID+`"}`, true); w.Code != 200 {
		t.Fatalf("aplicar: %d %s", w.Code, w.Body)
	}
	if got := s.estadoSkill("claude"); got != "al día" {
		t.Fatalf("después = %q", got)
	}
	if w = pedir(t, h, "POST", "/api/configurar/skill", `{"herramienta":"claude"}`, true); w.Code != 409 {
		t.Fatalf("al día: %d %s", w.Code, w.Body)
	}
	// Fuera de la lista cerrada: la carpeta de skills de otra cosa, o un archivo de un proyecto.
	if s.permitidoKitArranque(filepath.Join(s.Casa, ".claude", "skills", "otra", "SKILL.md")) ||
		s.permitidoKitArranque(filepath.Join(s.Casa, ".claude", "skills", NombreSkill+"-x", "SKILL.md")) {
		t.Fatal("solo la carpeta de la skill de Atalaya")
	}
	if w = pedir(t, h, "POST", "/api/configurar/deshacer", `{"id":"`+prop.ID+`"}`, true); w.Code != 200 {
		t.Fatalf("deshacer: %d %s", w.Code, w.Body)
	}
	if got := s.estadoSkill("claude"); got != "falta" {
		t.Fatalf("deshecha = %q", got)
	}
}

// El arranque: solo la vista previa (aplicarla tocaría el inicio de este equipo). En Windows, una orden de PowerShell
// que crea el acceso directo en la carpeta de inicio, y su inversa lo borra.
func TestArranquePropuesta(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("la orden de la prueba es la de Windows")
	}
	s, _ := conRegistro(t, "")
	t.Setenv("APPDATA", t.TempDir())
	w := pedir(t, s.Handler(), "POST", "/api/configurar/arranque", `{"activar":true}`, true)
	if w.Code != 200 {
		t.Fatalf("proponer: %d %s", w.Code, w.Body)
	}
	var prop configurar.Propuesta
	_ = json.Unmarshal(w.Body.Bytes(), &prop)
	c := prop.Cambios[0]
	orden := strings.Join(c.Argumentos, " ")
	if c.Programa != "powershell" || !strings.Contains(orden, `Startup\Atalaya.lnk`) || !strings.Contains(strings.Join(c.Deshacer, " "), "Remove-Item") {
		t.Fatalf("cambio = %+v", c)
	}
	if w = pedir(t, s.Handler(), "POST", "/api/configurar/arranque", `{"activar":false}`, true); w.Code != 409 {
		t.Fatalf("desactivar lo que no está: %d %s", w.Code, w.Body)
	}
}
