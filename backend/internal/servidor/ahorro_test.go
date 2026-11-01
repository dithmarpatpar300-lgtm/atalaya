package servidor

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atalaya/internal/avance"
	"atalaya/internal/claude"
	"atalaya/internal/cuotas"
)

func repetida(c Consulta) bool { return c.Ahorro != nil && c.Repetida }

func TestMarcarRepetidas(t *testing.T) {
	h := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cs := []Consulta{
		{Hora: h, Agente: "a", Herramienta: "estado", Argumentos: `{"a":1,"b":2}`},
		// Las mismas claves en otro orden, justo a los 5 minutos: repite.
		{Hora: h.Add(5 * time.Minute), Agente: "a", Herramienta: "estado", Argumentos: `{"b":2,"a":1}`},
		// Otro agente, otra herramienta, otros argumentos: no.
		{Hora: h.Add(time.Minute), Agente: "b", Herramienta: "estado", Argumentos: `{"a":1,"b":2}`},
		{Hora: h.Add(time.Minute), Agente: "a", Herramienta: "retomar", Argumentos: `{"a":1,"b":2}`},
		{Hora: h.Add(6 * time.Minute), Agente: "a", Herramienta: "estado", Argumentos: `{"a":3}`},
		// Sin argumentos («» y «{}» son lo mismo): repite.
		{Hora: h.Add(time.Minute), Agente: "a", Herramienta: "salud"},
		{Hora: h.Add(2 * time.Minute), Agente: "a", Herramienta: "salud", Argumentos: `{}`},
		// Codex sin argumentos conocidos: no se puede saber.
		{Hora: h, Agente: "c", Herramienta: "salud", argumentosDesconocidos: true},
		{Hora: h.Add(time.Minute), Agente: "c", Herramienta: "salud", argumentosDesconocidos: true},
		// A más de 5 minutos: no.
		{Hora: h, Agente: "d", Herramienta: "salud"},
		{Hora: h.Add(5*time.Minute + time.Second), Agente: "d", Herramienta: "salud"},
		// Sin agente (un curl): no.
		{Hora: h, Herramienta: "salud"},
		{Hora: h.Add(time.Second), Herramienta: "salud"},
	}
	marcarRepetidas(cs)
	for i, c := range cs {
		if repetida(c) != (i == 1 || i == 6) {
			t.Errorf("consulta %d: repetida = %v", i, repetida(c))
		}
	}
}

// Un plan que apunta al propio AVANCE.md cuenta una vez; un que_hizo sin argumentos (Codex) queda sin medir.
func TestEvitaSinRepetirYCodex(t *testing.T) {
	proj := filepath.Join(t.TempDir(), "Demo")
	escribirPrueba(t, filepath.Join(proj, "docs", "trazabilidad", "F7", "AVANCE.md"), "Avance: 1 % — x\nPlan: docs/trazabilidad/F7/AVANCE.md\n")
	avance.OlvidarMemoria()
	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(t.TempDir()),
		Proyectos: []Proyecto{{Nombre: "Demo", Raiz: proj, Registrado: true}}}
	cs := []Consulta{
		{Herramienta: "estado", Proyecto: "Demo", Agente: "codex~1", argumentosDesconocidos: true},
		{Herramienta: "que_hizo", Proyecto: "Demo", Agente: "codex~1", argumentosDesconocidos: true},
	}
	s.medirAhorro(cs)
	if e := cs[0]; rutasEvita(e.Ahorro) != "docs/trazabilidad/F7/AVANCE.md" || !e.AhorroEsCota || e.AhorroBytes != e.AlternativaBytes {
		t.Errorf("estado: %+v", *e.Ahorro)
	}
	if q := cs[1]; !q.SinMedir || q.AhorroEsCota || len(q.Evita) != 0 {
		t.Errorf("que_hizo de Codex: %+v", *q.Ahorro)
	}
}

func TestRutaPlan(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "p")
	if r := rutaPlan(raiz, "`docs/planes/fase-2.md` §4"); r != filepath.Join(raiz, "docs", "planes", "fase-2.md") {
		t.Errorf("plan: %q", r)
	}
	// Una ruta absoluta de este sistema: «C:/…» en Linux es relativa.
	fuera := filepath.ToSlash(filepath.Join(t.TempDir(), "fuera.md"))
	for _, p := range []string{"", "../otro/plan.md", fuera} {
		if r := rutaPlan(raiz, p); r != "" {
			t.Errorf("plan %q fuera del proyecto: %q", p, r)
		}
	}
}

