// Package codexvivo lee la traza en vivo de las llamadas a Codex:
// scripts/codex.ps1 escribe los eventos de `codex exec --json` en
// %LOCALAPPDATA%\Atalaya\codex\<fecha>-<tema>.jsonl y una ficha <lo mismo>.meta.json que completa al
// terminar. Cada llamada se vuelve un agente con sus pasos: las órdenes que ejecutó, lo que razonó, lo que respondió.
// Solo lee, y cada archivo se sigue desde el último byte leído.
package codexvivo

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/secretos"
)

// Meta: la ficha que escribe codex.ps1.
type Meta struct {
	Tema          string `json:"tema"`
	Clase         string `json:"clase"`
	Modelo        string `json:"modelo"`
	Effort        string `json:"effort"`
	Fase          string `json:"fase"`
	Raiz          string `json:"raiz"`
	Version       string `json:"version"`
	MinutosMax    int    `json:"minutos_max"`
	Inicio        string `json:"inicio"`
	Base          string `json:"base"`
	Fin           string `json:"fin"`
	Salida        *int   `json:"salida"`
	Segundos      int    `json:"segundos"`
	TiempoAgotado bool   `json:"tiempo_agotado"`
	Informe       string `json:"informe"`
	// Sesion: el id de la sesión de Claude Code que corrió codex.ps1 (CLAUDE_CODE_SESSION_ID); vacío en las
	// llamadas de antes o lanzadas a mano.
	Sesion string `json:"sesion"`
}

// Llamada: una ejecución de codex.ps1.
type Llamada struct {
	ID        string        `json:"id"`
	Nombre    string        `json:"nombre"` // <fecha>-<tema>[-hhmmss]
	Meta      Meta          `json:"meta"`
	Inicio    time.Time     `json:"inicio"`
	Ultimo    time.Time     `json:"ultimo"`
	Terminada bool          `json:"terminada"`
	Tokens    claude.Tokens `json:"tokens"`
	Pasos     []claude.Paso `json:"-"`

	// Una sesión interactiva de Codex (sesiones.go), no una llamada de codex.ps1: Origen es el que anota Codex
	// («cli» la TUI, «vscode» la app, «exec»…), EnTurno si hay un turno abierto, Ruta su rollout y
	// Carpeta, donde trabaja (Meta.Raiz es la raíz del proyecto que la contiene: Lector.RaizDe).
	Interactiva bool   `json:"interactiva,omitempty"`
	Origen      string `json:"origen,omitempty"`
	EnTurno     bool   `json:"en_turno,omitempty"`
	// TurnoInformado: Codex dijo dónde empieza y termina cada turno (task_started y task_complete).
	TurnoInformado bool   `json:"-"`
	Ruta           string `json:"ruta,omitempty"`
	Carpeta        string `json:"carpeta,omitempty"`

	detalles map[int]*claude.DetallePaso
	ses      *sesion
	items    map[string]int // id del item -> índice del paso
	offset   int64
	tam      int64
	mod      time.Time
}

// Lector sigue la carpeta y, si Sesiones no está vacía, las sesiones interactivas de esa carpeta de Codex
// (%USERPROFILE%\.codex; sesiones.go).
type Lector struct {
	Dir      string
	Sesiones string
	// RaizDe: la raíz del proyecto que contiene una carpeta, o "" si ninguno (una sesión abierta en backend/ es del
	// proyecto de arriba). Sin ella, la carpeta misma.
	RaizDe func(carpeta string) string
	Ahora  func() time.Time
	mu     sync.RWMutex
	ll     map[string]*Llamada

	indice   []string // rutas de rollouts del índice de Codex (rutasDelIndice)
	indiceEn time.Time
}

func NuevoLector(dir string) *Lector {
	return &Lector{Dir: dir, Ahora: time.Now, ll: map[string]*Llamada{}}
}

// Prefijo de los ids de las llamadas en vivo.
const Prefijo = "codexvivo~"

// LimiteTexto: lo más que se guarda de la salida de un paso.
const LimiteTexto = 64 * 1024

