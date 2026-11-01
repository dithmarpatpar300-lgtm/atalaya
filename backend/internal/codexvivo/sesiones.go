package codexvivo

// Las sesiones interactivas de Codex: la TUI (`codex` a secas) y lo que se abre desde la app. Codex guarda cada una
// en <CODEX_HOME>/sessions/AAAA/MM/DD/rollout-<fecha>-<id>.jsonl (al archivarla, en archived_sessions/), una línea
// JSON por cosa: session_meta (la carpeta, el origen, la versión), turn_context (el modelo y el effort de cada turno),
// response_item (mensajes, razonamiento, herramientas y sus salidas) y event_msg (inicio y fin del turno, tokens,
// salida de las órdenes). state_5.sqlite es su índice: de ahí solo se toma la ruta de las que no estén donde se
// buscan. Las comprimidas (.jsonl.zst, las viejas) no se leen. Todo en solo lectura; las llamadas de codex.ps1 no
// pasan por aquí (corren con --ephemeral y no dejan rollout).

import (
	"bufio"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/secretos"

	_ "modernc.org/sqlite"
)

// PrefijoSesion: el nombre de una sesión interactiva entre las llamadas («sesion-<id de Codex>»).
const PrefijoSesion = "sesion-"

// VentanaSesiones: solo se siguen las sesiones que escribieron en los últimos 30 días.
const VentanaSesiones = 30 * 24 * time.Hour

// sesion: lo que se lleva de una sesión mientras se lee.
type sesion struct {
	llamadas map[string]int // call_id -> índice del paso
	turno    int
}

// reIDRollout: el id de Codex al final del nombre del archivo.
var reIDRollout = regexp.MustCompile(`([0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12})\.jsonl$`)

// actualizarSesiones lee lo nuevo de cada sesión interactiva. Sin carpeta de Codex (otra versión, otro sistema), nada.
func (l *Lector) actualizarSesiones() []string {
	if l.Sesiones == "" {
		return nil
	}
	if fi, err := os.Stat(l.Sesiones); err != nil || !fi.IsDir() {
		return nil
	}
	ahora := l.Ahora()
	desde := ahora.Add(-VentanaSesiones)
	// Por id, la ruta más reciente: una sesión archivada se mudó de carpeta y sigue siendo la misma.
	rutas := map[string]string{}
	mods := map[string]time.Time{}
	ver := func(r string) {
		m := reIDRollout.FindStringSubmatch(filepath.Base(r))
		if m == nil {
			return
		}
		fi, err := os.Stat(r)
		if err != nil || fi.IsDir() || fi.ModTime().Before(desde) {
			return
		}
		id := strings.ToLower(m[1])
		if fi.ModTime().After(mods[id]) {
			rutas[id], mods[id] = r, fi.ModTime()
		}
	}
	for d := desde; !d.After(ahora.Add(24 * time.Hour)); d = d.Add(24 * time.Hour) {
		m, _ := filepath.Glob(filepath.Join(l.Sesiones, "sessions", d.Format("2006"), d.Format("01"), d.Format("02"), "rollout-*.jsonl"))
		for _, r := range m {
			ver(r)
		}
	}
	m, _ := filepath.Glob(filepath.Join(l.Sesiones, "archived_sessions", "rollout-*.jsonl"))
	for _, r := range m {
		ver(r)
	}
	for _, r := range l.rutasDelIndice(desde) {
		ver(r)
	}

	var cambiados []string
	for id, r := range rutas {
		fi, err := os.Stat(r)
		if err != nil {
			continue
		}
		nombre := PrefijoSesion + id
		l.mu.Lock()
		c := l.ll[nombre]
		if c == nil || c.Ruta != r || fi.Size() < c.offset {
			c = &Llamada{ID: Prefijo + nombre, Nombre: nombre, Interactiva: true, Ruta: r,
				detalles: map[int]*claude.DetallePaso{}, items: map[string]int{}, ses: &sesion{llamadas: map[string]int{}}}
			l.ll[nombre] = c
		}
		if fi.Size() != c.tam || !fi.ModTime().Equal(c.mod) {
			l.leerSesion(c, r, fi)
			cambiados = append(cambiados, c.ID)
		}
		l.mu.Unlock()
	}
	return cambiados
}

