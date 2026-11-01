package servidor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReleerRegistro(t *testing.T) {
	ruta := filepath.Join(t.TempDir(), "proyectos.json")
	if err := os.WriteFile(ruta, []byte("[1]"), 0o600); err != nil {
		t.Fatal(err)
	}
	var siguiente []Proyecto
	var fallo error
	s := &Servidor{Proyectos: []Proyecto{{Nombre: "A"}}, RutaRegistro: ruta,
		Registro: func() ([]Proyecto, error) { return siguiente, fallo }}

	if s.releerRegistro() {
		t.Error("la primera vuelta solo toma la firma: la lista del arranque ya está leída")
	}
	cambiar := func(texto string) {
		if err := os.WriteFile(ruta, []byte(texto), 0o600); err != nil {
			t.Fatal(err)
		}
		// La hora del archivo puede no moverse dentro del mismo tick: el tamaño sí.
		_ = os.Chtimes(ruta, time.Now(), time.Now().Add(time.Second))
	}

	siguiente = []Proyecto{{Nombre: "A"}, {Nombre: "B"}}
	cambiar("[1, 2]")
	if !s.releerRegistro() || len(s.listaProyectos()) != 2 {
		t.Fatalf("tras registrar B: %+v", s.listaProyectos())
	}
	if s.releerRegistro() {
		t.Error("sin cambios no hay relectura")
	}

	fallo = errors.New("roto")
	cambiar("[1, 2, 3]")
	if s.releerRegistro() || len(s.listaProyectos()) != 2 {
		t.Errorf("un registro roto deja la lista anterior: %+v", s.listaProyectos())
	}
}
