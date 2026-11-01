// Package servidor expone la API de solo lectura y el frontend embebido, y avisa
// por SSE cuando algo cambia. Escucha solo en 127.0.0.1.
package servidor

import (
	"context"
	"encoding/json"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"atalaya/internal/alertas"
	"atalaya/internal/avisos"
	"atalaya/internal/claude"
	"atalaya/internal/codexvivo"
	"atalaya/internal/configurar"
	"atalaya/internal/cuotas"
	"atalaya/internal/datapedia"
	"atalaya/internal/delegadas"
	"atalaya/internal/entorno"
	"atalaya/internal/trazabilidad"
	"atalaya/internal/usoclaude"
)

// Version del servidor.
const Version = "1.2.1"

// Proyecto conocido de antemano (para las llamadas a Codex y los documentos).
type Proyecto struct {
	Nombre string `json:"nombre"`
	Raiz   string `json:"raiz"`
	// Del registro proyectos.json (internal/proyectos): la sigla de su tarjeta y el nivel del contrato (0 a 3).
	// Registrado es falso en un proyecto que solo llegó por -proyecto.
	Sigla      string `json:"sigla,omitempty"`
	Nivel      int    `json:"nivel"`
	Registrado bool   `json:"registrado,omitempty"`
	// FueraDatapedia: el dueño lo sacó de la datapedia (proyectos.json, «"datapedia": false»).
	FueraDatapedia bool `json:"fuera_datapedia,omitempty"`
}

// Servidor reúne los lectores y la difusión de cambios.
type Servidor struct {
	Cuotas    *cuotas.Lector
	UsoClaude *usoclaude.Lector
	Claude    *claude.Indice
	Proyectos []Proyecto
	Web       fs.FS // el frontend exportado; nil si no se compiló
	Ahora     func() time.Time
	// Datapedia: nil si no se pudo abrir el índice (el resto del panel sigue).
	Datapedia *datapedia.Indice
	// CodexVivo: la traza en vivo de las llamadas a Codex; nil si no se usa.
	CodexVivo *codexvivo.Lector
	// Delegadas: las fichas de scripts/claude.ps1, que cuelgan cada sesión de Claude Code de quien la lanzó (la
	// aplica el índice de Claude); nil si no se usa.
	Delegadas *delegadas.Lector
	// Avisos y Notificador: las notificaciones de lo que te espera (avisos.go, avisos); nil si están apagados. Base
	// es la dirección del panel, para que un clic lo abra en su sección.
	Avisos      *avisos.Vigia
	Notificador avisos.Notificador
	Base        string
	// RutaAvisos: avisos.json, qué tipos están encendidos; vacía, todos y sin guardar.
	RutaAvisos string
	// Diario: la carpeta del diario (diario.go, avisos); vacía, sin diario.
	Diario   string
	diarioMu sync.Mutex
	// diarioRecortado: el día en que el diario se recortó por última vez a los días elegidos (con diarioMu).
	diarioRecortado string
	confAvisos      struct {
		mu    sync.Mutex
		c     avisos.Config
		leida bool
	}
	// RutaRegistro y Registro: proyectos.json y cómo volver a armar la lista cuando cambia (registro.go); sin
	// ellos, la lista del arranque no cambia.
	RutaRegistro string
	Registro     func() ([]Proyecto, error)
	// Casa y CodexHome: dónde buscar Claude Code y Codex (configurar.go, /api/entorno); vacías, no se buscan.
	Casa, CodexHome string
	// Datos: la carpeta propia de Atalaya, con las copias y el historial de la puerta de escritura (configurar.go).
	Datos    string
	cacheEnt cacheEntorno
	// clientes: cuándo llegó y consultó cada herramienta por el MCP (mcp_clientes.go).
	clientes clientesMCP
	// escrituras: las del registro de consultas, que van aparte para no demorar la respuesta (consultas_registro.go).
	escrituras sync.WaitGroup
	// Kit: la carpeta del kit atalaya-proyecto (junto a atalaya.exe), de donde se copia la skill; vacía si no está.
	Kit string
	// Compilacion: identifica este atalaya.exe (versión, tamaño y hora del ejecutable); vacía, la versión.
	Compilacion string
	// skills: el estado de la skill de cada herramienta, guardado mientras sus archivos no cambien (kit_arranque.go).
	skills estadosSkill
	// Demo: corre el mundo de -demo; la puerta no corre programas ni toca el arranque del equipo.
	Demo bool
	// herramientasPrueba: en las pruebas, las herramientas en lugar de buscarlas en el equipo.
	herramientasPrueba []entorno.Herramienta
	puertaUna          sync.Once
	puertaConf         *configurar.Puerta

	difusion      difusor
	vueltas       int
	lista         atomic.Pointer[[]Proyecto]
	firmaRegistro string
	registroVisto bool
	registroMu    sync.Mutex // firmaRegistro y registroVisto: los toca Vigilar y también el panel (configurar.go)
	// Las preguntas por el MCP que esperan o ya tienen respuesta (preguntas.go).
	preguntas almacenPreguntas

	reglas alertas.Lector
	// Los documentos hallados de cada subagente (nombres.go).
	docs cacheDocs
	// lanz: los lanzamientos de codex.ps1, guardados 30 s (nombres.go, lanzamientosRecientes).
	lanz struct {
		mu    sync.Mutex
		ls    []claude.LanzamientoCodex
		hasta time.Time
	}
	// resumenes: /api/consultas/resumen guardado poco tiempo y hasta el próximo aviso (ahorro.go).
	resumenes memoResumen

	mu           sync.RWMutex
	codex        map[string][]trazabilidad.LlamadaCodex // por proyecto
	firma        map[string]string                      // archivos vigilados -> tamaño y hora
	alertas      []alertas.Alerta
	firmaAlertas string
	firmaTmp     string     // las carpetas temporales de los proyectos (proceso.go)
	consultas    []Consulta // las llamadas a /mcp (mcp.go)
}

