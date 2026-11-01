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

// LanzamientoCodex: una llamada a scripts/codex.ps1 desde una sesión, con la mirada a la cuota anterior.
type LanzamientoCodex struct {
	Proyecto string
	Sesion   string // ruta del transcrito
	Hora     time.Time
	Paso     int
	Orden    string
	// OrdenNorm: la orden normalizada (Normalizar), para buscar en ella sin normalizarla en cada pedido.
	OrdenNorm string
	Mirada    Mirada
}

var reTemporal = regexp.MustCompile(`(?i)[\w.-]*-tmp[\\/]+(\d{4}-\d\d-\d\d-[\w.-]+)`)

// normalizada: la orden normalizada; la guardada al leer el paso, o calculada si el paso no la trae.
func (p *Paso) normalizada() string {
	if p.ordenNorm == "" && p.orden != "" {
		return Normalizar(p.orden)
	}
	return p.ordenNorm
}

// Marcas de un paso: lo que la sesión, la salud y los lanzamientos de Codex preguntan de cada paso en cada pedido
// del panel. Dependen solo de lo que el paso trae al leerse (tipo, herramienta, orden y archivo editado), así que se
// calculan una vez en agregar: con expresiones regulares sobre miles de pasos por pedido, era lo más caro del
// servidor (F7 P1).
const (
	marcado        uint8 = 1 << iota // las demás ya están calculadas
	marcaCuota                       // miraCuota
	marcaCommit                      // un «Ejecuta» con git … commit
	marcaDocEstado                   // edita un documento de estado (sin mirar Error: llega después, con el resultado)
	marcaCodex                       // un «Ejecuta» que corre codex.ps1
)

// marcasDe: las marcas de un paso, calculadas.
func marcasDe(p *Paso) uint8 {
	m := marcado
	if miraCuota(p) {
		m |= marcaCuota
	}
	if p.Tipo == "Ejecuta" {
		o := p.normalizada()
		if reGitCommit.MatchString(o) {
			m |= marcaCommit
		}
		if strings.Contains(o, "codex.ps1") {
			m |= marcaCodex
		}
	}
	if p.editado != "" && reDocEstado.MatchString(p.editado) {
		m |= marcaDocEstado
	}
	return m
}

// tiene: si el paso lleva la marca; un paso que no pasó por agregar (las pruebas) la calcula en el momento.
func (p *Paso) tiene(m uint8) bool {
	if p.marcas&marcado == 0 {
		return marcasDe(p)&m != 0
	}
	return p.marcas&m != 0
}

// miraCuota: el paso lee la cuota (Cuotas, get_usage, o un script del proyecto que corre «cuotas una-vez»).
func miraCuota(p *Paso) bool {
	if strings.HasSuffix(p.Herramienta, "get_usage") {
		return true
	}
	if p.Tipo != "Ejecuta" {
		return false
	}
	o := p.normalizada()
	return (strings.Contains(o, "cuotas") && (strings.Contains(o, "una-vez") || strings.Contains(o, "importar"))) || strings.Contains(o, "retomar.ps1")
}

// miradaAntes: la última mirada a la cuota entre los pasos [0, i). Es la regla; LanzamientosCodex y Revisiones la
// calculan en una sola pasada, llevando la última mirada mientras avanzan (revision_test.go la prueba).
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
	// Sin la línea, nada que contar; con ella, se recorre sin partir el encargo entero (se para en la línea).
	if !strings.Contains(encargo, "Asignación:") && !strings.Contains(encargo, "Assignment:") {
		return 0
	}
	n := 0
	for l := range strings.Lines(encargo) {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		n++
		// El filtro antes de la expresión: sin él, cada línea de cada encargo pasaba por ella en cada pedido.
		if (strings.Contains(l, "Asignación:") || strings.Contains(l, "Assignment:")) && reAsignacion.MatchString(l) {
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
	// Las llamadas que se buscan, por sesión: guardar en el mapa solo esas, no cada herramienta de cada sesión.
	buscadas := map[string]map[string]bool{}
	for _, a := range ix.arch {
		if ag := a.agente; ag.Tipo == "subagente" && ag.ToolUseID != "" {
			if buscadas[ag.Padre] == nil {
				buscadas[ag.Padre] = map[string]bool{}
			}
			buscadas[ag.Padre][ag.ToolUseID] = true
		}
	}
	// Por sesión, una sola pasada: cada llamada (por su id) con su paso y la última mirada a la cuota antes de él.
	type lanzado struct {
		p      *Paso
		mirada Mirada
	}
	llamadas := map[string]map[string]lanzado{}
	llamadasDe := func(s *archivo) map[string]lanzado {
		if m, ok := llamadas[s.agente.ID]; ok {
			return m
		}
		m := map[string]lanzado{}
		quiere := buscadas[s.agente.ID]
		var mirada Mirada
		for _, p := range s.pasos {
			if p.idUso != "" && quiere[p.idUso] {
				if _, visto := m[p.idUso]; !visto {
					m[p.idUso] = lanzado{p, mirada}
				}
			}
			if p.tiene(marcaCuota) {
				mirada = Mirada{Hora: p.Hora, Paso: p.N, Orden: p.orden}
			}
		}
		llamadas[s.agente.ID] = m
		return m
	}
	out := map[string]Revision{}
	for _, a := range ix.arch {
		ag := a.agente
		if ag.Tipo != "subagente" || ag.Pasos == 0 || ag.Ultimo.Before(desde) {
			continue
		}
		// Las temporales ya están al día en el agente (agregar las suma con cada paso): recorrer todos sus pasos con
		// una expresión regular en cada pedido era la mitad del costo de la salud del proceso.
		r := Revision{LineaAsignacion: lineaAsignacion(ag.Encargo), Transcrito: a.ruta, Temporales: append([]string(nil), ag.Temporales...)}
		if s := sesiones[ag.Padre]; s != nil && ag.ToolUseID != "" {
			r.Sesion = s.ruta
			if l, ok := llamadasDe(s)[ag.ToolUseID]; ok {
				r.Lanzado, r.PasoLanzado, r.Mirada = l.p.Hora, l.p.N, l.mirada
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
		// Una sola pasada: la última mirada a la cuota se lleva mientras se avanza (buscarla hacia atrás desde cada
		// lanzamiento recorría miles de pasos por lanzamiento, en cada pedido del panel).
		var mirada Mirada
		for _, p := range a.pasos {
			if p.tiene(marcaCodex) && !p.Hora.Before(desde) {
				out = append(out, LanzamientoCodex{Proyecto: a.agente.Proyecto, Sesion: a.ruta, Hora: p.Hora, Paso: p.N, Orden: p.orden,
					OrdenNorm: p.normalizada(), Mirada: mirada})
			}
			if p.tiene(marcaCuota) {
				mirada = Mirada{Hora: p.Hora, Paso: p.N, Orden: p.orden}
			}
		}
	}
	return out
}