// rutasDelIndice: las rutas de los rollouts que state_5.sqlite tiene por recientes, cada 30 s. La base se abre solo
// lectura (mode=ro y query_only); si no está o no se puede leer, ninguna.
func (l *Lector) rutasDelIndice(desde time.Time) []string {
	if l.Ahora().Sub(l.indiceEn) < 30*time.Second {
		return l.indice
	}
	l.indiceEn = l.Ahora()
	l.indice = nil
	ruta := filepath.Join(l.Sesiones, "state_5.sqlite")
	if _, err := os.Stat(ruta); err != nil {
		return nil
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(ruta)+"?mode=ro&_pragma=query_only(1)&_pragma=busy_timeout(500)")
	if err != nil {
		return nil
	}
	defer db.Close()
	filas, err := db.Query(`SELECT rollout_path FROM threads WHERE updated_at >= ?`, desde.Unix())
	if err != nil {
		return nil
	}
	defer filas.Close()
	for filas.Next() {
		var r string
		if filas.Scan(&r) == nil && strings.HasSuffix(r, ".jsonl") {
			l.indice = append(l.indice, r)
		}
	}
	return l.indice
}

// lineaRollout: una línea del rollout.
type lineaRollout struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// carga: los campos que se usan de cualquier payload (cada tipo trae los suyos).
type carga struct {
	Type string `json:"type"`
	// session_meta
	ID         string          `json:"id"`
	Timestamp  string          `json:"timestamp"`
	Cwd        string          `json:"cwd"`
	CliVersion string          `json:"cli_version"`
	Source     json.RawMessage `json:"source"`
	// turn_context
	Model           string `json:"model"`
	Effort          string `json:"effort"`
	ReasoningEffort string `json:"reasoning_effort"`
	// response_item
	Role string `json:"role"`
	// Lista de partes con texto; en turn_context, summary es un texto («auto»): por eso, crudos (textoDe).
	Content   json.RawMessage `json:"content"`
	Summary   json.RawMessage `json:"summary"`
	Name      string          `json:"name"`
	Arguments string          `json:"arguments"`
	Input     string          `json:"input"`
	CallID    string          `json:"call_id"`
	Output    json.RawMessage `json:"output"`
	Status    string          `json:"status"`
	Action    *struct {
		Command json.RawMessage `json:"command"`
		Query   string          `json:"query"`
	} `json:"action"`
	// event_msg
	Message          string `json:"message"`
	ExitCode         *int   `json:"exit_code"`
	AggregatedOutput string `json:"aggregated_output"`
	Info             *struct {
		Total *usoTokens `json:"total_token_usage"`
	} `json:"info"`
	Total *usoTokens `json:"total_token_usage"`
}

type usoTokens struct {
	Input  int64 `json:"input_tokens"`
	Cached int64 `json:"cached_input_tokens"`
	Output int64 `json:"output_tokens"`
}

// leerSesion sigue el rollout desde donde quedó. Cada línea trae su hora.
func (l *Lector) leerSesion(c *Llamada, ruta string, fi os.FileInfo) {
	f, err := os.Open(ruta)
	if err != nil {
		return
	}
	defer f.Close()
	if _, err := f.Seek(c.offset, io.SeekStart); err != nil {
		return
	}
	r := bufio.NewReaderSize(f, 256*1024)
	ofs := c.offset
	for {
		b, err := r.ReadBytes('\n')
		if err != nil {
			break // una línea a medio escribir queda para la próxima vuelta
		}
		ofs += int64(len(b))
		var ln lineaRollout
		if json.Unmarshal(b, &ln) != nil {
			continue
		}
		var p carga
		if json.Unmarshal(ln.Payload, &p) != nil {
			continue
		}
		hora, err := time.Parse(time.RFC3339Nano, ln.Timestamp)
		if err != nil {
			hora = l.Ahora()
		}
		c.aplicarRollout(ln.Type, p, hora)
		if hora.After(c.Ultimo) {
			c.Ultimo = hora
		}
	}
	c.offset, c.tam, c.mod = ofs, fi.Size(), fi.ModTime()
	if c.Inicio.IsZero() {
		c.Inicio = c.Ultimo
	}
}

