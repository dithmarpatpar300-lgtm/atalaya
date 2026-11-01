package servidor

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"atalaya/internal/claude"
	"atalaya/internal/cuotas"
)

func proyectoConAvance(t *testing.T) (*Servidor, string) {
	t.Helper()
	raiz := t.TempDir()
	tz := filepath.Join(raiz, "docs", "trazabilidad")
	_ = os.MkdirAll(filepath.Join(tz, "F5"), 0o755)
	_ = os.WriteFile(filepath.Join(tz, "F5", "AVANCE.md"), []byte("Avance: 83 % — T1b: pruebas de inventario\nPlan: docs/planes/fase-5.md §4\n\n"+
		"| Paquete | Qué | Horas | Estado | % | Evidencia |\n|---|---|---|---|---|---|\n"+
		"| W1 | Versión | 3 | hecho | 100 | F5/02.md |\n| T1 | Inventario | 4 | en curso | 85 | prueba de inventario |\n| W12 | Firma | 3 | sin empezar | 0 | |\n\n"+
		"- 13:24 · T1b: pruebas de inventario\n"), 0o644)
	_ = os.WriteFile(filepath.Join(tz, "ESTADO-para-retomar.md"), []byte("# Estado\n\nIntro.\n\n## Dónde estamos\n\n- Fase 5 en marcha.\n\n## Lo que hace el dueño\n\n1. Decidir.\n"), 0o644)
	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(t.TempDir()), Proyectos: []Proyecto{{Nombre: "TD", Raiz: raiz}}}
	return s, raiz
}

func TestEstadoMCP(t *testing.T) {
	s, _ := proyectoConAvance(t)
	got := s.estadoMCP("TD")
	// (3 + 3,4 + 0) de 10 h = 64 %.
	for _, q := range []string{"fase F5: 64,0 %", "6,4 de 10 h", "Ahora: T1b: pruebas de inventario", "T1 · Inventario · 4 h · en curso · 85 % · prueba de inventario", "Falta: unas 3,6 h", "13:24 · T1b"} {
		if !strings.Contains(got, q) {
			t.Errorf("falta %q en:\n%s", q, got)
		}
	}
	// La primera línea (83 %) no coincide con la tabla (64 %): se avisa.
	if !strings.Contains(got, "Avisos de la tabla: la primera línea dice 83 % y los paquetes dan 64,0 %") {
		t.Errorf("sin aviso:\n%s", got)
	}
	vacio := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(t.TempDir()), Proyectos: []Proyecto{{Nombre: "X", Raiz: t.TempDir()}}}
	if got := vacio.estadoMCP("X"); !strings.Contains(got, "no tiene docs/trazabilidad/F<n>/AVANCE.md") {
		t.Errorf("sin avance: %s", got)
	}
}

func TestRetomarMCP(t *testing.T) {
	s, _ := proyectoConAvance(t)
	got := s.retomarMCP("TD")
	for _, q := range []string{"Avance: fase F5 al 64,0 %", "Avisos del avance: la primera línea dice 83 %", "Agentes vivos: 0", "Alertas abiertas: 0", "## Dónde estamos", "Fase 5 en marcha", "git status"} {
		if !strings.Contains(got, q) {
			t.Errorf("falta %q en:\n%s", q, got)
		}
	}
	if strings.Contains(got, "Lo que hace el dueño") {
		t.Error("de ESTADO solo va la sección «Dónde estamos»")
	}
}

func TestVerAvance(t *testing.T) {
	s, _ := proyectoConAvance(t)
	r := httptest.NewRequest("GET", "/api/avance?proyecto=TD", nil)
	r.Host = "localhost"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	var out respuestaAvance
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || len(out.Fases) != 1 || out.Fases[0].Calculado != 64 || out.Sin != "" {
		t.Fatalf("respuesta %d: %s", w.Code, w.Body.String())
	}
}
