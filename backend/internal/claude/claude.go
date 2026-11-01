// Package claude lee los transcritos de Claude Code (~/.claude/projects/**/*.jsonl)
// y arma, por cada sesión y cada subagente, sus tokens y sus pasos. Solo lee, y
// lee cada archivo una vez: lo que crece se sigue leyendo desde el último byte.
// La orden y la salida de cada paso no se guardan en memoria: se releen del
// archivo por su posición cuando el panel las pide.
package claude

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"atalaya/internal/secretos"
)

// Tokens sumados de un agente.
type Tokens struct {
	Entrada      int64 `json:"entrada"`
	CacheEscrita int64 `json:"cache_escrita"`
	CacheLeida   int64 `json:"cache_leida"`
	Salida       int64 `json:"salida"`
}

func (t *Tokens) sumar(u Tokens) {
	t.Entrada += u.Entrada
	t.CacheEscrita += u.CacheEscrita
	t.CacheLeida += u.CacheLeida
	t.Salida += u.Salida
}

// Agente: una sesión de Claude Code o uno de sus subagentes.
type Agente struct {
	ID        string `json:"id"`
	Proyecto  string `json:"proyecto"`
	Raiz      string `json:"raiz"` // cwd del transcrito
	Proveedor string `json:"proveedor"`
	Tipo      string `json:"tipo"` // sesion | subagente
	Rol       string `json:"rol"`
	Tarea     string `json:"tarea"`
	// Titulo: el nombre que Claude Code le dio a la sesión (custom-title, si no ai-title, si no summary); vacío si
	// no tiene. Solo las sesiones lo llevan.
	Titulo      string `json:"titulo,omitempty"`
	Modelo      string `json:"modelo"`
	Effort      string `json:"effort"`
	Clase       string `json:"clase"`
	Presupuesto string `json:"presupuesto"`
	Base        string `json:"base"`
	Asignacion  string `json:"asignacion"`
	// Lo demás de la línea «Asignación», partido en sus campos.
	Nivel          string `json:"nivel,omitempty"`
	ClaseDetalle   string `json:"clase_detalle,omitempty"` // «I1 (diseño cerrado, lógica pura)»
	RolAsignado    string `json:"rol_asignado,omitempty"`  // «medidor (general-purpose)»
	ModeloAsignado string `json:"modelo_asignado,omitempty"`
	Cuota          string `json:"cuota,omitempty"` // la postura de la cuota al lanzar
	// Archivos nombrados en el encargo (entre comillas invertidas): la frontera que se le dio.
	Nombrados []string  `json:"nombrados,omitempty"`
	Encargo   string    `json:"encargo"`
	Bitacora  string    `json:"bitacora"` // ruta relativa a Raiz, si el encargo la nombra
	Informe   string    `json:"informe"`
	Padre     string    `json:"padre,omitempty"`
	Inicio    time.Time `json:"inicio"`
	Ultimo    time.Time `json:"ultimo"`
	Tokens    Tokens    `json:"tokens"`
	Pasos     int       `json:"pasos"`
	// Pendiente: el último paso es una herramienta que todavía no devolvió su resultado.
	Pendiente bool `json:"-"`
	// Cerrado: el subagente entregó (SubagentHandback) o la sesión terminó su turno.
	Cerrado bool      `json:"-"`
	Mod     time.Time `json:"-"`
	// ToolUseID: la llamada a Agent que lanzó a este subagente (de su .meta.json).
	ToolUseID string `json:"-"`
	// Temporales: las carpetas «<fecha>-…» bajo una raíz «…-tmp» que nombran sus pasos (temporalesDe), al día con
	// cada paso que se lee; con ellas se buscan el acta y la bitácora de un agente que no las nombra.
	Temporales []string `json:"-"`
}

// Paso de la actividad.
type Paso struct {
	N           int       `json:"n"`
	Hora        time.Time `json:"hora"`
	Tipo        string    `json:"tipo"`
	Herramienta string    `json:"herramienta"`
	Resumen     string    `json:"resumen"`
	Meta        string    `json:"meta"`
	Error       bool      `json:"error"`
	Hecho       bool      `json:"hecho"`

	// La continuidad (ver continuidad.go).
	Turno int    `json:"turno"` // cuántos pedidos de la persona van hasta aquí
	Etapa string `json:"etapa"` // el paso del plan (update_plan o TodoWrite) en curso
	Lote  int    `json:"lote"`  // el mensaje de Claude que lo emitió: las llamadas de un mismo lote van en paralelo
	// En paralelo: cuántas herramientas lanzó el mismo mensaje (1 si fue sola).
	Paralelo    int    `json:"paralelo"`
	Hilo        string `json:"hilo"`        // de qué trata: el archivo que lee o edita; vacío si no se sabe
	Profundidad int    `json:"profundidad"` // cuántos desvíos lo contienen
	Desvio      bool   `json:"desvio"`      // aquí empieza un desvío del hilo anterior
	Continua    int    `json:"continua"`    // el paso anterior del mismo hilo cuando se vuelve tras un desvío; -1 si no
	Subagente   string `json:"subagente,omitempty"`

	// Tokens del paso. TokEntra: lo que su resultado mete al contexto (≈ bytes ÷ 4). TokSalida: la salida del
	// mensaje que lo emitió, compartida por todas las llamadas de su lote. TokContexto: el contexto de la petición
	// que lo emitió. TokTras: el contexto de la petición siguiente, que ya lleva su resultado (continuidad.go).
	TokEntra    int64 `json:"tok_entra"`
	TokSalida   int64 `json:"tok_salida"`
	TokContexto int64 `json:"tok_contexto"`
	TokTras     int64 `json:"tok_tras"`

	// Dónde releer la orden y la salida.
	ofsUso, ofsRes int64
	bloque         int    // índice del bloque dentro del mensaje
	idUso          string // tool_use id
	editado        string // el archivo que edita o escribe, ruta completa normalizada (huellas.go)
	orden          string // la orden de un «Ejecuta», recortada, para buscarla (buscar.go)
	// ordenNorm: la orden ya normalizada (Normalizar), una vez al leerla: la salud, las sesiones y los lanzamientos de
	// Codex la miran en cada pedido del panel, y normalizar miles de órdenes cada vez era lo más caro del servidor.
	ordenNorm string
	marcas    uint8 // marcasDe, una vez en agregar (revision.go)
}

