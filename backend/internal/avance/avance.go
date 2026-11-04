// Package avance lee el avance de cada fase de un proyecto: docs/trazabilidad/F<n>/AVANCE.md, la bitácora del
// coordinador. Tiene la misma forma que la bitácora de un agente («Avance: NN % — qué pasa
// ahora» y los hitos «- HH:MM · texto»), más una tabla de paquetes con sus horas, su estado y su %:
//
//	Avance: 81 % — W6b: estrés de H9
//	Plan: docs/planes/fase-5.md §4
//
//	| Paquete | Qué | Horas | Estado | % | Evidencia |
//	| W1 | Versión y ensayador | 3 | hecho | 100 | F5/02-ensayo-0.4.0.md |
//
// Atalaya calcula el % de la fase pesando cada paquete por sus horas, y avisa si no coincide con el declarado:
// la cuenta es repetible y no depende de quien la sume. Solo lee.
package avance

import (
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"atalaya/internal/secretos"
	"atalaya/internal/trazabilidad"
)

// Estados de un paquete.
const (
	Hecho      = "hecho"
	EnCurso    = "en curso"
	SinEmpezar = "sin empezar"
	Bloqueado  = "bloqueado"
)

// Paquete de la fase.
type Paquete struct {
	ID        string  `json:"id"`
	Que       string  `json:"que"`
	Horas     float64 `json:"horas"`
	Estado    string  `json:"estado"`         // uno de los cuatro, siempre
	Nota      string  `json:"nota,omitempty"` // lo que la celda Estado dice además del estado, tal como está escrito
	Pct       int     `json:"pct"`
	Evidencia string  `json:"evidencia,omitempty"`
}

// Fase leída de su AVANCE.md.
type Fase struct {
	Fase  string `json:"fase"` // «F5»
	Ruta  string `json:"ruta"` // relativa a la raíz del proyecto
	Plan  string `json:"plan,omitempty"`
	Ahora string `json:"ahora"` // el texto de la primera línea
	// Declarado: el % de la primera línea (-1 si no lo dice); Calculado: Σ(horas × %) ÷ Σ horas, con un decimal.
	Declarado int                 `json:"declarado"`
	Calculado float64             `json:"calculado"`
	HorasTot  float64             `json:"horas"`
	HorasHech float64             `json:"horas_hechas"`
	Paquetes  []Paquete           `json:"paquetes"`
	Hitos     []trazabilidad.Hito `json:"hitos"`
	// Avisos: lo que la tabla tiene mal (un % fuera de 0–100, un estado que no cuadra con su %, el declarado distinto).
	Avisos []string `json:"avisos"`
}

var (
	rePlan   = regexp.MustCompile(`^Plan:\s*(.+)$`)
	reFase   = regexp.MustCompile(`^F(\d+)$`)
	reNum    = regexp.MustCompile(`-?\d+(?:[.,]\d+)?`)
	reSep    = regexp.MustCompile(`^\|?\s*:?-{2,}`)
	sinTilde = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u")
)

// columna: el nombre de una cabecera, normalizado («Est. (h)» → «horas»).
func columna(c string) string {
	c = sinTilde.Replace(strings.ToLower(strings.Trim(strings.TrimSpace(c), "*")))
	switch {
	case c == "paquete" || c == "package" || c == "id":
		return "id"
	case c == "que" || c == "what" || c == "descripcion":
		return "que"
	case c == "estado" || c == "status": // antes que «est», que es el comienzo de «Est. (h)»
		return "estado"
	case strings.HasPrefix(c, "hora") || c == "hours" || strings.HasPrefix(c, "est"):
		return "horas"
	case c == "%" || c == "avance" || c == "pct":
		return "pct"
	case c == "evidencia" || c == "evidence" || c == "prueba":
		return "evidencia"
	}
	return ""
}

func numero(s string) (float64, bool) {
	m := reNum.FindString(s)
	if m == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(strings.Replace(m, ",", ".", 1), 64)
	return v, err == nil
}

// limpiarEstado: la celda en minúsculas, sin tildes ni negritas, para compararla.
func limpiarEstado(s string) string {
	return sinTilde.Replace(strings.ToLower(strings.TrimSpace(strings.Trim(s, "*"))))
}