func (c *Llamada) aplicarRollout(tipo string, p carga, hora time.Time) {
	switch tipo {
	case "session_meta":
		c.Carpeta, c.Meta.Version = p.Cwd, p.CliVersion
		c.Origen = origenDe(p.Source)
		if t, err := time.Parse(time.RFC3339Nano, p.Timestamp); err == nil {
			c.Inicio = t
		}
	case "turn_context":
		if p.Cwd != "" {
			c.Carpeta = p.Cwd
		}
		if p.Model != "" {
			c.Meta.Modelo = p.Model
		}
		if e := p.Effort + p.ReasoningEffort; e != "" {
			c.Meta.Effort = e
		}
	case "token_usage_record":
		c.fijarTokens(p.Total)
	case "event_msg":
		c.eventoSesion(p, hora)
	case "response_item":
		c.itemSesion(p, hora)
	}
}

// textoDe: el texto de una lista de partes ([{"type", "text"}]), una línea por parte; o el texto mismo.
func textoDe(raw json.RawMessage) string {
	var partes []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &partes) == nil {
		var t []string
		for _, x := range partes {
			if x.Text != "" {
				t = append(t, x.Text)
			}
		}
		return strings.Join(t, "\n")
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

// origenDe: «cli», «vscode», «exec»… o «subagente» si el origen es un objeto.
func origenDe(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	if len(raw) > 0 && raw[0] == '{' {
		return "subagente"
	}
	return ""
}

func (c *Llamada) fijarTokens(u *usoTokens) {
	if u == nil {
		return
	}
	// Codex informa el total acumulado de la sesión: se reemplaza, no se suma.
	c.Tokens = claude.Tokens{Entrada: u.Input - u.Cached, CacheLeida: u.Cached, Salida: u.Output}
}

func (c *Llamada) eventoSesion(p carga, hora time.Time) {
	switch p.Type {
	case "task_started", "turn_started":
		c.EnTurno = true
	case "task_complete", "turn_complete", "turn_aborted":
		c.EnTurno = false
		// Lo que quedó sin salida al cerrar el turno ya no va a tenerla.
		for i := range c.Pasos {
			c.Pasos[i].Hecho = true
		}
	case "user_message":
		texto := strings.TrimSpace(p.Message)
		if texto == "" {
			return
		}
		c.ses.turno++
		if c.Meta.Tema == "" {
			c.Meta.Tema = secretos.Tapar(primeraLinea(texto))
		}
		c.agregarSesion(claude.Paso{Hora: hora, Tipo: "Pide", Herramienta: "Mensaje", Resumen: primeraLinea(texto),
			Meta: fmt.Sprintf("%d caracteres", len([]rune(texto))), Hecho: true}, &claude.DetallePaso{Orden: "—", Salida: texto})
	case "token_count":
		if p.Info != nil {
			c.fijarTokens(p.Info.Total)
		}
	case "exec_command_end":
		// La salida de la orden con su código: completa el paso de su llamada.
		if i, ok := c.ses.llamadas[p.CallID]; ok && p.ExitCode != nil {
			c.completarOrden(i, *p.ExitCode, p.AggregatedOutput)
		}
	case "error":
		c.agregarSesion(claude.Paso{Hora: hora, Tipo: "Error", Herramienta: "codex", Resumen: primeraLinea(p.Message), Meta: "error", Error: true, Hecho: true},
			&claude.DetallePaso{Orden: "—", Salida: p.Message})
	}
}

func (c *Llamada) itemSesion(p carga, hora time.Time) {
	switch p.Type {
	case "message":
		if p.Role != "assistant" {
			return // lo del usuario llega limpio por event_msg (user_message), sin el contexto que Codex le agrega
		}
		texto := textoDe(p.Content)
		if strings.TrimSpace(texto) == "" {
			return
		}
		c.agregarSesion(claude.Paso{Hora: hora, Tipo: "Responde", Herramienta: "Respuesta", Resumen: primeraLinea(texto),
			Meta: fmt.Sprintf("%d caracteres", len([]rune(texto))), Hecho: true}, &claude.DetallePaso{Orden: "—", Salida: texto})
	case "reasoning":
		texto := textoDe(p.Summary)
		if strings.TrimSpace(texto) == "" {
			return // razonamiento cifrado, sin resumen
		}
		c.agregarSesion(claude.Paso{Hora: hora, Tipo: "Razona", Herramienta: "Razonamiento", Resumen: primeraLinea(texto),
			Meta: fmt.Sprintf("%d caracteres", len([]rune(texto))), Hecho: true}, &claude.DetallePaso{Orden: "—", Salida: texto})
	case "function_call", "custom_tool_call":
		args := p.Arguments
		if p.Type == "custom_tool_call" {
			args = p.Input
		}
		paso, d := pasoDeHerramienta(p.Name, args)
		paso.Hora = hora
		c.agregarSesion(paso, d)
		if p.CallID != "" {
			c.ses.llamadas[p.CallID] = len(c.Pasos) - 1
		}
	case "local_shell_call":
		var orden string
		if p.Action != nil {
			orden = ordenDe(p.Action.Command)
		}
		c.agregarSesion(claude.Paso{Hora: hora, Tipo: "Ejecuta", Herramienta: "shell", Resumen: primeraLinea(orden)}, &claude.DetallePaso{Orden: orden})
		if p.CallID != "" {
			c.ses.llamadas[p.CallID] = len(c.Pasos) - 1
		}
	case "web_search_call":
		q := ""
		if p.Action != nil {
			q = p.Action.Query
		}
		c.agregarSesion(claude.Paso{Hora: hora, Tipo: "Consulta", Herramienta: "web_search", Resumen: primeraLinea(q), Hecho: true},
			&claude.DetallePaso{Orden: secretos.Tapar(q)})
	case "function_call_output", "custom_tool_call_output":
		i, ok := c.ses.llamadas[p.CallID]
		if !ok {
			return
		}
		if c.Pasos[i].Tipo == "Ejecuta" && c.Pasos[i].Hecho {
			return // ya la completó exec_command_end, con la salida sin el encabezado que agrega Codex
		}
		salida, codigo := salidaDe(p.Output)
		if c.Pasos[i].Tipo == "Ejecuta" && codigo != nil {
			c.completarOrden(i, *codigo, salida)
			return
		}
		c.Pasos[i].Hecho = true
		if d := c.detalles[i]; d != nil && salida != "" {
			d.Salida = secretos.Tapar(recortar(salida))
			if c.Pasos[i].Tipo == "Ejecuta" {
				d.Stdout = d.Salida
				c.Pasos[i].Meta = fmt.Sprintf("%d líneas", contarLineas(salida))
			}
		}
		c.Pasos[i].TokEntra = (int64(len(salida)) + 3) / 4
	}
}

func (c *Llamada) completarOrden(i, codigo int, salida string) {
	p := &c.Pasos[i]
	p.Hecho, p.Error = true, codigo != 0
	p.Meta = fmt.Sprintf("salida %d · %d líneas", codigo, contarLineas(salida))
	p.TokEntra = (int64(len(salida)) + 3) / 4
	if d := c.detalles[i]; d != nil {
		d.Salida = secretos.Tapar(recortar(salida))
		d.Stdout = d.Salida
	}
}

func (c *Llamada) agregarSesion(p claude.Paso, d *claude.DetallePaso) {
	p.N = len(c.Pasos)
	p.Turno, p.Lote, p.Paralelo, p.Continua = c.ses.turno, p.N+1, 1, -1
	p.Resumen = secretos.Tapar(p.Resumen)
	d.Orden = secretos.Tapar(recortar(d.Orden))
	d.Salida = secretos.Tapar(recortar(d.Salida))
	c.Pasos = append(c.Pasos, p)
	c.detalles[p.N] = d
}

// reArchivoParche: las rutas de un apply_patch («*** Update File: ruta»).
var reArchivoParche = regexp.MustCompile(`(?m)^\*\*\* (Add|Update|Delete) File: (.+?)\s*$`)

// pasoDeHerramienta traduce una llamada a herramienta de la sesión a un paso del panel.
func pasoDeHerramienta(nombre, args string) (claude.Paso, *claude.DetallePaso) {
	p := claude.Paso{Herramienta: nombre}
	d := &claude.DetallePaso{Orden: args}
	var a struct {
		Command json.RawMessage `json:"command"`
		Cmd     string          `json:"cmd"`
		Input   string          `json:"input"`
		Plan    []struct {
			Step   string `json:"step"`
			Status string `json:"status"`
		} `json:"plan"`
	}
	_ = json.Unmarshal([]byte(args), &a)
	switch {
	case nombre == "shell" || nombre == "shell_command" || nombre == "exec_command" || nombre == "container.exec" || nombre == "local_shell":
		orden := a.Cmd
		if orden == "" {
			orden = ordenDe(a.Command)
		}
		p.Tipo, p.Herramienta = "Ejecuta", "shell"
		p.Resumen, d.Orden = primeraLinea(orden), orden
	case nombre == "apply_patch":
		parche := args
		if a.Input != "" {
			parche = a.Input
		}
		p.Tipo = "Edita"
		var rutas []string
		for _, m := range reArchivoParche.FindAllStringSubmatch(parche, -1) {
			rutas = append(rutas, strings.ToLower(m[1])+" "+m[2])
		}
		p.Resumen, d.Orden = primeraLinea(strings.Join(rutas, ", ")), parche
		if len(rutas) == 1 {
			p.Hilo = strings.SplitN(rutas[0], " ", 2)[1]
		}
	case nombre == "update_plan":
		p.Tipo, p.Hecho = "Planifica", true
		var lista []string
		pendiente := ""
		for _, s := range a.Plan {
			marca := "[ ]"
			switch s.Status {
			case "completed":
				marca = "[x]"
			case "in_progress":
				marca = "[>]"
				if pendiente == "" {
					pendiente = s.Step
				}
			}
			lista = append(lista, marca+" "+s.Step)
		}
		p.Resumen, d.Orden, d.Salida = primeraLinea(pendiente), "—", strings.Join(lista, "\n")
	case strings.Contains(nombre, "__"):
		// Una herramienta de un servidor MCP: «servidor__herramienta» (o «mcp__servidor__herramienta»).
		partes := strings.Split(strings.TrimPrefix(nombre, "mcp__"), "__")
		servidor, herr := partes[0], strings.Join(partes[1:], "__")
		p.Tipo, p.Resumen = "Usa", servidor+"."+herr
		if claude.EsServidorAtalaya(servidor) {
			p.Tipo, p.Herramienta = "Atalaya", "mcp__atalaya__"+herr
		}
	default:
		p.Tipo, p.Resumen = "Usa", nombre
	}
	return p, d
}

// ordenDe: la orden de un shell, que llega como texto o como lista («powershell.exe -Command …», «bash -lc …»):
// de la lista, lo que va tras -Command o -c.
func ordenDe(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var lista []string
	if json.Unmarshal(raw, &lista) != nil || len(lista) == 0 {
		return ""
	}
	for i, x := range lista[:len(lista)-1] {
		switch strings.ToLower(x) {
		case "-c", "-lc", "-command", "/c":
			return strings.Join(lista[i+1:], " ")
		}
	}
	return strings.Join(lista, " ")
}

// reCodigoSalida: «Exit code: 0» o «Process exited with code 1» al principio de la salida de una orden.
var reCodigoSalida = regexp.MustCompile(`(?i)^(?:exit code:|process exited with code)\s*(-?\d+)`)

// salidaDe: el texto de la salida de una herramienta y, si lo dice, su código. Llega como texto, como un JSON
// {"output", "metadata": {"exit_code"}} dentro de ese texto, o como lista de partes.
func salidaDe(raw json.RawMessage) (string, *int) {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		if s = textoDe(raw); s == "" {
			s = string(raw)
		}
	}
	var envuelta struct {
		Output   *string `json:"output"`
		Metadata struct {
			ExitCode *int `json:"exit_code"`
		} `json:"metadata"`
	}
	if strings.HasPrefix(strings.TrimSpace(s), "{") && json.Unmarshal([]byte(s), &envuelta) == nil && envuelta.Output != nil {
		return *envuelta.Output, envuelta.Metadata.ExitCode
	}
	if m := reCodigoSalida.FindStringSubmatch(strings.TrimSpace(s)); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil {
			return s, &n
		}
	}
	return s, nil
}

// Activa: para una sesión interactiva, la que escribió hace menos de 2 min, o la que tiene un turno abierto y
// escribió hace menos de 15 (una orden larga calla un rato).
func (c Llamada) activaSesion(ahora time.Time) bool {
	quieta := ahora.Sub(c.Ultimo)
	return quieta < 2*time.Minute || (c.EnTurno && quieta < 15*time.Minute)
}

// EnEspera: una sesión interactiva que terminó su turno hace menos de 30 min y seguramente sigue abierta, esperando
// al usuario. Codex no deja huella al cerrar la TUI: pasada media hora, se da por terminada.
func (c Llamada) EnEspera(ahora time.Time) bool {
	return c.Interactiva && !c.Activa(ahora) && ahora.Sub(c.Ultimo) < 30*time.Minute
}