// Linea de un diff, como la guarda Claude Code en structuredPatch.
type Trozo struct {
	ViejoDesde  int      `json:"viejo_desde"`
	ViejoLineas int      `json:"viejo_lineas"`
	NuevoDesde  int      `json:"nuevo_desde"`
	NuevoLineas int      `json:"nuevo_lineas"`
	Lineas      []string `json:"lineas"` // cada una empieza con " ", "+" o "-"
}

// DetallePaso: la orden exacta y su salida, en crudo y ya separadas.
type DetallePaso struct {
	Paso
	Orden  string `json:"orden"`
	Salida string `json:"salida"`
	// Lo estructurado, cuando el transcrito lo trae.
	Archivo      string  `json:"archivo,omitempty"`
	Diff         []Trozo `json:"diff,omitempty"`
	Stdout       string  `json:"stdout,omitempty"`
	Stderr       string  `json:"stderr,omitempty"`
	Interrumpido bool    `json:"interrumpido,omitempty"`
	// Lo crudo: la entrada de la herramienta y su resultado tal como están en el transcrito (JSON).
	EntradaCruda string `json:"entrada_cruda,omitempty"`
	SalidaCruda  string `json:"salida_cruda,omitempty"`
}

type archivo struct {
	ruta     string
	offset   int64
	tam      int64
	mod      time.Time
	agente   *Agente
	pasos    []*Paso
	abiertos map[string]*Paso // tool_use id -> paso sin resultado
	usos     map[string]Tokens
	// Continuidad mientras se lee.
	turno     int
	ultimoMsg string
	lote      int
	delLote   []*Paso // los pasos del mensaje en curso, para darles su usage cuando llega
	// Cada petición a la API (un mensaje de Claude con su usage) y cada compactación (contexto.go).
	peticiones     []Peticion
	porPeticion    map[string]int
	compactaciones []Compactacion
	plan           []string
	planActual     int
	// Un encargo que solo dice «Lee tu encargo completo en `…encargo-x.md`»: el nombre del archivo, y el id del
	// Read que lo abre; su resultado es el encargo de verdad (encargoLeido).
	encargoArchivo, encargoRead string
	// Las decisiones que la sesión te pidió (decisiones.go): los botones que esperan tu mensaje siguiente y las
	// preguntas que esperan su resultado, por el id de su tool_use.
	decisiones []*Decision
	botones    *Decision
	preguntas  map[string]*Decision
	// Los nombres de la sesión (tituloDe): la última línea custom-title, su custom-title.json, la última ai-title y
	// el último summary.
	tituloPropio, tituloArchivo, tituloIA, resumen string
}

// Indice guarda todos los transcritos leídos.
type Indice struct {
	Raiz string
	// Delegadas: la ficha de scripts/claude.ps1 de una sesión, si la tiene (delegadas.go); nil, ninguna.
	Delegadas func(id string) (Delegacion, bool)
	mu        sync.RWMutex
	arch      map[string]*archivo
}

func NuevoIndice(raiz string) *Indice { return &Indice{Raiz: raiz, arch: map[string]*archivo{}} }

// Actualizar mira los archivos y lee lo nuevo. Devuelve los ids de los agentes que cambiaron.
func (ix *Indice) Actualizar() ([]string, error) {
	rutas, err := listar(ix.Raiz)
	if err != nil {
		return nil, err
	}
	var cambiados []string
	for _, r := range rutas {
		fi, err := os.Stat(r)
		if err != nil {
			continue
		}
		ix.mu.RLock()
		a := ix.arch[r]
		ix.mu.RUnlock()
		if a != nil && a.tam == fi.Size() && a.mod.Equal(fi.ModTime()) {
			continue
		}
		ix.mu.Lock()
		if a == nil || fi.Size() < a.offset {
			a = nuevoArchivo(r)
			ix.arch[r] = a
		}
		err = a.leer(fi)
		ix.mu.Unlock()
		if err == nil && a.agente.ID != "" {
			cambiados = append(cambiados, a.agente.ID)
		}
	}
	cambiados = append(cambiados, ix.soltarBorrados(rutas)...)
	return cambiados, nil
}

// soltarBorrados quita del índice los transcritos que ya no están en el disco (Claude Code los borra al limpiar sesiones
// viejas): si no, el índice crece sin fin y sus agentes siguen saliendo. Solo los que `os.Stat` confirma que no existen:
// uno que no salió en el listado por un fallo pasajero se queda.
func (ix *Indice) soltarBorrados(listadas []string) []string {
	vistas := make(map[string]bool, len(listadas))
	for _, r := range listadas {
		vistas[r] = true
	}
	ix.mu.RLock()
	var candidatas []string
	for r := range ix.arch {
		if !vistas[r] {
			candidatas = append(candidatas, r)
		}
	}
	ix.mu.RUnlock()
	var ids []string
	for _, r := range candidatas {
		if _, err := os.Stat(r); !errors.Is(err, os.ErrNotExist) {
			continue
		}
		ix.mu.Lock()
		if a := ix.arch[r]; a != nil {
			if a.agente.ID != "" {
				ids = append(ids, a.agente.ID)
			}
			delete(ix.arch, r)
		}
		ix.mu.Unlock()
	}
	return ids
}

func listar(raiz string) ([]string, error) {
	sesiones, err := filepath.Glob(filepath.Join(raiz, "*", "*.jsonl"))
	if err != nil {
		return nil, err
	}
	subs, _ := filepath.Glob(filepath.Join(raiz, "*", "*", "subagents", "*.jsonl"))
	return append(sesiones, subs...), nil
}

func nuevoArchivo(ruta string) *archivo {
	a := &archivo{ruta: ruta, abiertos: map[string]*Paso{}, usos: map[string]Tokens{}, agente: &Agente{Proveedor: "Claude"}, planActual: -1}
	base := strings.TrimSuffix(filepath.Base(ruta), ".jsonl")
	if filepath.Base(filepath.Dir(ruta)) == "subagents" {
		sesion := filepath.Base(filepath.Dir(filepath.Dir(ruta)))
		idAg := strings.TrimPrefix(base, "agent-")
		a.agente.ID = sesion + "~" + idAg
		a.agente.Tipo = "subagente"
		a.agente.Padre = sesion
		a.agente.Rol = "Subagente"
		leerMeta(strings.TrimSuffix(ruta, ".jsonl")+".meta.json", a.agente)
	} else {
		a.agente.ID = base
		a.agente.Tipo = "sesion"
		a.agente.Rol = "Sesión"
	}
	return a
}

