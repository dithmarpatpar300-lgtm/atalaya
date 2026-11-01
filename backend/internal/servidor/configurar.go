package servidor

// La configuración desde el panel (fase 6, docs/plan-configuracion.md). /api/entorno dice qué hay en el equipo: Claude
// Code y Codex, sus versiones, si ya hablan con Atalaya, y las carpetas donde trabajaron sus agentes en los últimos 30
// días. /api/configurar/... es la puerta de escritura (internal/configurar): primero la vista previa, después aplicar
// con su id, y el historial con deshacer. Las escrituras solo las acepta del panel (delPanel); el MCP no las tiene.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"atalaya/internal/configurar"
	"atalaya/internal/entorno"
	"atalaya/internal/proyectos"
)

// VentanaEntorno: las carpetas recientes son las de los agentes de los últimos 30 días.
const VentanaEntorno = 30 * 24 * time.Hour

// Entorno: lo que el asistente necesita saber del equipo antes de preguntar nada.
type Entorno struct {
	Sistema      string                `json:"sistema"`
	Arquitectura string                `json:"arquitectura"`
	Version      string                `json:"version"`
	Herramientas []entorno.Herramienta `json:"herramientas"`
	Carpetas     []CarpetaReciente     `json:"carpetas"`
	// Principal: la herramienta con más sesiones en los últimos 30 días («claude» o «codex»); vacía si ninguna.
	Principal string `json:"principal"`
	Registro  string `json:"registro"`
}

// CarpetaReciente: una carpeta donde trabajaron agentes, con lo que ya tiene. El nombre es el de la carpeta, que es
// como Atalaya agrupa a los agentes.
type CarpetaReciente struct {
	Ruta       string         `json:"ruta"`
	Nombre     string         `json:"nombre"`
	Claude     int            `json:"claude"` // sesiones de Claude Code
	Codex      int            `json:"codex"`  // sesiones interactivas de Codex
	Ultimo     time.Time      `json:"ultimo"`
	Registrada bool           `json:"registrada"`
	Marcas     entorno.Marcas `json:"marcas"`
}

// cacheEntorno: buscar las herramientas corre «--version», que tarda; se guarda 5 min.
type cacheEntorno struct {
	mu   sync.Mutex
	hs   []entorno.Herramienta
	hora time.Time
}

func (s *Servidor) herramientas() []entorno.Herramienta {
	if s.Casa == "" {
		return []entorno.Herramienta{}
	}
	s.cacheEnt.mu.Lock()
	defer s.cacheEnt.mu.Unlock()
	if s.cacheEnt.hs == nil || time.Since(s.cacheEnt.hora) > 5*time.Minute {
		s.cacheEnt.hs = entorno.Herramientas(entorno.Opciones{Casa: s.Casa, CodexHome: s.CodexHome})
		s.cacheEnt.hora = time.Now()
	}
	return s.cacheEnt.hs
}

// carpetasRecientes junta las raíces de las sesiones de Claude Code y de las interactivas de Codex.
func (s *Servidor) carpetasRecientes() []CarpetaReciente {
	desde := s.ahora().Add(-VentanaEntorno)
	por := map[string]*CarpetaReciente{}
	sumar := func(ruta string, ultimo time.Time, claude bool) {
		if ruta == "" || ultimo.Before(desde) {
			return
		}
		ruta = filepath.Clean(ruta)
		k := strings.ToLower(ruta)
		c := por[k]
		if c == nil {
			c = &CarpetaReciente{Ruta: ruta, Nombre: filepath.Base(ruta)}
			por[k] = c
		}
		if claude {
			c.Claude++
		} else {
			c.Codex++
		}
		if ultimo.After(c.Ultimo) {
			c.Ultimo = ultimo
		}
	}
	if s.Claude != nil {
		for _, a := range s.Claude.Agentes() {
			if a.Tipo == "sesion" {
				sumar(a.Raiz, a.Ultimo, true)
			}
		}
	}
	if s.CodexVivo != nil {
		for _, c := range s.CodexVivo.Llamadas() {
			if c.Interactiva {
				r := c.Meta.Raiz
				if r == "" {
					r = c.Carpeta
				}
				sumar(r, c.Ultimo, false)
			}
		}
	}
	registradas := s.listaProyectos()
	// Una carpeta dentro de otra de la lista, o de un proyecto registrado (un worktree en .claude/worktrees, una
	// subcarpeta), no es un proyecto aparte: sus sesiones cuentan para la que la contiene.
	dentro := func(hija, madre string) bool {
		rel, err := filepath.Rel(madre, hija)
		return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
	}
	for k, c := range por {
		for k2, m := range por {
			if k2 != k && dentro(c.Ruta, m.Ruta) {
				m.Claude, m.Codex = m.Claude+c.Claude, m.Codex+c.Codex
				if c.Ultimo.After(m.Ultimo) {
					m.Ultimo = c.Ultimo
				}
				delete(por, k)
				break
			}
		}
	}
	for k, c := range por {
		for _, p := range registradas {
			if p.Raiz != "" && dentro(c.Ruta, p.Raiz) {
				delete(por, k)
				break
			}
		}
	}
	out := make([]CarpetaReciente, 0, len(por))
	for _, c := range por {
		for _, p := range registradas {
			if p.Raiz != "" && configurar.MismaRuta(p.Raiz, c.Ruta) {
				c.Registrada = true
			}
		}
		if esCarpeta(c.Ruta) {
			c.Marcas = entorno.MarcasDe(c.Ruta)
		}
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ultimo.After(out[j].Ultimo) })
	if len(out) > 30 {
		out = out[:30]
	}
	return out
}

