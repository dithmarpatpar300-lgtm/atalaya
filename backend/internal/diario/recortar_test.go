package diario

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecortar(t *testing.T) {
	ahora := time.Date(2026, 10, 31, 12, 0, 0, 0, time.Local)
	for _, dias := range []int{30, 0} {
		t.Run(time.Duration(dias).String(), func(t *testing.T) {
			dir := t.TempDir()
			for _, edad := range []int{10, 30, 31, 400} {
				if err := os.WriteFile(ruta(dir, Fecha(ahora.AddDate(0, 0, -edad))), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for _, nombre := range []string{"notas.json", "2025-02-30.json"} {
				if err := os.WriteFile(filepath.Join(dir, nombre), []byte("{}"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			borrados, err := Recortar(dir, dias, ahora)
			esperados := 0
			if dias == 30 {
				esperados = 2
			}
			if err != nil || borrados != esperados {
				t.Fatalf("borrados: %d, error: %v", borrados, err)
			}
			for _, edad := range []int{10, 30, 31, 400} {
				_, err := os.Stat(ruta(dir, Fecha(ahora.AddDate(0, 0, -edad))))
				if debeExistir := dias == 0 || edad <= 30; (err == nil) != debeExistir {
					t.Errorf("archivo de hace %d días: %v", edad, err)
				}
			}
			for _, nombre := range []string{"notas.json", "2025-02-30.json"} {
				if _, err := os.Stat(filepath.Join(dir, nombre)); err != nil {
					t.Errorf("%s: %v", nombre, err)
				}
			}
		})
	}
	if n, err := Recortar(filepath.Join(t.TempDir(), "ausente"), 30, ahora); n != 0 || err != nil {
		t.Fatalf("carpeta ausente: %d %v", n, err)
	}
}