func (s *Servidor) ahora() time.Time {
	if s.Ahora != nil {
		return s.Ahora()
	}
	return time.Now()
}

// Vigilar lee todo al empezar y luego mira los archivos cada `cada`.
func (s *Servidor) Vigilar(ctx context.Context, cada time.Duration) {
	s.Cuotas.Leer(ctx)
	s.vuelta(ctx, true)
	t := time.NewTicker(cada)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.vuelta(ctx, false)
		}
	}
}

func (s *Servidor) vuelta(ctx context.Context, primera bool) {
	if s.UsoClaude != nil && s.UsoClaude.Toca() {
		go s.UsoClaude.Leer(ctx, false)
	}
	if !primera && s.Cuotas.Toca() {
		go func() {
			s.Cuotas.Leer(ctx)
			s.difusion.enviar(Evento{Tipo: "cuotas"})
		}()
	}
	// Las fichas antes que los transcritos: una sesión delegada nueva sale ya colgada de quien la lanzó.
	delegadasCambiadas := s.Delegadas.Actualizar()
	ids, err := s.Claude.Actualizar()
	if err != nil {
		log.Printf("transcritos: %v", err)
	}
	ids = append(ids, delegadasCambiadas...)
	cambioDocs := s.releerRegistro()
	cambioDocs = s.releerTrazabilidad() || cambioDocs
	if s.CodexVivo != nil {
		vivos, err := s.CodexVivo.Actualizar()
		if err != nil {
			log.Printf("codex en vivo: %v", err)
		}
		ids = append(ids, vivos...)
	}
	if primera {
		// Un navegador que se conectó mientras se leían los transcritos vio listas vacías: que vuelva a pedirlas ya,
		// sin esperar a que la datapedia termine su primer índice (puede tardar decenas de segundos).
		s.difusion.enviar(Evento{Tipo: "agentes"})
	}
	// La datapedia se pone al día cada 5 vueltas (10 s): recorrer unos cientos de Markdown no necesita más.
	if s.Datapedia != nil && s.vueltas%5 == 0 {
		r, err := s.Datapedia.Reindexar()
		if err != nil {
			log.Printf("datapedia: %v", err)
		} else if primera {
			log.Printf("datapedia: %d documentos (%d nuevos) en %s", r.Total, r.Nuevos, r.Duracion.Round(time.Millisecond))
		} else if r.Cambio() {
			s.difusion.enviar(Evento{Tipo: "datapedia"})
		}
	}
	s.vueltas++
	cambioAlertas := s.actualizarAlertas()
	cambioTmp := s.actualizarTemporales()
	if primera || s.vueltas%cadaDiario == 0 {
		s.escribirDiario()
	}
	if primera || s.vueltas%cadaAvisos == 0 {
		s.mirarAvisos()
	}
	if primera {
		// Y otra vez al terminar, por las alertas y la datapedia ya indexada; las consultas también, porque su ahorro
		// (que_hizo) necesita a los agentes ya leídos y lo que el panel pidió antes salió a medias.
		s.difusion.enviar(Evento{Tipo: "agentes"})
		s.difusion.enviar(Evento{Tipo: "alertas"})
		s.difusion.enviar(Evento{Tipo: "datapedia"})
		s.difusion.enviar(Evento{Tipo: "consultas"})
		return
	}
	if len(ids) > 0 || cambioDocs {
		// Lo que el panel pida con este aviso mira las bitácoras en el disco, no en la memoria de 1 s.
		trazabilidad.OlvidarAvance()
		s.difusion.enviar(Evento{Tipo: "agentes", IDs: ids})
	}
	if cambioAlertas {
		s.difusion.enviar(Evento{Tipo: "alertas"})
	}
	if cambioTmp {
		s.difusion.enviar(Evento{Tipo: "proceso"})
	}
}

