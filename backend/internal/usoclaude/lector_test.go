package usoclaude

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestArranqueConHistorial(t *testing.T) {
	ahora := time.Now().Truncate(time.Second)
	dir := t.TempDir()
	filas := []fila{
		{T: ahora.Add(-21 * time.Minute).Unix(), Ventanas: map[string]float64{"session": 13, "weekly_all": 49}, Reinicios: map[string]int64{"session": ahora.Add(-time.Minute).Unix(), "weekly_all": ahora.Add(time.Hour).Unix()}},
		{T: ahora.Add(-time.Hour).Unix(), Ventanas: map[string]float64{"session": 2}, Reinicios: map[string]int64{"session": ahora.Unix()}},
	}
	f, err := os.Create(filepath.Join(dir, "uso-claude.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fila := range filas {
		if err := json.NewEncoder(f).Encode(fila); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	l := NuevoLector(dir, false)
	l.Exe = "falso"
	l.Ahora = func() time.Time { return ahora }
	e := l.Estado(2)
	if e.Estado != "viejo" || e.TomadaEn != filas[0].T || e.EdadS != 1260 || len(e.Ventanas) != 2 || !e.Ventanas[0].Vencida || e.Ventanas[1].Vencida {
		t.Fatalf("estado restaurado: %+v", e)
	}
	if len(e.Historial) != 2 || e.Historial[0].R["session"] != filas[0].Reinicios["session"] {
		t.Fatalf("historial: %+v", e.Historial)
	}
	l.leyendo = true
	if e = l.Estado(2); e.Estado != "viejo" {
		t.Fatalf("durante la lectura: %+v", e)
	}
	l.Ahora = func() time.Time { return ahora.Add(-2 * time.Minute) }
	if e = l.Estado(2); e.Estado != "ok" {
		t.Fatalf("lectura reciente: %+v", e)
	}
}

func TestRitmo(t *testing.T) {
	ahora := time.Date(2026, 11, 1, 14, 0, 0, 0, time.UTC)
	l := NuevoLector(t.TempDir(), true)
	l.Demo = false
	l.Exe = "falso"
	l.Ahora = func() time.Time { return ahora }
	l.inicio = ahora
	n := 0
	l.Correr = func(ctx context.Context, exe, dir string, args []string) ([]byte, error) {
		n++
		if _, ok := ctx.Deadline(); !ok {
			t.Error("sin plazo")
		}
		fs, e := os.ReadDir(dir)
		if e != nil || len(fs) != 0 {
			t.Error("temporal no vacío")
		}
		return []byte(salidaBuena), nil
	}
	if l.Toca() {
		t.Fatal("antes del minuto")
	}
	ahora = ahora.Add(time.Minute)
	if !l.Leer(context.Background(), false) || n != 1 {
		t.Fatal("primera")
	}
	if l.Leer(context.Background(), true) {
		t.Fatal("demasiado pronto")
	}
	ahora = ahora.Add(2 * time.Minute)
	if !l.Leer(context.Background(), true) {
		t.Fatal("forzada")
	}
	ahora = ahora.Add(9 * time.Minute)
	if l.Toca() {
		t.Fatal("antes de diez minutos")
	}
	ahora = ahora.Add(time.Minute)
	if !l.Toca() {
		t.Fatal("faltó lectura periódica")
	}
}

func TestResolverNpm(t *testing.T) {
	dir := t.TempDir()
	atajo := filepath.Join(dir, "claude.cmd")
	if e := os.WriteFile(atajo, nil, 0o600); e != nil {
		t.Fatal(e)
	}
	want := filepath.Join(dir, "node_modules", "@anthropic-ai", "claude-code", "bin", "claude.exe")
	if got := Resolver(atajo, "windows"); got != want {
		t.Fatal(got)
	}
	if got := Resolver(filepath.Join(dir, "claude"), "windows"); got != want {
		t.Fatal(got)
	}
}

func TestLecturaConcurrenteYLimite(t *testing.T) {
	ahora := time.Date(2026, 11, 1, 14, 0, 0, 0, time.UTC)
	l := NuevoLector(t.TempDir(), true)
	l.Demo = false
	l.Exe = "falso"
	l.Ahora = func() time.Time { return ahora }
	inicio, salir := make(chan struct{}), make(chan struct{})
	l.Correr = func(context.Context, string, string, []string) ([]byte, error) {
		close(inicio)
		<-salir
		return []byte(`{"type":"result","result":"rate limited until 3:10pm"}`), nil
	}
	termino := make(chan bool, 1)
	go func() { termino <- l.Leer(context.Background(), true) }()
	<-inicio
	if l.Estado(1).Estado != "leyendo" || l.Leer(context.Background(), true) {
		t.Fatal("lecturas simultáneas")
	}
	close(salir)
	<-termino
	ahora = ahora.Add(10 * time.Minute)
	if l.Toca() || l.Leer(context.Background(), true) {
		t.Fatal("no respetó el límite")
	}
}