// normalizarEstado lleva las formas comunes a los cuatro estados; "" si la celda no dice ninguno.
func normalizarEstado(s string) string {
	s = limpiarEstado(s)
	switch {
	case strings.HasPrefix(s, "hech") || strings.HasPrefix(s, "done") || strings.HasPrefix(s, "complete") || s == "cerrado" || s == "listo":
		return Hecho
	case strings.HasPrefix(s, "en curso") || strings.HasPrefix(s, "curso") || strings.HasPrefix(s, "en marcha") || strings.HasPrefix(s, "in progress") || strings.HasPrefix(s, "ongoing"):
		return EnCurso
	case strings.HasPrefix(s, "bloq") || strings.HasPrefix(s, "blocked"):
		return Bloqueado
	case strings.HasPrefix(s, "sin empezar") || strings.HasPrefix(s, "pendiente") || strings.HasPrefix(s, "not started") || strings.HasPrefix(s, "pending") || s == "todo" || s == "to do" || s == "":
		return SinEmpezar
	}
	return ""
}

// soloEstado: las celdas que dicen el estado y nada más. Lo demás («hecho, sin commit», «G3a cerrado…») se guarda como nota.
var soloEstado = map[string]bool{
	"": true, "hecho": true, "hecha": true, "done": true, "complete": true, "completed": true, "cerrado": true, "listo": true,
	"en curso": true, "curso": true, "en marcha": true, "in progress": true, "ongoing": true,
	"bloqueado": true, "bloqueada": true, "blocked": true,
	"sin empezar": true, "pendiente": true, "not started": true, "pending": true, "todo": true, "to do": true,
}

// leerEstado separa la celda Estado en el estado y su nota. Una frase que no dice ninguno de los cuatro deja el estado
// en "": lo decide calcular con el % de la fila.
func leerEstado(celda string) (estado, nota string) {
	estado = normalizarEstado(celda)
	if !soloEstado[limpiarEstado(celda)] {
		nota = plano(strings.TrimSpace(strings.Trim(celda, "*")))
	}
	return estado, nota
}

// estadoPorPct: el estado de un paquete que no lo dice en palabras.
func estadoPorPct(pct int) string {
	switch {
	case pct >= 100:
		return Hecho
	case pct <= 0:
		return SinEmpezar
	}
	return EnCurso
}

// plano quita las comillas de código de la Markdown: el panel muestra el texto, no su marcado.
func plano(s string) string {
	return strings.ReplaceAll(s, "`", "")
}

// celdas de una fila de tabla Markdown; «\|» es una barra dentro de la celda, no el fin de la celda.
func celdas(l string) []string {
	const barra = "\x00"
	l = strings.ReplaceAll(strings.TrimSpace(l), `\|`, barra)
	l = strings.TrimPrefix(l, "|")
	l = strings.TrimSuffix(l, "|")
	out := strings.Split(l, "|")
	for i := range out {
		out[i] = strings.ReplaceAll(strings.TrimSpace(out[i]), barra, "|")
	}
	return out
}

// Parsear lee el texto de un AVANCE.md.
func Parsear(texto, fase, ruta string) Fase {
	b := trazabilidad.ParsearAvance(texto, ruta)
	f := Fase{Fase: fase, Ruta: ruta, Ahora: b.Texto, Declarado: b.Porcentaje, Hitos: b.Hitos, Avisos: []string{}}
	for _, l := range strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n") {
		if m := rePlan.FindStringSubmatch(strings.TrimSpace(l)); m != nil {
			f.Plan = strings.TrimSpace(m[1])
			break
		}
	}
	f.Paquetes = paquetes(texto, false)
	f.calcular()
	return f
}

