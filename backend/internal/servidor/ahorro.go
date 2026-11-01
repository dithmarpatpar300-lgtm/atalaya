package servidor

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"atalaya/internal/avance"
	"atalaya/internal/codexvivo"
)

// El ahorro de las consultas MCP (F7 C2): lo que un agente no tuvo que leer porque le preguntó a Atalaya. Cada
// herramienta reemplaza unos documentos (la lectura evitada); su tamaño de hoy menos lo que respondió Atalaya es el
// ahorro. Si el agente igual leyó el documento poco después, no hubo ahorro. Todo es de solo lectura.

// Ahorro: lo que la respuesta de una consulta evitó leer.
type Ahorro struct {
	Evita             []Evitado `json:"evita"`
	AlternativaBytes  int64     `json:"alternativa_bytes"`
	AlternativaTokens int64     `json:"alternativa_tokens"`
	AhorroBytes       int64     `json:"ahorro_bytes"`
	AhorroTokens      int64     `json:"ahorro_tokens"`
	// AhorroEsCota: no se sabe cuánto respondió Atalaya (la traza de Codex solo trae una estimación), así que el
	// ahorro es la lectura evitada entera: una cota de arriba.
	AhorroEsCota bool `json:"ahorro_es_cota"`
	// SinMedir: buscar, archivo y preguntar no reemplazan un documento que se pueda nombrar.
	SinMedir bool `json:"sin_medir"`
	// Releyo: el mismo agente leyó uno de los documentos de Evita en sus pasos siguientes; el ahorro no cuenta.
	Releyo bool `json:"releyo"`
	// Repetida: la misma herramienta con los mismos argumentos, del mismo agente, en los VentanaRepetida anteriores.
	Repetida bool `json:"repetida"`
}

// Evitado: un documento que la respuesta reemplaza, con su tamaño de hoy.
type Evitado struct {
	Ruta  string `json:"ruta"` // relativa a la raíz del proyecto, con «/»
	Bytes int64  `json:"bytes"`
	abs   string // entera y normalizada, para compararla con lo que el agente leyó
}

// VentanaRepetida: una consulta igual a otra del mismo agente dentro de este tiempo es una repetición.
const VentanaRepetida = 5 * time.Minute

// tokens: bytes ÷ 4, como el panel.
func tokens(b int64) int64 { return b / 4 }

// normaRuta: entera, con «/» y en minúsculas: Windows no distingue mayúsculas y los transcritos mezclan «\» y «/».
func normaRuta(r string) string {
	if r == "" {
		return ""
	}
	return strings.ToLower(filepath.ToSlash(filepath.Clean(r)))
}

// documento: el archivo si existe, con su tamaño de ahora (un Stat: el tamaño es lo único que se mira).
func documento(raiz, ruta string) (Evitado, bool) {
	abs := ruta
	if !filepath.IsAbs(abs) {
		if raiz == "" {
			return Evitado{}, false
		}
		abs = filepath.Join(raiz, filepath.FromSlash(ruta))
	}
	fi, err := os.Stat(abs)
	if err != nil || fi.IsDir() {
		return Evitado{}, false
	}
	rel := filepath.ToSlash(abs)
	if raiz != "" {
		if r, err := filepath.Rel(raiz, abs); err == nil && !strings.HasPrefix(r, "..") {
			rel = filepath.ToSlash(r)
		}
	}
	return Evitado{Ruta: rel, Bytes: fi.Size(), abs: normaRuta(abs)}, true
}

// rutaPlan: el documento de la línea «Plan:» de una fase («docs/planes/fase-5.md §4»), si está dentro del proyecto;
// la misma regla que usa el paquete avance para leer los paquetes del plan.
func rutaPlan(raiz, plan string) string {
	campos := strings.Fields(strings.Trim(plan, "`"))
	if raiz == "" || len(campos) == 0 {
		return ""
	}
	rel := strings.Trim(campos[0], "`")
	ruta := filepath.Join(raiz, filepath.FromSlash(rel))
	if dentro, err := filepath.Rel(raiz, ruta); err != nil || filepath.IsAbs(rel) || dentro == ".." || strings.HasPrefix(dentro, ".."+string(filepath.Separator)) {
		return ""
	}
	return ruta
}

// argsConsulta: lo que se mira de los argumentos de una consulta.
type argsConsulta struct {
	Agente   string `json:"agente"`
	Proyecto string `json:"proyecto"`
}