// releerTrazabilidad vuelve a leer los registros de Codex y mira si cambió alguna bitácora.
func (s *Servidor) releerTrazabilidad() bool {
	firma := map[string]string{}
	codex := map[string][]trazabilidad.LlamadaCodex{}
	for _, p := range s.listaProyectos() {
		// También el avance de cada fase y el estado: el panel se pone al día solo cuando el coordinador los mueve.
		pats := []string{"F*/avance/*.md", "F*/codex/*.md", "F*/AVANCE.md", "*ESTADO-para-retomar.md"}
		for _, pat := range pats {
			rutas, _ := filepath.Glob(filepath.Join(p.Raiz, "docs", "trazabilidad", filepath.FromSlash(pat)))
			for _, r := range rutas {
				if fi, err := os.Stat(r); err == nil {
					firma[r] = strconv.FormatInt(fi.Size(), 10) + "@" + fi.ModTime().String()
				}
			}
		}
		if ll, err := trazabilidad.LeerCodex(p.Nombre, p.Raiz); err == nil {
			codex[p.Nombre] = ll
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cambio := len(firma) != len(s.firma)
	for k, v := range firma {
		if s.firma[k] != v {
			cambio = true
		}
	}
	s.firma, s.codex = firma, codex
	return cambio
}

// Handler arma las rutas.
func (s *Servidor) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/salud", s.salud)
	mux.HandleFunc("GET /api/cuotas", s.cuotas)
	mux.HandleFunc("GET /api/uso-claude", s.verUsoClaude)
	mux.HandleFunc("POST /api/uso-claude/leer", s.leerUsoClaude)
	mux.HandleFunc("GET /api/proyectos", s.proyectos)
	mux.HandleFunc("GET /api/agentes", s.agentes)
	mux.HandleFunc("GET /api/agentes/{id}", s.agente)
	mux.HandleFunc("GET /api/agentes/{id}/pasos/{n}", s.paso)
	mux.HandleFunc("GET /api/agentes/{id}/contexto", s.contexto)
	mux.HandleFunc("GET /api/alertas", s.verAlertas)
	mux.HandleFunc("GET /api/pasos", s.buscarPasos)
	mux.HandleFunc("GET /api/consultas", s.verConsultas)
	mux.HandleFunc("GET /api/consultas/resumen", s.verResumenConsultas)
	mux.HandleFunc("GET /api/proceso", s.verProceso)
	mux.HandleFunc("GET /api/estadisticas", s.verEstadisticas)
	mux.HandleFunc("GET /api/avance", s.verAvance)
	mux.HandleFunc("GET /api/portafolio", s.verPortafolio)
	mux.HandleFunc("GET /api/linea", s.verLinea)
	mux.HandleFunc("GET /api/decisiones", s.verDecisiones)
	mux.HandleFunc("GET /api/avisos", s.verAvisos)
	mux.HandleFunc("POST /api/avisos", s.configurarAvisos)
	mux.HandleFunc("POST /api/avisos/probar", s.probarAviso)
	mux.HandleFunc("GET /api/diario", s.verDiario)
	mux.HandleFunc("GET /api/preguntas", s.verPreguntas)
	mux.HandleFunc("GET /api/preguntas/{id}", s.verPregunta)
	mux.HandleFunc("POST /api/preguntas/{id}/responder", s.responderPregunta)
	mux.HandleFunc("GET /api/agentes/{id}/sesion", s.verSesion)
	mux.HandleFunc("GET /api/agentes/{id}/exportar", s.exportar)
	mux.HandleFunc("POST /mcp", s.mcp)
	mux.HandleFunc("GET /mcp", s.mcpOtro)
	mux.HandleFunc("DELETE /mcp", s.mcpOtro)
	mux.HandleFunc("GET /api/datapedia", s.buscarDatapedia)
	mux.HandleFunc("GET /api/datapedia/{id}", s.documentoDatapedia)
	mux.HandleFunc("GET /api/doc", s.doc)
	mux.HandleFunc("GET /api/eventos", s.eventos)
	mux.HandleFunc("GET /api/entorno", s.verEntorno)
	mux.HandleFunc("GET /api/configurar/historial", s.verHistorialConfiguracion)
	mux.HandleFunc("POST /api/configurar/proyectos", s.proponerProyectos)
	mux.HandleFunc("GET /api/configurar/preferencias", s.verPreferencias)
	mux.HandleFunc("POST /api/configurar/preferencias", s.proponerPreferencias)
	mux.HandleFunc("POST /api/configurar/skill", s.proponerSkill)
	mux.HandleFunc("POST /api/configurar/arranque", s.proponerArranque)
	mux.HandleFunc("POST /api/configurar/mcp", s.proponerMCP)
	mux.HandleFunc("POST /api/configurar/aplicar", s.aplicarConfiguracion)
	mux.HandleFunc("POST /api/configurar/deshacer", s.deshacerConfiguracion)
	if s.Web != nil {
		mux.Handle("GET /", conCache(http.FileServerFS(s.Web)))
	} else {
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "El frontend no está compilado: corre scripts/compilar.ps1, o usa `npm run dev` en frontend/.", http.StatusNotFound)
		})
	}
	return soloLocal(mux)
}

