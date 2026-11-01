package servidor

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/cuotas"
)

const avancePrueba = `Avance: 50 % — W2: pruebas
Plan: docs/planes/fase-2.md

| Paquete | Qué | Horas | Estado | % | Evidencia |
| --- | --- | --- | --- | --- | --- |
| W1 | Base | 3 | hecho | 100 | F2/01.md |
| W2 | Pruebas | 4 | en curso | 50 | |
| W3 | Cierre | 3 | sin empezar | 0 | |
`

func escribirPrueba(t *testing.T, ruta, texto string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(texto), 0o644); err != nil {
		t.Fatal(err)
	}
}

// Un proyecto con medio contrato, su AVANCE.md y una sesión viva que consultó a Atalaya hoy (y ayer); otro que no
// está en el disco ni en el registro.
func TestPortafolio(t *testing.T) {
	raiz := filepath.Join(t.TempDir(), "Mi Proyecto")
	_ = os.MkdirAll(filepath.Join(raiz, ".git"), 0o755)
	_ = os.MkdirAll(filepath.Join(raiz, ".claude", "agents"), 0o755)
	escribirPrueba(t, filepath.Join(raiz, "AGENTS.md"), "# Agentes\n")
	escribirPrueba(t, filepath.Join(raiz, "docs", "trazabilidad", "00-ESTADO-para-retomar.md"), "# Estado\n")
	escribirPrueba(t, filepath.Join(raiz, "docs", "trazabilidad", "F2", "AVANCE.md"), avancePrueba)

	ahora := time.Now()
	ayer := ahora.Add(-26 * time.Hour)
	cwd, _ := json.Marshal(raiz)
	ts := func(t time.Time) string { return t.UTC().Format(time.RFC3339) }
	sesion := `{"type":"user","timestamp":"` + ts(ayer) + `","cwd":` + string(cwd) + `,"message":{"role":"user","content":"Retoma"}}
{"type":"assistant","timestamp":"` + ts(ayer) + `","message":{"id":"m0","role":"assistant","content":[{"type":"tool_use","id":"t0","name":"mcp__atalaya__retomar","input":{}}]}}
{"type":"user","timestamp":"` + ts(ayer) + `","message":{"role":"user","content":[{"tool_use_id":"t0","type":"tool_result","content":"ok"}]}}
{"type":"assistant","timestamp":"` + ts(ahora) + `","message":{"id":"m1","role":"assistant","content":[{"type":"tool_use","id":"t1","name":"mcp__atalaya__salud","input":{}}]}}
{"type":"user","timestamp":"` + ts(ahora) + `","message":{"role":"user","content":[{"tool_use_id":"t1","type":"tool_result","content":"ok"}]}}
{"type":"assistant","timestamp":"` + ts(ahora) + `","message":{"id":"m2","role":"assistant","content":[{"type":"tool_use","id":"t2","name":"Bash","input":{"command":"go test ./..."}}]}}
`
	transcritos := t.TempDir()
	escribirPrueba(t, filepath.Join(transcritos, "D--Mi-Proyecto", "ses1.jsonl"), sesion)

	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(transcritos), Ahora: func() time.Time { return ahora }}
	if _, err := s.Claude.Actualizar(); err != nil {
		t.Fatal(err)
	}
	s.Proyectos = []Proyecto{
		{Nombre: "Mi Proyecto", Raiz: raiz, Sigla: "MP", Nivel: 3, Registrado: true},
		{Nombre: "Fantasma", Raiz: filepath.Join(t.TempDir(), "no-existe")},
	}

	r := httptest.NewRequest("GET", "/api/portafolio", nil)
	r.Host = "localhost"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	var ps []proyectoPortafolio
	if err := json.Unmarshal(w.Body.Bytes(), &ps); err != nil || len(ps) != 2 {
		t.Fatalf("respuesta: %v %s", err, w.Body.String())
	}

	p := ps[0]
	if p.Nombre != "Mi Proyecto" || p.Sigla != "MP" || p.Nivel != 3 || !p.Existe || p.Fase != "F2" {
		t.Errorf("la ficha: %+v", p)
	}
	esta := map[string]bool{}
	rutas := map[string]string{}
	for _, c := range p.Contrato {
		esta[c.ID], rutas[c.ID] = c.Esta, c.Ruta
	}
	if len(p.Contrato) != 7 {
		t.Fatalf("el contrato tiene 7 archivos: %+v", p.Contrato)
	}
	for id, want := range map[string]bool{"agents": true, "mcp": false, "estado": true, "avance": true, "operacion": false, "subagentes": true, "registro": true} {
		if esta[id] != want {
			t.Errorf("contrato %s: %v, quería %v", id, esta[id], want)
		}
	}
	if rutas["estado"] != "docs/trazabilidad/00-ESTADO-para-retomar.md" || rutas["avance"] != "docs/trazabilidad/F2/AVANCE.md" {
		t.Errorf("rutas del contrato: %v", rutas)
	}
	a := p.Avance
	if a == nil || a.Calculado != 50 || a.Horas != 10 || a.Hechas != 3 || a.EnCurso != 2 || a.Avanzadas != 5 || a.Ahora != "W2: pruebas" {
		t.Errorf("el avance: %+v", a)
	}
	if p.Vivos != 1 {
		t.Errorf("agentes vivos: %d", p.Vivos)
	}
	if p.Consultas != 1 {
		t.Errorf("consultas de hoy (la de ayer no cuenta): %d", p.Consultas)
	}
	if p.Salud.Total != 0 || p.Salud.EnRegla != 0 {
		t.Errorf("sin subagentes, la salud es 0 de 0: %+v", p.Salud)
	}

	f := ps[1]
	if f.Existe || f.Avance != nil || f.Fase != "" || f.Sigla != "FA" || f.Vivos != 0 {
		t.Errorf("el que no existe: %+v", f)
	}
	for _, c := range f.Contrato {
		if c.Esta {
			t.Errorf("el que no existe no tiene %s", c.ID)
		}
	}
}

