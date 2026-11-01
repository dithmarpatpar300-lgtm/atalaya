// Package alertas vigila las reglas del centro de operaciones de un proyecto sobre sus agentes vivos: el tiempo de
// cada agente, el contexto, la falta de actividad y la frontera de archivos entre agentes en paralelo.
//
// Las reglas se leen de los documentos del proyecto (AGENTS.md), no se copian aquí: si la regla cambia, la alerta
// la sigue. Lo que el documento no fija va con un valor por omisión, y la regla lo dice (Omision).
package alertas

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"atalaya/internal/claude"
)

// Niveles, del más grave al más leve.
const (
	Error = "error"
	Aviso = "aviso"
	Info  = "info"
)

// Regla vigilada, con de dónde salió su umbral.
type Regla struct {
	ID     string `json:"id"`
	Nombre string `json:"nombre"`
	Umbral string `json:"umbral"`
	Nota   string `json:"nota"`
	// Fuente: «AGENTS.md:25», o vacía si el umbral es el valor por omisión.
	Fuente  string `json:"fuente,omitempty"`
	Omision bool   `json:"omision"`
}

// Reglas de un proyecto, ya en números.
type Reglas struct {
	Tope     time.Duration // lo más que dura un agente
	Margen   time.Duration // cuánto antes del tope se avisa
	Contexto float64       // fracción de la ventana
	Quieto   time.Duration // sin pasos nuevos mientras trabaja
	Disipa   time.Duration // cuánto queda a la vista un cruce de archivos ya cerrado
	Lista    []Regla
	fuentes  map[string]string

	// Las reglas del proceso (la salud del proceso, internal/proceso), leídas del mismo AGENTS.md.
	Frescura time.Duration // lo más vieja que puede estar la lectura de la cuota al lanzar
	TmpRaiz  string        // «C:\Mi-Proyecto-tmp»: dónde van los temporales de los agentes
	Retomar  string        // el script que lista los temporales que quedan («scripts/ops/retomar.ps1»)
}

// Fuente de una regla («AGENTS.md:31»), o vacía si el documento no la escribe.
func (r Reglas) Fuente(id string) string { return r.fuentes[id] }

// Por omisión: valores predeterminados para las reglas.
const (
	topeOmision     = 30 * time.Minute
	margenOmision   = 5 * time.Minute
	contextoOmision = 0.80
	quietoOmision   = 5 * time.Minute
	disipaOmision   = 30 * time.Minute
	frescuraOmision = 10 * time.Minute
)

var (
	reTope     = regexp.MustCompile(`(?i)ning[uú]n agente pasa de (\d+) minutos`)
	reFrontera = regexp.MustCompile(`(?i)nunca hay dos sobre el mismo archivo`)
	reMinutos  = regexp.MustCompile(`(?i)(\d+(?:[.,]\d+)?)\s*(min|h)`)

	// Las líneas de AGENTS.md que escriben las reglas del proceso: con encontrarlas basta para citar la fuente.
	reglasProceso = []struct {
		id string
		re *regexp.Regexp
	}{
		{"asignacion", regexp.MustCompile(`(?i)primera l[ií]nea es la asignaci[oó]n`)},
		{"effort", regexp.MustCompile(`(?i)effort se fija siempre expl[ií]cito`)},
		{"cuota", regexp.MustCompile(`(?i)se mira la cuota`)},
		{"bitacora", regexp.MustCompile(`(?i)cada agente lleva una bit[aá]cora`)},
		{"acta", regexp.MustCompile(`(?i)el acta es obligatoria`)},
	}
	reFrescura = regexp.MustCompile(`(?i)dato de Claude tiene m[aá]s de (\d+) min`)
	reTmpRaiz  = regexp.MustCompile("(?i)temporales en `([A-Za-z]:\\\\[^\\\\`<]+)")
	reRetomar  = regexp.MustCompile("`([\\w./-]+\\.ps1)` los lista")
)

