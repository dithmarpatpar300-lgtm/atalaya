package datapedia

import (
	"path/filepath"
	"testing"
)

func TestProyectosFuera(t *testing.T) {
	base := t.TempDir()
	raizA, raizB := filepath.Join(base, "a"), filepath.Join(base, "b")
	escribir(t, filepath.Join(raizA, "README.md"), "# Proyecto A\n\nPalabra compartida.")
	escribir(t, filepath.Join(raizB, "README.md"), "# Proyecto B\n\nPalabra compartida.")
	memoria := filepath.Join(base, "memoria")
	escribir(t, filepath.Join(memoria, carpetaClaude(raizA), "memory", "nota.md"), "# Palabra compartida")
	ix, err := Abrir(filepath.Join(base, "db", "indice.db"), memoria, []Proyecto{{"A", raizA}, {"B", raizB}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Cerrar() })
	ix.Virtuales = func() []Virtual {
		return []Virtual{{Clave: "traza:a", Proyecto: "A", Cuerpo: "Palabra compartida"}}
	}
	if r, err := ix.Reindexar(); err != nil || r.Nuevos != 4 {
		t.Fatalf("primer reindexado: %+v %v", r, err)
	}
	ix.Fuera = func(proyecto string) bool { return proyecto == "A" }
	if r, err := ix.Reindexar(); err != nil || r.Borrados != 3 {
		t.Fatalf("reindexado con A fuera: %+v %v", r, err)
	}
	docs, err := ix.Buscar(Filtro{Texto: "compartida"})
	if err != nil || len(docs) != 1 || docs[0].Proyecto != "B" {
		t.Fatalf("documentos restantes: %+v %v", docs, err)
	}
}
