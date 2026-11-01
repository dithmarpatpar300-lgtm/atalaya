package servidor

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/codexvivo"
	"atalaya/internal/trazabilidad"
)

func TestArmarMarkdown(t *testing.T) {
	loc := time.FixedZone("PE", -5*3600)
	ini := time.Date(2026, 10, 31, 10, 18, 0, 0, loc)
	v := AgenteVista{Proyecto: "Tienda Demo", Tipo: "subagente", Clase: "I2", Modelo: "claude-sonnet-5-5", Effort: "high", Estado: Entregado,
		Inicio: ini, Ultimo: ini.Add(29 * time.Minute), Base: "main en cbcc0ae", Asignacion: "nivel enfocado · clase I2",
		Encargo: "**Asignación: nivel enfocado · clase I2.** Si vas a pasarte, dilo.\n\nQue la cola respete stock.\nSegunda línea.", Bitacora: "docs/trazabilidad/F5/avance/codificador-t1a.md",
		Avance: &trazabilidad.Avance{Porcentaje: 100}, Acta: "docs/trazabilidad/F5/temporales/x.md", ActaEstado: "cerrada", Pasos: 3,
		Tokens: claude.Tokens{Entrada: 98, CacheEscrita: 164_200, CacheLeida: 3_810_000, Salida: 36_400}}
	raiz := `C:\Proyectos\Tienda Demo`
	pasos := []claude.Paso{
		{N: 0, Hora: ini, Tipo: "Lee", Resumen: "carrito.rs", Turno: 1},
		{N: 1, Hora: ini.Add(24 * time.Minute), Tipo: "Edita", Resumen: "carrito.rs", Turno: 1},
		{N: 2, Hora: ini.Add(27 * time.Minute), Tipo: "Ejecuta", Resumen: "scripts/check.ps1", Meta: "salida 0", Turno: 1, TokContexto: 104_700},
	}
	larga := strings.Repeat("ok\n", 45)
	detalle := func(n int) (claude.DetallePaso, bool) {
		switch n {
		case 1:
			return claude.DetallePaso{Archivo: raiz + `\crates\pedidos\src\carrito.rs`,
				Diff: []claude.Trozo{{ViejoDesde: 398, ViejoLineas: 1, NuevoDesde: 398, NuevoLineas: 2, Lineas: []string{"-a", "+b", "+c"}}}}, true
		case 2:
			return claude.DetallePaso{Salida: larga}, true
		}
		return claude.DetallePaso{}, false
	}

	md := armarMarkdown("codificador-t1a", v, raiz, pasos, leerOpciones(nil), detalle, loc)
	for _, quiere := range []string{
		"# codificador-t1a\n",
		"> Tienda Demo · 2026-10-31 · I2 · sonnet-5-5/high · 10:18 – 10:47 (29 min) · base main en cbcc0ae",
		"**Asignación:** nivel enfocado · clase I2",
		"> Que la cola respete stock.\n> Segunda línea.",
		"- Bitácora: docs/trazabilidad/F5/avance/codificador-t1a.md (100 %)",
		"| crates/pedidos/src/carrito.rs | 2 | 1 |",
		"- 10:45 Ejecuta · scripts/check.ps1 (salida 0)",
		"  … (5 líneas más)",
		"| 98 | 164,2 k | 3,81 M | 36,4 k |",
		"Contexto al final: 104,7 k tokens.",
	} {
		if !strings.Contains(md, quiere) {
			t.Errorf("falta %q en:\n%s", quiere, md)
		}
	}
	if strings.Contains(md, "## Diffs") {
		t.Error("los diffs van apagados por omisión")
	}
	if strings.Count(md, "Asignación") != 1 {
		t.Errorf("la asignación se repite en el encargo citado:\n%s", md)
	}

	o := leerOpciones(map[string][]string{"encargo": {"0"}, "diffs": {"1"}, "salidas": {"ninguna"}, "tokens": {"0"}})
	md = armarMarkdown("codificador-t1a", v, raiz, pasos, o, detalle, loc)
	if strings.Contains(md, "## Encargo") || strings.Contains(md, "## Tokens") || strings.Contains(md, "```\n  ok") {
		t.Errorf("secciones apagadas que salen:\n%s", md)
	}
	if !strings.Contains(md, "## Diffs") || !strings.Contains(md, "@@ -398,1 +398,2 @@\n-a\n+b\n+c\n") {
		t.Errorf("diff:\n%s", md)
	}
}