// proyectoDeConsulta: el proyecto que respondió estado, retomar o salud: el de los argumentos, o el de la dirección del
// MCP (el del agente), o el mayor, como hace llamar. Falso si los argumentos nombran uno que no existe.
func (s *Servidor) proyectoDeConsulta(c Consulta, a argsConsulta) (string, bool) {
	p := a.Proyecto
	if p == "" {
		p = c.Proyecto
	}
	if p != "" {
		n, ok := s.proyectoMCP(p)
		if !ok {
			if a.Proyecto != "" {
				return "", false
			}
			n = "" // el del agente no está registrado: llamar tampoco lo habría usado
		}
		p = n
	}
	return s.proyectoOMayor(p), true
}

// evitaDe: los documentos que reemplaza la respuesta de c, por herramienta. memo guarda lo de este pedido: cientos de
// consultas de estado del mismo proyecto comparten una sola resolución.
func (s *Servidor) evitaDe(c Consulta, memo map[string][]Evitado) []Evitado {
	var a argsConsulta
	if c.Argumentos != "" {
		_ = json.Unmarshal([]byte(c.Argumentos), &a)
	}
	clave := c.Herramienta + "\x00" + c.Proyecto + "\x00" + a.Proyecto + "\x00" + a.Agente
	if ev, ok := memo[clave]; ok {
		return ev
	}
	var ev []Evitado
	vistos := map[string]bool{}
	agregar := func(raiz, ruta string) {
		// Una vez cada documento: la línea «Plan:» de una fase puede apuntar a su propio AVANCE.md.
		if d, ok := documento(raiz, ruta); ok && !vistos[d.abs] {
			vistos[d.abs] = true
			ev = append(ev, d)
		}
	}
	switch c.Herramienta {
	case "estado", "retomar", "salud":
		p, ok := s.proyectoDeConsulta(c, a)
		raiz := s.raizDe(p)
		if !ok || raiz == "" {
			break
		}
		if c.Herramienta == "retomar" {
			agregar(raiz, "docs/trazabilidad/ESTADO-para-retomar.md")
		}
		if c.Herramienta != "salud" {
			if f := avance.Actual(raiz); f != nil {
				agregar(raiz, f.Ruta)
				if c.Herramienta == "estado" {
					if r := rutaPlan(raiz, f.Plan); r != "" {
						agregar(raiz, r)
					}
				}
			}
		}
		if c.Herramienta != "estado" {
			agregar(raiz, "AGENTS.md")
		}
	case "que_hizo":
		ev = s.evitaQueHizo(a)
	}
	memo[clave] = ev
	return ev
}

// cacheQueHizo: elegir al agente de que_hizo recorre todos los agentes; lo hallado vale vidaDocs, como los documentos
// de un agente (nombres.go).
var cacheQueHizo = struct {
	sync.Mutex
	m map[string]queHizoGuardado
}{m: map[string]queHizoGuardado{}}

type queHizoGuardado struct {
	hasta time.Time
	ev    []Evitado
}

// evitaQueHizo: la bitácora, el acta y el informe del agente que que_hizo elige hoy para esos argumentos.
func (s *Servidor) evitaQueHizo(a argsConsulta) []Evitado {
	proyecto := ""
	if a.Proyecto != "" {
		p, ok := s.proyectoMCP(a.Proyecto)
		if !ok {
			return nil
		}
		proyecto = p
	}
	clave := a.Agente + "\x00" + proyecto
	ahora := time.Now()
	cacheQueHizo.Lock()
	g, ok := cacheQueHizo.m[clave]
	cacheQueHizo.Unlock()
	if ok && ahora.Before(g.hasta) {
		return g.ev
	}
	var ev []Evitado
	if cands := s.agenteQueHizo(a.Agente, proyecto); len(cands) > 0 {
		v := cands[0]
		raiz := s.raizDe(v.Proyecto)
		if !strings.HasPrefix(v.ID, codexvivo.Prefijo) && !strings.HasPrefix(v.ID, "codex~") {
			// Los documentos de un agente de Claude son relativos a la raíz de su repositorio.
			if ag, _, ok := s.Claude.Agente(v.ID, 1); ok {
				v, raiz = s.vistaClaude(ag, true), ag.Raiz
			}
		}
		vistos := map[string]bool{}
		for _, r := range []string{v.Bitacora, v.Acta, v.Informe} {
			if r == "" {
				continue
			}
			if d, ok := documento(raiz, r); ok && !vistos[d.abs] {
				vistos[d.abs] = true
				ev = append(ev, d)
			}
		}
	}
	cacheQueHizo.Lock()
	if len(cacheQueHizo.m) > 256 {
		clear(cacheQueHizo.m)
	}
	// Sin documentos, poco: al arrancar el índice todavía no tiene a los agentes, y 30 s de «nada» serían falsos.
	vida := vidaDocs
	if len(ev) == 0 {
		vida = avance.VidaMemoria
	}
	cacheQueHizo.m[clave] = queHizoGuardado{hasta: ahora.Add(vida), ev: ev}
	cacheQueHizo.Unlock()
	return ev
}