// js: una cadena en JSON, para meter rutas de verdad en los transcritos de prueba.
func js(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// proyectoAhorro arma un proyecto con su avance, su plan y sus documentos, y los transcritos de una sesión y de su
// subagente que consultan a Atalaya y leen esos documentos.
func proyectoAhorro(t *testing.T) (*Servidor, string) {
	t.Helper()
	proj := filepath.Join(t.TempDir(), "Demo")
	_ = os.MkdirAll(filepath.Join(proj, ".git"), 0o755)
	doc := func(rel, texto string) string {
		r := filepath.Join(proj, filepath.FromSlash(rel))
		escribirPrueba(t, r, texto)
		return r
	}
	doc("AGENTS.md", strings.Repeat("a", 100))
	estado := doc("docs/trazabilidad/ESTADO-para-retomar.md", strings.Repeat("e", 200))
	f1 := doc("docs/trazabilidad/F1/AVANCE.md", "Avance: 100 % — listo\n")
	f2 := doc("docs/trazabilidad/F2/AVANCE.md", "Avance: 10 % — en curso\nPlan: docs/planes/fase-2.md §1\n")
	plan := doc("docs/planes/fase-2.md", strings.Repeat("p", 300))
	doc("docs/trazabilidad/F2/avance/medidor-t1.md", strings.Repeat("b", 400))
	avance.OlvidarMemoria()

	cwd := js(proj)
	uso := func(hora, id, herr, input string) string {
		return `{"type":"assistant","timestamp":"2026-10-31T` + hora + `Z","cwd":` + cwd + `,"message":{"id":"m` + id + `","model":"claude-opus-5-5","role":"assistant","content":[{"type":"tool_use","id":"` + id + `","name":"` + herr + `","input":` + input + `}],"usage":{"input_tokens":1,"output_tokens":10}}}` + "\n"
	}
	res := func(hora, id, texto string, err bool) string {
		e := "false"
		if err {
			e = "true"
		}
		return `{"type":"user","timestamp":"2026-10-31T` + hora + `Z","message":{"role":"user","content":[{"tool_use_id":"` + id + `","type":"tool_result","content":[{"type":"text","text":` + js(texto) + `}],"is_error":` + e + `}]}}` + "\n"
	}
	lee := func(ruta string) string { return `{"file_path":` + js(ruta) + `}` }
	sesion := `{"type":"user","timestamp":"2026-10-31T17:00:00Z","cwd":` + cwd + `,"message":{"role":"user","content":"Arranca F2."}}` + "\n" +
		uso("17:01:00", "tu_e", "mcp__atalaya__estado", `{}`) + res("17:01:01", "tu_e", strings.Repeat("r", 50), false) +
		// Leyó el AVANCE.md justo después: el estado no le ahorró nada.
		uso("17:02:00", "tu_r1", "Read", lee(f2)) + res("17:02:01", "tu_r1", "Avance", false) +
		uso("17:03:00", "tu_s", "mcp__atalaya__salud", `{"proyecto":"Demo"}`) + res("17:03:01", "tu_s", "ok", false) +
		uso("17:04:00", "tu_s2", "mcp__atalaya__salud", `{"proyecto":"Demo"}`) + res("17:04:01", "tu_s2", "ok", false) +
		uso("17:05:00", "tu_b", "mcp__atalaya__buscar", `{"texto":"x"}`) + res("17:05:01", "tu_b", "nada", false) +
		uso("17:06:00", "tu_q", "mcp__atalaya__que_hizo", `{"agente":"medidor-t1"}`) + res("17:06:01", "tu_q", strings.Repeat("q", 10), false) +
		uso("17:07:00", "tu_x", "mcp__atalaya__estado", `{"proyecto":"Nadie"}`) + res("17:07:01", "tu_x", "No conozco el proyecto", true)
	sub := `{"isSidechain":true,"type":"user","timestamp":"2026-10-31T17:09:00Z","cwd":` + cwd + `,"message":{"role":"user","content":"**Asignación: nivel enfocado · clase M · rol medidor**\nMide T1. Tu bitácora: ` + "`docs/trazabilidad/F2/avance/medidor-t1.md`" + `."}}` + "\n" +
		// Leyó el ESTADO y el plan sin consultar antes: podría haber consultado.
		uso("17:10:00", "tu_l1", "Read", lee(estado)) + res("17:10:01", "tu_l1", "Estado", false) +
		uso("17:10:30", "tu_l2", "Read", lee(plan)) + res("17:10:31", "tu_l2", "Plan", false) +
		// El AVANCE.md de una plantilla dentro del proyecto no es el del proyecto: no cuenta.
		uso("17:10:40", "tu_l4", "Read", lee(filepath.Join(proj, "kit", "docs", "trazabilidad", "F0", "AVANCE.md"))) + res("17:10:41", "tu_l4", "F0", false) +
		// retomar sin resultado todavía: el ahorro es una cota.
		uso("17:11:00", "tu_t", "mcp__atalaya__retomar", `{}`) +
		// Después de retomar, leer un AVANCE.md ya no cuenta como lectura evitable; el de F1 no es el que reemplazó.
		uso("17:12:00", "tu_l3", "Read", lee(f1)) + res("17:12:01", "tu_l3", "F1", false)

	tr := t.TempDir()
	dir := filepath.Join(tr, "D--Demo")
	_ = os.MkdirAll(filepath.Join(dir, "ses1", "subagents"), 0o755)
	_ = os.WriteFile(filepath.Join(dir, "ses1.jsonl"), []byte(sesion), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "ses1", "subagents", "agent-med.jsonl"), []byte(strings.ReplaceAll(sub, `{"type"`, `{"isSidechain":true,"type"`)), 0o644)
	ahora := time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC)
	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(tr), Ahora: func() time.Time { return ahora },
		Proyectos: []Proyecto{{Nombre: "Demo", Raiz: proj, Registrado: true}}}
	if _, err := s.Claude.Actualizar(); err != nil {
		t.Fatal(err)
	}
	return s, proj
}