func TestExportarCodexVivo(t *testing.T) {
	dir := t.TempDir()
	base := filepath.Join(dir, "2026-10-31-prueba")
	if err := os.WriteFile(base+".meta.json", []byte(`{"tema":"prueba","clase":"I1","modelo":"gpt-6.1-terra","effort":"high","inicio":"2026-10-31T10:00:00Z","fin":"2026-10-31T10:01:00Z"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base+".jsonl", []byte(`{"type":"item.completed","item":{"id":"c1","type":"command_execution","command":"go test","aggregated_output":"salida de prueba","exit_code":0,"status":"completed"}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := nuevo(t)
	s.CodexVivo = codexvivo.NuevoLector(dir)
	if _, err := s.CodexVivo.Actualizar(); err != nil {
		t.Fatal(err)
	}
	ll := s.CodexVivo.Llamadas()
	if len(ll) != 1 {
		t.Fatalf("llamadas: %+v", ll)
	}
	for _, caso := range []struct {
		query  string
		salida bool
	}{{"salidas=completas", true}, {"salidas=ninguna", false}} {
		r := httptest.NewRequest("GET", "/api/agentes/"+ll[0].ID+"/exportar?"+caso.query, nil)
		r.Host = "localhost"
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		var out respuestaExportar
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != 200 {
			t.Fatalf("respuesta %d: %s (%v)", w.Code, w.Body.String(), err)
		}
		if !strings.Contains(out.Markdown, "go test") || strings.Contains(out.Markdown, "salida de prueba") != caso.salida {
			t.Errorf("%s: %s", caso.query, out.Markdown)
		}
	}
}

func TestExportarCodexRegistro(t *testing.T) {
	raiz := t.TempDir()
	if err := os.MkdirAll(filepath.Join(raiz, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(raiz, "docs", "informe.md"), []byte("Resultado comprobado"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := nuevo(t)
	s.Proyectos = []Proyecto{{Nombre: "Proyecto", Raiz: raiz}}
	s.codex = map[string][]trazabilidad.LlamadaCodex{"Proyecto": {{ID: "codex~F1~prueba", Proyecto: "Proyecto", Tema: "revisar", Clase: "E1", Modelo: "gpt-6.1-sol", Effort: "high", Informe: "docs/informe.md", Terminada: time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC), Segundos: 60, Entrada: 10}}}
	r := httptest.NewRequest("GET", "/api/agentes/codex~F1~prueba/exportar", nil)
	r.Host = "localhost"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	var out respuestaExportar
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil || w.Code != 200 {
		t.Fatalf("respuesta %d: %s (%v)", w.Code, w.Body.String(), err)
	}
	for _, texto := range []string{"Codex · revisar", "E1", "high", "## Encargo", "## Resumen", "docs/informe.md", "## Informe", "Resultado comprobado", "## Tokens"} {
		if !strings.Contains(out.Markdown, texto) {
			t.Errorf("falta %q: %s", texto, out.Markdown)
		}
	}
}

func TestArchivoSeguro(t *testing.T) {
	for entra, sale := range map[string]string{
		"codificador-t1a":                 "codificador-t1a",
		"Retomar Tienda Demo con Atalaya": "retomar-tienda-demo-con-atalaya",
		"Medición: ¿año?":                 "medicion-ano",
		"":                                "agente",
	} {
		if got := archivoSeguro(entra); got != sale {
			t.Errorf("%q → %q, quiere %q", entra, got, sale)
		}
	}
}