// conCache: los archivos de /_next/static llevan su hash en el nombre y no cambian nunca; el resto (index.html) se
// revalida siempre, para que tras reiniciar con un frontend nuevo el navegador no siga mostrando el viejo.
func conCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/_next/static/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		h.ServeHTTP(w, r)
	})
}

// soloLocal rechaza un Host que no sea este equipo (defensa contra DNS rebinding) y fija cabeceras de seguridad.
func soloLocal(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if hh, _, err := net.SplitHostPort(host); err == nil {
			host = hh
		}
		if host != "localhost" && host != "127.0.0.1" && host != "[::1]" && host != "::1" {
			http.Error(w, "solo localhost", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Frame-Options", "DENY")
		h.ServeHTTP(w, r)
	})
}

func escribir(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(v)
}

func fallo(w http.ResponseWriter, codigo int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(codigo)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (s *Servidor) salud(w http.ResponseWriter, _ *http.Request) {
	escribir(w, map[string]any{"version": Version, "hora": s.ahora().Unix(), "transcritos": s.Claude.Raiz, "cuotas": s.Cuotas.Exe, "proyectos": s.listaProyectos(), "registro": s.RutaRegistro})
}

func (s *Servidor) cuotas(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("leer") == "1" {
		est := s.Cuotas.Forzar(r.Context())
		s.difusion.enviar(Evento{Tipo: "cuotas"})
		escribir(w, est)
		return
	}
	escribir(w, s.Cuotas.Estado())
}

// AgenteVista es lo que el panel muestra de un agente, de Claude o de Codex.
type AgenteVista struct {
	ID string `json:"id"`
	// Nombre: el único nombre del agente en todo el panel (nombreVista); ModeloCorto, «sonnet-5-5/high» (modeloCorto).
	Nombre      string               `json:"nombre"`
	ModeloCorto string               `json:"modelo_corto"`
	Proyecto    string               `json:"proyecto"`
	Proveedor   string               `json:"proveedor"`
	Tipo        string               `json:"tipo"`
	Rol         string               `json:"rol"`
	Tarea       string               `json:"tarea"`
	Titulo      string               `json:"titulo,omitempty"`      // el nombre de la sesión en Claude Code, si tiene
	Interactiva bool                 `json:"interactiva,omitempty"` // una sesión interactiva de Codex, no una llamada
	Modelo      string               `json:"modelo"`
	Effort      string               `json:"effort"`
	Clase       string               `json:"clase"`
	Presupuesto string               `json:"presupuesto"`
	Base        string               `json:"base"`
	Estado      string               `json:"estado"`
	Inicio      time.Time            `json:"inicio"`
	Ultimo      time.Time            `json:"ultimo"`
	Tokens      claude.Tokens        `json:"tokens"`
	Avance      *trazabilidad.Avance `json:"avance"`
	Bitacora    string               `json:"bitacora"`
	Informe     string               `json:"informe"`
	Pasos       int                  `json:"pasos"`
	// Padre: la sesión que lo lanzó. Un subagente lo trae de su transcrito; una llamada a Codex, de la sesión que
	// corrió codex.ps1 con su tema (sesionDeCodex). Vacío en una llamada cuya sesión no está en los transcritos.
	Padre      string `json:"padre,omitempty"`
	Asignacion string `json:"asignacion,omitempty"`
	Encargo    string `json:"encargo,omitempty"`
	// Solo en el detalle: la asignación partida, los archivos que nombra el encargo y sus entregables.
	Nivel          string   `json:"nivel,omitempty"`
	ClaseDetalle   string   `json:"clase_detalle,omitempty"`
	RolAsignado    string   `json:"rol_asignado,omitempty"`
	ModeloAsignado string   `json:"modelo_asignado,omitempty"`
	Cuota          string   `json:"cuota,omitempty"`
	Nombrados      []string `json:"nombrados,omitempty"`
	InformeExiste  bool     `json:"informe_existe,omitempty"`
	Acta           string   `json:"acta,omitempty"`
	ActaEstado     string   `json:"acta_estado,omitempty"` // abierta | cerrada
}

// Estados.
const (
	Trabajando = "Trabajando"
	EnEspera   = "En espera"
	Entregado  = "Entregado"
	Inactiva   = "Inactiva"
)

// EstadoClaude decide el estado de una sesión o un subagente por lo último que escribió.
func EstadoClaude(a claude.Agente, ahora time.Time) string {
	quieto := ahora.Sub(a.Mod)
	activo := quieto < 2*time.Minute || (a.Pendiente && quieto < 15*time.Minute)
	if a.Tipo == "subagente" {
		if a.Cerrado || !activo {
			return Entregado
		}
		return Trabajando
	}
	switch {
	case activo && !a.Cerrado:
		return Trabajando
	case quieto < 30*time.Minute:
		return EnEspera
	}
	return Inactiva
}

func activo(estado string) bool { return estado == Trabajando || estado == EnEspera }

func (s *Servidor) vistaClaude(a claude.Agente, conEncargo bool) AgenteVista {
	return s.vistaConEquipo(a, conEncargo, nil)
}

// vistaConEquipo: vistaClaude con las sesiones de equipo trabajando ya sabidas (equipoTrabajando), para quien arma
// muchas vistas de una vez; nil las busca si hace falta.
func (s *Servidor) vistaConEquipo(a claude.Agente, conEncargo bool, equipo map[string]bool) AgenteVista {
	v := AgenteVista{ID: a.ID, Proyecto: a.Proyecto, Proveedor: "Claude", Tipo: a.Tipo, Rol: a.Rol, Tarea: a.Tarea, Titulo: a.Titulo, Modelo: a.Modelo,
		ModeloCorto: modeloCorto(a.Modelo, a.Effort), Effort: a.Effort, Clase: a.Clase, Presupuesto: a.Presupuesto, Base: a.Base,
		Estado: s.estadoClaude(a, equipo), Inicio: a.Inicio, Ultimo: a.Ultimo, Tokens: a.Tokens, Bitacora: a.Bitacora, Informe: a.Informe,
		Pasos: a.Pasos, Padre: a.Padre}
	d := s.docsDe(a)
	v.Nombre = nombreVista(v, d)
	v.Bitacora = d.Bitacora
	a.Bitacora = d.Bitacora
	if a.Bitacora != "" && a.Raiz != "" {
		if av, err := trazabilidad.LeerAvance(a.Raiz, a.Bitacora); err == nil {
			v.Avance = av
		}
	}
	if conEncargo {
		v.Asignacion, v.Encargo = a.Asignacion, a.Encargo
		v.Nivel, v.ClaseDetalle, v.RolAsignado, v.ModeloAsignado, v.Cuota, v.Nombrados = a.Nivel, a.ClaseDetalle, a.RolAsignado, a.ModeloAsignado, a.Cuota, a.Nombrados
		if a.Raiz != "" {
			if a.Informe != "" {
				_, err := os.Stat(filepath.Join(a.Raiz, filepath.FromSlash(a.Informe)))
				v.InformeExiste = err == nil
			}
			if a.Tipo == "subagente" && d.Acta != "" {
				v.Acta, v.ActaEstado = d.Acta, estadoActa(a.Raiz, d.Acta)
			}
		}
	}
	return v
}

// bitacoraPorRol: si el encargo no nombra la bitácora, la del mismo rol que se escribió mientras el agente trabajaba.
func bitacoraPorRol(a claude.Agente) string {
	return documentoPorRol(a, "avance", strings.ToLower(a.Rol)+"*.md")
}

// actaPorRol: el acta de temporales del agente («2026-10-30-codificador-c3max.md»), por su rol y su hora.
func actaPorRol(a claude.Agente) string {
	return documentoPorRol(a, "temporales", "*-"+strings.ToLower(a.Rol)+"*.md")
}

// estadoActa: un acta que todavía dice «(en curso)» en alguna sección sigue abierta.
func estadoActa(raiz, rel string) string {
	b, err := os.ReadFile(filepath.Join(raiz, filepath.FromSlash(rel)))
	if err != nil {
		return ""
	}
	// «(en curso)» en una sección, o el «## Cierre» de la plantilla aún en «(pendiente)».
	if s := string(b); strings.Contains(s, "(en curso)") || strings.Contains(s, "(pendiente)") {
		return "abierta"
	}
	return "cerrada"
}

// documentoPorRol: en docs/trazabilidad/F*/<carpeta>, el archivo del patrón que se escribió mientras el agente
// trabajaba (o hasta 10 min después), el más cercano a su último paso.
func documentoPorRol(a claude.Agente, carpeta, patron string) string {
	rutas, _ := filepath.Glob(filepath.Join(a.Raiz, "docs", "trazabilidad", "F*", carpeta, patron))
	mejor, mejorDif := "", time.Duration(1<<62)
	for _, r := range rutas {
		fi, err := os.Stat(r)
		if err != nil || fi.ModTime().Before(a.Inicio.Add(-time.Minute)) || fi.ModTime().After(a.Ultimo.Add(10*time.Minute)) {
			continue
		}
		dif := a.Ultimo.Sub(fi.ModTime()).Abs()
		if dif < mejorDif {
			rel, err := filepath.Rel(a.Raiz, r)
			if err == nil {
				mejor, mejorDif = filepath.ToSlash(rel), dif
			}
		}
	}
	return mejor
}

var rolCodex = map[string]string{"R1": "Búsqueda en el código", "E1": "Auditoría", "E2": "Segunda opinión"}

func vistaCodex(c trazabilidad.LlamadaCodex) AgenteVista {
	rol := rolCodex[c.Clase]
	if rol == "" {
		rol = "Codex"
	}
	v := AgenteVista{ID: c.ID, Proyecto: c.Proyecto, Proveedor: "Codex", Tipo: "codex", Rol: rol, Tarea: c.Tema, Modelo: c.Modelo,
		Effort: c.Effort, Clase: c.Clase, Base: c.Base, Estado: Entregado, Ultimo: c.Terminada,
		Inicio: c.Terminada.Add(-time.Duration(c.Segundos) * time.Second), Informe: c.Informe, Pasos: 1,
		Tokens: claude.Tokens{Entrada: c.Entrada, CacheLeida: c.EnCache, Salida: c.TokSalida}}
	v.Avance = &trazabilidad.Avance{Porcentaje: 100, Texto: "entregado · salida " + strconv.Itoa(c.Salida) + " · contraste: " + c.Contraste, Hitos: []trazabilidad.Hito{}}
	v.Nombre, v.ModeloCorto = nombreVista(v, docsAgente{}), modeloCorto(v.Modelo, v.Effort)
	return v
}

func (s *Servidor) todos() []AgenteVista {
	var out []AgenteVista
	equipo := s.equipoTrabajando()
	for _, a := range s.Claude.Agentes() {
		out = append(out, s.vistaConEquipo(a, false, equipo))
	}
	// Las llamadas a Codex con traza en vivo reemplazan a su fila del registro, que solo aporta el contraste. Cada
	// una cuelga de la sesión que corrió codex.ps1 (padreCodexVivo y sesionDeCodex).
	ls := s.Claude.LanzamientosCodex(time.Time{})
	vivos, reg := s.codexEnVivo()
	cubiertas := map[string]bool{}
	for _, c := range vivos {
		k := claveRegistro(c)
		cubiertas[k] = true
		v := s.vistaCodexVivo(c, reg[k])
		v.Padre = padreCodexVivo(ls, c)
		out = append(out, v)
	}
	s.mu.RLock()
	for _, ll := range s.codex {
		for _, c := range ll {
			if !cubiertas[c.ID] {
				v := vistaCodex(c)
				v.Padre = sesionDeCodex(ls, c.Proyecto, c.Tema, inicioRegistro(c))
				out = append(out, v)
			}
		}
	}
	s.mu.RUnlock()
	// Una llamada a Codex que trabaja pone a trabajar a su sesión (estadoClaude ya lo vio por equipoTrabajando; esto
	// cubre la que solo se enlaza por su registro).
	for i := range out {
		if out[i].Tipo != "sesion" && out[i].Estado == Trabajando && out[i].Padre != "" {
			equipo[out[i].Padre] = true
		}
	}
	for i := range out {
		if out[i].Tipo == "sesion" && out[i].Estado != Trabajando && equipo[out[i].ID] && s.ahora().Sub(out[i].Ultimo) <= 24*time.Hour {
			out[i].Estado = Trabajando
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		ai, aj := activo(out[i].Estado), activo(out[j].Estado)
		if ai != aj {
			return ai
		}
		return out[i].Ultimo.After(out[j].Ultimo)
	})
	return out
}

type proyectoVista struct {
	Nombre  string `json:"nombre"`
	Agentes int    `json:"agentes"`
	Activos int    `json:"activos"`
	Ultimo  int64  `json:"ultimo"`
}

func (s *Servidor) proyectos(w http.ResponseWriter, _ *http.Request) {
	por := map[string]*proyectoVista{}
	for _, p := range s.listaProyectos() {
		por[p.Nombre] = &proyectoVista{Nombre: p.Nombre}
	}
	for _, a := range s.todos() {
		if a.Proyecto == "" {
			continue
		}
		p := por[a.Proyecto]
		if p == nil {
			p = &proyectoVista{Nombre: a.Proyecto}
			por[a.Proyecto] = p
		}
		p.Agentes++
		if activo(a.Estado) {
			p.Activos++
		}
		if u := a.Ultimo.Unix(); u > p.Ultimo {
			p.Ultimo = u
		}
	}
	out := make([]proyectoVista, 0, len(por))
	for _, p := range por {
		out = append(out, *p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ultimo > out[j].Ultimo })
	escribir(w, out)
}

func (s *Servidor) agentes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	proyecto, filtro := q.Get("proyecto"), q.Get("filtro")
	out := []AgenteVista{}
	for _, a := range s.todos() {
		if proyecto != "" && a.Proyecto != proyecto {
			continue
		}
		if (filtro == "activos" && !activo(a.Estado)) || (filtro == "hechos" && activo(a.Estado)) {
			continue
		}
		out = append(out, a)
	}
	// Las sesiones viejas sin subagentes ni actividad reciente no aportan: se cortan a 200.
	if len(out) > 200 {
		out = out[:200]
	}
	escribir(w, out)
}

