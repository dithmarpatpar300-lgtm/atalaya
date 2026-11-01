// Package trazabilidad lee lo que el centro de operaciones deja escrito en cada
// proyecto: las bitácoras de avance (docs/trazabilidad/F*/avance/*.md) y el
// registro de llamadas a Codex (docs/trazabilidad/F*/codex/REGISTRO.md).
package trazabilidad

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"atalaya/internal/secretos"
)

// Avance de una bitácora.
type Avance struct {
	Porcentaje int    `json:"porcentaje"` // -1 si la bitácora no lo dice
	Texto      string `json:"texto"`
	Hitos      []Hito `json:"hitos"`
	Ruta       string `json:"ruta"`
	// Desfase: la primera línea quedó atrás de su último hito con % (el % que vale es el del hito).
	Desfase string `json:"desfase,omitempty"`
}

type Hito struct {
	Hora  string `json:"hora"`
	Texto string `json:"texto"`
}

var (
	reAvance = regexp.MustCompile(`^(?:Avance|Progress):\s*(\d{1,3})\s*%\s*(?:[—–-]\s*(.*))?$`)
	reHito   = regexp.MustCompile(`^-\s*(\d{1,2}:\d{2})\s*·\s*(.+)$`)
)

// LeerAvance lee la bitácora `rel` (relativa a `raiz`).
func LeerAvance(raiz, rel string) (*Avance, error) {
	// El texto y los hitos van al panel: los secretos se tapan como en los transcritos (LeerTapado guarda lo leído
	// mientras el archivo no cambie: se lee para cada agente en cada pedido).
	ruta := filepath.Join(raiz, filepath.FromSlash(rel))
	clave := ruta + "\x00" + rel
	ahora := time.Now()
	parseadosMu.Lock()
	p, ok := parseados[clave]
	parseadosMu.Unlock()
	// Leída hace menos de VidaAvance: tal cual, sin mirar el archivo (el Stat de Windows, por agente y por pedido,
	// era lo que quedaba de esta lectura).
	if ok && ahora.Sub(p.visto) < VidaAvance {
		return copiaAvance(p.av), nil
	}
	texto, err := secretos.LeerTapado(ruta)
	if err != nil {
		return nil, err
	}
	// Y lo parseado se guarda junto al texto del que salió: mientras LeerTapado devuelva el mismo (el archivo no
	// cambió), no se vuelve a pasar cada línea por las expresiones. Comparar el texto entero es casi gratis cuando es
	// el mismo string guardado, y si el archivo cambió el texto es otro y se parsea de nuevo.
	if !ok || p.texto != texto {
		p = parseado{texto: texto, av: ParsearAvance(texto, rel)}
	}
	p.visto = ahora
	parseadosMu.Lock()
	if len(parseados) >= MaxParseados {
		clear(parseados)
	}
	parseados[clave] = p
	parseadosMu.Unlock()
	return copiaAvance(p.av), nil
}

// OlvidarAvance: la próxima lectura de cada bitácora vuelve a mirar el archivo (lo parseado se queda: solo se
// rehace si el texto cambió). La llama el servidor antes de avisar al panel de un cambio, para que lo que el panel
// pide con ese aviso no salga de la memoria corta.
func OlvidarAvance() {
	parseadosMu.Lock()
	for k, p := range parseados {
		p.visto = time.Time{}
		parseados[k] = p
	}
	parseadosMu.Unlock()
}

// copiaAvance: quien la recibe puede tocarla sin cambiar la guardada.
func copiaAvance(g *Avance) *Avance {
	av := *g
	av.Hitos = slices.Clone(g.Hitos)
	return &av
}

// MaxParseados: cuántas bitácoras parseadas se guardan (una por agente; pasado el tope se empieza de nuevo).
const MaxParseados = 512

// VidaAvance: cuánto vale una bitácora leída sin volver a mirar el archivo. Corta, como la de avance.Leer: el panel
// nunca muestra un avance más viejo que esto.
var VidaAvance = time.Second