type meta struct {
	AgentType   string `json:"agentType"`
	Description string `json:"description"`
	Model       string `json:"model"`
	Effort      string `json:"effort"`
	ToolUseID   string `json:"toolUseId"`
}

func leerMeta(ruta string, ag *Agente) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return
	}
	var m meta
	if json.Unmarshal(b, &m) != nil {
		return
	}
	if m.AgentType != "" {
		ag.Rol = mayuscula(m.AgentType)
	}
	ag.Tarea, ag.Modelo, ag.Effort, ag.ToolUseID = m.Description, m.Model, m.Effort, m.ToolUseID
}

func mayuscula(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	return strings.ToUpper(string(r[0])) + string(r[1:])
}

// linea es lo que se usa de cada registro del transcrito.
type linea struct {
	Type        string    `json:"type"`
	IsSidechain bool      `json:"isSidechain"`
	IsMeta      bool      `json:"isMeta"`
	Timestamp   time.Time `json:"timestamp"`
	Cwd         string    `json:"cwd"`
	Message     *mensaje  `json:"message"`
	// El nombre de la sesión: {"type":"custom-title","customTitle":…}, {"type":"ai-title","aiTitle":…} o
	// {"type":"summary","summary":…}.
	CustomTitle *string `json:"customTitle"`
	AiTitle     string  `json:"aiTitle"`
	Summary     string  `json:"summary"`
	// Una compactación: {"type":"system","subtype":"compact_boundary","compactMetadata":{…}}.
	Subtype         string `json:"subtype"`
	CompactMetadata *struct {
		Trigger    string `json:"trigger"`
		PreTokens  int64  `json:"preTokens"`
		PostTokens int64  `json:"postTokens"`
	} `json:"compactMetadata"`
}

type mensaje struct {
	ID         string          `json:"id"`
	Model      string          `json:"model"`
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content"`
	StopReason string          `json:"stop_reason"`
	Usage      *struct {
		Input       int64 `json:"input_tokens"`
		CacheCreate int64 `json:"cache_creation_input_tokens"`
		CacheRead   int64 `json:"cache_read_input_tokens"`
		Output      int64 `json:"output_tokens"`
	} `json:"usage"`
}

type bloque struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   json.RawMessage `json:"content"`
	IsError   bool            `json:"is_error"`
}

func bloques(raw json.RawMessage) (texto string, bs []bloque) {
	if len(raw) == 0 {
		return "", nil
	}
	if raw[0] == '"' {
		_ = json.Unmarshal(raw, &texto)
		return texto, nil
	}
	_ = json.Unmarshal(raw, &bs)
	return "", bs
}

func (a *archivo) leer(fi os.FileInfo) error {
	f, err := os.Open(a.ruta)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Seek(a.offset, io.SeekStart); err != nil {
		return err
	}
	r := bufio.NewReaderSize(f, 256*1024)
	ofs := a.offset
	for {
		b, err := r.ReadBytes('\n')
		if err != nil {
			// Una línea a medio escribir se deja para la próxima vuelta.
			break
		}
		a.procesar(b, ofs)
		ofs += int64(len(b))
	}
	a.offset = ofs
	a.tam, a.mod = fi.Size(), fi.ModTime()
	a.agente.Mod = fi.ModTime()
	if a.agente.Tipo == "sesion" {
		// Claude Code guarda también el nombre en <carpeta de la sesión>/custom-title.json; se relee cuando el
		// transcrito cambia (renombrar la sesión escribe una línea custom-title nueva).
		a.tituloArchivo = leerTituloArchivo(filepath.Join(strings.TrimSuffix(a.ruta, ".jsonl"), "custom-title.json"))
		a.agente.Titulo = a.tituloDe()
	}
	return nil
}

// tituloDe: el nombre que se le puso a la sesión manda; si no, el que propuso Claude Code; si no, su resumen.
func (a *archivo) tituloDe() string {
	for _, t := range []string{a.tituloPropio, a.tituloArchivo, a.tituloIA, a.resumen} {
		if t != "" {
			return t
		}
	}
	return ""
}

func leerTituloArchivo(ruta string) string {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return ""
	}
	var m struct {
		CustomTitle string `json:"customTitle"`
	}
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	return limpiarTitulo(m.CustomTitle)
}

// limpiarTitulo: una sola línea, sin secretos y con tope.
func limpiarTitulo(t string) string {
	return secretos.Tapar(primeraLinea(t, 160))
}

// anotarTitulo guarda el nombre de la sesión de una línea custom-title, ai-title o summary; la última manda.
func (a *archivo) anotarTitulo(l linea) {
	if a.agente.Tipo != "sesion" {
		return
	}
	switch l.Type {
	case "custom-title":
		if l.CustomTitle != nil {
			a.tituloPropio = limpiarTitulo(*l.CustomTitle) // vacío: se le quitó el nombre
		}
	case "ai-title":
		if t := limpiarTitulo(l.AiTitle); t != "" {
			a.tituloIA = t
		}
	case "summary":
		if t := limpiarTitulo(l.Summary); t != "" {
			a.resumen = t
		}
	}
	a.agente.Titulo = a.tituloDe()
}