type detalle struct {
	Agente AgenteVista   `json:"agente"`
	Pasos  []claude.Paso `json:"pasos"`
	// Desde: el número del primer paso devuelto (los anteriores se omiten).
	Desde int `json:"desde"`
}

func (s *Servidor) codexPorID(id string) (trazabilidad.LlamadaCodex, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, ll := range s.codex {
		for _, c := range ll {
			if c.ID == id {
				return c, true
			}
		}
	}
	return trazabilidad.LlamadaCodex{}, false
}

func (s *Servidor) raizDe(proyecto string) string {
	for _, p := range s.listaProyectos() {
		if p.Nombre == proyecto {
			return p.Raiz
		}
	}
	// Un proyecto que solo se conoce por sus transcritos: la raíz de su repositorio.
	for _, a := range s.Claude.Agentes() {
		if a.Proyecto == proyecto && a.Raiz != "" {
			return a.Raiz
		}
	}
	return ""
}

func (s *Servidor) agente(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	max := 200
	if n, err := strconv.Atoi(r.URL.Query().Get("max")); err == nil && n > 0 && n <= 5000 {
		max = n
	}
	if strings.HasPrefix(id, codexvivo.Prefijo) && s.CodexVivo != nil {
		c, ok := s.CodexVivo.Llamada(id)
		if !ok {
			fallo(w, http.StatusNotFound, "no existe esa llamada")
			return
		}
		_, reg := s.codexEnVivo()
		v := s.vistaCodexVivo(c, reg[claveRegistro(c)])
		v.Padre = padreCodexVivo(s.Claude.LanzamientosCodex(time.Time{}), c)
		escribir(w, detalle{Agente: v, Pasos: c.Pasos})
		return
	}
	if c, ok := s.codexPorID(id); ok {
		v := vistaCodex(c)
		v.Padre = sesionDeCodex(s.Claude.LanzamientosCodex(time.Time{}), c.Proyecto, c.Tema, inicioRegistro(c))
		paso := claude.Paso{N: 0, Continua: -1, Paralelo: 1, Turno: 1, Hora: c.Terminada, Tipo: "Responde", Herramienta: "codex.ps1", Resumen: "Informe de " + c.Tema, Meta: strconv.Itoa(c.Segundos) + " s · salida " + strconv.Itoa(c.Salida), Hecho: true, Error: c.Salida != 0}
		escribir(w, detalle{Agente: v, Pasos: []claude.Paso{paso}})
		return
	}
	// ?antes=n: los `max` pasos de antes del paso n (una página vieja de la actividad); sin él, los últimos.
	antes := -1
	if n, err := strconv.Atoi(r.URL.Query().Get("antes")); err == nil && n >= 0 {
		antes = n
	}
	a, pasos, ok := s.Claude.AgenteAntes(id, antes, max)
	if !ok {
		fallo(w, http.StatusNotFound, "no existe ese agente")
		return
	}
	desde := 0
	if len(pasos) > 0 {
		desde = pasos[0].N
	}
	escribir(w, detalle{Agente: s.vistaClaude(a, true), Pasos: pasos, Desde: desde})
}

