package servidor

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/configurar"
	"atalaya/internal/cuotas"
	"atalaya/internal/entorno"
	"atalaya/internal/proyectos"
)

// conRegistro: un servidor con proyectos.json en una carpeta temporal y su carpeta de datos.
func conRegistro(t *testing.T, contenido string) (*Servidor, string) {
	t.Helper()
	dir := t.TempDir()
	ruta := filepath.Join(dir, "proyectos.json")
	if contenido != "" {
		_ = os.WriteFile(ruta, []byte(contenido), 0o644)
	}
	s := nuevo(t)
	s.RutaRegistro, s.Datos = ruta, filepath.Join(dir, "Atalaya")
	s.Registro = func() ([]Proyecto, error) {
		ps, err := proyectos.Leer(ruta)
		if err != nil {
			return nil, err
		}
		var out []Proyecto
		for _, p := range ps {
			out = append(out, Proyecto{Nombre: p.Nombre, Raiz: p.Ruta, Sigla: p.Sigla, Nivel: p.Nivel, Registrado: true})
		}
		return out, nil
	}
	s.releerRegistro() // toma la firma del arranque, como Vigilar
	return s, ruta
}

func pedir(t *testing.T, h http.Handler, metodo, ruta, cuerpo string, panel bool) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest(metodo, ruta, strings.NewReader(cuerpo))
	r.Host = "127.0.0.1:7420"
	if panel {
		r.Header.Set(cabPanel, "1")
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// Sin la cabecera del panel, o desde otro origen, nada se escribe.
func TestConfigurarSoloDesdeElPanel(t *testing.T) {
	s, ruta := conRegistro(t, "")
	h := s.Handler()
	cuerpo := `{"agregar":[{"ruta":` + comillas(t.TempDir()) + `,"nivel":1}]}`
	for _, rr := range []string{"/api/configurar/proyectos", "/api/configurar/aplicar", "/api/configurar/deshacer"} {
		if w := pedir(t, h, "POST", rr, cuerpo, false); w.Code != http.StatusForbidden {
			t.Errorf("%s sin la cabecera: %d", rr, w.Code)
		}
	}
	r := httptest.NewRequest("POST", "/api/configurar/proyectos", strings.NewReader(cuerpo))
	r.Host = "127.0.0.1:7420"
	r.Header.Set(cabPanel, "1")
	r.Header.Set("Origin", "https://otra.example")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("desde otro origen: %d", w.Code)
	}
	if _, err := os.Stat(ruta); !os.IsNotExist(err) {
		t.Error("se escribió el registro")
	}
}

