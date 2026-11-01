package servidor

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/cuotas"
)

func TestNombreVista(t *testing.T) {
	for _, c := range []struct {
		v    AgenteVista
		d    docsAgente
		want string
	}{
		{AgenteVista{Tipo: "sesion", Proyecto: "Tienda Demo", Titulo: "Auditar T1"}, docsAgente{}, "Tienda Demo"},
		{AgenteVista{Tipo: "subagente", Rol: "general-purpose", Tarea: "Medir T1"},
			docsAgente{Bitacora: "docs/trazabilidad/F5/avance/medidor-t1b.md"}, "medidor-t1b"},
		{AgenteVista{Tipo: "subagente", Rol: "general-purpose", Tarea: "Medir T1"},
			docsAgente{Acta: "docs/trazabilidad/F5/temporales/2026-10-31-medidor-t1b.md"}, "medidor-t1b"},
		{AgenteVista{Tipo: "subagente", Rol: "general-purpose", Tarea: "Medir T1"}, docsAgente{}, "Medir T1"},
		{AgenteVista{Tipo: "subagente", Rol: "Explore", Tarea: "Buscar el lector"}, docsAgente{}, "Buscar el lector"},
		{AgenteVista{Tipo: "subagente", Rol: "Auditor", Clase: "E2", Tarea: "P3"}, docsAgente{}, "Auditor E2"},
		{AgenteVista{Tipo: "codex", Rol: "Auditoría", Tarea: "e2-x"}, docsAgente{}, "Codex · e2-x"},
	} {
		if got := nombreVista(c.v, c.d); got != c.want {
			t.Errorf("nombreVista(%+v) = %q, quería %q", c.v, got, c.want)
		}
	}
}

func TestModeloCorto(t *testing.T) {
	for _, c := range [][3]string{{"claude-sonnet-5-5", "high", "sonnet-5-5/high"}, {"claude-opus-5-5", "", "opus-5-5"},
		{"gpt-6.1", "terra", "gpt-6.1/terra"}, {"", "high", ""}} {
		if got := modeloCorto(c[0], c[1]); got != c[2] {
			t.Errorf("modeloCorto(%q, %q) = %q", c[0], c[1], got)
		}
	}
	if c := claseDeVista(AgenteVista{Clase: "I2", Modelo: "claude-sonnet-5-5", Effort: "high"}); c != "I2 · sonnet-5-5/high" {
		t.Errorf("claseDeVista: %q", c)
	}
}

// Un general-purpose sin bitácora en el encargo: su acta y su bitácora se hallan por la carpeta temporal que usó.
func TestDocsPorCarpetaTemporal(t *testing.T) {
	raiz := t.TempDir()
	escribirPrueba(t, filepath.Join(raiz, "docs", "trazabilidad", "F5", "temporales", "2026-10-31-medidor-t1b.md"), "# Acta\n")
	escribirPrueba(t, filepath.Join(raiz, "docs", "trazabilidad", "F5", "avance", "medidor-t1b.md"), "Avance: 100 % — listo\n")
	ahora := time.Now()
	a := claude.Agente{ID: "s~a", Tipo: "subagente", Rol: "general-purpose", Tarea: "Medir T1", Raiz: raiz, Inicio: ahora.Add(-10 * time.Minute),
		Ultimo: ahora, Temporales: []string{"2026-10-31-med", "2026-10-31-medidor-t1b"}}
	d := buscarDocs(a)
	if d.Acta != "docs/trazabilidad/F5/temporales/2026-10-31-medidor-t1b.md" || d.Bitacora != "docs/trazabilidad/F5/avance/medidor-t1b.md" {
		t.Fatalf("documentos: %+v", d)
	}
	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(t.TempDir())}
	v := s.vistaClaude(a, true)
	if v.Nombre != "medidor-t1b" || v.Acta != d.Acta || v.Avance == nil || v.Avance.Porcentaje != 100 {
		t.Errorf("la vista: nombre %q, acta %q, avance %+v", v.Nombre, v.Acta, v.Avance)
	}
}

// Una sesión quieta cuyo subagente sigue trabajando está «Trabajando», y su vista de sesión trae sus decisiones.
func TestSesionConEquipoTrabajando(t *testing.T) {
	raiz := t.TempDir()
	dirProy := filepath.Join(raiz, "C--Proyectos-Tienda-Demo")
	dir := filepath.Join(dirProy, "ses1", "subagents")
	_ = os.MkdirAll(dir, 0o755)
	hora := time.Now().Add(-20 * time.Minute).UTC().Format(time.RFC3339)
	sesion := `{"type":"user","timestamp":"` + hora + `","cwd":"C:\\Proyectos\\Tienda Demo","message":{"role":"user","content":"Audita P3"}}
{"type":"assistant","timestamp":"` + hora + `","message":{"id":"m1","role":"assistant","content":[{"type":"text","text":"Lanzo al auditor."}]}}
`
	rutaSes := filepath.Join(dirProy, "ses1.jsonl")
	_ = os.WriteFile(rutaSes, []byte(sesion), 0o644)
	vieja := time.Now().Add(-20 * time.Minute)
	_ = os.Chtimes(rutaSes, vieja, vieja)
	_ = os.WriteFile(filepath.Join(dir, "agent-abc.jsonl"), []byte(transcritoMCP), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "agent-abc.meta.json"), []byte(`{"agentType":"auditor","description":"P3: auditoría E2"}`), 0o644)
	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(raiz)}
	if _, err := s.Claude.Actualizar(); err != nil {
		t.Fatal(err)
	}
	por := map[string]AgenteVista{}
	for _, a := range s.todos() {
		por[a.ID] = a
	}
	if ses := por["ses1"]; ses.Estado != Trabajando || ses.Nombre != "Tienda Demo" {
		t.Errorf("la sesión: estado %q, nombre %q", ses.Estado, ses.Nombre)
	}
	if sub := por["ses1~abc"]; sub.Estado != Trabajando || sub.Nombre != "auditor-p3max" || sub.ModeloCorto != "opus-5-5" {
		t.Errorf("el subagente: %+v", sub)
	}
	r := httptest.NewRequest("GET", "/api/agentes/ses1/sesion", nil)
	r.Host = "localhost"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if b := w.Body.String(); !strings.Contains(b, `"decisiones":[]`) || !strings.Contains(b, `"nombre":"auditor-p3max"`) {
		t.Errorf("la vista de la sesión: %s", b)
	}
}