// Leer arma las reglas desde el AGENTS.md de la raíz del proyecto; sin raíz o sin archivo, todo va por omisión.
func Leer(raiz string) Reglas {
	r := Reglas{Tope: topeOmision, Margen: margenOmision, Contexto: contextoOmision, Quieto: quietoOmision, Disipa: disipaOmision, Frescura: frescuraOmision,
		fuentes: map[string]string{}}
	if raiz != "" {
		if f, err := os.Open(filepath.Join(raiz, "AGENTS.md")); err == nil {
			sc := bufio.NewScanner(f)
			sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for n := 1; sc.Scan(); n++ {
				t := sc.Text()
				if m := reTope.FindStringSubmatch(t); m != nil && r.fuentes["tiempo"] == "" {
					if v, err := strconv.Atoi(m[1]); err == nil && v > 0 {
						r.Tope = time.Duration(v) * time.Minute
						r.fuentes["tiempo"] = "AGENTS.md:" + strconv.Itoa(n)
					}
				}
				if reFrontera.MatchString(t) && r.fuentes["frontera"] == "" {
					r.fuentes["frontera"] = "AGENTS.md:" + strconv.Itoa(n)
				}
				for _, rp := range reglasProceso {
					if r.fuentes[rp.id] == "" && rp.re.MatchString(t) {
						r.fuentes[rp.id] = "AGENTS.md:" + strconv.Itoa(n)
					}
				}
				if m := reFrescura.FindStringSubmatch(t); m != nil && r.fuentes["frescura"] == "" {
					if v, err := strconv.Atoi(m[1]); err == nil && v > 0 {
						r.Frescura = time.Duration(v) * time.Minute
						r.fuentes["frescura"] = "AGENTS.md:" + strconv.Itoa(n)
					}
				}
				if m := reTmpRaiz.FindStringSubmatch(t); m != nil && r.TmpRaiz == "" {
					r.TmpRaiz = m[1]
					r.fuentes["temporales"] = "AGENTS.md:" + strconv.Itoa(n)
				}
				if m := reRetomar.FindStringSubmatch(t); m != nil && r.Retomar == "" {
					r.Retomar = m[1]
				}
			}
			f.Close()
		}
	}
	r.Lista = []Regla{
		{ID: "tiempo", Nombre: "Tiempo del agente", Umbral: fmt.Sprintf("%d min", int(r.Tope.Minutes())),
			Nota:   fmt.Sprintf("Aviso a los %d min y alerta al cumplirse. El presupuesto del encargo manda si lo trae.", int((r.Tope - r.Margen).Minutes())),
			Fuente: r.fuentes["tiempo"], Omision: r.fuentes["tiempo"] == ""},
		{ID: "contexto", Nombre: "Contexto", Umbral: fmt.Sprintf("%d %%", int(r.Contexto*100)),
			Nota: "De la ventana que se infiere de las peticiones del agente.", Omision: true},
		{ID: "quieto", Nombre: "Sin actividad", Umbral: fmt.Sprintf("%d min", int(r.Quieto.Minutes())),
			Nota: "Trabajando, pero sin pasos nuevos: suele ser una orden larga que no termina.", Omision: true},
		{ID: "frontera", Nombre: "Frontera de archivos", Umbral: "siempre",
			Nota: fmt.Sprintf("Dos agentes vivos a la vez editan el mismo archivo: error entre dos en paralelo, aviso entre la sesión y su subagente. "+
				"Se cierra sola cuando uno de los dos deja de trabajar; el aviso de cierre queda %d min (por omisión).", int(r.Disipa.Minutes())),
			Fuente: r.fuentes["frontera"], Omision: r.fuentes["frontera"] == ""},
	}
	return r
}

// Lector guarda las reglas de cada raíz y las relee solo cuando AGENTS.md cambia.
type Lector struct {
	mu    sync.Mutex
	cache map[string]reglasLeidas
}

type reglasLeidas struct {
	firma  string
	reglas Reglas
}