func (a *archivo) procesar(b []byte, ofs int64) {
	var l linea
	if json.Unmarshal(b, &l) != nil {
		return
	}
	switch l.Type {
	case "custom-title", "ai-title", "summary":
		a.anotarTitulo(l)
		return
	}
	if l.Type == "system" && l.Subtype == "compact_boundary" && l.CompactMetadata != nil {
		a.compactaciones = append(a.compactaciones, Compactacion{Hora: l.Timestamp, Antes: l.CompactMetadata.PreTokens,
			Despues: l.CompactMetadata.PostTokens, Motivo: l.CompactMetadata.Trigger, Peticion: len(a.peticiones)})
		return
	}
	if l.Message == nil {
		return
	}
	ag := a.agente
	// En el archivo de una sesión, las líneas de subagentes (formato viejo) no son de la sesión.
	if ag.Tipo == "sesion" && l.IsSidechain {
		return
	}
	if !l.Timestamp.IsZero() {
		if ag.Inicio.IsZero() {
			ag.Inicio = l.Timestamp
		}
		ag.Ultimo = l.Timestamp
	}
	if ag.Raiz == "" && l.Cwd != "" {
		ag.Raiz = raizDelRepositorio(l.Cwd)
		ag.Proyecto = filepath.Base(ag.Raiz)
	}
	m := l.Message
	texto, bs := bloques(m.Content)
	switch l.Type {
	case "user":
		if l.IsMeta {
			return
		}
		if texto != "" {
			a.pedido(texto, l.Timestamp, ofs)
			return
		}
		for _, bl := range bs {
			switch bl.Type {
			case "text":
				a.pedido(bl.Text, l.Timestamp, ofs)
			case "tool_result":
				if p := a.abiertos[bl.ToolUseID]; p != nil {
					delete(a.abiertos, bl.ToolUseID)
					p.ofsRes, p.Hecho, p.Error = ofs, true, bl.IsError
					if bl.ToolUseID == a.encargoRead && !bl.IsError {
						a.encargoLeido(textoResultado(bl.Content))
					}
					if a.preguntas[bl.ToolUseID] != nil {
						a.responderPreguntas(bl.ToolUseID, b, l.Timestamp, bl.IsError)
					}
					p.Meta = metaResultado(p, bl)
					p.TokEntra = (int64(len(textoResultado(bl.Content))) + 3) / 4
				}
			}
		}
		ag.Pendiente = len(a.abiertos) > 0
	case "assistant":
		// El modelo real del mensaje gana al alias del .meta.json («opus»).
		if m.Model != "" && !strings.HasPrefix(m.Model, "<") {
			ag.Modelo = m.Model
		}
		if m.Usage != nil && m.ID != "" {
			nuevo := Tokens{Entrada: m.Usage.Input, CacheEscrita: m.Usage.CacheCreate, CacheLeida: m.Usage.CacheRead, Salida: m.Usage.Output}
			viejo := a.usos[m.ID]
			a.usos[m.ID] = nuevo
			ag.Tokens.sumar(Tokens{nuevo.Entrada - viejo.Entrada, nuevo.CacheEscrita - viejo.CacheEscrita, nuevo.CacheLeida - viejo.CacheLeida, nuevo.Salida - viejo.Salida})
		}
		// Un mismo mensaje de Claude llega en varias líneas (un bloque cada una): todas son el mismo lote.
		if m.ID == "" || m.ID != a.ultimoMsg {
			a.lote++
			a.ultimoMsg = m.ID
			a.delLote = a.delLote[:0]
		}
		if m.Usage != nil && m.ID != "" {
			a.anotarPeticion(m.ID, l.Timestamp, a.usos[m.ID])
		}
		for i, bl := range bs {
			switch bl.Type {
			case "text":
				if t := strings.TrimSpace(bl.Text); t != "" {
					p := &Paso{Hora: l.Timestamp, Tipo: "Responde", Herramienta: "Respuesta", Resumen: primeraLinea(t, 160), Meta: fmt.Sprintf("%d caracteres", len([]rune(t))), Hecho: true, ofsUso: ofs, ofsRes: -1, bloque: i}
					a.agregar(p)
					a.delLote = append(a.delLote, p)
				}
			case "tool_use":
				p := &Paso{Hora: l.Timestamp, Tipo: tipoDe(bl.Name), Herramienta: bl.Name, Resumen: resumenDe(bl.Name, bl.Input), ofsUso: ofs, ofsRes: -1, bloque: i, idUso: bl.ID}
				if ag.Tipo == "sesion" {
					a.anotarDecision(bl.Name, bl.Input, bl.ID, l.Timestamp, len(a.pasos))
				}
				if a.encargoArchivo != "" && a.encargoRead == "" && bl.Name == "Read" && strings.HasSuffix(strings.ToLower(rutaLeida(bl.Input)), a.encargoArchivo) {
					a.encargoRead = bl.ID
				}
				if a.planificar(bl.Name, bl.Input) {
					p.Tipo = "Planifica"
					if et := a.etapa(); et != "" {
						p.Resumen = recortar(et, 160)
					}
				}
				p.Hilo = hiloDe(bl.Name, bl.Input)
				if p.Tipo == "Edita" || p.Tipo == "Escribe" {
					p.editado = claveArchivo(bl.Input)
				}
				if p.Tipo == "Ejecuta" {
					// Una orden que corre codex.ps1 va entera (hasta 4000): varias llamadas en una sola orden dejan
					// «codex.ps1» y los temas lejos del principio, y sin ellos la llamada no encuentra su sesión.
					orden, tope := secretos.Tapar(ordenDe(bl.Name, bl.Input)), 400
					if strings.Contains(strings.ToLower(orden), "codex.ps1") {
						tope = 4000
					}
					p.orden = recortar(orden, tope)
					p.ordenNorm = Normalizar(p.orden)
				}
				a.agregar(p)
				a.delLote = append(a.delLote, p)
				a.abiertos[bl.ID] = p
				if bl.Name == "SubagentHandback" {
					ag.Cerrado = true
				}
			}
		}
		// El usage se repite (y crece la salida) en cada línea del mensaje: el último manda para todo el lote.
		if m.Usage != nil {
			for _, p := range a.delLote {
				p.TokSalida = m.Usage.Output
				p.TokContexto = m.Usage.Input + m.Usage.CacheCreate + m.Usage.CacheRead
			}
		}
		ag.Pendiente = len(a.abiertos) > 0
		if ag.Tipo == "sesion" {
			ag.Cerrado = m.StopReason == "end_turn" && !ag.Pendiente
		}
	}
}

var (
	// Los bloques que la app pone delante de lo que escribes: avisos del sistema, el artifact que estás mirando, el
	// archivo abierto en el editor. Con su cierre o sin él (`<x/>`).
	reAvisos = func() []*regexp.Regexp {
		var out []*regexp.Regexp
		for _, tag := range []string{"system-reminder", "artifact-view-context", "artifact-content-authored-by-others", "ide_selection", "ide_opened_file"} {
			out = append(out, regexp.MustCompile(`(?s)^\s*<`+tag+`\b[^>]*?(?:/>|>.*?</`+tag+`>)`))
		}
		return out
	}()
	rePegado = regexp.MustCompile(`</?pasted_content[^>]*>`)
)

// sinAvisos quita los bloques de la app que van delante de un mensaje tuyo.
func sinAvisos(t string) string {
	t = strings.TrimSpace(t)
	for quito := true; quito; {
		quito = false
		for _, re := range reAvisos {
			if re.MatchString(t) {
				t, quito = strings.TrimSpace(re.ReplaceAllString(t, "")), true
			}
		}
	}
	return t
}

