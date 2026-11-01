package datapedia

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func escribir(t *testing.T, ruta, texto string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(texto), 0o644); err != nil {
		t.Fatal(err)
	}
}

func preparar(t *testing.T) (*Indice, string, string) {
	t.Helper()
	base := t.TempDir()
	raiz := filepath.Join(base, "Proyectos", "Tienda Demo")
	memoria := filepath.Join(base, "claude")
	escribir(t, filepath.Join(raiz, "AGENTS.md"), "# Tienda Demo\n\nReglas del proyecto.")
	escribir(t, filepath.Join(raiz, "docs", "operacion", "AGENTES.md"), "# Centro de operaciones\n\nLa **bitácora** de cada agente lleva su avance.")
	escribir(t, filepath.Join(raiz, "docs", "trazabilidad", "F5", "avance", "codificador-p3.md"), "Avance: 100 % — entregado\n\n- 21:32 · 5 % · leído el ADR 0013")
	escribir(t, filepath.Join(raiz, "docs", "adr", "0013-reintentos.md"), "# ADR 0013 · Reintentos\n\nEl Retry-After máximo manda. URL https://x.example/a?sig=abcdefghijkl")
	escribir(t, filepath.Join(raiz, "docs", "notas.env.md"), "# no\n\nclave")
	escribir(t, filepath.Join(raiz, "docs", "notas.txt"), "no es markdown")
	escribir(t, filepath.Join(memoria, carpetaClaude(raiz), "memory", "cuotas.md"),
		"---\nname: cuotas-herramienta\ndescription: Mirar la cuota antes de repartir\nmetadata:\n  type: reference\n---\n\nCodex primero.")
	escribir(t, filepath.Join(memoria, "C--Proyectos-Agenda-Clinica-Demo", "memory", "entorno.md"), "---\nname: entorno\n---\nTodo en C:.")
	ix, err := Abrir(filepath.Join(base, "db", "atalaya.db"), memoria, []Proyecto{{"Tienda Demo", raiz}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ix.Cerrar() })
	return ix, raiz, memoria
}

func TestReindexarYBuscar(t *testing.T) {
	ix, raiz, _ := preparar(t)
	r, err := ix.Reindexar()
	if err != nil {
		t.Fatal(err)
	}
	if r.Nuevos != 6 || r.Total != 6 {
		t.Fatalf("debía indexar 6 (sin .env ni .txt): %+v", r)
	}

	// Sin tildes encuentra con tildes, y la última palabra vale como prefijo.
	docs, err := ix.Buscar(Filtro{Texto: "bitacora"})
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 || docs[0].Tipo != "regla" || !strings.Contains(docs[0].Fragmento, "\x02bitácora\x03") {
		t.Fatalf("bitacora: %+v", docs)
	}
	docs, _ = ix.Buscar(Filtro{Texto: "retry-aft"})
	if len(docs) != 1 || docs[0].Titulo != "ADR 0013 · Reintentos" || docs[0].Tipo != "adr" {
		t.Fatalf("retry-aft: %+v", docs)
	}
	// Lo que parece un operador de FTS5 se busca como texto y no rompe la consulta.
	if _, err := ix.Buscar(Filtro{Texto: `"AND (OR NEAR -* ^`}); err != nil {
		t.Fatalf("consulta con operadores: %v", err)
	}

	// La memoria: título y resumen del frontmatter, y el proyecto por la carpeta de Claude Code.
	docs, _ = ix.Buscar(Filtro{Tipo: "memoria"})
	if len(docs) != 2 {
		t.Fatalf("memoria: %+v", docs)
	}
	por := map[string]Documento{}
	for _, d := range docs {
		por[d.Proyecto] = d
	}
	if m := por["Tienda Demo"]; m.Titulo != "cuotas-herramienta" || m.Resumen != "Mirar la cuota antes de repartir" || m.Ruta != "memory/cuotas.md" {
		t.Errorf("memoria de Tienda Demo: %+v", m)
	}
	if _, ok := por["Agenda-Clinica-Demo"]; !ok {
		t.Errorf("la memoria de Agenda-Clinica-Demo debía tomar el nombre de su carpeta: %+v", docs)
	}

	// El texto guardado ya va con los secretos tapados.
	id, err := ix.PorRuta("Tienda Demo", "docs/adr/0013-reintentos.md")
	if err != nil {
		t.Fatal(err)
	}
	d, err := ix.Documento(id)
	if err != nil || strings.Contains(d.Cuerpo, "abcdefghijkl") || !strings.Contains(d.Cuerpo, "sig=[tapado]") {
		t.Fatalf("documento: %+v %v", d, err)
	}

	cuentas, _ := ix.Cuentas("", "Tienda Demo")
	total := 0
	for _, c := range cuentas {
		total += c.Cantidad
	}
	if total != 5 {
		t.Errorf("cuentas de Tienda Demo: %+v", cuentas)
	}

	// Un segundo pase sin cambios no toca nada; un cambio y un borrado sí.
	if r, _ := ix.Reindexar(); r.Cambio() {
		t.Errorf("sin cambios: %+v", r)
	}
	adr := filepath.Join(raiz, "docs", "adr", "0013-reintentos.md")
	escribir(t, adr, "# ADR 0013 · Reintentos\n\nAhora habla de jitter.")
	_ = os.Chtimes(adr, time.Now().Add(time.Minute), time.Now().Add(time.Minute))
	_ = os.Remove(filepath.Join(raiz, "AGENTS.md"))
	r, _ = ix.Reindexar()
	if r.Cambiados != 1 || r.Borrados != 1 || r.Nuevos != 0 {
		t.Errorf("tras cambiar y borrar: %+v", r)
	}
	if docs, _ := ix.Buscar(Filtro{Texto: "jitter"}); len(docs) != 1 {
		t.Errorf("el texto nuevo no se indexó: %+v", docs)
	}
	if docs, _ := ix.Buscar(Filtro{Texto: "Retry"}); len(docs) != 0 {
		t.Errorf("el texto viejo quedó en el índice: %+v", docs)
	}
}