// Reglas de la raíz, del caché si AGENTS.md no cambió.
func (l *Lector) Reglas(raiz string) Reglas {
	firma := ""
	if raiz != "" {
		if fi, err := os.Stat(filepath.Join(raiz, "AGENTS.md")); err == nil {
			firma = strconv.FormatInt(fi.Size(), 10) + "@" + fi.ModTime().String()
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cache == nil {
		l.cache = map[string]reglasLeidas{}
	}
	if c, ok := l.cache[raiz]; ok && c.firma == firma {
		return c.reglas
	}
	r := Leer(raiz)
	l.cache[raiz] = reglasLeidas{firma: firma, reglas: r}
	return r
}

// Agente: lo que la evaluación necesita de uno.
type Agente struct {
	ID       string
	Nombre   string // «Codificador I2», «Sesión»
	Proyecto string
	// Subagente o llamada a Codex: los únicos con presupuesto de tiempo (la sesión coordinadora no lo tiene).
	ConTope     bool
	Trabajando  bool // trabaja ahora
	Activo      bool // trabaja o espera a la persona
	Presupuesto string
	Inicio      time.Time
	Ultimo      time.Time
	Contexto    int64
	Ventana     int64
	Ediciones   []claude.Edicion
	// Padre: la sesión que lo lanzó (un subagente o una llamada a Codex); vacío en una sesión.
	Padre string
}

// Alerta sobre un agente (o dos, en la frontera).
type Alerta struct {
	// ID estable mientras la situación sea la misma: lo que la persona ya vio no vuelve a contarse como nuevo.
	ID       string    `json:"id"`
	Nivel    string    `json:"nivel"`
	Regla    string    `json:"regla"`
	Fuente   string    `json:"fuente,omitempty"`
	Proyecto string    `json:"proyecto"`
	Agente   string    `json:"agente"`
	Nombre   string    `json:"nombre"`
	Titulo   string    `json:"titulo"`
	Texto    string    `json:"texto"`
	Hora     time.Time `json:"hora"`
	// Para la barra: Valor de Max (minutos, tokens). Max 0: sin barra.
	Valor float64 `json:"valor"`
	Max   float64 `json:"max"`
	// Frontera: el archivo y el otro agente.
	Archivo string `json:"archivo,omitempty"`
	Otro    string `json:"otro,omitempty"`
	// Chip: el texto corto que va en la fila del agente.
	Chip string `json:"chip"`
}

var orden = map[string]int{Error: 0, Aviso: 1, Info: 2}

// Evaluar aplica las reglas de cada proyecto a sus agentes. `reglas` va por proyecto; uno que falte usa los valores
// por omisión.
func Evaluar(reglas map[string]Reglas, ags []Agente, ahora time.Time) []Alerta {
	var out []Alerta
	de := func(p string) Reglas {
		if r, ok := reglas[p]; ok {
			return r
		}
		return Leer("")
	}
	for _, a := range ags {
		r := de(a.Proyecto)
		out = append(out, tiempo(r, a, ahora)...)
		if a.Activo && a.Ventana > 0 && float64(a.Contexto) >= r.Contexto*float64(a.Ventana) {
			pct := float64(a.Contexto) / float64(a.Ventana) * 100
			out = append(out, Alerta{ID: "contexto:" + a.ID, Nivel: Aviso, Regla: "contexto", Proyecto: a.Proyecto, Agente: a.ID, Nombre: a.Nombre,
				Titulo: "Contexto", Hora: a.Ultimo, Valor: float64(a.Contexto), Max: float64(a.Ventana), Chip: fmt.Sprintf("contexto %.0f %%", pct),
				Texto: fmt.Sprintf("%s usa el %.0f %% de su ventana de %s. La próxima compactación resume y pierde detalle.", a.Nombre, pct, cortos(a.Ventana))})
		}
		if a.Trabajando && ahora.Sub(a.Ultimo) >= r.Quieto {
			min := int(ahora.Sub(a.Ultimo).Minutes())
			out = append(out, Alerta{ID: "quieto:" + a.ID, Nivel: Info, Regla: "quieto", Proyecto: a.Proyecto, Agente: a.ID, Nombre: a.Nombre,
				Titulo: "Sin actividad", Hora: a.Ultimo, Chip: fmt.Sprintf("quieto %d min", min),
				Texto: fmt.Sprintf("%s no da un paso hace %d min; su último paso sigue esperando resultado.", a.Nombre, min)})
		}
	}
	out = append(out, frontera(reglas, ags, ahora)...)
	sort.SliceStable(out, func(i, j int) bool {
		if orden[out[i].Nivel] != orden[out[j].Nivel] {
			return orden[out[i].Nivel] < orden[out[j].Nivel]
		}
		return out[i].Hora.After(out[j].Hora)
	})
	return out
}

// MinutosDe: el presupuesto de un encargo en minutos: «30 min», «45 minutos», «1 h», «1,5 h», «2 horas»; 0 si no
// se entiende.
func MinutosDe(presupuesto string) float64 {
	m := reMinutos.FindStringSubmatch(presupuesto)
	if m == nil {
		return 0
	}
	v, err := strconv.ParseFloat(strings.Replace(m[1], ",", ".", 1), 64)
	if err != nil || v <= 0 {
		return 0
	}
	if strings.HasPrefix(strings.ToLower(m[2]), "h") {
		v *= 60
	}
	return v
}

// TopeDe: lo más que puede durar un agente: el presupuesto de su encargo si lo trae, si no el tope del proyecto.
// Lo usan las alertas, las estadísticas y la línea de tiempo, para que las tres corten en el mismo minuto.
func TopeDe(r Reglas, presupuesto string) time.Duration {
	if m := MinutosDe(presupuesto); m > 0 {
		return time.Duration(m * float64(time.Minute))
	}
	return r.Tope
}

func tiempo(r Reglas, a Agente, ahora time.Time) []Alerta {
	if !a.ConTope || !a.Trabajando || a.Inicio.IsZero() {
		return nil
	}
	tope := TopeDe(r, a.Presupuesto)
	dur := ahora.Sub(a.Inicio)
	if dur < tope-r.Margen {
		return nil
	}
	min, tmin := int(dur.Minutes()), int(tope.Minutes())
	al := Alerta{Regla: "tiempo", Fuente: r.fuentes["tiempo"], Proyecto: a.Proyecto, Agente: a.ID, Nombre: a.Nombre, Titulo: "Presupuesto de tiempo",
		Hora: a.Inicio.Add(tope - r.Margen), Valor: dur.Minutes(), Max: tope.Minutes(), Chip: fmt.Sprintf("%d de %d min", min, tmin)}
	if dur >= tope {
		al.ID, al.Nivel, al.Hora = "tiempo:"+a.ID+":pasado", Error, a.Inicio.Add(tope)
		al.Texto = fmt.Sprintf("%s lleva %d min y su presupuesto era de %d. Toca avisarte y recortar el alcance.", a.Nombre, min, tmin)
	} else {
		al.ID, al.Nivel = "tiempo:"+a.ID+":aviso", Aviso
		al.Texto = fmt.Sprintf("%s lleva %d de %d min. Toca avisarte antes de que se cumpla.", a.Nombre, min, tmin)
	}
	return []Alerta{al}
}

// Cruce: dos agentes del mismo proyecto que editaron el mismo archivo mientras los dos estaban vivos. Una sesión
// que espera a su subagente no edita mientras él trabaja, así que el trabajo en serie (la sesión edita, delega, el
// subagente edita) no es un cruce.
type Cruce struct {
	A, B         Agente
	Ruta         string
	HoraA, HoraB time.Time // la última edición de cada uno dentro del rato en que los dos vivían
	// Coordinador: uno lanzó al otro (la sesión y su subagente). Él puede verlo y decidir; dos agentes en paralelo, no.
	Coordinador bool
	// Fin: cuándo terminó el riesgo, cuando el primero de los dos dejó de trabajar; cero si los dos siguen activos.
	Fin time.Time
}

// Cruces de los agentes, del mismo proyecto, en el orden de la lista.
func Cruces(ags []Agente) []Cruce {
	var out []Cruce
	for i := 0; i < len(ags); i++ {
		for j := i + 1; j < len(ags); j++ {
			a, b := ags[i], ags[j]
			if a.Proyecto != b.Proyecto || len(a.Ediciones) == 0 || len(b.Ediciones) == 0 {
				continue
			}
			desde, hasta := maxT(a.Inicio, b.Inicio), minT(a.Ultimo, b.Ultimo)
			if !desde.Before(hasta) {
				continue
			}
			ea, eb := enVentana(a.Ediciones, desde, hasta), enVentana(b.Ediciones, desde, hasta)
			claves := make([]string, 0, len(ea))
			for clave := range ea {
				if _, ok := eb[clave]; ok {
					claves = append(claves, clave)
				}
			}
			sort.Strings(claves)
			for _, clave := range claves {
				c := Cruce{A: a, B: b, Ruta: ea[clave].Ruta, HoraA: ea[clave].Hora, HoraB: eb[clave].Hora,
					Coordinador: (a.Padre != "" && a.Padre == b.ID) || (b.Padre != "" && b.Padre == a.ID)}
				switch {
				case a.Activo && b.Activo:
				case !a.Activo && !b.Activo:
					c.Fin = minT(a.Ultimo, b.Ultimo)
				case !a.Activo:
					c.Fin = a.Ultimo
				default:
					c.Fin = b.Ultimo
				}
				out = append(out, c)
			}
		}
	}
	return out
}

// Terminado: el de los dos que dejó de trabajar primero (el que cerró el cruce).
func (c Cruce) Terminado() Agente {
	if !c.A.Activo && (c.B.Activo || !c.A.Ultimo.After(c.B.Ultimo)) {
		return c.A
	}
	return c.B
}

// frontera: el cruce está abierto mientras los dos siguen activos: error entre dos agentes en paralelo, aviso entre la
// sesión y su subagente. Se cierra solo cuando uno de los dos deja de trabajar (el riesgo de pisarse terminó), y deja
// un aviso informativo, sin chip en la fila, durante `Disipa`, para revisar una vez el archivo.
func frontera(reglas map[string]Reglas, ags []Agente, ahora time.Time) []Alerta {
	var out []Alerta
	for _, c := range Cruces(ags) {
		a, b := c.A, c.B
		p, q := a, b
		if p.ID > q.ID {
			p, q = q, p
		}
		r := reglas[a.Proyecto]
		if r.fuentes == nil {
			r = Leer("")
		}
		quienes := fmt.Sprintf("%s (%s) y %s (%s) editaron %s mientras los dos trabajaban.", a.Nombre, c.HoraA.Local().Format("15:04"), b.Nombre,
			c.HoraB.Local().Format("15:04"), c.Ruta)
		al := Alerta{ID: "frontera:" + c.Ruta + ":" + p.ID + "+" + q.ID, Regla: "frontera", Fuente: r.fuentes["frontera"], Proyecto: a.Proyecto,
			Agente: a.ID, Nombre: a.Nombre, Otro: b.ID, Archivo: c.Ruta, Titulo: "Frontera de archivos", Hora: maxT(c.HoraA, c.HoraB)}
		switch {
		case c.Fin.IsZero():
			al.Nivel, al.Chip = Error, "mismo archivo"
			al.Texto = quienes + " Que lo lleve uno solo, y revisa que estén los cambios de los dos."
			if c.Coordinador {
				al.Nivel = Aviso
				al.Texto = quienes + " El coordinador no edita lo que está en la frontera de su subagente mientras trabaja: espera a que entregue."
			}
		case ahora.Sub(c.Fin) < r.Disipa:
			al.ID += ":cerrado"
			al.Nivel, al.Titulo, al.Hora = Info, "Cruce cerrado", c.Fin
			al.Texto = fmt.Sprintf("%s Se cerró a las %s, cuando %s dejó de trabajar: revisa una vez que el archivo tenga los cambios de los dos (git diff).",
				quienes, c.Fin.Local().Format("15:04"), c.Terminado().Nombre)
		default:
			continue
		}
		out = append(out, al)
	}
	return out
}

// enVentana: la última edición de cada archivo (por su clave) dentro de [desde, hasta].
func enVentana(es []claude.Edicion, desde, hasta time.Time) map[string]claude.Edicion {
	m := map[string]claude.Edicion{}
	for _, e := range es {
		if e.Clave == "" || e.Hora.Before(desde) || e.Hora.After(hasta) {
			continue
		}
		if e.Hora.After(m[e.Clave].Hora) {
			m[e.Clave] = e
		}
	}
	return m
}

func maxT(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func minT(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}

func cortos(n int64) string {
	if n >= 1_000_000 {
		return strings.Replace(strings.TrimSuffix(fmt.Sprintf("%.1f", float64(n)/1e6), ".0"), ".", ",", 1) + " M"
	}
	return fmt.Sprintf("%d k", n/1000)
}

// Firma de un conjunto de alertas: cambia cuando aparece o se va una, o cambia de nivel o de chip.
func Firma(as []Alerta) string {
	var b bytes.Buffer
	for _, a := range as {
		b.WriteString(a.ID)
		b.WriteByte('|')
		b.WriteString(a.Chip)
		b.WriteByte('\n')
	}
	return b.String()
}