func (s *Servidor) paso(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		fallo(w, http.StatusBadRequest, "número de paso no válido")
		return
	}
	if strings.HasPrefix(id, codexvivo.Prefijo) && s.CodexVivo != nil {
		d, err := s.CodexVivo.Detalle(id, n)
		if err != nil {
			fallo(w, http.StatusNotFound, err.Error())
			return
		}
		escribir(w, d)
		return
	}
	if c, ok := s.codexPorID(id); ok {
		d := claude.DetallePaso{Paso: claude.Paso{N: 0, Continua: -1, Paralelo: 1, Tipo: "Responde", Herramienta: "codex.ps1", Hecho: true}}
		d.Orden = "codex exec --ephemeral -s read-only -m " + c.Modelo + " -c model_reasoning_effort=" + c.Effort + " --json -C <copia de " + c.Base + ">"
		if c.Informe != "" {
			d.Salida, _ = trazabilidad.LeerDoc(s.raizDe(c.Proyecto), c.Informe)
		}
		escribir(w, d)
		return
	}
	d, err := s.Claude.Detalle(id, n)
	if err != nil {
		fallo(w, http.StatusNotFound, err.Error())
		return
	}
	escribir(w, d)
}

// contexto: la serie de peticiones, las compactaciones y qué llenó el contexto (claude/contexto.go). Una llamada
// a Codex no la tiene: su registro solo guarda los totales.
func (s *Servidor) contexto(w http.ResponseWriter, r *http.Request) {
	c, err := s.Claude.Contexto(r.PathValue("id"))
	if err != nil {
		fallo(w, http.StatusNotFound, err.Error())
		return
	}
	escribir(w, c)
}