type parseado struct {
	texto string
	av    *Avance
	visto time.Time // cuándo se miró el archivo por última vez
}

var (
	parseadosMu sync.Mutex
	parseados   = map[string]parseado{}
)

func ParsearAvance(texto, rel string) *Avance {
	av := &Avance{Porcentaje: -1, Ruta: rel, Hitos: []Hito{}}
	for _, l := range strings.Split(strings.ReplaceAll(texto, "\r\n", "\n"), "\n") {
		l = strings.TrimSpace(l)
		if m := reAvance.FindStringSubmatch(l); m != nil && av.Porcentaje < 0 {
			av.Porcentaje, _ = strconv.Atoi(m[1])
			av.Texto = strings.TrimSpace(m[2])
		} else if m := reHito.FindStringSubmatch(l); m != nil {
			av.Hitos = append(av.Hitos, Hito{Hora: m[1], Texto: m[2]})
		}
	}
	// Un agente que anota hitos con su % («- 11:07 · 75 % · …») y se olvida de la primera línea deja la cabecera
	// atrás: el último hito con % manda si dice más, y Desfase lo cuenta (para que se vea y se corrija).
	for i := len(av.Hitos) - 1; i >= 0; i-- {
		if m := reHitoPct.FindStringSubmatch(av.Hitos[i].Texto); m != nil {
			if p, _ := strconv.Atoi(m[1]); p > av.Porcentaje {
				if av.Porcentaje >= 0 {
					av.Desfase = fmt.Sprintf("la primera línea dice %d %%; el último hito (%s), %d %%", av.Porcentaje, av.Hitos[i].Hora, p)
				}
				av.Porcentaje = p
			}
			break
		}
	}
	return av
}

// reHitoPct: el % con que empieza el texto de un hito («75 % · corpus escrito…»).
var reHitoPct = regexp.MustCompile(`^(\d{1,3})\s*%`)

// LlamadaCodex: una fila de REGISTRO.md.
type LlamadaCodex struct {
	ID        string    `json:"id"`
	Proyecto  string    `json:"proyecto"`
	Fase      string    `json:"fase"`
	Fecha     string    `json:"fecha"`
	Tema      string    `json:"tema"`
	Clase     string    `json:"clase"`
	Modelo    string    `json:"modelo"`
	Effort    string    `json:"effort"`
	Version   string    `json:"version"`
	Base      string    `json:"base"`
	Segundos  int       `json:"segundos"`
	Salida    int       `json:"salida"`
	Entrada   int64     `json:"entrada"` // sin la parte en caché
	EnCache   int64     `json:"en_cache"`
	TokSalida int64     `json:"tok_salida"`
	Contraste string    `json:"contraste"`
	Informe   string    `json:"informe"` // relativa a la raíz del proyecto
	Terminada time.Time `json:"terminada"`
}

var (
	reSegundos = regexp.MustCompile(`^(\d+)\s*s$`)
	reSalida   = regexp.MustCompile(`^salida\s+(-?\d+)$`)
	reTokens   = regexp.MustCompile(`^(\d+)\s+entrada\s+\((\d+)\s+en\s+cache\)\s*/\s*(\d+)\s+salida$`)
)

// LeerCodex lee los registros de todas las fases del proyecto.
func LeerCodex(proyecto, raiz string) ([]LlamadaCodex, error) {
	regs, err := filepath.Glob(filepath.Join(raiz, "docs", "trazabilidad", "F*", "codex", "REGISTRO.md"))
	if err != nil {
		return nil, err
	}
	var out []LlamadaCodex
	for _, reg := range regs {
		fase := filepath.Base(filepath.Dir(filepath.Dir(reg)))
		f, err := os.Open(reg)
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			ll, ok := ParsearFilaCodex(sc.Text())
			if !ok {
				continue
			}
			ll.Proyecto, ll.Fase = proyecto, fase
			ll.ID = "codex~" + fase + "~" + ll.Fecha + "-" + ll.Tema
			rel := "docs/trazabilidad/" + fase + "/codex/" + ll.Fecha + "-" + ll.Tema + ".md"
			if fi, err := os.Stat(filepath.Join(raiz, filepath.FromSlash(rel))); err == nil {
				ll.Informe, ll.Terminada = rel, fi.ModTime()
			}
			out = append(out, ll)
		}
		f.Close()
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Terminada.After(out[j].Terminada) })
	return out, nil
}

