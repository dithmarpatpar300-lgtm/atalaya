package diario

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGuardarConservaLoBorrado(t *testing.T) {
	dir := t.TempDir()
	t0 := time.Date(2026, 10, 31, 10, 0, 0, 0, time.Local)
	ayer := Dia{Fecha: "2026-10-31",
		Agentes:    []Agente{{ID: "a", Nombre: "codificador-w6a", Inicio: t0, Minutos: 12}, {ID: "b", Nombre: "auditor", Inicio: t0.Add(time.Hour), Activo: true, Minutos: 3}},
		Decisiones: []Decision{{ID: "d1", Hora: t0, Estado: "pendiente"}},
		Alertas:    []Alerta{{ID: "tiempo:a:pasado", Hora: t0.Add(30 * time.Minute)}}}
	if ok, err := Guardar(dir, ayer); err != nil || !ok {
		t.Fatalf("primera escritura: %v %v", ok, err)
	}
	if ok, _ := Guardar(dir, ayer); ok {
		t.Error("sin cambios no se reescribe")
	}

	// Al día siguiente, el transcrito de «a» se borró: la memoria solo trae «b», ya terminado, y la decisión
	// respondida; la alerta se vuelve a ver más tarde.
	hoy := Dia{Fecha: "2026-10-31",
		Agentes:    []Agente{{ID: "b", Nombre: "auditor", Inicio: t0.Add(time.Hour), Minutos: 20}},
		Decisiones: []Decision{{ID: "d1", Hora: t0, Estado: "respondida"}},
		Alertas:    []Alerta{{ID: "tiempo:a:pasado", Hora: t0.Add(2 * time.Hour)}}}
	if ok, err := Guardar(dir, hoy); err != nil || !ok {
		t.Fatalf("segunda escritura: %v %v", ok, err)
	}
	d, err := Leer(dir, "2026-10-31")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Agentes) != 2 || d.Agentes[0].ID != "a" || d.Agentes[1].Minutos != 20 || d.Agentes[1].Activo {
		t.Errorf("agentes: %+v", d.Agentes)
	}
	if d.Decisiones[0].Estado != "respondida" {
		t.Errorf("la decisión se actualiza: %+v", d.Decisiones)
	}
	if !d.Alertas[0].Hora.Equal(t0.Add(30 * time.Minute)) {
		t.Errorf("la alerta guarda la primera vez que se vio: %v", d.Alertas[0].Hora)
	}
	if _, err := os.Stat(filepath.Join(dir, "2026-10-31.json.tmp")); !os.IsNotExist(err) {
		t.Error("no queda el archivo temporal")
	}
}

func TestFechas(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"2026-10-30.json", "2026-10-31.json", "avisos.json", "2026-10-29.json.tmp"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("{}"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got := Fechas(dir)
	if len(got) != 2 || got[0] != "2026-10-31" || got[1] != "2026-10-30" {
		t.Errorf("%v", got)
	}
	if _, err := Leer(dir, "../avisos"); err == nil {
		t.Error("una fecha que no es AAAA-MM-DD no se lee (ni sale de la carpeta)")
	}
	if d, err := Leer(dir, "2026-01-01"); err != nil || d.Fecha != "2026-01-01" || len(d.Agentes) != 0 {
		t.Errorf("un día sin archivo es un día vacío: %+v %v", d, err)
	}
}