func (a *archivo) pedido(texto string, hora time.Time, ofs int64) {
	// Los avisos del sistema y lo que la app pone delante (el artifact que mirabas) no son el pedido.
	t := sinAvisos(texto)
	// Un texto pegado llega envuelto en <pasted_content>: es un pedido de la persona, no un aviso del sistema.
	t = strings.TrimSpace(rePegado.ReplaceAllString(t, ""))
	if t == "" || strings.HasPrefix(t, "<") {
		return // avisos del sistema y órdenes internas
	}
	ag := a.agente
	if ag.Encargo == "" {
		ag.Encargo = secretos.Tapar(recortar(t, 6000))
		leerAsignacion(ag, t)
		if ag.Tarea == "" {
			ag.Tarea = lineaDeTarea(quitarAsignacion(t), 140)
		}
		if ag.Asignacion == "" && ag.Tipo != "sesion" {
			if m := reEncargoArchivo.FindStringSubmatch(t); m != nil {
				a.encargoArchivo = strings.ToLower(m[1])
			}
		}
	}
	if ag.Tipo == "sesion" {
		ag.Cerrado = false
		a.responderBotones(t, hora)
	}
	// Un pedido nuevo abre un turno; el plan del turno anterior ya no rige.
	a.turno++
	a.plan, a.planActual = nil, -1
	a.agregar(&Paso{Hora: hora, Tipo: "Pide", Herramienta: "Mensaje", Resumen: primeraLinea(quitarAsignacion(t), 160), Meta: fmt.Sprintf("%d caracteres", len([]rune(t))), Hecho: true, ofsUso: ofs, ofsRes: -1, bloque: -1})
}

var (
	// «Lee tu encargo completo en `C:\…\scratchpad\encargo-medidor-w6b.md`»: el nombre del archivo.
	reEncargoArchivo = regexp.MustCompile("(?i)encargo[^`\\n]*`[^`\\n]*?[\\\\/]?([\\w.-]*encargo[\\w.-]*\\.md)`")
	// El Read devuelve «     1→texto» o «1\ttexto»: el número de línea no es del encargo.
	reNumLinea = regexp.MustCompile(`(?m)^\s*\d+(?:→|\t)`)
)

// encargoLeido: el agente abrió el archivo de su encargo; ese texto es el encargo, con su asignación y su tarea.
func (a *archivo) encargoLeido(texto string) {
	t := strings.TrimSpace(reNumLinea.ReplaceAllString(texto, ""))
	if t == "" {
		return
	}
	ag := a.agente
	ag.Encargo = secretos.Tapar(recortar(t, 6000))
	leerAsignacion(ag, t)
	// La tarea que salió del puntero («Lee tu encargo completo en…») no dice nada; la de .meta.json se queda.
	if ag.Tarea == "" || strings.Contains(strings.ToLower(ag.Tarea), "encargo") {
		ag.Tarea = lineaDeTarea(quitarAsignacion(t), 140)
	}
	a.encargoArchivo = ""
}

func rutaLeida(input json.RawMessage) string {
	var m struct {
		Ruta string `json:"file_path"`
	}
	_ = json.Unmarshal(input, &m)
	return m.Ruta
}

// raizDelRepositorio sube desde cwd hasta la carpeta con .git: una sesión abierta en
// «Tienda Demo\docs\operacion» es del proyecto Tienda Demo. Sin .git, el propio cwd.
func raizDelRepositorio(cwd string) string {
	for d := cwd; ; {
		if _, err := os.Stat(filepath.Join(d, ".git")); err == nil {
			return d
		}
		padre := filepath.Dir(d)
		if padre == d {
			return cwd
		}
		d = padre
	}
}

func (a *archivo) agregar(p *Paso) {
	p.N = len(a.pasos)
	p.Turno, p.Lote, p.Continua = a.turno, a.lote, -1
	if p.Etapa == "" {
		p.Etapa = a.etapa()
	}
	p.marcas = marcasDe(p)
	a.pasos = append(a.pasos, p)
	a.agente.Pasos = len(a.pasos)
	if a.agente.Tipo == "subagente" {
		a.agente.Temporales = sumarTemporales(a.agente.Temporales, p)
	}
}

var (
	// La línea «Asignación» termina en el cierre de la negrita («…».**) o en el fin de línea; lo que sigue en la
	// misma línea («Si vas a pasarte…») no es parte de ella.
	reAsignacion  = regexp.MustCompile(`(?m)^\*{0,2}Asignación:\s*(.+?)(?:\*\*|$)`)
	reClase       = regexp.MustCompile(`(?i)\bclase\s+([A-Z]\d?)\b`)
	rePresupuesto = regexp.MustCompile(`(?i)presupuesto\s+([^·*\n]+)`)
	reBase        = regexp.MustCompile("(?i)\\bbase\\s+([^·*\\n]+)")
	reBitacora    = regexp.MustCompile(`docs/trazabilidad/[\w.-]+/avance/[\w.-]+\.md`)
	reBitacoraRel = regexp.MustCompile(`\bavance/([\w.-]+\.md)`)
	reFase        = regexp.MustCompile(`docs/trazabilidad/(F\d+)/`)
	reInforme     = regexp.MustCompile(`docs/trazabilidad/F\d+/\d\d-[\w.-]+\.md`)
)

var (
	reModelo     = regexp.MustCompile(`^[A-Za-z][\w.-]*/[A-Za-z]+$`)
	reNombrado   = regexp.MustCompile("`([\\w./\\\\-]+\\.(?:rs|ts|tsx|mts|go|md|ps1|json|toml|js|mjs|css|ya?ml|sh|txt|html))`")
	maxNombrados = 24
)

// partirAsignacion lee «nivel enfocado · clase I1 (…) · rol codificador · sonnet/medium · presupuesto 30 min ·
// base … · cuota: Claude normal 4 %/11 % · Codex normal». La cuota va al final y puede llevar « · » propios.
func partirAsignacion(ag *Agente, asig string) {
	partes := strings.Split(asig, " · ")
	for i, p := range partes {
		p = strings.TrimSpace(p)
		switch {
		case strings.HasPrefix(p, "nivel "):
			ag.Nivel = strings.TrimSpace(strings.TrimPrefix(p, "nivel "))
		case strings.HasPrefix(p, "clase "):
			ag.ClaseDetalle = strings.TrimSpace(strings.TrimPrefix(p, "clase "))
		case strings.HasPrefix(p, "rol "):
			ag.RolAsignado = strings.TrimSpace(strings.TrimPrefix(p, "rol "))
		case strings.HasPrefix(p, "cuota:"), strings.HasPrefix(p, "cuota "):
			// Una plantilla puede escribir «cuota Codex normal · Claude normal (…)», sin los dos puntos.
			resto := append([]string{strings.TrimSpace(p[len("cuota "):])}, partes[i+1:]...)
			ag.Cuota = strings.TrimRight(strings.Join(resto, " · "), ". ")
			return
		case reModelo.MatchString(p):
			ag.ModeloAsignado = p
		}
	}
}

