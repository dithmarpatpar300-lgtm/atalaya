package delegadas

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/pruebas"
)

// comillas: un texto como valor de JSON.
func comillas(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func escribir(t *testing.T, ruta, texto string) {
	t.Helper()
	if err := os.WriteFile(ruta, []byte(texto), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Una sesión de Claude Code lanzada por claude.ps1 desde Codex: se cuelga de la sesión de Codex, es del repositorio
// (no de la copia) y se cierra cuando la ficha dice que terminó.
func TestDelegacionDesdeCodex(t *testing.T) {
	dir := t.TempDir()
	raiz := pruebas.Nativa(`C:\repos\Tienda`)
	ruta := filepath.Join(dir, "2026-11-01-diseno-cola.meta.json")
	escribir(t, ruta, "\xef\xbb\xbf"+`{"tema":"diseno-cola","clase":"I2","effort":"high","raiz":`+comillas(raiz)+
		`,"minutos_max":20,"sesion_claude":"abc-123","sesion":"019a-hilo","padre_herramienta":"codex"}`)
	l := NuevoLector(dir)
	if got := l.Actualizar(); len(got) != 1 || got[0] != "abc-123" {
		t.Fatalf("cambiadas = %v", got)
	}
	d, ok := l.Delegacion("abc-123")
	if !ok {
		t.Fatal("sin delegación")
	}
	if d.Padre != "codexvivo~sesion-019a-hilo" || d.Rol != "Claude" || d.Presupuesto != "20 min" || d.Terminada {
		t.Fatalf("delegación = %+v", d)
	}
	if _, ok := l.Delegacion("otra"); ok {
		t.Fatal("una sesión sin ficha no es delegada")
	}
	if got := l.Actualizar(); len(got) != 0 {
		t.Fatalf("sin cambios, cambiadas = %v", got)
	}

	// Al terminar, la ficha se completa; el índice la ve como un subagente cerrado del repositorio.
	escribir(t, ruta, `{"tema":"diseno-cola","clase":"I2","raiz":`+comillas(raiz)+
		`,"sesion_claude":"abc-123","sesion":"019a-hilo","padre_herramienta":"codex","fin":"2026-11-01T10:00:00-05:00","salida":0}`)
	ahora := time.Now().Add(time.Second)
	os.Chtimes(ruta, ahora, ahora)
	if got := l.Actualizar(); len(got) != 1 {
		t.Fatalf("la ficha completa no se releyó: %v", got)
	}
	d, _ = l.Delegacion("abc-123")
	if !d.Terminada || d.Raiz != raiz {
		t.Fatalf("al terminar = %+v", d)
	}

	// claude.ps1 borra las fichas viejas: la sesión vuelve a ser una más, y avisa que cambió.
	if err := os.Remove(ruta); err != nil {
		t.Fatal(err)
	}
	if got := l.Actualizar(); len(got) != 1 || got[0] != "abc-123" {
		t.Fatalf("al borrarse, cambiadas = %v", got)
	}
	if _, ok := l.Delegacion("abc-123"); ok {
		t.Fatal("la ficha borrada sigue en memoria")
	}
}

func TestPadre(t *testing.T) {
	casos := []struct {
		f    Ficha
		want string
	}{
		{Ficha{Sesion: "h1", PadreHerramienta: "codex"}, "codexvivo~sesion-h1"},
		{Ficha{Sesion: "s1", PadreHerramienta: "claude"}, "s1"},
		{Ficha{}, ""},
	}
	for _, c := range casos {
		if got := Padre(c.f); got != c.want {
			t.Errorf("Padre(%+v) = %q, quiero %q", c.f, got, c.want)
		}
	}
}

// El índice de Claude aplica la ficha: con padre, la sesión pasa a subagente del padre y toma el proyecto del
// repositorio.
func TestIndiceConDelegacion(t *testing.T) {
	transcritos := t.TempDir()
	copia := filepath.Join(transcritos, "D--tmp-arbol")
	os.MkdirAll(copia, 0o755)
	linea := `{"type":"user","sessionId":"abc-123","cwd":` + comillas(pruebas.Nativa(`C:\tmp\arbol`)) +
		`,"timestamp":"2026-11-01T09:00:00Z","message":{"role":"user","content":"Diseña la cola"}}` + "\n"
	escribir(t, filepath.Join(copia, "abc-123.jsonl"), linea)
	ix := claude.NuevoIndice(transcritos)
	raiz := pruebas.Nativa(`C:\repos\Tienda`)
	ix.Delegadas = func(id string) (claude.Delegacion, bool) {
		if id != "abc-123" {
			return claude.Delegacion{}, false
		}
		return claude.Delegacion{Raiz: raiz, Padre: "codexvivo~sesion-h1", Rol: "Claude · I2", Tarea: "diseno-cola", Terminada: true}, true
	}
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	ags := ix.Agentes()
	if len(ags) != 1 {
		t.Fatalf("agentes = %d", len(ags))
	}
	a := ags[0]
	if a.Tipo != "subagente" || a.Padre != "codexvivo~sesion-h1" || a.Proyecto != "Tienda" || a.Raiz != raiz || !a.Cerrado || a.Tarea != "diseno-cola" {
		t.Fatalf("agente = %+v", a)
	}
	if b, _, ok := ix.Agente("abc-123", 0); !ok || b.Padre != a.Padre {
		t.Fatalf("Agente() sin la ficha: %+v", b)
	}
}