// paquetes: las filas de las tablas con columna «Paquete» (o «ID»); con soloPrimera, las de la primera de ellas. Una
// fila con el nombre y nada más no es un paquete sino un marcador, como «| (W1 a W12, al aprobarse el plan) | | |»:
// se salta, para que no se pinte como un paquete de nombre largo, sin horas ni estado.
func paquetes(texto string, soloPrimera bool) []Paquete {
	out := []Paquete{}
	var cols []string
	for _, l := range strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n") {
		t := strings.TrimSpace(l)
		if !strings.HasPrefix(t, "|") {
			if len(cols) > 0 && soloPrimera {
				break
			}
			cols = nil // una tabla termina con la primera línea que no es fila
			continue
		}
		if reSep.MatchString(t) {
			continue
		}
		cs := celdas(t)
		if cols == nil {
			cols = make([]string, len(cs))
			hayID := false
			for i, c := range cs {
				cols[i] = columna(c)
				hayID = hayID || cols[i] == "id"
			}
			if !hayID {
				cols = []string{} // no es la tabla de paquetes: sus filas se saltan
			}
			continue
		}
		if len(cols) == 0 {
			continue
		}
		// Una barra sin escapar en la última columna abre celdas de más: siguen siendo esa columna, no se pierden.
		if ultima := len(cols) - 1; len(cs) > len(cols) {
			cs = append(cs[:ultima:ultima], strings.Join(cs[ultima:], " | "))
		}
		p := Paquete{Pct: -1}
		soloNombre := true
		for i, c := range cs {
			if i >= len(cols) {
				break
			}
			if cols[i] != "id" && c != "" {
				soloNombre = false
			}
			switch cols[i] {
			case "id":
				p.ID = strings.Trim(c, "*` ")
			case "que":
				p.Que = plano(c)
			case "horas":
				p.Horas, _ = numero(c)
			case "estado":
				p.Estado, p.Nota = leerEstado(c)
			case "pct":
				if v, ok := numero(c); ok {
					p.Pct = int(math.Round(v))
				}
			case "evidencia":
				p.Evidencia = plano(c)
			}
		}
		if p.ID == "" || soloNombre || strings.EqualFold(p.ID, "total") {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (f *Fase) calcular() {
	var suma float64
	for i := range f.Paquetes {
		p := &f.Paquetes[i]
		if p.Pct < 0 {
			switch p.Estado {
			case Hecho:
				p.Pct = 100
			case SinEmpezar:
				p.Pct = 0
			default:
				f.Avisos = append(f.Avisos, p.ID+" no dice su %: cuenta como 0")
				p.Pct = 0
			}
		}
		if p.Pct < 0 || p.Pct > 100 {
			f.Avisos = append(f.Avisos, p.ID+" tiene "+strconv.Itoa(p.Pct)+" %: fuera de 0 a 100")
			p.Pct = int(math.Max(0, math.Min(100, float64(p.Pct))))
		}
		if p.Estado == "" {
			p.Estado = estadoPorPct(p.Pct)
		}
		switch {
		case p.Estado == Hecho && p.Pct != 100:
			f.Avisos = append(f.Avisos, p.ID+" está hecho pero dice "+strconv.Itoa(p.Pct)+" %")
		case p.Estado == SinEmpezar && p.Pct != 0:
			f.Avisos = append(f.Avisos, p.ID+" está sin empezar pero dice "+strconv.Itoa(p.Pct)+" %")
		}
		if p.Horas <= 0 {
			f.Avisos = append(f.Avisos, p.ID+" no dice sus horas: no pesa en el %")
			continue
		}
		f.HorasTot += p.Horas
		suma += p.Horas * float64(p.Pct) / 100
		if p.Estado == Hecho {
			f.HorasHech += p.Horas
		}
	}
	if f.HorasTot > 0 {
		f.Calculado = math.Round(suma/f.HorasTot*1000) / 10
	}
	if f.Declarado >= 0 && len(f.Paquetes) > 0 && math.Abs(float64(f.Declarado)-f.Calculado) >= 1 {
		f.Avisos = append(f.Avisos, "la primera línea dice "+strconv.Itoa(f.Declarado)+" % y los paquetes dan "+
			strings.Replace(strconv.FormatFloat(f.Calculado, 'f', 1, 64), ".", ",", 1)+" %")
	}
}

// VidaMemoria: cuánto vale lo leído de un proyecto. Un pedido del panel pregunta la fase de cada agente (cientos, del
// mismo proyecto); con esto comparten una sola lectura. Es lo que tarda en dar una vuelta Atalaya, así que lo que
// cambió en el disco se ve igual de pronto.
const VidaMemoria = 2 * time.Second

var memoria = struct {
	sync.Mutex
	m map[string]leidas
}{m: map[string]leidas{}}

type leidas struct {
	cuando time.Time
	fases  []Fase
}

// Leer todas las fases con AVANCE.md de un proyecto, de la más alta a la más baja. Lo que devuelve se comparte entre
// quienes lo piden en el mismo par de segundos: es de solo lectura.
func Leer(raiz string) []Fase {
	if raiz == "" {
		return nil
	}
	ahora := time.Now()
	memoria.Lock()
	if l, ok := memoria.m[raiz]; ok && ahora.Sub(l.cuando) < VidaMemoria {
		memoria.Unlock()
		return l.fases
	}
	memoria.Unlock()
	fases := leer(raiz)
	memoria.Lock()
	if len(memoria.m) > 256 {
		clear(memoria.m)
	}
	memoria.m[raiz] = leidas{cuando: ahora, fases: fases}
	memoria.Unlock()
	return fases
}

// OlvidarMemoria: la próxima lectura va al disco (para quien acaba de cambiar un AVANCE.md y lo quiere ver ya).
func OlvidarMemoria() {
	memoria.Lock()
	clear(memoria.m)
	memoria.Unlock()
}

func leer(raiz string) []Fase {
	rutas, _ := filepath.Glob(filepath.Join(raiz, "docs", "trazabilidad", "F*", "AVANCE.md"))
	var out []Fase
	for _, r := range rutas {
		fase := filepath.Base(filepath.Dir(r))
		if !reFase.MatchString(fase) {
			continue
		}
		// El texto, el plan y los hitos llegan al panel y al MCP: un token pegado por error no sale de aquí. LeerTapado
		// guarda lo leído mientras el archivo no cambie (cada pedido del panel lee las fases de cada proyecto).
		texto, err := secretos.LeerTapado(r)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(raiz, r)
		f := Parsear(texto, fase, filepath.ToSlash(rel))
		f.paquetesDelPlan(raiz)
		if info, err := os.Stat(r); err == nil {
			f.horaDelUltimoHito(info.ModTime())
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return numFase(out[i].Fase) > numFase(out[j].Fase) })
	return out
}

// paquetesDelPlan: una fase recién abierta todavía no tiene su tabla en AVANCE.md, pero su plan (la línea «Plan:»)
// sí trae la de paquetes con sus horas. Entonces se leen de ahí, todos sin empezar, y un aviso dice de dónde salen.
// El plan tiene que estar dentro del proyecto.
func (f *Fase) paquetesDelPlan(raiz string) {
	if len(f.Paquetes) > 0 || f.Plan == "" {
		return
	}
	campos := strings.Fields(strings.Trim(f.Plan, "`"))
	if len(campos) == 0 {
		return
	}
	rel := strings.Trim(campos[0], "`")
	ruta := filepath.Join(raiz, filepath.FromSlash(rel))
	if dentro, err := filepath.Rel(raiz, ruta); err != nil || filepath.IsAbs(rel) || dentro == ".." || strings.HasPrefix(dentro, ".."+string(filepath.Separator)) {
		return
	}
	texto, err := secretos.LeerTapado(ruta)
	if err != nil {
		return
	}
	ps := paquetes(texto, true)
	if len(ps) == 0 {
		return
	}
	for i := range ps {
		ps[i].Estado, ps[i].Nota = SinEmpezar, ""
		ps[i].Pct = -1
	}
	f.Paquetes = ps
	f.Avisos = []string{}
	f.HorasTot, f.HorasHech, f.Calculado = 0, 0, 0
	f.calcular()
	f.Avisos = append([]string{"los paquetes salen del plan (" + rel + "): AVANCE.md todavía no tiene su tabla"}, f.Avisos...)
}

// horaDelUltimoHito: el último hito (van del más nuevo al más viejo) se escribe al guardar el archivo, así que su hora no
// puede ser posterior a la de guardado. Si lo es, la hora se inventó al escribirlo y la bitácora miente sobre cuándo
// pasó cada cosa: se avisa. Se tolera un minuto de redondeo, y una diferencia de más de 12 h se toma como un hito de
// la víspera (los hitos solo dicen HH:MM).
func (f *Fase) horaDelUltimoHito(guardado time.Time) {
	if len(f.Hitos) == 0 {
		return
	}
	h, err := time.ParseInLocation("15:04", f.Hitos[0].Hora, guardado.Location())
	if err != nil {
		return
	}
	hito := time.Date(guardado.Year(), guardado.Month(), guardado.Day(), h.Hour(), h.Minute(), 0, 0, guardado.Location())
	adelanto := hito.Sub(guardado.Truncate(time.Minute))
	if adelanto > time.Minute && adelanto <= 12*time.Hour {
		f.Avisos = append(f.Avisos, "el último hito dice "+f.Hitos[0].Hora+" y el archivo se guardó a las "+
			guardado.Format("15:04")+": la hora del hito es posterior y no puede ser la real")
	}
}

func numFase(f string) int {
	n, _ := strconv.Atoi(strings.TrimPrefix(f, "F"))
	return n
}

// Actual: la fase más alta con avance, o nil.
func Actual(raiz string) *Fase {
	fs := Leer(raiz)
	if len(fs) == 0 {
		return nil
	}
	return &fs[0]
}

// EnCursoAhora: el primer paquete en curso (el que se muestra como «ahora»), o nil.
func (f Fase) EnCursoAhora() *Paquete {
	for i := range f.Paquetes {
		if f.Paquetes[i].Estado == EnCurso {
			return &f.Paquetes[i]
		}
	}
	return nil
}