// nombrados: los archivos que el encargo nombra entre comillas invertidas, por su nombre, sin repetir.
func nombrados(t string) []string {
	visto := map[string]bool{}
	var out []string
	for _, m := range reNombrado.FindAllStringSubmatch(t, -1) {
		nombre := m[1]
		if i := strings.LastIndexAny(nombre, `/\`); i >= 0 {
			nombre = nombre[i+1:]
		}
		if visto[nombre] {
			continue
		}
		visto[nombre] = true
		out = append(out, nombre)
		if len(out) == maxNombrados {
			break
		}
	}
	return out
}

// FaseSinNombre: la fase de una bitácora que el encargo nombra sin ella («avance/codificador-w6a.md»). El índice no sabe
// en qué fase va cada proyecto; el servidor sí (faseActual), y la pone.
const FaseSinNombre = "F?"

func leerAsignacion(ag *Agente, t string) {
	m := reAsignacion.FindStringSubmatch(t)
	if m == nil {
		return
	}
	asig := strings.Trim(m[1], "* ")
	ag.Asignacion = asig
	partirAsignacion(ag, asig)
	ag.Nombrados = nombrados(t)
	if c := reClase.FindStringSubmatch(asig); c != nil {
		ag.Clase = strings.ToUpper(c[1])
	}
	if p := rePresupuesto.FindStringSubmatch(asig); p != nil {
		ag.Presupuesto = strings.TrimSpace(p[1])
	}
	if b := reBase.FindStringSubmatch(asig); b != nil {
		ag.Base = strings.Trim(strings.TrimSpace(b[1]), "`")
		ag.Base = strings.ReplaceAll(ag.Base, "`", "")
	}
	if b := reBitacora.FindString(t); b != "" {
		ag.Bitacora = b
	} else if b := reBitacoraRel.FindStringSubmatch(t); b != nil {
		// Sin fase en el encargo, queda FaseSinNombre: el servidor la cambia por la fase en curso del proyecto.
		fase := FaseSinNombre
		if f := reFase.FindStringSubmatch(t); f != nil {
			fase = f[1]
		}
		ag.Bitacora = "docs/trazabilidad/" + fase + "/avance/" + b[1]
	}
	// El informe propio es el último que nombra el encargo (los primeros suelen ser lecturas previas).
	if inf := reInforme.FindAllString(t, -1); len(inf) > 0 {
		ag.Informe = inf[len(inf)-1]
	}
}

func quitarAsignacion(t string) string {
	t = reAsignacion.ReplaceAllString(t, "")
	return strings.TrimSpace(t)
}

func primeraLinea(t string, max int) string {
	t = strings.TrimSpace(t)
	if i := strings.IndexByte(t, '\n'); i >= 0 {
		t = t[:i]
	}
	t = strings.Trim(t, "*# ")
	return recortar(t, max)
}

// lineaDeTarea: la primera línea con contenido; se saltan los rótulos («Task (from …):», «Contexto:»).
func lineaDeTarea(t string, max int) string {
	for _, l := range strings.Split(t, "\n") {
		l = strings.Trim(strings.TrimSpace(l), "*# ")
		if l != "" && !strings.HasSuffix(l, ":") {
			return recortar(l, max)
		}
	}
	return primeraLinea(t, max)
}

func recortar(t string, max int) string {
	r := []rune(t)
	if len(r) <= max {
		return t
	}
	return string(r[:max-1]) + "…"
}

func tipoDe(herr string) string {
	switch herr {
	case "Read", "NotebookRead":
		return "Lee"
	case "Edit", "MultiEdit", "NotebookEdit":
		return "Edita"
	case "Write":
		return "Escribe"
	case "Bash", "PowerShell", "BashOutput":
		return "Ejecuta"
	case "Grep", "Glob", "ToolSearch":
		return "Busca"
	case "Agent", "Task", "SendMessage":
		return "Delega"
	case "WebFetch", "WebSearch":
		return "Consulta"
	case "SubagentHandback":
		return "Entrega"
	}
	if _, ok := HerramientaAtalaya(herr); ok {
		return "Atalaya"
	}
	if strings.HasSuffix(herr, "__wrap_up") {
		return "Cierra"
	}
	return "Usa"
}

func resumenDe(herr string, input json.RawMessage) string {
	var m map[string]any
	_ = json.Unmarshal(input, &m)
	cad := func(k string) string { s, _ := m[k].(string); return s }
	var s string
	switch herr {
	case "Read", "Write", "Edit", "MultiEdit", "NotebookEdit":
		s = corta(cad("file_path"))
		if o, ok := m["offset"].(float64); ok {
			s += fmt.Sprintf(", desde la línea %d", int(o))
		}
	case "Bash", "PowerShell":
		s = cad("description")
		if s == "" {
			s = cad("command")
		}
	case "Grep", "Glob":
		s = cad("pattern")
	case "Agent", "Task":
		s = cad("description")
	case "WebFetch":
		s = cad("url")
	case "WebSearch":
		s = cad("query")
	case "SubagentHandback":
		s = "Entrega su informe"
	default:
		for _, k := range []string{"description", "query", "title", "prompt", "message"} {
			if s = cad(k); s != "" {
				break
			}
		}
	}
	return secretos.Tapar(primeraLinea(s, 160))
}

// corta deja la ruta relativa al proyecto cuando se puede.
func corta(ruta string) string {
	r := strings.ReplaceAll(ruta, "\\", "/")
	for _, marca := range []string{"/crates/", "/docs/", "/scripts/", "/backend/", "/frontend/", "/src/"} {
		if i := strings.Index(r, marca); i >= 0 {
			return r[i+1:]
		}
	}
	return r
}

func metaResultado(p *Paso, bl bloque) string {
	if bl.IsError {
		return "error"
	}
	salida := textoResultado(bl.Content)
	n := strings.Count(salida, "\n")
	if salida != "" && !strings.HasSuffix(salida, "\n") {
		n++
	}
	if p.Tipo == "Ejecuta" {
		if strings.Contains(salida, "Exit code") {
			if i := strings.Index(salida, "Exit code"); i >= 0 {
				return primeraLinea(salida[i:], 30)
			}
		}
		return fmt.Sprintf("salida 0 · %d líneas", n)
	}
	if n == 1 {
		return "1 línea"
	}
	return fmt.Sprintf("%d líneas", n)
}

func textoResultado(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == '"' {
		var s string
		_ = json.Unmarshal(raw, &s)
		return s
	}
	var partes []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	_ = json.Unmarshal(raw, &partes)
	var sb strings.Builder
	for _, p := range partes {
		switch p.Type {
		case "text":
			sb.WriteString(p.Text)
		case "image":
			sb.WriteString("[imagen]")
		}
	}
	return sb.String()
}

// Agentes devuelve una copia de todos los agentes, del más reciente al más viejo.
func (ix *Indice) Agentes() []Agente {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	out := make([]Agente, 0, len(ix.arch))
	for _, a := range ix.arch {
		if a.agente.Pasos == 0 {
			continue
		}
		out = append(out, ix.conDelegacion(*a.agente))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ultimo.After(out[j].Ultimo) })
	return out
}