func (s *Servidor) doc(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	raiz := s.raizDe(q.Get("proyecto"))
	if raiz == "" {
		fallo(w, http.StatusNotFound, "proyecto desconocido")
		return
	}
	texto, err := trazabilidad.LeerDoc(raiz, q.Get("ruta"))
	if err != nil {
		fallo(w, http.StatusNotFound, err.Error())
		return
	}
	escribir(w, map[string]string{"ruta": q.Get("ruta"), "texto": texto})
}

// Evento que se difunde por SSE.
type Evento struct {
	Tipo string   `json:"tipo"`
	IDs  []string `json:"ids,omitempty"`
}

type difusor struct {
	mu   sync.Mutex
	subs map[chan Evento]struct{}
	// generacion: cuántos avisos salieron. Lo que se guarda para el panel vale solo dentro de una misma generación:
	// con cada aviso el panel vuelve a pedir, y lo que pida tiene que salir fresco.
	generacion atomic.Uint64
}

func (d *difusor) alta() chan Evento {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.subs == nil {
		d.subs = map[chan Evento]struct{}{}
	}
	c := make(chan Evento, 16)
	d.subs[c] = struct{}{}
	return c
}

func (d *difusor) baja(c chan Evento) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.subs, c)
}

func (d *difusor) enviar(e Evento) {
	d.generacion.Add(1)
	d.mu.Lock()
	defer d.mu.Unlock()
	for c := range d.subs {
		select {
		case c <- e:
		default: // un navegador lento pierde un aviso, no frena a los demás
		}
	}
}

