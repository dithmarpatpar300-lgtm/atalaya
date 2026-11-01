// Package estadisticas cuenta cuánto cuesta cada clase de tarea: minutos, tokens de salida
// y tokens nuevos por clase de AGENTES.md §2, contra el presupuesto de las asignaciones y contra la configuración que
// manda la matriz (§3 para Claude, §14.2 para Codex). Sirve para recalibrar la matriz y decidir qué va a Codex.
//
// La matriz se lee del documento que AGENTS.md nombra, no se copia aquí: si cambia, las estadísticas la siguen.
package estadisticas

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Config de una clase en la matriz: el modelo (familia en Claude, nombre en Codex) y el effort.
type Config struct {
	Modelo string `json:"modelo"`
	Effort string `json:"effort"`
	Fuente string `json:"fuente"` // «AGENTES.md:72»
}

// Texto: «sonnet/high».
func (c Config) Texto() string {
	if c.Modelo == "" {
		return ""
	}
	return c.Modelo + "/" + c.Effort
}

// Matriz leída de AGENTES.md.
type Matriz struct {
	Ruta    string            // relativa a la raíz del proyecto; vacía si AGENTS.md no nombra el documento
	Orden   []string          // las clases en el orden de §3
	Nombres map[string]string // «M» → «Mecánica»; «R» vale para R1, R2 y R3
	Claude  map[string]Config
	Codex   map[string]Config
	// La meta de agentes en Opus de §12 («menos del 40 %»), en fracción; 0 si no está escrita.
	MetaOpus   float64
	FuenteOpus string
}

var (
	reDocAgentes = regexp.MustCompile("`(docs/[\\w/.-]*AGENTES\\.md)`")
	reNombre     = regexp.MustCompile(`^\| \*\*([A-Z]\d?) · ([^*|]+)\*\*`)
	reFila       = regexp.MustCompile(`^\| ([A-Z]\d?) \|`)
	reFamilia    = regexp.MustCompile(`(?i)\b(haiku|sonnet|opus|fable)\b`)
	reEffort     = regexp.MustCompile("`(low|medium|high|xhigh|max)`")
	reCodex      = regexp.MustCompile("Codex `([\\w.-]+)` · `(\\w+)`")
	reMetaOpus   = regexp.MustCompile(`(?i)opus.*menos del (\d+) ?%`)
)

// RutaAgentes: el documento de agentes que cita AGENTS.md («docs/operacion/AGENTES.md», o «docs/operacion-neutro/…» en
// un proyecto que ya tenía su docs/operacion); vacía si AGENTS.md no cita ninguno.
func RutaAgentes(raiz string) string {
	if raiz == "" {
		return ""
	}
	b, err := os.ReadFile(filepath.Join(raiz, "AGENTS.md"))
	if err != nil {
		return ""
	}
	if r := reDocAgentes.FindStringSubmatch(string(b)); r != nil {
		return r[1]
	}
	return ""
}