// medirAhorro llena el Ahorro de cada consulta y marca las repetidas.
func (s *Servidor) medirAhorro(cs []Consulta) {
	memo := map[string][]Evitado{}
	for i := range cs {
		c := &cs[i]
		a := &Ahorro{Evita: []Evitado{}}
		c.Ahorro = a
		switch c.Herramienta {
		case "que_hizo":
			// Sin los argumentos (la traza de Codex no los trae) no se sabe por qué agente preguntó.
			if c.argumentosDesconocidos {
				a.SinMedir = true
			} else if !c.Error {
				a.Evita = append(a.Evita, s.evitaDe(*c, memo)...)
			}
		case "estado", "retomar", "salud":
			// Una respuesta con error no reemplazó ninguna lectura.
			if !c.Error {
				a.Evita = append(a.Evita, s.evitaDe(*c, memo)...)
			}
		default:
			a.SinMedir = true
		}
		for _, d := range a.Evita {
			a.AlternativaBytes += d.Bytes
		}
		a.Releyo = releyo(c.leidas, a.Evita)
		switch {
		case a.SinMedir || a.Releyo:
		case c.bytesConocidos:
			a.AhorroBytes = max(0, a.AlternativaBytes-int64(c.Bytes))
		default:
			a.AhorroBytes, a.AhorroEsCota = a.AlternativaBytes, true
		}
		a.AlternativaTokens, a.AhorroTokens = tokens(a.AlternativaBytes), tokens(a.AhorroBytes)
	}
	marcarRepetidas(cs)
}

// releyo: si alguna de las rutas leídas es uno de los documentos evitados.
func releyo(leidas []string, ev []Evitado) bool {
	for _, l := range leidas {
		n := normaRuta(l)
		for _, d := range ev {
			if n == d.abs {
				return true
			}
		}
	}
	return false
}

// marcarRepetidas: por agente y de la más vieja a la más nueva, una consulta repite otra anterior de la misma
// herramienta y los mismos argumentos a VentanaRepetida o menos. Solo se comparan las que tienen agente y argumentos
// conocidos: una traza de Codex sin ellos diría «repetida» de dos consultas distintas. Por agente, para no comparar
// todas contra todas en cada pedido.
func marcarRepetidas(cs []Consulta) {
	porAgente := map[string][]int{}
	for i, c := range cs {
		if c.Agente != "" && !c.argumentosDesconocidos {
			porAgente[c.Agente] = append(porAgente[c.Agente], i)
		}
	}
	for _, is := range porAgente {
		sort.SliceStable(is, func(x, y int) bool { return cs[is[x]].Hora.Before(cs[is[y]].Hora) })
		for k, i := range is {
			c := &cs[i]
			for m := k - 1; m >= 0; m-- {
				p := cs[is[m]]
				if c.Hora.Sub(p.Hora) > VentanaRepetida {
					break
				}
				if p.Herramienta == c.Herramienta && mismaEntrada(p.Argumentos, c.Argumentos) {
					if c.Ahorro == nil {
						c.Ahorro = &Ahorro{Evita: []Evitado{}}
					}
					c.Repetida = true
					break
				}
			}
		}
	}
}

