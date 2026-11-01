package servidor

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"atalaya/internal/claude"
)

// Un solo nombre, un solo modelo y un solo estado por agente (auditoría, puntos 16, 17 y 25): se resuelven aquí, una
// vez, y la lista, la línea, la salud, las alertas, las decisiones, las consultas, exportar, el equipo de la sesión y
// el MCP los copian de AgenteVista en vez de armar el suyo.

// docsAgente: los documentos de un subagente. La bitácora que nombra el encargo; si no nombra ninguna, la de su rol
// escrita mientras trabajaba; y si tampoco, la que lleva el nombre de su carpeta temporal. Ajena: la de ese nombre
// existe, pero se escribió después de que entregó (la retomó otro agente).
type docsAgente struct {
	Bitacora, Ajena, Acta string
}

// buscarDocs: un general-purpose no tiene acta ni bitácora con su tipo de agente en el nombre: se buscan por la
// carpeta temporal que usó, que lleva el mismo nombre que su acta («2026-10-31-medidor-w5» ↔
// temporales/2026-10-31-medidor-w5.md) y el rol de su bitácora («medidor-w5» ↔ avance/medidor-w5.md).
func buscarDocs(a claude.Agente) docsAgente {
	d := docsAgente{Bitacora: a.Bitacora}
	if f := faseActual(a.Raiz); f != "" && a.Raiz != "" {
		d.Bitacora = strings.Replace(d.Bitacora, "/"+claude.FaseSinNombre+"/", "/"+f+"/", 1)
	}
	if a.Tipo != "subagente" || a.Raiz == "" {
		return d
	}
	if d.Bitacora == "" {
		d.Bitacora = bitacoraPorRol(a)
	}
	d.Acta = actaPorRol(a)
	temps := sinCortadas(a.Temporales)
	if d.Acta == "" {
		for _, t := range temps {
			if r := documentoPorNombre(a, "temporales", t, false); r != "" {
				d.Acta = r
				break
			}
		}
	}
	if d.Bitacora != "" {
		return d
	}
	var candidatos []string
	for _, t := range temps {
		candidatos = append(candidatos, reFechaRol.ReplaceAllString(t, ""))
	}
	if d.Acta != "" {
		candidatos = append(candidatos, nombreDoc(d.Acta))
	}
	for _, c := range candidatos {
		if r := documentoPorNombre(a, "avance", c, true); r != "" {
			d.Bitacora = r
			break
		}
		if r := documentoPorNombre(a, "avance", c, false); r != "" {
			d.Ajena = r
			break
		}
	}
	return d
}

// cacheDocs: buscarDocs recorre carpetas en el disco y se pide en cada vuelta (alertas) y en cada lista; lo hallado
// vale 30 s mientras el agente no dé otro paso.
type cacheDocs struct {
	mu sync.Mutex
	m  map[string]docsGuardados
}

type docsGuardados struct {
	clave string
	hasta time.Time
	docs  docsAgente
}

const vidaDocs = 30 * time.Second

func (s *Servidor) docsDe(a claude.Agente) docsAgente {
	if a.Tipo != "subagente" || a.Raiz == "" {
		return docsAgente{Bitacora: a.Bitacora}
	}
	// Sin la hora del último paso: un agente activo la cambia a cada paso y el caché no serviría. Una bitácora o una carpeta nueva sí lo renuevan.
	clave := a.Bitacora + "|" + strings.Join(a.Temporales, ",")
	ahora := time.Now()
	s.docs.mu.Lock()
	g, ok := s.docs.m[a.ID]
	s.docs.mu.Unlock()
	if ok && g.clave == clave && ahora.Before(g.hasta) {
		return g.docs
	}
	d := buscarDocs(a)
	s.docs.mu.Lock()
	if s.docs.m == nil {
		s.docs.m = map[string]docsGuardados{}
	}
	s.docs.m[a.ID] = docsGuardados{clave: clave, hasta: ahora.Add(vidaDocs), docs: d}
	s.docs.mu.Unlock()
	return d
}

// nombreDoc: «docs/trazabilidad/F5/temporales/2026-10-31-medidor-w6b.md» → «medidor-w6b».
func nombreDoc(ruta string) string {
	return reFechaRol.ReplaceAllString(strings.TrimSuffix(filepath.Base(filepath.FromSlash(ruta)), ".md"), "")
}

