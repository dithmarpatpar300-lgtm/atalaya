package claude

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMapa(t *testing.T) {
	raiz := t.TempDir()
	dir := filepath.Join(raiz, "C--proyecto")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	linea := `{"type":"user","timestamp":"2026-10-31T17:20:24Z","message":{"role":"user","content":"Hola"}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "ses1.jsonl"), []byte(strings.Repeat(linea, 250)), 0o644); err != nil {
		t.Fatal(err)
	}
	ix := NuevoIndice(raiz)
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	var a *archivo
	for _, x := range ix.arch {
		a = x
	}
	if a == nil || len(a.pasos) != 250 {
		t.Fatalf("transcrito: %+v", a)
	}
	a.pasos[0].Tipo = "Edita"
	a.pasos[1].Tipo = "Escribe"
	a.pasos[1].Error = true
	m, ok := ix.Mapa("ses1")
	if !ok || m.Total != 250 || m.PorCelda != 3 || len(m.Celdas) != 84 {
		t.Fatalf("mapa: %+v, existe: %v", m, ok)
	}
	primera := m.Celdas[0]
	hora := time.Date(2026, 10, 31, 17, 20, 24, 0, time.UTC)
	if primera.N != 0 || !primera.Hora.Equal(hora) || primera.Pasos != 3 || primera.Edita != 0.67 || primera.Errores != 1 || primera.Pedidos != 1 {
		t.Errorf("primera celda: %+v", primera)
	}
	suma := 0
	for _, c := range m.Celdas {
		suma += c.Pasos
	}
	ultima := m.Celdas[len(m.Celdas)-1]
	if suma != 250 || ultima.N != 249 || ultima.Pasos != 1 {
		t.Errorf("suma %d, última: %+v", suma, ultima)
	}
	a.pasos = nil
	vacio, ok := ix.Mapa("ses1")
	if !ok || vacio.Total != 0 || vacio.PorCelda != 1 || vacio.Celdas == nil || len(vacio.Celdas) != 0 {
		t.Errorf("vacío: %+v, existe: %v", vacio, ok)
	}
	if _, ok := ix.Mapa("ausente"); ok {
		t.Error("el agente ausente no debe existir")
	}
}
