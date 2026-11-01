package usoclaude

import (
	"path/filepath"
	"testing"
	"time"
)

func TestHistorial(t *testing.T) {
	ahora := time.Now()
	ruta := filepath.Join(t.TempDir(), "uso-claude.jsonl")
	fs := []fila{{T: ahora.Add(-31 * 24 * time.Hour).Unix(), Ventanas: map[string]float64{"session": 1}}, {T: ahora.Unix(), Ventanas: map[string]float64{"session": 14}, Reinicios: map[string]int64{"session": ahora.Unix() + 100}}}
	if _, e := guardar(ruta, fs, ahora); e != nil {
		t.Fatal(e)
	}
	got, e := cargar(ruta)
	if e != nil || len(got) != 1 || got[0].Ventanas["session"] != 14 {
		t.Fatal(got, e)
	}
}

func TestCargaYFiltro(t *testing.T) {
	ahora := time.Now()
	dir := t.TempDir()
	ruta := filepath.Join(dir, "uso-claude.jsonl")
	filas := []fila{{T: ahora.Add(-2 * time.Hour).Unix(), Ventanas: map[string]float64{"session": 1}}, {T: ahora.Unix(), Ventanas: map[string]float64{"session": 2}}}
	if _, err := guardar(ruta, filas, ahora); err != nil {
		t.Fatal(err)
	}
	l := NuevoLector(dir, false)
	l.Ahora = func() time.Time { return ahora }
	h := l.Estado(1).Historial
	if len(h) != 1 || h[0].V["session"] != 2 {
		t.Fatal(h)
	}
}