// nombreVista: el nombre del agente en todo el panel. Una sesión, por su proyecto (su título de Claude Code va
// aparte); un subagente, por sus documentos («medidor-w6b»), si no tiene y es general-purpose o Explore, por su tarea,
// y si no, por su rol y su clase («Auditor E2»); una llamada a Codex, «Codex · <tema>». Sin el id: va aparte.
func nombreVista(v AgenteVista, d docsAgente) string {
	switch v.Tipo {
	case "sesion":
		return primeroConTexto(v.Proyecto, v.Titulo, "Sesión")
	case "codex":
		if v.Tarea == "" {
			return "Codex"
		}
		return "Codex · " + v.Tarea
	}
	if doc := primeroConTexto(d.Bitacora, d.Ajena, d.Acta); doc != "" {
		return nombreDoc(doc)
	}
	if (strings.EqualFold(v.Rol, "general-purpose") || strings.EqualFold(v.Rol, "explore")) && v.Tarea != "" {
		return v.Tarea
	}
	// Una sesión lanzada por claude.ps1 (internal/delegadas), como las llamadas a Codex: la herramienta y su tema.
	if v.Rol == "Claude" && v.Tarea != "" {
		return "Claude · " + v.Tarea
	}
	return strings.TrimSpace(v.Rol + " " + v.Clase)
}

// nombreDe: nombreVista de un agente de los transcritos, sin armar toda su vista.
func (s *Servidor) nombreDe(a claude.Agente) string {
	return nombreVista(AgenteVista{Tipo: a.Tipo, Proyecto: a.Proyecto, Titulo: a.Titulo, Rol: a.Rol, Clase: a.Clase, Tarea: a.Tarea}, s.docsDe(a))
}

// modeloCorto: «claude-sonnet-5-5» con effort «high» → «sonnet-5-5/high»; «gpt-6.1» y «terra» → «gpt-6.1/terra».
func modeloCorto(modelo, effort string) string {
	m := strings.TrimPrefix(modelo, "claude-")
	if m != "" && effort != "" {
		m += "/" + effort
	}
	return m
}

// equipoTrabajando: las sesiones que tienen un subagente o una llamada a Codex suyos trabajando.
func (s *Servidor) equipoTrabajando() map[string]bool {
	ahora := s.ahora()
	out := map[string]bool{}
	for _, a := range s.Claude.Agentes() {
		if a.Tipo == "subagente" && a.Padre != "" && EstadoClaude(a, ahora) == Trabajando {
			out[a.Padre] = true
		}
	}
	if s.CodexVivo != nil {
		var ls []claude.LanzamientoCodex
		leidos := false
		for _, c := range s.CodexVivo.Llamadas() {
			if !c.Activa(ahora) {
				continue
			}
			if !leidos {
				ls, leidos = s.lanzamientosRecientes(ahora), true
			}
			if p := padreCodexVivo(ls, c); p != "" {
				out[p] = true
			}
		}
	}
	return out
}

// lanzamientosRecientes: los lanzamientos de codex.ps1 de todas las sesiones, guardados 30 s. Recorrerlos es leer
// todos los pasos, y equipoTrabajando lo pide en cada vuelta de 2 s mientras corre un Codex.
func (s *Servidor) lanzamientosRecientes(ahora time.Time) []claude.LanzamientoCodex {
	s.lanz.mu.Lock()
	defer s.lanz.mu.Unlock()
	if s.lanz.ls == nil || ahora.After(s.lanz.hasta) {
		s.lanz.ls, s.lanz.hasta = s.Claude.LanzamientosCodex(time.Time{}), ahora.Add(vidaDocs)
	}
	return s.lanz.ls
}

// estadoClaude: EstadoClaude, salvo que una sesión con su equipo trabajando no está «En espera»: trabaja por él. equipo
// puede ser nil: se arma solo si hace falta (una sesión quieta de las últimas 24 h).
func (s *Servidor) estadoClaude(a claude.Agente, equipo map[string]bool) string {
	ahora := s.ahora()
	est := EstadoClaude(a, ahora)
	if a.Tipo != "sesion" || est == Trabajando || ahora.Sub(a.Ultimo) > 24*time.Hour {
		return est
	}
	if equipo == nil {
		equipo = s.equipoTrabajando()
	}
	if equipo[a.ID] {
		return Trabajando
	}
	return est
}
