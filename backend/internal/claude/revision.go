package claude

import (
	"regexp"
	"slices"
	"strings"
	"time"
)

// Lo que la salud del proceso (internal/proceso) necesita saber de los transcritos: dónde va la línea «Asignación»,
// cuándo se lanzó cada subagente, si la sesión que lo lanzó miró la cuota antes y qué carpetas temporales usó.

// Revision de un subagente.
type Revision struct {
	// LineaAsignacion: en qué línea con contenido del encargo va «Asignación:» (1 = la primera); 0 si no la trae.
	LineaAsignacion int
	// Transcrito: la ruta del transcrito del subagente, la prueba de lo que dice su encargo.
	Transcrito string
	// El «Delega» que lo lanzó, en el transcrito de la sesión; Lanzado es cero si no se encontró.
	Lanzado     time.Time
	Sesion      string // ruta del transcrito de la sesión
	PasoLanzado int
	Mirada      Mirada
	// Temporales: las carpetas «<fecha>-…» bajo una raíz «…-tmp» que nombran sus pasos.
	Temporales []string
}

// Mirada a la cuota: el último paso de la sesión que la leyó antes de lanzar. Hora cero si no hubo ninguno.
type Mirada struct {
	Hora  time.Time
	Paso  int
	Orden string
}

// LanzamientoCodex: una llamada a scripts/ops/codex.ps1 desde una sesión, con la mirada a la cuota anterior.
type LanzamientoCodex struct {
	Proyecto string
	Sesion   string // ruta del transcrito
	Hora     time.Time
	Paso     int
	Orden    string
	Mirada   Mirada
}

var reTemporal = regexp.MustCompile(`(?i)[\w.-]*-tmp[\\/]+(\d{4}-\d\d-\d\d-[\w.-]+)`)

// miraCuota: el paso lee la cuota (Cuotas, get_usage, o retomar.ps1, que corre «cuotas una-vez»).
func miraCuota(p *Paso) bool {
	if strings.HasSuffix(p.Herramienta, "get_usage") {
		return true
	}
	if p.Tipo != "Ejecuta" {
		return false
	}
	o := Normalizar(p.orden)
	return (strings.Contains(o, "cuotas") && (strings.Contains(o, "una-vez") || strings.Contains(o, "importar"))) || strings.Contains(o, "retomar.ps1")
}

// miradaAntes: la última mirada a la cuota entre los pasos [0, i).
func miradaAntes(pasos []*Paso, i int) Mirada {
	for j := i - 1; j >= 0; j-- {
		if miraCuota(pasos[j]) {
			return Mirada{Hora: pasos[j].Hora, Paso: pasos[j].N, Orden: pasos[j].orden}
		}
	}
	return Mirada{}
}

// lineaAsignacion: la línea con contenido (contando desde 1) que empieza con «Asignación:».
func lineaAsignacion(encargo string) int {
	n := 0
	for _, l := range strings.Split(encargo, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		n++
		if reAsignacion.MatchString(l) {
			return n
		}
	}
	return 0
}

// temporalesDe: las carpetas temporales que nombran los pasos (su archivo o su orden), sin repetir.
func temporalesDe(pasos []*Paso) []string {
	var out []string
	for _, p := range pasos {
		out = sumarTemporales(out, p)
	}
	return out
}

// sumarTemporales: las carpetas que nombra un paso, sumadas a las que ya se tenían.
func sumarTemporales(l []string, p *Paso) []string {
	for _, m := range reTemporal.FindAllStringSubmatch(p.Hilo+"\n"+p.orden, -1) {
		nombre := strings.TrimRight(m[1], ".")
		if !slices.Contains(l, nombre) {
			l = append(l, nombre)
		}
	}
	return l
}

// Revisiones de los subagentes que trabajaron desde `desde`, por id.
func (ix *Indice) Revisiones(desde time.Time) map[string]Revision {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	sesiones := map[string]*archivo{}
	for _, a := range ix.arch {
		if a.agente.Tipo == "sesion" {
			sesiones[a.agente.ID] = a
		}
	}
	out := map[string]Revision{}
	for _, a := range ix.arch {
		ag := a.agente
		if ag.Tipo != "subagente" || ag.Pasos == 0 || ag.Ultimo.Before(desde) {
			continue
		}
		r := Revision{LineaAsignacion: lineaAsignacion(ag.Encargo), Transcrito: a.ruta, Temporales: temporalesDe(a.pasos)}
		if s := sesiones[ag.Padre]; s != nil && ag.ToolUseID != "" {
			r.Sesion = s.ruta
			for i, p := range s.pasos {
				if p.idUso == ag.ToolUseID {
					r.Lanzado, r.PasoLanzado, r.Mirada = p.Hora, p.N, miradaAntes(s.pasos, i)
					break
				}
			}
		}
		out[ag.ID] = r
	}
	return out
}

// LanzamientosCodex: cada «Ejecuta» de una sesión que corrió codex.ps1 desde `desde`.
func (ix *Indice) LanzamientosCodex(desde time.Time) []LanzamientoCodex {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	var out []LanzamientoCodex
	for _, a := range ix.arch {
		if a.agente.Tipo != "sesion" || a.agente.Ultimo.Before(desde) {
			continue
		}
		for i, p := range a.pasos {
			if p.Tipo == "Ejecuta" && !p.Hora.Before(desde) && strings.Contains(Normalizar(p.orden), "codex.ps1") {
				out = append(out, LanzamientoCodex{Proyecto: a.agente.Proyecto, Sesion: a.ruta, Hora: p.Hora, Paso: p.N, Orden: p.orden,
					Mirada: miradaAntes(a.pasos, i)})
			}
		}
	}
	return out
}