// La salud se guarda mientras no cambia lo que la mueve, y se rehace si cambian las temporales o pasan 30 s.
func TestSaludDeHoyGuardada(t *testing.T) {
	ahora := time.Date(2026, 10, 31, 12, 0, 0, 0, time.Local)
	s := nuevo(t)
	s.Ahora = func() time.Time { return ahora }
	p := Proyecto{Nombre: "X", Raiz: t.TempDir()}
	if got := s.saludDeHoy(p, ahora, nil); got.Total != 0 {
		t.Fatalf("primera: %+v", got)
	}
	muSaludes.Lock()
	g := saludes[s]["X"]
	g.salud = saludPortafolio{EnRegla: 9, Total: 9}
	saludes[s]["X"] = g
	muSaludes.Unlock()
	if got := s.saludDeHoy(p, ahora.Add(10*time.Second), nil); got.Total != 9 {
		t.Errorf("debía salir de lo guardado: %+v", got)
	}
	if got := s.saludDeHoy(p, ahora.Add(40*time.Second), nil); got.Total != 0 {
		t.Errorf("a los 30 s se rehace: %+v", got)
	}
	muSaludes.Lock()
	g = saludes[s]["X"]
	g.salud = saludPortafolio{EnRegla: 9, Total: 9}
	saludes[s]["X"] = g
	muSaludes.Unlock()
	s.firmaTmp = "otra"
	if got := s.saludDeHoy(p, ahora.Add(41*time.Second), nil); got.Total != 0 {
		t.Errorf("con otras temporales se rehace: %+v", got)
	}
	if got := s.saludDeHoy(p, ahora.Add(42*time.Second), []string{"a=Trabajando"}); got.Total != 0 {
		t.Errorf("con otro agente se rehace: %+v", got)
	}
}