// FilaAhorro: las cifras de un grupo de consultas en el resumen.
type FilaAhorro struct {
	Clave             string `json:"clave"`
	N                 int    `json:"n"`
	Bytes             int64  `json:"bytes"`
	AlternativaBytes  int64  `json:"alternativa_bytes"`
	AlternativaTokens int64  `json:"alternativa_tokens"`
	AhorroBytes       int64  `json:"ahorro_bytes"`
	AhorroTokens      int64  `json:"ahorro_tokens"`
	Releyo            int    `json:"releyo"`
	Repetidas         int    `json:"repetidas"`
	Errores           int    `json:"errores"`
}

func (f *FilaAhorro) sumar(c Consulta) {
	f.N++
	f.Bytes += int64(c.Bytes)
	if c.Error {
		f.Errores++
	}
	if a := c.Ahorro; a != nil {
		f.AlternativaBytes += a.AlternativaBytes
		f.AhorroBytes += a.AhorroBytes
		if a.Releyo {
			f.Releyo++
		}
		if a.Repetida {
			f.Repetidas++
		}
	}
	f.AlternativaTokens, f.AhorroTokens = tokens(f.AlternativaBytes), tokens(f.AhorroBytes)
}

// LecturaEvitable: un paso que leyó un documento que estado o retomar le habrían dado.
type LecturaEvitable struct {
	Hora     time.Time `json:"hora"`
	Agente   string    `json:"agente"`
	Nombre   string    `json:"nombre"`
	Proyecto string    `json:"proyecto"`
	Paso     int       `json:"paso"`
	Ruta     string    `json:"ruta"` // relativa a la raíz del proyecto
	Bytes    int64     `json:"bytes"`
}

// PodriaHaberConsultado: las lecturas evitables y lo que habrían ahorrado.
type PodriaHaberConsultado struct {
	Lecturas []LecturaEvitable `json:"lecturas"`
	Bytes    int64             `json:"bytes"`
	Tokens   int64             `json:"tokens"`
}

// ResumenConsultas: GET /api/consultas/resumen.
type ResumenConsultas struct {
	Dias                  int                   `json:"dias"`
	Proyecto              string                `json:"proyecto,omitempty"`
	Desde                 time.Time             `json:"desde"`
	Total                 FilaAhorro            `json:"total"`
	PorHerramienta        []FilaAhorro          `json:"por_herramienta"`
	PorProyecto           []FilaAhorro          `json:"por_proyecto"`
	PorRol                []FilaAhorro          `json:"por_rol"`
	PorDia                []FilaAhorro          `json:"por_dia"`
	PodriaHaberConsultado PodriaHaberConsultado `json:"podria_haber_consultado"`
}

// rolConsulta: quién la hizo, para el resumen.
func rolConsulta(c Consulta) string {
	switch c.tipo {
	case "sesion":
		return "sesion"
	case "subagente":
		return "subagente"
	case "codex":
		return "codex"
	}
	return "sin agente"
}

func (s *Servidor) verResumenConsultas(w http.ResponseWriter, r *http.Request) {
	dias := DiasConsultas
	if n, err := strconv.Atoi(r.URL.Query().Get("dias")); err == nil && n > 0 {
		// Más allá de DiasConsultas no hay transcritos leídos: el resumen mentiría por omisión.
		dias = min(n, DiasConsultas)
	}
	escribir(w, s.resumenConsultas(dias, strings.TrimSpace(r.URL.Query().Get("proyecto"))))
}

func (s *Servidor) resumenConsultas(dias int, proyecto string) ResumenConsultas {
	desde := s.ahora().AddDate(0, 0, -dias)
	res := ResumenConsultas{Dias: dias, Proyecto: proyecto, Desde: desde, Total: FilaAhorro{Clave: "total"}}
	grupos := [4]map[string]*FilaAhorro{{}, {}, {}, {}}
	for _, c := range s.consultasDesde(desde) {
		if c.Hora.Before(desde) || (proyecto != "" && c.Proyecto != proyecto) {
			continue
		}
		res.Total.sumar(c)
		for g, k := range []string{c.Herramienta, primeroConTexto(c.Proyecto, "sin proyecto"), rolConsulta(c), c.Hora.Local().Format("2006-01-02")} {
			f := grupos[g][k]
			if f == nil {
				f = &FilaAhorro{Clave: k}
				grupos[g][k] = f
			}
			f.sumar(c)
		}
	}
	filas := func(m map[string]*FilaAhorro, porClave bool) []FilaAhorro {
		out := []FilaAhorro{}
		for _, f := range m {
			out = append(out, *f)
		}
		sort.Slice(out, func(i, j int) bool {
			if !porClave && out[i].N != out[j].N {
				return out[i].N > out[j].N
			}
			return out[i].Clave < out[j].Clave
		})
		return out
	}
	res.PorHerramienta, res.PorProyecto, res.PorRol, res.PorDia = filas(grupos[0], false), filas(grupos[1], false), filas(grupos[2], false), filas(grupos[3], true)
	res.PodriaHaberConsultado = s.podriaHaberConsultado(desde, proyecto)
	return res
}

