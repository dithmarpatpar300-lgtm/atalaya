package trazabilidad

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsearAvance(t *testing.T) {
	av := ParsearAvance("Avance: 85 % — check en verde\r\n\r\nDesglose: …\r\n- 21:32 · 5 % · leído el ADR\n- 22:08 · 100 % · informe hecho\n", "x.md")
	if av.Porcentaje != 85 || av.Texto != "check en verde" || len(av.Hitos) != 2 || av.Hitos[1].Hora != "22:08" {
		t.Errorf("%+v", av)
	}
	if av := ParsearAvance("sin avance", "y.md"); av.Porcentaje != -1 {
		t.Errorf("sin dato debería ser -1: %+v", av)
	}
}

func TestParsearFilaCodex(t *testing.T) {
	fila := "- 2026-10-30 · e2-c3max-sol · E2 · gpt-6.1-sol/high · codex-cli 0.166.0 · commit a8b0b0d + cambios sin commit (19 archivos nuevos) · 107 s · salida 0 · 377272 entrada (319872 en cache) / 3968 salida · contraste: —"
	ll, ok := ParsearFilaCodex(fila)
	if !ok {
		t.Fatal("no la leyó")
	}
	if ll.Tema != "e2-c3max-sol" || ll.Clase != "E2" || ll.Modelo != "gpt-6.1-sol" || ll.Effort != "high" || ll.Segundos != 107 || ll.Salida != 0 {
		t.Errorf("%+v", ll)
	}
	if ll.Entrada != 377272-319872 || ll.EnCache != 319872 || ll.TokSalida != 3968 || ll.Contraste != "—" {
		t.Errorf("tokens o contraste: %+v", ll)
	}
	if ll.Base != "commit a8b0b0d + cambios sin commit (19 archivos nuevos)" {
		t.Errorf("base %q", ll.Base)
	}
	// El contraste puede traer « · » propio.
	ll, ok = ParsearFilaCodex("- 2026-10-30 · r1 · R1 · luna/medium · cli · commit f · 34 s · salida 0 · 10 entrada (4 en cache) / 2 salida · contraste: bien · 0 falsos")
	if !ok || ll.Contraste != "bien · 0 falsos" {
		t.Errorf("%+v %v", ll, ok)
	}
	if _, ok := ParsearFilaCodex("Una linea por llamada"); ok {
		t.Error("el encabezado no es una fila")
	}
}

func TestLeerDocLimitaLaRuta(t *testing.T) {
	raiz := t.TempDir()
	_ = os.MkdirAll(filepath.Join(raiz, "docs"), 0o755)
	_ = os.WriteFile(filepath.Join(raiz, "docs", "a.md"), []byte("hola ghp_abcdefghijklmnopqrstuvwxyz0123"), 0o644)
	_ = os.WriteFile(filepath.Join(raiz, "secreto.md"), []byte("no"), 0o644)
	if s, err := LeerDoc(raiz, "docs/a.md"); err != nil || s != "hola gh_[tapado]" {
		t.Errorf("%q %v", s, err)
	}
	for _, mala := range []string{"secreto.md", "../x/docs/a.md", "docs/../secreto.md", "docs/a.txt", `C:\docs\a.md`, "docsx/a.md"} {
		if _, err := LeerDoc(raiz, mala); err == nil {
			t.Errorf("%q debería rechazarse", mala)
		}
	}
}