// Agente devuelve uno y sus últimos `tope` pasos, ya enlazados (continuidad.go). La continuidad se calcula
// sobre todos los pasos, para que un desvío que empezó antes del corte cuente igual.
func (ix *Indice) Agente(id string, tope int) (Agente, []Paso, bool) {
	return ix.AgenteAntes(id, -1, tope)
}

// AgenteAntes: como Agente, pero los `tope` pasos que van justo antes del paso `antes` (el panel pide así las páginas
// viejas de una actividad que ya no caben en la primera respuesta). Con antes < 0, los últimos.
func (ix *Indice) AgenteAntes(id string, antes, tope int) (Agente, []Paso, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	var a *archivo
	for _, x := range ix.arch {
		if x.agente.ID == id {
			a = x
			break
		}
	}
	if a == nil {
		return Agente{}, nil, false
	}
	pasos := make([]Paso, len(a.pasos))
	for i, p := range a.pasos {
		pasos[i] = *p
	}
	enlazar(pasos)
	// Los subagentes que lanzó cada «Delega»: su .meta.json guarda el id de esa llamada.
	lanzados := map[string]string{}
	for _, x := range ix.arch {
		if x.agente.Padre == a.agente.ID && x.agente.ToolUseID != "" {
			lanzados[x.agente.ToolUseID] = x.agente.ID
		}
	}
	for i := range pasos {
		if id, ok := lanzados[pasos[i].idUso]; ok {
			pasos[i].Subagente = id
		}
	}
	// El paso N está en la posición N (agregar numera en orden).
	if antes >= 0 && antes < len(pasos) {
		pasos = pasos[:antes]
	}
	if tope > 0 && len(pasos) > tope {
		pasos = pasos[len(pasos)-tope:]
	}
	return ix.conDelegacion(*a.agente), pasos, true
}

// LimiteSalida: lo más que se devuelve de la salida de un paso.
const LimiteSalida = 64 * 1024

// Detalle relee del transcrito la orden y la salida del paso n.
func (ix *Indice) Detalle(id string, n int) (DetallePaso, error) {
	ix.mu.RLock()
	var ruta string
	var p Paso
	for _, a := range ix.arch {
		if a.agente.ID == id && n >= 0 && n < len(a.pasos) {
			ruta, p = a.ruta, *a.pasos[n]
			break
		}
	}
	ix.mu.RUnlock()
	if ruta == "" {
		return DetallePaso{}, errors.New("no existe ese paso")
	}
	d := DetallePaso{Paso: p}
	l, err := lineaEn(ruta, p.ofsUso)
	if err != nil {
		return d, err
	}
	texto, bs := bloques(l.Message.Content)
	switch {
	case p.bloque < 0 && texto != "":
		d.Orden = texto
	case p.bloque < 0:
		for _, b := range bs {
			if b.Type == "text" {
				d.Orden += b.Text
			}
		}
	case p.bloque < len(bs):
		b := bs[p.bloque]
		if b.Type == "text" {
			d.Orden = "—"
			d.Salida = b.Text
		} else {
			d.Orden = ordenDe(b.Name, b.Input)
			d.EntradaCruda = sangrar(b.Input)
			d.Diff = diffDeEntrada(b.Name, b.Input)
			var in struct {
				FilePath string `json:"file_path"`
			}
			_ = json.Unmarshal(b.Input, &in)
			d.Archivo = in.FilePath
		}
	}
	if p.ofsRes >= 0 {
		if crudo, err := lineaCruda(ruta, p.ofsRes); err == nil {
			var lr struct {
				Message       *mensaje        `json:"message"`
				ToolUseResult json.RawMessage `json:"toolUseResult"`
			}
			if json.Unmarshal(crudo, &lr) == nil && lr.Message != nil {
				_, rs := bloques(lr.Message.Content)
				for _, b := range rs {
					if b.Type == "tool_result" && b.ToolUseID == p.idUso {
						d.Salida = textoResultado(b.Content)
						d.SalidaCruda = sangrar(b.Content)
					}
				}
				d.estructurar(lr.ToolUseResult)
			}
		}
	} else if p.Tipo != "Responde" && p.Tipo != "Pide" {
		d.Salida = "En curso"
	}
	d.Orden = secretos.Tapar(recortarBytes(d.Orden, LimiteSalida))
	d.Salida = secretos.Tapar(recortarBytes(d.Salida, LimiteSalida))
	d.Stdout = secretos.Tapar(recortarBytes(d.Stdout, LimiteSalida))
	d.Stderr = secretos.Tapar(recortarBytes(d.Stderr, LimiteSalida))
	d.EntradaCruda = secretos.Tapar(recortarBytes(d.EntradaCruda, LimiteSalida))
	d.SalidaCruda = secretos.Tapar(recortarBytes(d.SalidaCruda, LimiteSalida))
	for i := range d.Diff {
		for j, l := range d.Diff[i].Lineas {
			d.Diff[i].Lineas[j] = secretos.Tapar(l)
		}
	}
	return d, nil
}

