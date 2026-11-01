package servidor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/codexvivo"
	"atalaya/internal/trazabilidad"
)

// Las llamadas a Codex en vivo y las de REGISTRO.md son la misma llamada vista de dos maneras: la traza
// tiene los pasos y el registro, el contraste del coordinador. Donde están las dos, se muestra la traza con lo del
// registro encima; las llamadas anteriores a la traza siguen saliendo solo del registro.

var reSufijoHora = regexp.MustCompile(`-\d{6}$`)

// claveRegistro: el id que tendría en REGISTRO.md («codex~F5~2026-10-31-e2-tema»).
func claveRegistro(c codexvivo.Llamada) string {
	return "codex~" + c.Meta.Fase + "~" + reSufijoHora.ReplaceAllString(c.Nombre, "")
}

// RaizDeCarpeta: la raíz del proyecto registrado que contiene esa carpeta (la más honda, si se anidan); "" si ninguno.
// Una sesión de Codex abierta en backend\ es del proyecto de arriba (codexvivo.Lector.RaizDe).
func (s *Servidor) RaizDeCarpeta(carpeta string) string {
	if carpeta == "" {
		return ""
	}
	// Con «/» y sin distinguir mayúsculas, como el resto de las comparaciones de rutas del panel.
	norma := func(r string) string {
		return strings.TrimSuffix(strings.ToLower(strings.ReplaceAll(filepath.Clean(r), `\`, "/")), "/")
	}
	c := norma(carpeta)
	mejor := ""
	for _, p := range s.listaProyectos() {
		if p.Raiz == "" || len(p.Raiz) <= len(mejor) {
			continue
		}
		if r := norma(p.Raiz); c == r || strings.HasPrefix(c, r+"/") {
			mejor = p.Raiz
		}
	}
	return mejor
}

// rolSesionCodex: el rol de una sesión interactiva de Codex por su origen.
var rolSesionCodex = map[string]string{"exec": "Codex · exec", "vscode": "Codex · app", "subagente": "Codex · subagente"}

// vistaSesionCodex: una sesión interactiva de Codex (codexvivo/sesiones.go). Trabaja mientras su turno está abierto,
// espera al usuario la media hora siguiente y después se da por terminada.
func (s *Servidor) vistaSesionCodex(c codexvivo.Llamada) AgenteVista {
	rol := rolSesionCodex[c.Origen]
	if rol == "" {
		rol = "Codex · interactiva"
	}
	ahora := s.ahora()
	estado := Inactiva
	switch {
	case c.Activa(ahora):
		estado = Trabajando
	case c.EnEspera(ahora):
		estado = EnEspera
	}
	v := AgenteVista{ID: c.ID, Proyecto: s.proyectoDeRaiz(c.Meta.Raiz), Proveedor: "Codex", Tipo: "codex", Rol: rol, Tarea: c.Meta.Tema,
		Interactiva: true, Modelo: c.Meta.Modelo, Effort: c.Meta.Effort, Estado: estado, Inicio: c.Inicio, Ultimo: c.Ultimo,
		Tokens: c.Tokens, Pasos: len(c.Pasos)}
	v.Nombre, v.ModeloCorto = nombreVista(v, docsAgente{}), modeloCorto(v.Modelo, v.Effort)
	texto := fmt.Sprintf("%d pasos", len(c.Pasos))
	if c.EnTurno {
		texto = "en curso · " + texto
	}
	v.Avance = &trazabilidad.Avance{Porcentaje: -1, Texto: texto, Hitos: []trazabilidad.Hito{}}
	return v
}

func (s *Servidor) vistaCodexVivo(c codexvivo.Llamada, reg *trazabilidad.LlamadaCodex) AgenteVista {
	if c.Interactiva {
		return s.vistaSesionCodex(c)
	}
	rol := rolCodex[c.Meta.Clase]
	if rol == "" {
		rol = "Codex"
	}
	estado := Entregado
	if c.Activa(s.ahora()) {
		estado = Trabajando
	}
	v := AgenteVista{ID: c.ID, Proyecto: filepath.Base(c.Meta.Raiz), Proveedor: "Codex", Tipo: "codex", Rol: rol, Tarea: c.Meta.Tema,
		Modelo: c.Meta.Modelo, Effort: c.Meta.Effort, Clase: c.Meta.Clase, Base: c.Meta.Base, Estado: estado, Inicio: c.Inicio,
		Ultimo: c.Ultimo, Tokens: c.Tokens, Informe: c.Meta.Informe, Pasos: len(c.Pasos)}
	v.Nombre, v.ModeloCorto = nombreVista(v, docsAgente{}), modeloCorto(v.Modelo, v.Effort)
	if c.Meta.MinutosMax > 0 {
		v.Presupuesto = fmt.Sprintf("%d min", c.Meta.MinutosMax)
	}
	switch {
	case c.Terminada:
		texto := "entregado"
		if c.Meta.Salida != nil {
			texto += " · salida " + strconv.Itoa(*c.Meta.Salida)
		}
		if c.Meta.TiempoAgotado {
			texto += " · se agotó el tiempo"
		}
		texto += fmt.Sprintf(" · %d s", c.Meta.Segundos)
		if reg != nil && reg.Contraste != "" {
			texto += " · contraste: " + reg.Contraste
		}
		v.Avance = &trazabilidad.Avance{Porcentaje: 100, Texto: texto, Hitos: []trazabilidad.Hito{}}
	default:
		v.Avance = &trazabilidad.Avance{Porcentaje: -1, Texto: fmt.Sprintf("en curso · %d pasos", len(c.Pasos)), Hitos: []trazabilidad.Hito{}}
	}
	// Los tokens de Codex llegan al final del turno; mientras tanto, los del registro si ya los hay.
	if reg != nil && c.Tokens == (claude.Tokens{}) {
		v.Tokens.Entrada, v.Tokens.CacheLeida, v.Tokens.Salida = reg.Entrada, reg.EnCache, reg.TokSalida
	}
	return v
}

// sesionDeCodex: el id de la sesión de Claude Code que lanzó la llamada (el nombre de su transcrito sin «.jsonl»,
// que es el id de su agente): la que corrió codex.ps1 con su tema, la más cercana antes de su inicio, con 2 min de
// margen (lanzamientoDe). Vacío si no está en los transcritos: la llamada queda como raíz, «sin sesión».
func sesionDeCodex(ls []claude.LanzamientoCodex, proyecto, tema string, inicio time.Time) string {
	l, ok := lanzamientoDe(ls, proyecto, tema, inicio)
	if !ok {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(l.Sesion), ".jsonl")
}

// padreCodexVivo: la sesión de una llamada en vivo. La que anotó codex.ps1 en su meta, si la hay (exacta); si no,
// la que se deduce de los transcritos (sesionDeCodex).
func padreCodexVivo(ls []claude.LanzamientoCodex, c codexvivo.Llamada) string {
	if c.Interactiva {
		return "" // la abrió una persona, no una sesión de Claude Code
	}
	if c.Meta.Sesion != "" {
		return c.Meta.Sesion
	}
	return sesionDeCodex(ls, filepath.Base(c.Meta.Raiz), c.Meta.Tema, c.Inicio)
}

// archivosDelParche: los archivos que cambió una llamada con -Escribe, de las cabeceras «diff --git» de su parche
// (docs/trazabilidad/<fase>/codex/<fecha>-<tema>.patch, que deja codex.ps1). Codex edita con apply_patch y eso no sale
// en su traza --json: el parche es la única huella. Vacío si no hay parche.
func archivosDelParche(c codexvivo.Llamada) []string {
	if len(c.Nombre) < 10 || c.Meta.Raiz == "" || c.Meta.Fase == "" || c.Meta.Tema == "" {
		return nil
	}
	f, err := os.Open(filepath.Join(c.Meta.Raiz, "docs", "trazabilidad", c.Meta.Fase, "codex", c.Nombre[:10]+"-"+c.Meta.Tema+".patch"))
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 4*1024*1024)
	for sc.Scan() {
		if r, ok := strings.CutPrefix(sc.Text(), "diff --git a/"); ok {
			if i := strings.Index(r, " b/"); i > 0 {
				out = append(out, r[:i])
			}
		}
	}
	return out
}

// inicioRegistro: cuándo empezó una llamada de REGISTRO.md (la hora de su informe menos lo que tardó). Sin informe
// no hay hora: se toma el fin de su fecha, para que la busque entre los lanzamientos de ese día.
func inicioRegistro(c trazabilidad.LlamadaCodex) time.Time {
	if !c.Terminada.IsZero() {
		return c.Terminada.Add(-time.Duration(c.Segundos) * time.Second)
	}
	if d, err := time.ParseInLocation("2006-01-02", c.Fecha, time.Local); err == nil {
		return d.Add(24*time.Hour - time.Second)
	}
	return time.Time{}
}

// codexEnVivo: las llamadas con traza y, por clave del registro, la fila de REGISTRO.md que les corresponde.
func (s *Servidor) codexEnVivo() ([]codexvivo.Llamada, map[string]*trazabilidad.LlamadaCodex) {
	if s.CodexVivo == nil {
		return nil, nil
	}
	ll := s.CodexVivo.Llamadas()
	reg := map[string]*trazabilidad.LlamadaCodex{}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range ll {
		k := claveRegistro(c)
		for _, filas := range s.codex {
			for i := range filas {
				if filas[i].ID == k {
					f := filas[i]
					reg[k] = &f
				}
			}
		}
	}
	return ll, reg
}