func esCarpeta(ruta string) bool {
	fi, err := os.Stat(ruta)
	return err == nil && fi.IsDir()
}

func (s *Servidor) verEntorno(w http.ResponseWriter, _ *http.Request) {
	e := Entorno{Sistema: runtime.GOOS, Arquitectura: runtime.GOARCH, Version: Version, Herramientas: s.herramientas(),
		Carpetas: s.carpetasRecientes(), Registro: s.RutaRegistro}
	claude, codex := 0, 0
	for _, c := range e.Carpetas {
		claude += c.Claude
		codex += c.Codex
	}
	switch {
	case claude > 0 && claude >= codex:
		e.Principal = "claude"
	case codex > 0:
		e.Principal = "codex"
	}
	escribir(w, e)
}

// puerta: la de escritura, con su lista cerrada; sin carpeta de datos no hay dónde guardar copias ni historial.
func (s *Servidor) puerta() (*configurar.Puerta, error) {
	s.puertaUna.Do(func() {
		if s.Datos == "" {
			return
		}
		s.puertaConf = &configurar.Puerta{
			Dir: s.Datos,
			Permitido: func(r string) bool {
				return s.RutaRegistro != "" && configurar.MismaRuta(r, s.RutaRegistro)
			},
			Programas: map[string]string{},
			Ahora:     s.ahora,
		}
	})
	if s.puertaConf == nil {
		return nil, errors.New("Atalaya corre sin carpeta de datos: no puede guardar copias ni el historial de cambios")
	}
	return s.puertaConf, nil
}

// pedidoProyectos: qué agregar al registro y qué quitar.
type pedidoProyectos struct {
	Agregar []struct {
		Ruta  string `json:"ruta"`
		Nivel int    `json:"nivel"`
		Sigla string `json:"sigla"`
	} `json:"agregar"`
	Quitar []string `json:"quitar"`
}

var reSigla = regexp.MustCompile(`^[A-Z0-9]{1,3}$`)

// cambioRegistro arma el proyectos.json nuevo: los que ya están, en su orden, menos los que se quitan, y los nuevos
// al final. El nombre de un proyecto nuevo es el de su carpeta (así agrupa Atalaya a sus agentes).
func cambioRegistro(actual []byte, existe bool, p pedidoProyectos) (string, string, error) {
	var lista []proyectos.Proyecto
	if existe && strings.TrimSpace(string(actual)) != "" {
		l, err := proyectos.Parsear(actual)
		if err != nil {
			return "", "", fmt.Errorf("%w: arréglalo a mano antes de cambiarlo desde el panel", err)
		}
		lista = l
	}
	var hechos []string
	for _, q := range p.Quitar {
		antes := len(lista)
		lista = sinProyecto(lista, q)
		if len(lista) == antes {
			return "", "", fmt.Errorf("%q no está en el registro", q)
		}
		hechos = append(hechos, "quitar "+q)
	}
	for _, a := range p.Agregar {
		ruta := strings.TrimSpace(a.Ruta)
		if !filepath.IsAbs(ruta) {
			return "", "", fmt.Errorf("%q no es una ruta completa", a.Ruta)
		}
		ruta = filepath.Clean(ruta)
		if !esCarpeta(ruta) {
			return "", "", fmt.Errorf("%s no es una carpeta de este equipo", ruta)
		}
		if a.Nivel < 0 || a.Nivel > 3 {
			return "", "", fmt.Errorf("el nivel va de 0 a 3, vino %d", a.Nivel)
		}
		nombre := filepath.Base(ruta)
		sigla := strings.ToUpper(strings.TrimSpace(a.Sigla))
		if sigla == "" {
			sigla = proyectos.Sigla(nombre)
		}
		if !reSigla.MatchString(sigla) {
			return "", "", fmt.Errorf("la sigla %q: de una a tres letras o cifras", a.Sigla)
		}
		for _, x := range lista {
			if strings.EqualFold(x.Nombre, nombre) || configurar.MismaRuta(x.Ruta, ruta) {
				return "", "", fmt.Errorf("%s ya está en el registro (como %q)", ruta, x.Nombre)
			}
		}
		lista = append(lista, proyectos.Proyecto{Nombre: nombre, Ruta: ruta, Sigla: sigla, Nivel: a.Nivel})
		hechos = append(hechos, fmt.Sprintf("agregar %s (nivel %d)", nombre, a.Nivel))
	}
	if len(hechos) == 0 {
		return "", "", errors.New("no hay nada que agregar ni quitar")
	}
	texto, err := formatoRegistro(lista)
	if err != nil {
		return "", "", err
	}
	if _, err := proyectos.Parsear([]byte(texto)); err != nil {
		return "", "", err
	}
	return texto, "Registro de proyectos: " + strings.Join(hechos, ", "), nil
}