// comillas: el texto como literal JSON, con sus comillas.
func comillas(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// El ciclo completo: la vista previa no escribe, aplicar escribe el registro con su forma de siempre y la portada lo
// ve, y deshacer lo deja como estaba.
func TestRegistrarUnProyectoDesdeElPanel(t *testing.T) {
	carpeta := filepath.Join(t.TempDir(), "Tienda Demo")
	_ = os.MkdirAll(carpeta, 0o755)
	viejo := `[{"nombre":"Blog","ruta":` + comillas(t.TempDir()) + `,"nivel":1}]`
	s, ruta := conRegistro(t, viejo)
	h := s.Handler()

	w := pedir(t, h, "POST", "/api/configurar/proyectos", `{"agregar":[{"ruta":`+comillas(carpeta)+`,"nivel":2}]}`, true)
	if w.Code != 200 {
		t.Fatalf("vista previa: %d %s", w.Code, w.Body)
	}
	var prop configurar.Propuesta
	_ = json.Unmarshal(w.Body.Bytes(), &prop)
	if len(prop.Cambios) != 1 || prop.Cambios[0].Antes != viejo || !strings.Contains(prop.Resumen, "agregar Tienda Demo (nivel 2)") {
		t.Fatalf("propuesta: %+v", prop)
	}
	if !strings.Contains(prop.Cambios[0].Despues, `{ "nombre": "Tienda Demo", "ruta": `) || !strings.Contains(prop.Cambios[0].Despues, `"sigla": "TD", "nivel": 2 }`) {
		t.Errorf("el registro nuevo:\n%s", prop.Cambios[0].Despues)
	}
	if b, _ := os.ReadFile(ruta); string(b) != viejo {
		t.Fatal("la vista previa escribió")
	}

	w = pedir(t, h, "POST", "/api/configurar/aplicar", `{"id":"`+prop.ID+`"}`, true)
	if w.Code != 200 {
		t.Fatalf("aplicar: %d %s", w.Code, w.Body)
	}
	if ps, err := proyectos.Leer(ruta); err != nil || len(ps) != 2 || ps[1].Nombre != "Tienda Demo" {
		t.Fatalf("registro: %+v %v", ps, err)
	}
	if l := s.listaProyectos(); len(l) != 2 {
		t.Errorf("la portada no lo ve: %+v", l)
	}

	w = pedir(t, h, "GET", "/api/configurar/historial", "", false)
	var hist []configurar.Entrada
	_ = json.Unmarshal(w.Body.Bytes(), &hist)
	if len(hist) != 1 || hist[0].ID != prop.ID {
		t.Fatalf("historial: %s", w.Body)
	}

	w = pedir(t, h, "POST", "/api/configurar/deshacer", `{"id":"`+prop.ID+`"}`, true)
	if w.Code != 200 {
		t.Fatalf("deshacer: %d %s", w.Code, w.Body)
	}
	if b, _ := os.ReadFile(ruta); string(b) != viejo {
		t.Errorf("deshecho: %s", b)
	}
}

// Si el panel aplica antes de que Vigilar haya mirado el registro (al arrancar lee primero las cuotas), el cambio no
// se pierde: la portada lo ve igual.
func TestAplicarAntesDeLaPrimeraVuelta(t *testing.T) {
	carpeta := filepath.Join(t.TempDir(), "Tienda")
	_ = os.MkdirAll(carpeta, 0o755)
	s, _ := conRegistro(t, "[]")
	s.registroVisto = false // como recién arrancado
	h := s.Handler()
	w := pedir(t, h, "POST", "/api/configurar/proyectos", `{"agregar":[{"ruta":`+comillas(carpeta)+`,"nivel":1}]}`, true)
	var prop configurar.Propuesta
	_ = json.Unmarshal(w.Body.Bytes(), &prop)
	if w = pedir(t, h, "POST", "/api/configurar/aplicar", `{"id":"`+prop.ID+`"}`, true); w.Code != 200 {
		t.Fatalf("aplicar: %d %s", w.Code, w.Body)
	}
	if l := s.listaProyectos(); len(l) != 1 || l[0].Nombre != "Tienda" {
		t.Errorf("la portada: %+v", l)
	}
	if s.releerRegistro() {
		t.Error("la vuelta siguiente no debería ver un cambio nuevo")
	}
}

// Lo que el registro no acepta: rutas que no son carpetas, repetidos, niveles fuera de 0 a 3, siglas raras, quitar
// lo que no está, y un registro roto que hay que arreglar a mano.
func TestRegistroQueSeRechaza(t *testing.T) {
	carpeta := filepath.Join(t.TempDir(), "Blog")
	_ = os.MkdirAll(carpeta, 0o755)
	s, _ := conRegistro(t, `[{"nombre":"Blog","ruta":`+comillas(carpeta)+`,"nivel":1}]`)
	h := s.Handler()
	malos := map[string]string{
		"relativa":      `{"agregar":[{"ruta":"proyectos/x","nivel":1}]}`,
		"no existe":     `{"agregar":[{"ruta":` + comillas(filepath.Join(carpeta, "no-existe")) + `,"nivel":1}]}`,
		"repetido":      `{"agregar":[{"ruta":` + comillas(carpeta) + `,"nivel":1}]}`,
		"nivel":         `{"agregar":[{"ruta":` + comillas(t.TempDir()) + `,"nivel":7}]}`,
		"sigla":         `{"agregar":[{"ruta":` + comillas(t.TempDir()) + `,"nivel":1,"sigla":"<b>"}]}`,
		"quitar ajeno":  `{"quitar":["Tienda"]}`,
		"nada":          `{}`,
		"no es un JSON": `{agregar`,
	}
	for nombre, cuerpo := range malos {
		if w := pedir(t, h, "POST", "/api/configurar/proyectos", cuerpo, true); w.Code != http.StatusBadRequest {
			t.Errorf("%s: %d %s", nombre, w.Code, w.Body)
		}
	}
	roto, _ := conRegistro(t, `[{"nombre":"Blog"`)
	if w := pedir(t, roto.Handler(), "POST", "/api/configurar/proyectos", `{"quitar":["Blog"]}`, true); w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), "arréglalo a mano") {
		t.Errorf("registro roto: %d %s", w.Code, w.Body)
	}
}