// ParsearFilaCodex lee «- fecha · tema · clase · modelo/effort · versión · base · N s · salida X · tokens · contraste: …».
func ParsearFilaCodex(linea string) (LlamadaCodex, bool) {
	var ll LlamadaCodex
	if !strings.HasPrefix(linea, "- ") {
		return ll, false
	}
	partes := strings.Split(strings.TrimPrefix(linea, "- "), " · ")
	if len(partes) < 9 {
		return ll, false
	}
	ll.Fecha, ll.Tema, ll.Clase = strings.TrimSpace(partes[0]), strings.TrimSpace(partes[1]), strings.TrimSpace(partes[2])
	if _, err := time.Parse("2006-01-02", ll.Fecha); err != nil {
		return ll, false
	}
	if m, e, ok := strings.Cut(partes[3], "/"); ok {
		ll.Modelo, ll.Effort = strings.TrimSpace(m), strings.TrimSpace(e)
	} else {
		ll.Modelo = strings.TrimSpace(partes[3])
	}
	ll.Version = strings.TrimSpace(partes[4])
	// La base puede llevar « · » propio: todo lo que haya entre la versión y los segundos.
	i := 5
	var base []string
	for ; i < len(partes) && !reSegundos.MatchString(strings.TrimSpace(partes[i])); i++ {
		base = append(base, partes[i])
	}
	ll.Base = strings.Join(base, " · ")
	if i+3 > len(partes) {
		return ll, false
	}
	if m := reSegundos.FindStringSubmatch(strings.TrimSpace(partes[i])); m != nil {
		ll.Segundos, _ = strconv.Atoi(m[1])
	}
	if m := reSalida.FindStringSubmatch(strings.TrimSpace(partes[i+1])); m != nil {
		ll.Salida, _ = strconv.Atoi(m[1])
	}
	if m := reTokens.FindStringSubmatch(strings.TrimSpace(partes[i+2])); m != nil {
		total, _ := strconv.ParseInt(m[1], 10, 64)
		ll.EnCache, _ = strconv.ParseInt(m[2], 10, 64)
		ll.TokSalida, _ = strconv.ParseInt(m[3], 10, 64)
		ll.Entrada = total - ll.EnCache
	}
	resto := strings.Join(partes[i+3:], " · ")
	ll.Contraste = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(resto), "contraste:"))
	return ll, true
}

// LeerDoc lee un .md bajo `raiz` sin salirse de las carpetas permitidas.
func LeerDoc(raiz, rel string) (string, error) {
	limpia := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(limpia) || strings.HasPrefix(limpia, "..") || !strings.EqualFold(filepath.Ext(limpia), ".md") {
		return "", errors.New("ruta no permitida")
	}
	permitida := false
	baja := strings.ToLower(limpia)
	if strings.HasPrefix(baja, "docs"+string(filepath.Separator)) {
		permitida = true
	}
	for _, f := range []string{"agents.md", "claude.md", "readme.md"} {
		if baja == f {
			permitida = true
		}
	}
	if !permitida {
		return "", errors.New("ruta no permitida")
	}
	b, err := os.ReadFile(filepath.Join(raiz, limpia))
	if err != nil {
		return "", err
	}
	const max = 512 * 1024
	if len(b) > max {
		b = b[:max]
	}
	return secretos.Tapar(string(b)), nil
}