// Actualizar lee lo nuevo de cada llamada. Devuelve los ids que cambiaron.
func (l *Lector) Actualizar() ([]string, error) {
	rutas, err := filepath.Glob(filepath.Join(l.Dir, "*.jsonl"))
	if err != nil {
		return nil, err
	}
	var cambiados []string
	for _, r := range rutas {
		nombre := strings.TrimSuffix(filepath.Base(r), ".jsonl")
		fi, err := os.Stat(r)
		if err != nil {
			continue
		}
		var meta Meta
		metaOK := false
		metaMod := time.Time{}
		if b, err := os.ReadFile(filepath.Join(l.Dir, nombre+".meta.json")); err == nil {
			metaOK = json.Unmarshal(trimBOM(b), &meta) == nil
			if mi, err := os.Stat(filepath.Join(l.Dir, nombre+".meta.json")); err == nil {
				metaMod = mi.ModTime()
			}
		}
		l.mu.Lock()
		c := l.ll[nombre]
		if c == nil || fi.Size() < c.offset {
			c = &Llamada{ID: Prefijo + nombre, Nombre: nombre, detalles: map[int]*claude.DetallePaso{}, items: map[string]int{}}
			l.ll[nombre] = c
		}
		// Una ficha que no se pudo leer (a medio escribir, o dañada) no borra la última buena.
		if !metaOK {
			meta = c.Meta
		}
		cambio := fi.Size() != c.tam || !fi.ModTime().Equal(c.mod) || meta.Fin != c.Meta.Fin
		c.Meta = meta
		if cambio {
			l.leer(c, r, fi)
			ultimo := fi.ModTime()
			if metaMod.After(ultimo) {
				ultimo = metaMod
			}
			c.Ultimo = ultimo
			cambiados = append(cambiados, c.ID)
		}
		c.Terminada = meta.Fin != ""
		if t, err := time.Parse(time.RFC3339Nano, meta.Inicio); err == nil {
			c.Inicio = t
		}
		if t, err := time.Parse(time.RFC3339Nano, meta.Fin); err == nil {
			c.Ultimo = t
		}
		l.mu.Unlock()
	}
	return append(cambiados, l.actualizarSesiones()...), nil
}

func trimBOM(b []byte) []byte {
	// El BOM de UTF-8 en bytes: PowerShell 5.1 lo escribe a veces al principio del JSON.
	return bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
}