func sinProyecto(l []proyectos.Proyecto, nombre string) []proyectos.Proyecto {
	out := l[:0:0]
	for _, p := range l {
		if !strings.EqualFold(p.Nombre, strings.TrimSpace(nombre)) {
			out = append(out, p)
		}
	}
	return out
}

// formatoRegistro: una línea por proyecto, como el ejemplo de la guía, para que se lea y se edite a mano.
func formatoRegistro(l []proyectos.Proyecto) (string, error) {
	var b strings.Builder
	b.WriteString("[\n")
	for i, p := range l {
		n, _ := json.Marshal(p.Nombre)
		r, _ := json.Marshal(p.Ruta)
		sg, _ := json.Marshal(p.Sigla)
		fmt.Fprintf(&b, "  { \"nombre\": %s, \"ruta\": %s, \"sigla\": %s, \"nivel\": %d }", n, r, sg, p.Nivel)
		if i < len(l)-1 {
			b.WriteString(",")
		}
		b.WriteString("\n")
	}
	b.WriteString("]\n")
	return b.String(), nil
}

// leerCuerpo: un JSON pequeño del panel.
func leerCuerpo(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(v); err != nil {
		fallo(w, http.StatusBadRequest, "el cuerpo no es un JSON válido")
		return false
	}
	return true
}

func (s *Servidor) proponerProyectos(w http.ResponseWriter, r *http.Request) {
	if !delPanel(r) {
		fallo(w, http.StatusForbidden, "solo el panel puede cambiar la configuración")
		return
	}
	var p pedidoProyectos
	if !leerCuerpo(w, r, &p) {
		return
	}
	if s.RutaRegistro == "" {
		fallo(w, http.StatusConflict, "Atalaya corre sin registro de proyectos (-proyectos)")
		return
	}
	pu, err := s.puerta()
	if err != nil {
		fallo(w, http.StatusConflict, err.Error())
		return
	}
	actual, errL := os.ReadFile(s.RutaRegistro)
	if errL != nil && !errors.Is(errL, os.ErrNotExist) {
		fallo(w, http.StatusInternalServerError, "no se pudo leer el registro: "+errL.Error())
		return
	}
	texto, resumen, err := cambioRegistro(actual, errL == nil, p)
	if err != nil {
		fallo(w, http.StatusBadRequest, err.Error())
		return
	}
	prop, err := pu.Proponer("proyectos", resumen, []configurar.Cambio{{Tipo: configurar.Archivo, Ruta: s.RutaRegistro, Despues: texto}})
	if err != nil {
		fallo(w, http.StatusBadRequest, err.Error())
		return
	}
	escribir(w, prop)
}

type pedidoID struct {
	ID string `json:"id"`
}

func (s *Servidor) aplicarConfiguracion(w http.ResponseWriter, r *http.Request) {
	s.conPuerta(w, r, func(ctx context.Context, pu *configurar.Puerta, id string) (configurar.Entrada, error) {
		return pu.Aplicar(ctx, id)
	})
}

func (s *Servidor) deshacerConfiguracion(w http.ResponseWriter, r *http.Request) {
	s.conPuerta(w, r, func(ctx context.Context, pu *configurar.Puerta, id string) (configurar.Entrada, error) {
		return pu.Deshacer(ctx, id)
	})
}

// conPuerta: aplicar y deshacer comparten la puerta del panel, el cuerpo {id} y la respuesta (la entrada del
// historial, también cuando falla a medias, con su error).
func (s *Servidor) conPuerta(w http.ResponseWriter, r *http.Request, hacer func(context.Context, *configurar.Puerta, string) (configurar.Entrada, error)) {
	if !delPanel(r) {
		fallo(w, http.StatusForbidden, "solo el panel puede cambiar la configuración")
		return
	}
	var p pedidoID
	if !leerCuerpo(w, r, &p) {
		return
	}
	pu, err := s.puerta()
	if err != nil {
		fallo(w, http.StatusConflict, err.Error())
		return
	}
	e, err := hacer(r.Context(), pu, p.ID)
	if err != nil && e.ID == "" {
		fallo(w, http.StatusConflict, err.Error())
		return
	}
	// El registro pudo cambiar: la portada lo ve ya, sin esperar a Vigilar.
	s.recargarRegistro()
	escribir(w, e)
}

func (s *Servidor) verHistorialConfiguracion(w http.ResponseWriter, _ *http.Request) {
	pu, err := s.puerta()
	if err != nil {
		escribir(w, []configurar.Entrada{})
		return
	}
	h, err := pu.Historial(50)
	if err != nil {
		fallo(w, http.StatusInternalServerError, err.Error())
		return
	}
	escribir(w, h)
}