// Las carpetas recientes: una sesión en el repositorio y otra en un worktree suyo (.claude/worktrees, con su propio
// .git) son una sola carpeta, la del repositorio, con las dos sesiones; y no está registrada.
func TestCarpetasRecientesJuntanLosWorktrees(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "Tienda")
	arbol := filepath.Join(repo, ".claude", "worktrees", "f6-x")
	_ = os.MkdirAll(filepath.Join(repo, ".git"), 0o755)
	_ = os.MkdirAll(arbol, 0o755)
	_ = os.WriteFile(filepath.Join(arbol, ".git"), []byte("gitdir: x"), 0o644)
	_ = os.WriteFile(filepath.Join(repo, "AGENTS.md"), []byte("# Tienda"), 0o644)
	transcritos := t.TempDir()
	hora := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	for i, cwd := range []string{repo, arbol} {
		dir := filepath.Join(transcritos, fmt.Sprintf("p%d", i))
		_ = os.MkdirAll(dir, 0o755)
		linea := `{"type":"user","timestamp":"` + hora + `","cwd":` + comillas(cwd) + `,"message":{"role":"user","content":"Hola"}}` + "\n"
		_ = os.WriteFile(filepath.Join(dir, fmt.Sprintf("ses%d.jsonl", i)), []byte(linea), 0o644)
	}
	s := &Servidor{Cuotas: cuotas.NuevoLector("x", t.TempDir()), Claude: claude.NuevoIndice(transcritos)}
	if _, err := s.Claude.Actualizar(); err != nil {
		t.Fatal(err)
	}
	cs := s.carpetasRecientes()
	if len(cs) != 1 || !configurar.MismaRuta(cs[0].Ruta, repo) || cs[0].Claude != 2 || cs[0].Registrada || !cs[0].Marcas.AgentsMD {
		t.Fatalf("carpetas: %+v", cs)
	}
	// Registrada, la carpeta sale marcada; y una subcarpeta suya no aparece aparte.
	s.Proyectos = []Proyecto{{Nombre: "Tienda", Raiz: repo}}
	if cs := s.carpetasRecientes(); len(cs) != 1 || !cs[0].Registrada {
		t.Errorf("registrada: %+v", cs)
	}
}

// Las preferencias del asistente: solo claude o codex, el secundario distinto del principal; se guardan en la carpeta
// de Atalaya por la misma puerta y se leen de vuelta.
func TestPreferencias(t *testing.T) {
	s, _ := conRegistro(t, "")
	h := s.Handler()
	for _, malo := range []string{`{"principal":"gemini"}`, `{"principal":"claude","secundario":"claude"}`, `{"principal":"codex","secundario":"x"}`} {
		if w := pedir(t, h, "POST", "/api/configurar/preferencias", malo, true); w.Code != http.StatusBadRequest {
			t.Errorf("%s: %d", malo, w.Code)
		}
	}
	w := pedir(t, h, "POST", "/api/configurar/preferencias", `{"principal":"codex","secundario":"claude","asistente":true}`, true)
	var prop configurar.Propuesta
	_ = json.Unmarshal(w.Body.Bytes(), &prop)
	if w.Code != 200 || prop.Resumen != "Preferencias: coordina Codex, con Claude Code de secundario" {
		t.Fatalf("%d %s", w.Code, w.Body)
	}
	if w = pedir(t, h, "POST", "/api/configurar/aplicar", `{"id":"`+prop.ID+`"}`, true); w.Code != 200 {
		t.Fatalf("aplicar: %d %s", w.Code, w.Body)
	}
	if p := s.preferencias(); p.Principal != "codex" || p.Secundario != "claude" || !p.Asistente {
		t.Errorf("guardadas: %+v", p)
	}
}