// estructurar toma lo que Claude Code guarda aparte del resultado (toolUseResult): el diff real con sus números
// de línea, y stdout, stderr e interrupción de una orden.
func (d *DetallePaso) estructurar(raw json.RawMessage) {
	if len(raw) == 0 || raw[0] != '{' {
		return
	}
	var r struct {
		FilePath        string `json:"filePath"`
		Type            string `json:"type"`
		Content         string `json:"content"`
		StructuredPatch []struct {
			OldStart int      `json:"oldStart"`
			OldLines int      `json:"oldLines"`
			NewStart int      `json:"newStart"`
			NewLines int      `json:"newLines"`
			Lines    []string `json:"lines"`
		} `json:"structuredPatch"`
		Stdout      *string `json:"stdout"`
		Stderr      *string `json:"stderr"`
		Interrupted bool    `json:"interrupted"`
	}
	if json.Unmarshal(raw, &r) != nil {
		return
	}
	if r.FilePath != "" {
		d.Archivo = r.FilePath
	}
	if len(r.StructuredPatch) > 0 {
		d.Diff = d.Diff[:0]
		for _, h := range r.StructuredPatch {
			d.Diff = append(d.Diff, Trozo{ViejoDesde: h.OldStart, ViejoLineas: h.OldLines, NuevoDesde: h.NewStart, NuevoLineas: h.NewLines, Lineas: h.Lines})
		}
	} else if r.Type == "create" && r.Content != "" {
		d.Diff = []Trozo{nuevoArchivoDiff(r.Content)}
	}
	if r.Stdout != nil {
		d.Stdout = *r.Stdout
	}
	if r.Stderr != nil {
		d.Stderr = *r.Stderr
	}
	d.Interrumpido = r.Interrupted
}

// LimiteLineasDiff: un archivo nuevo enorme no se manda entero como diff.
const LimiteLineasDiff = 2000

func nuevoArchivoDiff(contenido string) Trozo {
	lineas := strings.Split(strings.TrimSuffix(strings.ReplaceAll(contenido, "\r\n", "\n"), "\n"), "\n")
	t := Trozo{NuevoDesde: 1, NuevoLineas: len(lineas)}
	for i, l := range lineas {
		if i == LimiteLineasDiff {
			t.Lineas = append(t.Lineas, fmt.Sprintf(" … (%d líneas más)", len(lineas)-i))
			break
		}
		t.Lineas = append(t.Lineas, "+"+l)
	}
	return t
}

// diffDeEntrada: el diff que se deduce de la llamada, para cuando el resultado todavía no llegó o no trae
// structuredPatch. Sin números de línea: Claude Code no los pone en la entrada.
func diffDeEntrada(herr string, input json.RawMessage) []Trozo {
	var in struct {
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
		Content   string `json:"content"`
		Edits     []struct {
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		} `json:"edits"`
	}
	if json.Unmarshal(input, &in) != nil {
		return nil
	}
	switch herr {
	case "Edit":
		return []Trozo{DiffLineas(in.OldString, in.NewString)}
	case "MultiEdit":
		var out []Trozo
		for _, e := range in.Edits {
			out = append(out, DiffLineas(e.OldString, e.NewString))
		}
		return out
	case "Write":
		return []Trozo{nuevoArchivoDiff(in.Content)}
	}
	return nil
}

// DiffLineas compara dos textos línea a línea (subsecuencia común más larga) y marca lo que se quitó y lo que se puso.
func DiffLineas(viejo, nuevo string) Trozo {
	a := strings.Split(strings.ReplaceAll(viejo, "\r\n", "\n"), "\n")
	b := strings.Split(strings.ReplaceAll(nuevo, "\r\n", "\n"), "\n")
	t := Trozo{ViejoLineas: len(a), NuevoLineas: len(b)}
	// Textos enormes: sin tabla de LCS (cuadrática), todo lo viejo fuera y todo lo nuevo dentro.
	if len(a)*len(b) > 4_000_000 {
		for _, l := range a {
			t.Lineas = append(t.Lineas, "-"+l)
		}
		for _, l := range b {
			t.Lineas = append(t.Lineas, "+"+l)
		}
		return t
	}
	lcs := make([][]int32, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int32, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			t.Lineas = append(t.Lineas, " "+a[i])
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			t.Lineas = append(t.Lineas, "-"+a[i])
			i++
		default:
			t.Lineas = append(t.Lineas, "+"+b[j])
			j++
		}
	}
	for ; i < len(a); i++ {
		t.Lineas = append(t.Lineas, "-"+a[i])
	}
	for ; j < len(b); j++ {
		t.Lineas = append(t.Lineas, "+"+b[j])
	}
	return t
}

// sangrar devuelve el JSON con sangría, para la vista cruda.
func sangrar(raw json.RawMessage) string {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return string(raw)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return string(raw)
	}
	return string(b)
}

func recortarBytes(s string, max int) string {
	if len(s) <= max {
		return s
	}
	corte := max
	for corte > 0 && (s[corte]&0xC0) == 0x80 {
		corte--
	}
	return s[:corte] + fmt.Sprintf("\n… (%d bytes más)", len(s)-corte)
}

func lineaCruda(ruta string, ofs int64) ([]byte, error) {
	f, err := os.Open(ruta)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if _, err := f.Seek(ofs, io.SeekStart); err != nil {
		return nil, err
	}
	b, err := bufio.NewReaderSize(f, 256*1024).ReadBytes('\n')
	if err != nil && len(b) == 0 {
		return nil, err
	}
	return b, nil
}

func lineaEn(ruta string, ofs int64) (linea, error) {
	var l linea
	b, err := lineaCruda(ruta, ofs)
	if err != nil {
		return l, err
	}
	if err := json.Unmarshal(b, &l); err != nil || l.Message == nil {
		return l, errors.New("la línea del transcrito no se pudo leer")
	}
	return l, nil
}

// ordenDe escribe la llamada a la herramienta como la vería quien la lanzó.
func ordenDe(herr string, input json.RawMessage) string {
	var m map[string]any
	if json.Unmarshal(input, &m) != nil {
		return herr
	}
	cad := func(k string) string { s, _ := m[k].(string); return s }
	switch herr {
	case "Bash", "PowerShell":
		return cad("command")
	case "Read":
		s := "Read " + cad("file_path")
		if o, ok := m["offset"].(float64); ok {
			s += fmt.Sprintf("\n  offset %d", int(o))
		}
		if l, ok := m["limit"].(float64); ok {
			s += fmt.Sprintf(", limit %d", int(l))
		}
		return s
	case "Edit":
		return "Edit " + cad("file_path") + "\n\n− " + strings.ReplaceAll(cad("old_string"), "\n", "\n− ") + "\n\n+ " + strings.ReplaceAll(cad("new_string"), "\n", "\n+ ")
	case "Write":
		return "Write " + cad("file_path") + "\n\n" + cad("content")
	}
	b, _ := json.MarshalIndent(m, "", "  ")
	return herr + " " + string(b)
}