// LeerMatriz busca en AGENTS.md el documento de agentes y lee de él los nombres de las clases (§2), la matriz de
// Claude (§3), la de Codex (§14.2) y la meta de Opus (§12). Sin documento, la matriz va vacía.
func LeerMatriz(raiz string) Matriz {
	m := Matriz{Nombres: map[string]string{}, Claude: map[string]Config{}, Codex: map[string]Config{}}
	ruta := RutaAgentes(raiz)
	if ruta == "" {
		return m
	}
	f, err := os.Open(filepath.Join(raiz, filepath.FromSlash(ruta)))
	if err != nil {
		return m
	}
	defer f.Close()
	m.Ruta = ruta
	base := filepath.Base(ruta)
	seccion := ""
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for n := 1; sc.Scan(); n++ {
		t := sc.Text()
		if strings.HasPrefix(t, "## ") || strings.HasPrefix(t, "### ") {
			seccion = strings.Fields(t)[1]
			continue
		}
		fuente := base + ":" + strconv.Itoa(n)
		switch seccion {
		case "2.":
			if x := reNombre.FindStringSubmatch(t); x != nil {
				m.Nombres[x[1]] = strings.TrimSpace(x[2])
			}
		case "3.":
			x := reFila.FindStringSubmatch(t)
			if x == nil {
				continue
			}
			celdas := strings.Split(t, "|")
			if len(celdas) < 4 {
				continue
			}
			conf := celdas[3]
			fam, ef := reFamilia.FindStringSubmatch(conf), reEffort.FindStringSubmatch(conf)
			if fam != nil && ef != nil {
				m.Claude[x[1]] = Config{Modelo: strings.ToLower(fam[1]), Effort: ef[1], Fuente: fuente}
				m.Orden = append(m.Orden, x[1])
			}
		case "14.2":
			if x := reFila.FindStringSubmatch(t); x != nil {
				if c := reCodex.FindStringSubmatch(t); c != nil {
					m.Codex[x[1]] = Config{Modelo: c[1], Effort: c[2], Fuente: fuente}
				}
			}
		case "12.":
			if x := reMetaOpus.FindStringSubmatch(t); x != nil && m.FuenteOpus == "" {
				if v, err := strconv.Atoi(x[1]); err == nil {
					m.MetaOpus, m.FuenteOpus = float64(v)/100, fuente
				}
			}
		}
	}
	return m
}

// NombreDe una clase: «I2» → «Implementación con decisiones»; «R1» → el de «R».
func (m Matriz) NombreDe(clase string) string {
	if n, ok := m.Nombres[clase]; ok {
		return n
	}
	if len(clase) > 1 {
		return m.Nombres[clase[:1]]
	}
	return ""
}

// Agente: lo que cuenta de uno ya entregado.
type Agente struct {
	ID        string
	Nombre    string
	Fin       time.Time
	Clase     string // «I2»; vacía si su encargo no la dice
	Proveedor string // Claude | Codex
	Modelo    string // familia en Claude («sonnet»), nombre en Codex («gpt-6.1-terra»)
	Effort    string
	Minutos   float64
	Salida    int64
	Nuevos    int64 // entrada que no vino de la caché (incluida la que se escribió en ella)
	Leida     int64 // entrada leída de la caché
	// Presupuesto: los minutos que dio su asignación; 0 si no lo dice.
	Presupuesto float64
}

// Medida de un grupo: la mediana y el percentil 90.
type Medida struct {
	Mediana float64 `json:"mediana"`
	P90     float64 `json:"p90"`
}

// Clase de la tabla de barras: una clase con un proveedor.
type Clase struct {
	Clase     string        `json:"clase"`
	Nombre    string        `json:"nombre"`
	Proveedor string        `json:"proveedor"`
	N         int           `json:"n"`
	Minutos   Medida        `json:"minutos"`
	Salida    Medida        `json:"salida"`
	Nuevos    Medida        `json:"nuevos"`
	Agentes   []AgenteClase `json:"agentes"`
	// El presupuesto en minutos: la mediana de lo que dicen sus asignaciones, o el tope del proyecto.
	Presupuesto       float64 `json:"presupuesto"`
	PresupuestoFuente string  `json:"presupuesto_fuente"`
	// Lo que manda la matriz para esta clase y este proveedor, y cuántos corrieron así.
	Matriz       string `json:"matriz,omitempty"`
	MatrizFuente string `json:"matriz_fuente,omitempty"`
	EnMatriz     int    `json:"en_matriz"`
}

type AgenteClase struct {
	ID          string    `json:"id"`
	Nombre      string    `json:"nombre"`
	Proveedor   string    `json:"proveedor"`
	Modelo      string    `json:"modelo"`
	Minutos     float64   `json:"minutos"`
	Presupuesto float64   `json:"presupuesto"`
	Fin         time.Time `json:"fin"`
}

// Celda de la tabla clase × modelo.
type Celda struct {
	Clase    string  `json:"clase"`
	Modelo   string  `json:"modelo"`
	N        int     `json:"n"`
	Minutos  float64 `json:"minutos"` // la mediana
	EnMatriz bool    `json:"en_matriz"`
	Matriz   string  `json:"matriz,omitempty"` // lo que manda la matriz, para decir por qué no coincide
}