// Conectar Codex por MCP: la orden oficial con la dirección de este servidor, corrida sin shell por la puerta, y
// deshacer corre la inversa. Nada real se corre: las herramientas y el que corre son de mentira.
func TestConectarPorMCP(t *testing.T) {
	s, _ := conRegistro(t, "")
	s.herramientasPrueba = []entorno.Herramienta{
		{ID: "claude", Nombre: "Claude Code", Instalada: true, Ruta: `C:\bin\claude.exe`, MCP: true},
		{ID: "codex", Nombre: "Codex", Instalada: true, Ruta: `C:\npm\codex.cmd`},
	}
	pu, _ := s.puerta()
	var corridas []string
	pu.Correr = func(_ context.Context, exe string, args []string) (string, error) {
		corridas = append(corridas, exe+" "+strings.Join(args, " "))
		return "Added global MCP server 'atalaya'", nil
	}
	h := s.Handler()
	malos := map[string]int{
		`{"herramienta":"gemini","accion":"conectar"}`:   http.StatusBadRequest,
		`{"herramienta":"codex","accion":"borrar"}`:      http.StatusBadRequest,
		`{"herramienta":"claude","accion":"conectar"}`:   http.StatusConflict, // ya está conectada
		`{"herramienta":"codex","accion":"desconectar"}`: http.StatusConflict,
	}
	for cuerpo, codigo := range malos {
		if w := pedir(t, h, "POST", "/api/configurar/mcp", cuerpo, true); w.Code != codigo {
			t.Errorf("%s: %d, quería %d", cuerpo, w.Code, codigo)
		}
	}
	if w := pedir(t, h, "POST", "/api/configurar/mcp", `{"herramienta":"codex","accion":"conectar"}`, false); w.Code != http.StatusForbidden {
		t.Errorf("sin la cabecera del panel: %d", w.Code)
	}
	w := pedir(t, h, "POST", "/api/configurar/mcp", `{"herramienta":"codex","accion":"conectar"}`, true)
	var prop configurar.Propuesta
	_ = json.Unmarshal(w.Body.Bytes(), &prop)
	if w.Code != 200 || prop.Resumen != "Conectar Codex a Atalaya por MCP" || len(corridas) != 0 {
		t.Fatalf("vista previa: %d %s %q", w.Code, w.Body, corridas)
	}
	if w = pedir(t, h, "POST", "/api/configurar/aplicar", `{"id":"`+prop.ID+`"}`, true); w.Code != 200 {
		t.Fatalf("aplicar: %d %s", w.Code, w.Body)
	}
	if w = pedir(t, h, "POST", "/api/configurar/deshacer", `{"id":"`+prop.ID+`"}`, true); w.Code != 200 {
		t.Fatalf("deshacer: %d %s", w.Code, w.Body)
	}
	quiere := []string{`C:\npm\codex.cmd mcp add atalaya --url http://127.0.0.1:7420/mcp`, `C:\npm\codex.cmd mcp remove atalaya`}
	if strings.Join(corridas, "|") != strings.Join(quiere, "|") {
		t.Errorf("corridas: %q", corridas)
	}
	// Claude Code va para todos los proyectos (--scope user).
	if con, des := ordenesMCP("claude", "http://localhost:7421/mcp"); strings.Join(con, " ") != "mcp add --transport http --scope user atalaya http://localhost:7421/mcp" || strings.Join(des, " ") != "mcp remove --scope user atalaya" {
		t.Errorf("claude: %q %q", con, des)
	}
	for _, u := range []string{"http://127.0.0.1:7420/mcp", "http://localhost:80/mcp"} {
		if !reURLMCP.MatchString(u) {
			t.Errorf("%s debería valer", u)
		}
	}
	for _, u := range []string{"http://127.0.0.1:7420/mcp&calc", "http://evil.example:7420/mcp", "http://127.0.0.1/mcp", "http://127.0.0.1:7420/mcp \"x\""} {
		if reURLMCP.MatchString(u) {
			t.Errorf("%s no debería valer", u)
		}
	}
}

// /api/entorno responde aun sin herramientas que buscar, con el sistema y las carpetas.
func TestEntornoSinCasa(t *testing.T) {
	s, _ := conRegistro(t, "")
	w := pedir(t, s.Handler(), "GET", "/api/entorno", "", false)
	var e Entorno
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &e) != nil || e.Sistema == "" || e.Herramientas == nil || e.Carpetas == nil {
		t.Errorf("%d %s", w.Code, w.Body)
	}
}