// evento de `codex exec --json`.
type evento struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Item    *item  `json:"item"`
	Usage   *struct {
		Input  int64 `json:"input_tokens"`
		Cached int64 `json:"cached_input_tokens"`
		Output int64 `json:"output_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type item struct {
	ID               string `json:"id"`
	Type             string `json:"type"`
	Text             string `json:"text"`
	Command          string `json:"command"`
	AggregatedOutput string `json:"aggregated_output"`
	ExitCode         *int   `json:"exit_code"`
	Status           string `json:"status"`
	Query            string `json:"query"`
	Server           string `json:"server"`
	Tool             string `json:"tool"`
	Message          string `json:"message"`
	Changes          []struct {
		Path string `json:"path"`
		Kind string `json:"kind"`
	} `json:"changes"`
	Items []struct {
		Text      string `json:"text"`
		Completed bool   `json:"completed"`
	} `json:"items"`
}

// leer sigue el archivo desde donde quedó. Los eventos no traen hora: un paso nuevo lleva la hora en que se leyó;
// los de una llamada ya terminada que se lee de una vez se reparten entre su inicio y su fin.
func (l *Lector) leer(c *Llamada, ruta string, fi os.FileInfo) {
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
	nuevosDesde := len(c.Pasos)
	for {
		b, err := r.ReadBytes('\n')
		if err != nil {
			break // una línea a medio escribir queda para la próxima vuelta
		}
		ofs += int64(len(b))
		var e evento
		if json.Unmarshal(b, &e) != nil {
			continue
		}
		c.aplicar(e)
	}
	c.offset, c.tam, c.mod = ofs, fi.Size(), fi.ModTime()

	ahora := l.Ahora()
	inicio, errI := time.Parse(time.RFC3339Nano, c.Meta.Inicio)
	fin, errF := time.Parse(time.RFC3339Nano, c.Meta.Fin)
	repartir := nuevosDesde == 0 && errI == nil && errF == nil && len(c.Pasos) > 0
	for i := nuevosDesde; i < len(c.Pasos); i++ {
		switch {
		case repartir:
			c.Pasos[i].Hora = inicio.Add(fin.Sub(inicio) * time.Duration(i+1) / time.Duration(len(c.Pasos)+1))
		default:
			c.Pasos[i].Hora = ahora
		}
	}
}

func (c *Llamada) aplicar(e evento) {
	switch e.Type {
	case "turn.completed":
		if e.Usage != nil {
			c.Tokens.Entrada += e.Usage.Input - e.Usage.Cached
			c.Tokens.CacheLeida += e.Usage.Cached
			c.Tokens.Salida += e.Usage.Output
		}
	case "turn.failed", "error":
		msg := e.Message
		if e.Error != nil {
			msg = e.Error.Message
		}
		c.agregar(claude.Paso{Tipo: "Error", Herramienta: "codex", Resumen: primeraLinea(msg), Meta: "error", Error: true, Hecho: true},
			&claude.DetallePaso{Orden: "—", Salida: msg})
	case "item.started", "item.updated", "item.completed":
		if e.Item == nil {
			return
		}
		c.item(e.Item, e.Type == "item.completed")
	}
}

func (c *Llamada) item(it *item, terminado bool) {
	p, d := pasoDe(it, terminado)
	if i, ok := c.items[it.ID]; ok && it.ID != "" {
		// El mismo item que avanza (empezó, terminó): conserva su lugar en la línea de tiempo.
		viejo := c.Pasos[i]
		p.N, p.Hora, p.Turno, p.Lote, p.Paralelo, p.Continua = viejo.N, viejo.Hora, viejo.Turno, viejo.Lote, viejo.Paralelo, viejo.Continua
		c.Pasos[i] = p
		c.detalles[i] = d
		return
	}
	c.agregar(p, d)
	if it.ID != "" {
		c.items[it.ID] = len(c.Pasos) - 1
	}
}

func (c *Llamada) agregar(p claude.Paso, d *claude.DetallePaso) {
	p.N = len(c.Pasos)
	p.Turno, p.Lote, p.Paralelo, p.Continua = 1, p.N+1, 1, -1
	c.Pasos = append(c.Pasos, p)
	c.detalles[p.N] = d
}

// pasoDe traduce un item de Codex a un paso del panel.
func pasoDe(it *item, terminado bool) (claude.Paso, *claude.DetallePaso) {
	p := claude.Paso{Hecho: terminado || it.Status == "completed" || it.Status == "failed", Herramienta: it.Type}
	d := &claude.DetallePaso{}
	switch it.Type {
	case "command_execution":
		p.Tipo, p.Herramienta = "Ejecuta", "shell"
		p.Resumen = primeraLinea(it.Command)
		d.Orden, d.Salida = it.Command, it.AggregatedOutput
		d.Stdout = it.AggregatedOutput
		if it.ExitCode != nil {
			p.Meta = fmt.Sprintf("salida %d · %d líneas", *it.ExitCode, contarLineas(it.AggregatedOutput))
			p.Error = *it.ExitCode != 0
		} else if p.Hecho {
			p.Meta = fmt.Sprintf("%d líneas", contarLineas(it.AggregatedOutput))
		}
		p.TokEntra = (int64(len(it.AggregatedOutput)) + 3) / 4
	case "agent_message":
		p.Tipo, p.Herramienta = "Responde", "Respuesta"
		p.Resumen, d.Orden, d.Salida = primeraLinea(it.Text), "—", it.Text
		p.Meta = fmt.Sprintf("%d caracteres", len([]rune(it.Text)))
		p.Hecho = true
	case "reasoning":
		p.Tipo, p.Herramienta = "Razona", "Razonamiento"
		p.Resumen, d.Orden, d.Salida = primeraLinea(it.Text), "—", it.Text
		p.Meta = fmt.Sprintf("%d caracteres", len([]rune(it.Text)))
		p.Hecho = true
	case "file_change":
		p.Tipo = "Edita"
		var rutas []string
		for _, ch := range it.Changes {
			rutas = append(rutas, ch.Kind+" "+ch.Path)
		}
		p.Resumen = primeraLinea(strings.Join(rutas, ", "))
		d.Orden, d.Salida = strings.Join(rutas, "\n"), it.Status
		if len(it.Changes) == 1 {
			p.Hilo = it.Changes[0].Path
		}
	case "web_search":
		p.Tipo, p.Resumen, d.Orden = "Consulta", primeraLinea(it.Query), it.Query
	case "mcp_tool_call":
		p.Tipo, p.Resumen, d.Orden = "Usa", it.Server+"."+it.Tool, it.Server+"."+it.Tool
		if claude.EsServidorAtalaya(it.Server) {
			// Una consulta a Atalaya (mcp.go), con el mismo nombre de herramienta que en Claude Code.
			p.Tipo, p.Herramienta = "Atalaya", "mcp__atalaya__"+it.Tool
		}
	case "todo_list":
		p.Tipo = "Planifica"
		var lista []string
		pendiente := ""
		for _, t := range it.Items {
			marca := "[ ]"
			if t.Completed {
				marca = "[x]"
			} else if pendiente == "" {
				pendiente = t.Text
			}
			lista = append(lista, marca+" "+t.Text)
		}
		p.Resumen = primeraLinea(pendiente)
		d.Orden, d.Salida = "—", strings.Join(lista, "\n")
		p.Hecho = true
	case "error":
		p.Tipo, p.Error, p.Hecho = "Error", true, true
		p.Resumen, d.Orden, d.Salida = primeraLinea(it.Message), "—", it.Message
	default:
		p.Tipo, p.Resumen = "Usa", it.Type
	}
	p.Resumen = secretos.Tapar(p.Resumen)
	d.Orden = secretos.Tapar(recortar(d.Orden))
	d.Salida = secretos.Tapar(recortar(d.Salida))
	d.Stdout = secretos.Tapar(recortar(d.Stdout))
	return p, d
}

func primeraLinea(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	// La negrita de Markdown no va en un resumen de una línea.
	s = strings.TrimSpace(strings.ReplaceAll(strings.Trim(s, "# "), "**", ""))
	if r := []rune(s); len(r) > 160 {
		s = string(r[:159]) + "…"
	}
	return s
}

func contarLineas(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

func recortar(s string) string {
	if len(s) <= LimiteTexto {
		return s
	}
	corte := LimiteTexto
	for corte > 0 && (s[corte]&0xC0) == 0x80 {
		corte--
	}
	return s[:corte] + fmt.Sprintf("\n… (%d bytes más)", len(s)-corte)
}

// Llamadas devuelve una copia de todas, de la más reciente a la más vieja (sin los detalles).
func (l *Lector) Llamadas() []Llamada {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]Llamada, 0, len(l.ll))
	for _, c := range l.ll {
		cp := *c
		cp.Pasos = append([]claude.Paso{}, c.Pasos...)
		cp.detalles, cp.items, cp.ses = nil, nil, nil
		out = append(out, cp)
	}
	for i := range out {
		if out[i].Interactiva {
			out[i].Meta.Raiz = out[i].Carpeta
			if l.RaizDe != nil {
				if r := l.RaizDe(out[i].Carpeta); r != "" {
					out[i].Meta.Raiz = r
				}
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ultimo.After(out[j].Ultimo) })
	return out
}

// Llamada devuelve una por su id.
func (l *Lector) Llamada(id string) (Llamada, bool) {
	for _, c := range l.Llamadas() {
		if c.ID == id {
			return c, true
		}
	}
	return Llamada{}, false
}

// ErrSinPaso: no existe ese paso.
var ErrSinPaso = errors.New("no existe ese paso")

// Detalle devuelve la orden y la salida de un paso.
func (l *Lector) Detalle(id string, n int) (claude.DetallePaso, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	c := l.ll[strings.TrimPrefix(id, Prefijo)]
	if c == nil || n < 0 || n >= len(c.Pasos) {
		return claude.DetallePaso{}, ErrSinPaso
	}
	d := claude.DetallePaso{Paso: c.Pasos[n]}
	if x := c.detalles[n]; x != nil {
		d.Orden, d.Salida, d.Stdout = x.Orden, x.Salida, x.Stdout
	}
	return d, nil
}

// Activa: una llamada sin ficha de fin que escribió hace poco (codex.ps1 la cierra siempre, también al fallar); una
// sesión interactiva, según activaSesion.
func (c Llamada) Activa(ahora time.Time) bool {
	if c.Interactiva {
		return c.activaSesion(ahora)
	}
	return !c.Terminada && ahora.Sub(c.Ultimo) < 3*time.Minute
}