// Totales del periodo.
type Totales struct {
	Agentes  int     `json:"agentes"`
	Claude   int     `json:"claude"`
	Codex    int     `json:"codex"`
	Minutos  float64 `json:"minutos"`
	Salida   int64   `json:"salida"`
	Nuevos   int64   `json:"nuevos"`
	CachePct float64 `json:"cache_pct"` // de la entrada, cuánto vino de la caché
	Opus     int     `json:"opus"`      // agentes de Claude en Opus
	OpusPct  float64 `json:"opus_pct"`
	MetaOpus float64 `json:"meta_opus"` // 0 si §12 no la escribe
	// FuenteOpus: «AGENTES.md:208».
	FuenteOpus string `json:"fuente_opus,omitempty"`
}

// Informe de estadísticas.
type Informe struct {
	Totales Totales  `json:"totales"`
	Clases  []Clase  `json:"clases"`
	Modelos []string `json:"modelos"`
	Celdas  []Celda  `json:"celdas"`
	Matriz  string   `json:"matriz,omitempty"` // la ruta del documento de la matriz
}

// SinClase: el grupo de los agentes cuyo encargo no dice su clase.
const SinClase = "—"

var ordenFamilias = map[string]int{"haiku": 0, "sonnet": 1, "opus": 2, "fable": 3}

