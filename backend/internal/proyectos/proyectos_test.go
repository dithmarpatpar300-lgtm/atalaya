package proyectos

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParsear(t *testing.T) {
	ps, err := Parsear([]byte(`[
		{"nombre": "Tienda Demo", "ruta": "C:\\Proyectos\\Tienda Demo", "sigla": "TD", "nivel": 3},
		{"nombre": " Contador ", "ruta": "C:\\Proyectos\\Contador", "nivel": 0}
	]`))
	if err != nil {
		t.Fatal(err)
	}
	if len(ps) != 2 || ps[0].Sigla != "TD" || ps[0].Nivel != 3 || ps[1].Nombre != "Contador" || ps[1].Sigla != "CO" {
		t.Errorf("leído: %+v", ps)
	}
}

func TestParsearRechaza(t *testing.T) {
	for nombre, b := range map[string]string{
		"no es lista":    `{"nombre":"x"}`,
		"sin ruta":       `[{"nombre":"x"}]`,
		"repetido":       `[{"nombre":"X","ruta":"a"},{"nombre":"x","ruta":"b"}]`,
		"nivel fuera":    `[{"nombre":"x","ruta":"a","nivel":4}]`,
		"nivel negativo": `[{"nombre":"x","ruta":"a","nivel":-1}]`,
	} {
		if _, err := Parsear([]byte(b)); err == nil {
			t.Errorf("%s: debía fallar", nombre)
		}
	}
}

func TestLeerSinArchivo(t *testing.T) {
	_, err := Leer(filepath.Join(t.TempDir(), Archivo))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("sin archivo: %v", err)
	}
}

func TestSigla(t *testing.T) {
	for in, want := range map[string]string{"Tienda Demo": "TD", "Recetario-Demo": "RD", "Agenda Clinica Demo": "AC", "Atalaya": "AT", "Contador": "CO", "X": "X", "": "?", "ñandú azul": "ÑA"} {
		if got := Sigla(in); got != want {
			t.Errorf("Sigla(%q) = %q, quería %q", in, got, want)
		}
	}
}

func TestNombreDistinto(t *testing.T) {
	if NombreDistinto(Proyecto{Nombre: "Recetario-Demo", Ruta: `C:\Proyectos\Recetario-Demo\`}) {
		t.Error("el nombre es el de la carpeta")
	}
	if !NombreDistinto(Proyecto{Nombre: "Recetario Demo", Ruta: `C:\Proyectos\Recetario-Demo`}) {
		t.Error("el nombre no es el de la carpeta")
	}
}
