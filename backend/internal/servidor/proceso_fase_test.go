package servidor

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFaseDeAgente(t *testing.T) {
	raiz := t.TempDir()
	base := filepath.Join(raiz, "docs", "trazabilidad")
	for _, fase := range []string{"F1", "F2", "F3"} {
		if err := os.MkdirAll(filepath.Join(base, fase), 0755); err != nil {
			t.Fatal(err)
		}
	}
	inicio := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	for _, archivo := range []struct {
		ruta string
		hora time.Time
	}{
		{"F1/viejo.md", inicio.Add(-48 * time.Hour)},
		{"F2/AVANCE.md", inicio.Add(time.Hour)},
		{"F2/inicio.md", inicio},
	} {
		ruta := filepath.Join(base, filepath.FromSlash(archivo.ruta))
		if err := os.WriteFile(ruta, []byte("Avance: 0 %\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(ruta, archivo.hora, archivo.hora); err != nil {
			t.Fatal(err)
		}
	}
	if got := faseActual(raiz); got != "F2" {
		t.Fatalf("faseActual = %q, quiero F2", got)
	}
	for _, caso := range []struct {
		nombre string
		inicio time.Time
		ruta   string
		quiere string
	}{
		{"ruta anterior", inicio.Add(time.Hour), "docs/trazabilidad/F1/informe.md", "F1"},
		{"después", inicio.Add(time.Minute), "", "F2"},
		{"antes", inicio.Add(-time.Minute), "", ""},
		{"al inicio", inicio, "", ""},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if got := faseDeAgente(raiz, caso.inicio, caso.ruta); got != caso.quiere {
				t.Errorf("faseDeAgente = %q, quiero %q", got, caso.quiere)
			}
		})
	}
}