// Calcular arma el informe. `tope` (minutos) y su fuente son el presupuesto de las clases sin presupuesto escrito.
func Calcular(m Matriz, tope float64, topeFuente string, ags []Agente) Informe {
	inf := Informe{Matriz: m.Ruta, Clases: []Clase{}, Modelos: []string{}, Celdas: []Celda{}}
	t := &inf.Totales
	t.MetaOpus, t.FuenteOpus = m.MetaOpus, m.FuenteOpus
	var leida, entrada int64
	type clave struct{ clase, prov string }
	grupos := map[clave][]Agente{}
	celdas := map[[2]string][]Agente{}
	modelos := map[string]bool{}
	for _, a := range ags {
		if a.Clase == "" {
			a.Clase = SinClase
		}
		t.Agentes++
		if a.Proveedor == "Codex" {
			t.Codex++
		} else {
			t.Claude++
			if a.Modelo == "opus" {
				t.Opus++
			}
		}
		t.Minutos += a.Minutos
		t.Salida += a.Salida
		t.Nuevos += a.Nuevos
		leida += a.Leida
		entrada += a.Leida + a.Nuevos
		grupos[clave{a.Clase, a.Proveedor}] = append(grupos[clave{a.Clase, a.Proveedor}], a)
		modelo := a.Modelo
		if modelo == "" {
			modelo = "?"
		}
		modelos[modelo] = true
		celdas[[2]string{a.Clase, modelo}] = append(celdas[[2]string{a.Clase, modelo}], a)
	}
	if entrada > 0 {
		t.CachePct = float64(leida) / float64(entrada) * 100
	}
	if t.Claude > 0 {
		t.OpusPct = float64(t.Opus) / float64(t.Claude) * 100
	}

	posicion := map[string]int{}
	for i, c := range m.Orden {
		posicion[c] = i
	}
	pos := func(c string) int {
		if p, ok := posicion[c]; ok {
			return p
		}
		if c == SinClase {
			return 1 << 20
		}
		return 1 << 19
	}
	for k, g := range grupos {
		c := Clase{Clase: k.clase, Nombre: m.NombreDe(k.clase), Proveedor: k.prov, N: len(g), Agentes: []AgenteClase{}}
		sort.SliceStable(g, func(i, j int) bool { return g[i].Fin.After(g[j].Fin) })
		for i, a := range g {
			if i == 40 {
				break
			}
			modelo := a.Modelo
			if a.Effort != "" {
				modelo += "/" + a.Effort
			}
			c.Agentes = append(c.Agentes, AgenteClase{ID: a.ID, Nombre: a.Nombre, Proveedor: a.Proveedor,
				Modelo: modelo, Minutos: a.Minutos, Presupuesto: a.Presupuesto, Fin: a.Fin})
		}
		if k.clase == SinClase {
			c.Nombre = "sin clase en el encargo"
		}
		c.Minutos = medir(g, func(a Agente) float64 { return a.Minutos })
		c.Salida = medir(g, func(a Agente) float64 { return float64(a.Salida) })
		c.Nuevos = medir(g, func(a Agente) float64 { return float64(a.Nuevos) })
		var pres []float64
		for _, a := range g {
			if a.Presupuesto > 0 {
				pres = append(pres, a.Presupuesto)
			}
		}
		if len(pres) > 0 {
			c.Presupuesto, c.PresupuestoFuente = percentil(pres, 0.5), "las asignaciones de la clase"
		} else {
			c.Presupuesto, c.PresupuestoFuente = tope, topeFuente
		}
		conf, ok := configDe(m, k.clase, k.prov)
		if ok {
			c.Matriz, c.MatrizFuente = conf.Texto(), conf.Fuente
			for _, a := range g {
				if coincide(a, conf) {
					c.EnMatriz++
				}
			}
		}
		inf.Clases = append(inf.Clases, c)
	}
	sort.Slice(inf.Clases, func(i, j int) bool {
		a, b := inf.Clases[i], inf.Clases[j]
		if pos(a.Clase) != pos(b.Clase) {
			return pos(a.Clase) < pos(b.Clase)
		}
		if a.Clase != b.Clase {
			return a.Clase < b.Clase
		}
		return a.Proveedor < b.Proveedor // Claude antes que Codex
	})

	for mo := range modelos {
		inf.Modelos = append(inf.Modelos, mo)
	}
	sort.Slice(inf.Modelos, func(i, j int) bool {
		a, b := inf.Modelos[i], inf.Modelos[j]
		pa, oka := ordenFamilias[a]
		pb, okb := ordenFamilias[b]
		switch {
		case oka && okb:
			return pa < pb
		case oka != okb:
			return oka // las familias de Claude primero
		}
		return a < b
	})
	for k, g := range celdas {
		c := Celda{Clase: k[0], Modelo: k[1], N: len(g), Minutos: percentil(valores(g, func(a Agente) float64 { return a.Minutos }), 0.5)}
		if conf, ok := configDe(m, k[0], g[0].Proveedor); ok {
			c.Matriz = conf.Texto()
			c.EnMatriz = strings.EqualFold(conf.Modelo, k[1])
		}
		inf.Celdas = append(inf.Celdas, c)
	}
	sort.Slice(inf.Celdas, func(i, j int) bool {
		a, b := inf.Celdas[i], inf.Celdas[j]
		if a.Clase != b.Clase {
			return pos(a.Clase) < pos(b.Clase) || (pos(a.Clase) == pos(b.Clase) && a.Clase < b.Clase)
		}
		return a.Modelo < b.Modelo
	})
	return inf
}

// configDe: lo que manda la matriz para la clase con ese proveedor.
func configDe(m Matriz, clase, prov string) (Config, bool) {
	if prov == "Codex" {
		c, ok := m.Codex[clase]
		return c, ok
	}
	c, ok := m.Claude[clase]
	return c, ok
}

func coincide(a Agente, c Config) bool {
	return strings.EqualFold(a.Modelo, c.Modelo) && strings.EqualFold(a.Effort, c.Effort)
}

func valores(g []Agente, f func(Agente) float64) []float64 {
	out := make([]float64, len(g))
	for i, a := range g {
		out[i] = f(a)
	}
	return out
}

func medir(g []Agente, f func(Agente) float64) Medida {
	v := valores(g, f)
	return Medida{Mediana: percentil(v, 0.5), P90: percentil(v, 0.9)}
}

// percentil por rango más cercano: el valor que deja la fracción p de los datos a su izquierda. La mediana de un
// número par de datos es el promedio de los dos del medio.
func percentil(v []float64, p float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	if p == 0.5 && len(s)%2 == 0 {
		return (s[len(s)/2-1] + s[len(s)/2]) / 2
	}
	i := int(math.Ceil(p*float64(len(s)))) - 1
	if i < 0 {
		i = 0
	}
	return s[i]
}