// podriaHaberConsultado: los pasos que leyeron un AVANCE.md, un ESTADO-para-retomar.md o el plan de una fase, de
// agentes de proyectos registrados, sin una consulta estado o retomar del mismo agente en sus pasos anteriores.
// Solo los transcritos de Claude: la traza de Codex lee con órdenes de la consola, no con pasos «Lee».
func (s *Servidor) podriaHaberConsultado(desde time.Time, proyecto string) PodriaHaberConsultado {
	out := PodriaHaberConsultado{Lecturas: []LecturaEvitable{}}
	// Los planes de todas las fases de los proyectos registrados; el nombre del archivo filtra antes de releer la
	// ruta entera del transcrito.
	planes, nombres := map[string]bool{}, map[string]bool{"avance.md": true, "estado-para-retomar.md": true}
	for _, p := range s.listaProyectos() {
		for _, f := range avance.Leer(p.Raiz) {
			if r := rutaPlan(p.Raiz, f.Plan); r != "" {
				planes[normaRuta(r)] = true
				nombres[strings.ToLower(filepath.Base(r))] = true
			}
		}
	}
	quiere := func(hilo string) bool { return nombres[strings.ToLower(filepath.Base(filepath.FromSlash(hilo)))] }
	vistas := map[string]AgenteVista{}
	for _, l := range s.Claude.Lecturas(desde, quiere) {
		raizAgente := s.RaizDeCarpeta(l.Agente.Raiz)
		if raizAgente == "" || l.Ruta == "" || consultoAntes(l.AtalayaAntes) {
			continue
		}
		raizDoc := s.RaizDeCarpeta(filepath.Dir(l.Ruta))
		if raizDoc == "" {
			continue
		}
		// En su lugar del proyecto: el AVANCE.md de una plantilla (kit/…/docs/trazabilidad/F0) no es el que da estado.
		n, traz := normaRuta(l.Ruta), normaRuta(filepath.Join(raizDoc, "docs", "trazabilidad"))+"/"
		resto, enTraz := strings.CutPrefix(n, traz)
		fase, archivo, _ := strings.Cut(resto, "/")
		esAvance := enTraz && archivo == "avance.md" && reFaseDir(fase)
		if !(esAvance || (enTraz && resto == "estado-para-retomar.md") || planes[n]) {
			continue
		}
		v, ok := vistas[l.Agente.ID]
		if !ok {
			v = s.vistaClaude(l.Agente, false)
			vistas[l.Agente.ID] = v
		}
		pr := s.proyectoDeRaiz(raizAgente)
		if proyecto != "" && pr != proyecto {
			continue
		}
		e := LecturaEvitable{Hora: l.Paso.Hora, Agente: v.ID, Nombre: v.Nombre, Proyecto: pr, Paso: l.Paso.N, Ruta: filepath.ToSlash(l.Ruta)}
		if d, ok := documento(raizDoc, l.Ruta); ok {
			e.Ruta, e.Bytes = d.Ruta, d.Bytes
		} else {
			// Ya no está en el disco: lo que el paso metió al contexto.
			e.Bytes = l.Paso.TokEntra * 4
		}
		out.Lecturas = append(out.Lecturas, e)
		out.Bytes += e.Bytes
	}
	out.Tokens = tokens(out.Bytes)
	return out
}

// consultoAntes: si entre las herramientas de Atalaya de los pasos anteriores hubo un estado o un retomar.
func consultoAntes(hs []string) bool {
	for _, h := range hs {
		if h == "estado" || h == "retomar" {
			return true
		}
	}
	return false
}

// reFaseDir: «f7» (ya en minúsculas) es la carpeta de una fase.
func reFaseDir(d string) bool {
	n, err := strconv.Atoi(strings.TrimPrefix(d, "f"))
	return strings.HasPrefix(d, "f") && err == nil && n >= 0
}