func TestVirtuales(t *testing.T) {
	ix, _, _ := preparar(t)
	cuando := time.Unix(1_000_000, 0)
	cuerpo := "# Codificador · turno 2\n\n- 22:27 · **Edita** · `crates/pedidos.rs` — ↩ vuelve a `crates/pedidos.rs`"
	ix.Virtuales = func() []Virtual {
		return []Virtual{{Clave: "traza:a~b:2", Proyecto: "Tienda Demo", Tipo: "traza", Ruta: "trazas/a~b/turno-2", Titulo: "Codificador · turno 2", Resumen: "1 paso", Cuerpo: cuerpo, Mtime: cuando}}
	}
	if r, err := ix.Reindexar(); err != nil || r.Nuevos != 7 {
		t.Fatalf("%+v %v", r, err)
	}
	docs, _ := ix.Buscar(Filtro{Texto: "pedidos", Tipo: "traza"})
	if len(docs) != 1 || docs[0].Titulo != "Codificador · turno 2" || docs[0].Resumen != "1 paso" || docs[0].Ruta != "trazas/a~b/turno-2" {
		t.Fatalf("%+v", docs)
	}
	// Igual fecha y tamaño: no se toca. Si la traza desaparece, se borra.
	if r, _ := ix.Reindexar(); r.Cambio() {
		t.Errorf("sin cambios: %+v", r)
	}
	ix.Virtuales = func() []Virtual { return nil }
	if r, _ := ix.Reindexar(); r.Borrados != 1 {
		t.Errorf("la traza debía borrarse: %+v", r)
	}
}

func TestTipoDe(t *testing.T) {
	casos := map[string]string{
		"docs/trazabilidad/F5/avance/x.md":     "bitacora",
		"docs/trazabilidad/F5/temporales/x.md": "acta",
		"docs/trazabilidad/F5/codex/x.md":      "codex",
		"docs/trazabilidad/F5/08-p3.md":        "informe",
		"docs/adr/0001.md":                     "adr",
		"docs/archivo/adr/0001.md":             "archivo",
		"docs/02-motor.md":                     "documento",
	}
	for ruta, want := range casos {
		if got := TipoDe(ruta); got != want {
			t.Errorf("%s: %s, quería %s", ruta, got, want)
		}
	}
}

func TestConsulta(t *testing.T) {
	if got := Consulta(`retry-after "máx`); got != `"retry" "after" "máx"*` {
		t.Errorf("%q", got)
	}
	if Consulta(` -*" `) != "" {
		t.Error("sin palabras debía quedar vacía")
	}
}

func TestCambiarProyectos(t *testing.T) {
	ix, raiz, memoria := preparar(t)
	if _, err := ix.Reindexar(); err != nil {
		t.Fatal(err)
	}
	nueva := filepath.Join(t.TempDir(), "Atalaya")
	escribir(t, filepath.Join(nueva, "README.md"), "# Atalaya\n\nDocumento nuevo.")
	escribir(t, filepath.Join(memoria, carpetaClaude(nueva), "memory", "notas.md"), "# Memoria\n\nContenido.")
	proyectos := []Proyecto{{"Tienda Demo", raiz}, {"Atalaya", nueva}}
	ix.CambiarProyectos(proyectos)
	proyectos[1].Nombre = "Alterado"
	r, err := ix.Reindexar()
	if err != nil || r.Nuevos != 2 {
		t.Fatalf("proyecto agregado: %+v, %v", r, err)
	}
	docs, err := ix.Buscar(Filtro{Proyecto: "Atalaya"})
	if err != nil || len(docs) != 2 {
		t.Fatalf("documentos del proyecto: %+v, %v", docs, err)
	}
	mems, err := ix.Buscar(Filtro{Proyecto: "Atalaya", Tipo: "memoria"})
	if err != nil || len(mems) != 1 {
		t.Fatalf("memoria del proyecto: %+v, %v", mems, err)
	}
	ix.CambiarProyectos([]Proyecto{{"Renombrado", nueva}})
	r, err = ix.Reindexar()
	if err != nil || r.Cambiados != 3 || r.Borrados != 4 {
		t.Fatalf("proyecto renombrado y retirado: %+v, %v", r, err)
	}
	docs, err = ix.Buscar(Filtro{Proyecto: "Renombrado"})
	if err != nil || len(docs) != 2 {
		t.Fatalf("documentos renombrados: %+v, %v", docs, err)
	}
}

func TestCarpetaMemoriaProyectoRegistrado(t *testing.T) {
	ix, _, memoria := preparar(t)
	raiz := `C:\Proyectos\Atalaya`
	escribir(t, filepath.Join(memoria, "C--Proyectos-Atalaya", "memory", "nota.md"), "# Nota")
	ix.CambiarProyectos([]Proyecto{{Nombre: "Atalaya", Raiz: raiz}})
	if _, err := ix.Reindexar(); err != nil {
		t.Fatal(err)
	}
	docs, err := ix.Buscar(Filtro{Proyecto: "Atalaya", Tipo: "memoria"})
	if err != nil || len(docs) != 1 {
		t.Fatalf("memoria: %+v, %v", docs, err)
	}
}