func tam(t *testing.T, proj, rel string) int64 {
	t.Helper()
	fi, err := os.Stat(filepath.Join(proj, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return fi.Size()
}

func rutasEvita(a *Ahorro) string {
	var rs []string
	for _, d := range a.Evita {
		rs = append(rs, d.Ruta)
	}
	return strings.Join(rs, ",")
}

// La regla de cada herramienta, releyo, repetida y la cota, sobre /api/consultas.
func TestAhorroPorHerramienta(t *testing.T) {
	s, proj := proyectoAhorro(t)
	w := httptest.NewRecorder()
	s.verConsultas(w, httptest.NewRequest("GET", "/api/consultas", nil))
	var cs []Consulta
	if err := json.Unmarshal(w.Body.Bytes(), &cs); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(w.Body.String(), `"alternativa_bytes"`) || !strings.Contains(w.Body.String(), `"evita"`) {
		t.Fatalf("faltan los campos del ahorro: %s", w.Body.String())
	}
	por := map[string]Consulta{}
	for _, c := range cs {
		if c.Ahorro == nil {
			t.Fatalf("sin ahorro: %+v", c)
		}
		k := c.Herramienta
		if c.Herramienta == "salud" && c.Repetida {
			k = "salud2"
		}
		if c.Error {
			k = "error"
		}
		por[k] = c
	}
	if len(cs) != 7 {
		t.Fatalf("esperaba 7 consultas, hay %d", len(cs))
	}
	avF2, plan := tam(t, proj, "docs/trazabilidad/F2/AVANCE.md"), tam(t, proj, "docs/planes/fase-2.md")

	e := por["estado"]
	if rutasEvita(e.Ahorro) != "docs/trazabilidad/F2/AVANCE.md,docs/planes/fase-2.md" || e.AlternativaBytes != avF2+plan || e.AlternativaTokens != (avF2+plan)/4 {
		t.Errorf("estado evita %q, %d bytes", rutasEvita(e.Ahorro), e.AlternativaBytes)
	}
	if !e.Releyo || e.AhorroBytes != 0 {
		t.Errorf("estado: releyó el AVANCE.md, el ahorro no cuenta: %+v", *e.Ahorro)
	}

	s1, s2 := por["salud"], por["salud2"]
	if rutasEvita(s1.Ahorro) != "AGENTS.md" || s1.AlternativaBytes != 100 || s1.AhorroBytes != 98 || s1.AhorroEsCota || s1.Repetida || s1.Releyo {
		t.Errorf("salud: %+v", *s1.Ahorro)
	}
	if !s2.Repetida || s2.AhorroBytes != 98 {
		t.Errorf("la segunda salud repite la primera: %+v", *s2.Ahorro)
	}

	if b := por["buscar"]; !b.SinMedir || b.AlternativaBytes != 0 || len(b.Evita) != 0 || b.AhorroBytes != 0 {
		t.Errorf("buscar: %+v", *b.Ahorro)
	}

	if q := por["que_hizo"]; rutasEvita(q.Ahorro) != "docs/trazabilidad/F2/avance/medidor-t1.md" || q.AlternativaBytes != 400 || q.AhorroBytes != 390 {
		t.Errorf("que_hizo: %+v", *q.Ahorro)
	}

	if x := por["error"]; len(x.Evita) != 0 || x.AhorroBytes != 0 || x.SinMedir {
		t.Errorf("una consulta con error no evita nada: %+v", *x.Ahorro)
	}

	r := por["retomar"]
	if rutasEvita(r.Ahorro) != "docs/trazabilidad/ESTADO-para-retomar.md,docs/trazabilidad/F2/AVANCE.md,AGENTS.md" ||
		r.AlternativaBytes != 200+avF2+100 || !r.AhorroEsCota || r.AhorroBytes != r.AlternativaBytes || r.Releyo {
		t.Errorf("retomar sin resultado: %+v", *r.Ahorro)
	}

	// El registro guardado en el disco no lleva el ahorro.
	b, _ := json.Marshal(Consulta{Herramienta: "salud"})
	if strings.Contains(string(b), "evita") || strings.Contains(string(b), "repetida") {
		t.Errorf("una consulta sin medir no debe llevar el ahorro: %s", b)
	}
}

// Una consulta que repite otra de antes de la ventana de días también es repetida: se mira hacia atrás 5 minutos.
func TestRepetidaEnElBorde(t *testing.T) {
	s, _ := proyectoAhorro(t)
	// La ventana empieza entre la primera salud (17:03) y la segunda (17:04).
	s.Ahora = func() time.Time { return time.Date(2026, 11, 14, 17, 3, 30, 0, time.UTC) }
	var salud []Consulta
	for _, c := range s.consultasDesde(s.ahora().AddDate(0, 0, -DiasConsultas)) {
		if c.Herramienta == "salud" {
			salud = append(salud, c)
		}
	}
	if len(salud) != 1 || !salud[0].Repetida {
		t.Errorf("la salud de las 17:04 repite la de las 17:03, fuera de la ventana: %+v", salud)
	}
}

func TestPodriaHaberConsultado(t *testing.T) {
	s, proj := proyectoAhorro(t)
	p := s.podriaHaberConsultado(time.Time{}, "")
	var rs []string
	for _, l := range p.Lecturas {
		rs = append(rs, l.Ruta)
		if l.Proyecto != "Demo" || l.Agente != "ses1~med" || l.Paso <= 0 {
			t.Errorf("lectura: %+v", l)
		}
	}
	// El plan y el ESTADO, sí; el AVANCE.md de la sesión (después de estado) y el de F1 (después de retomar), no.
	if strings.Join(rs, ",") != "docs/planes/fase-2.md,docs/trazabilidad/ESTADO-para-retomar.md" {
		t.Errorf("lecturas evitables: %v", rs)
	}
	if want := tam(t, proj, "docs/planes/fase-2.md") + 200; p.Bytes != want || p.Tokens != want/4 {
		t.Errorf("bytes: %d, quería %d", p.Bytes, want)
	}
	if otro := s.podriaHaberConsultado(time.Time{}, "Otro"); len(otro.Lecturas) != 0 {
		t.Errorf("filtro por proyecto: %+v", otro)
	}
}

func TestResumenConsultas(t *testing.T) {
	s, _ := proyectoAhorro(t)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/api/consultas/resumen?dias=99", nil)
	r.Host = "localhost"
	s.Handler().ServeHTTP(w, r)
	var res ResumenConsultas
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("%v: %s", err, w.Body.String())
	}
	if res.Dias != DiasConsultas {
		t.Errorf("los días se acotan a %d: %d", DiasConsultas, res.Dias)
	}
	tot := res.Total
	if tot.N != 7 || tot.Errores != 1 || tot.Repetidas != 1 || tot.Releyo != 1 || tot.Bytes != 50+2+2+4+10+int64(len("No conozco el proyecto"))+0 {
		t.Errorf("total: %+v", tot)
	}
	var suma int64
	por := map[string]FilaAhorro{}
	for _, f := range res.PorHerramienta {
		suma += f.AhorroBytes
		por[f.Clave] = f
	}
	if suma != tot.AhorroBytes || por["estado"].N != 2 || por["estado"].Errores != 1 || por["salud"].Repetidas != 1 || por["salud"].AhorroBytes != 196 {
		t.Errorf("por herramienta: %+v", res.PorHerramienta)
	}
	if len(res.PorProyecto) != 1 || res.PorProyecto[0].Clave != "Demo" || res.PorProyecto[0].N != 7 {
		t.Errorf("por proyecto: %+v", res.PorProyecto)
	}
	roles := map[string]int{}
	for _, f := range res.PorRol {
		roles[f.Clave] = f.N
	}
	if roles["sesion"] != 6 || roles["subagente"] != 1 {
		t.Errorf("por rol: %+v", res.PorRol)
	}
	if len(res.PorDia) != 1 || res.PorDia[0].N != 7 {
		t.Errorf("por día: %+v", res.PorDia)
	}
	if len(res.PodriaHaberConsultado.Lecturas) != 2 {
		t.Errorf("podría haber consultado: %+v", res.PodriaHaberConsultado)
	}

	// Con un proyecto que no es: vacío, pero con sus listas.
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/api/consultas/resumen?proyecto=Otro", nil)
	r.Host = "localhost"
	s.Handler().ServeHTTP(w, r)
	if b := w.Body.String(); !strings.Contains(b, `"por_herramienta":[]`) || !strings.Contains(b, `"lecturas":[]`) || !strings.Contains(b, `"n":0`) {
		t.Errorf("otro proyecto: %s", b)
	}
}