func (s *Servidor) eventos(w http.ResponseWriter, r *http.Request) {
	fl, ok := w.(http.Flusher)
	if !ok {
		fallo(w, http.StatusInternalServerError, "sin flush")
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	c := s.difusion.alta()
	defer s.difusion.baja(c)
	// El saludo lleva la compilación: un panel abierto que, al reconectarse, ve otra (atalaya.exe se actualizó) se
	// recarga solo, en vez de seguir con el panel viejo contra el servidor nuevo.
	compilacion := s.Compilacion
	if compilacion == "" {
		compilacion = Version
	}
	hola, _ := json.Marshal(map[string]string{"compilacion": compilacion})
	_, _ = w.Write([]byte("retry: 3000\nevent: hola\ndata: " + string(hola) + "\n\n"))
	fl.Flush()
	latido := time.NewTicker(25 * time.Second)
	defer latido.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case e := <-c:
			b, _ := json.Marshal(e)
			_, _ = w.Write([]byte("event: cambio\ndata: " + string(b) + "\n\n"))
			fl.Flush()
		case <-latido.C:
			_, _ = w.Write([]byte(": latido\n\n"))
			fl.Flush()
		}
	}
}

// NombreProyecto: "Nombre=Ruta" de la línea de órdenes.
func NombreProyecto(s string) (Proyecto, bool) {
	n, r, ok := strings.Cut(s, "=")
	if !ok || strings.TrimSpace(n) == "" || strings.TrimSpace(r) == "" {
		return Proyecto{}, false
	}
	return Proyecto{Nombre: strings.TrimSpace(n), Raiz: strings.TrimSpace(r)}, true
}
