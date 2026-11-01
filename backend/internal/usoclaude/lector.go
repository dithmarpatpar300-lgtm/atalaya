package usoclaude

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"
)

type Estado struct {
	Estado        string    `json:"estado"`
	Detalle       string    `json:"detalle"`
	TomadaEn      int64     `json:"tomada_en"`
	EdadS         int64     `json:"edad_s"`
	ProximaEn     int64     `json:"proxima_en"`
	LimitadoHasta *int64    `json:"limitado_hasta"`
	Ventanas      []Ventana `json:"ventanas"`
	ExtraActivo   bool      `json:"extra_activo"`
	Historial     []Punto   `json:"historial"`
	// Fuente: de dónde salió la última lectura: «usage» (claude -p /usage) o «app» (la empujó el mod de Claude Code).
	Fuente string `json:"fuente,omitempty"`
}

type Lector struct {
	Ahora                    func() time.Time
	Correr                   func(context.Context, string, string, []string) ([]byte, error)
	Exe                      string
	Demo                     bool
	Cambio                   func()
	mu                       sync.Mutex
	inicio, ultimo, limitado time.Time
	leyendo                  bool
	estado                   Estado
	filas                    []fila
	ruta                     string
}

// Resolver evita lanzar los intermediarios de npm como si fueran ejecutables.
func Resolver(ruta, sistema string) string {
	if sistema != "windows" || ruta == "" {
		return ruta
	}
	ext := strings.ToLower(filepath.Ext(ruta))
	if ext != ".cmd" && ext != ".ps1" && ext != "" {
		return ruta
	}
	if ext == "" {
		if _, e := os.Stat(ruta + ".cmd"); e != nil {
			if _, e = os.Stat(ruta + ".ps1"); e != nil {
				return ruta
			}
		}
	}
	return filepath.Join(filepath.Dir(ruta), "node_modules", "@anthropic-ai", "claude-code", "bin", "claude.exe")
}

func buscar() string {
	if runtime.GOOS != "linux" {
		r, _ := exec.LookPath("claude")
		r = Resolver(r, runtime.GOOS)
		if _, err := os.Stat(r); err != nil {
			return ""
		}
		return r
	}
	for _, d := range filepath.SplitList(os.Getenv("PATH")) {
		if d == "" || strings.HasPrefix(d, "/mnt/") && len(d) >= 6 && ((d[5] >= 'a' && d[5] <= 'z') || (d[5] >= 'A' && d[5] <= 'Z')) && (len(d) == 6 || d[6] == '/') {
			continue
		}
		if r, e := exec.LookPath(filepath.Join(d, "claude")); e == nil {
			return r
		}
	}
	return ""
}

func correr(ctx context.Context, exe, dir string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	cmd.Dir = dir
	ocultar(cmd)
	return cmd.Output()
}

func NuevoLector(datos string, demo bool) *Lector {
	l := &Lector{Ahora: time.Now, Correr: correr, Demo: demo}
	l.inicio = l.Ahora()
	if !demo {
		l.Exe = buscar()
	}
	if datos != "" && !demo {
		l.ruta = filepath.Join(datos, "uso-claude.jsonl")
		var err error
		l.filas, err = cargar(l.ruta)
		if err != nil {
			l.estado.Estado = "error"
			l.estado.Detalle = "No se pudo cargar el historial"
		} else if len(l.filas) > 0 {
			f := l.filas[0]
			for _, otra := range l.filas[1:] {
				if otra.T >= f.T {
					f = otra
				}
			}
			l.estado.Estado = "ok"
			l.estado.TomadaEn = f.T
			for clave, porcentaje := range f.Ventanas {
				etiqueta, minutos := clave, 10080
				switch {
				case clave == "session":
					etiqueta, minutos = "5 horas", 300
				case clave == "weekly_all":
					etiqueta = "Semana · todos los modelos"
				case strings.HasPrefix(clave, "weekly_scoped:"):
					etiqueta = "Semana · " + strings.TrimPrefix(clave, "weekly_scoped:")
				}
				l.estado.Ventanas = append(l.estado.Ventanas, Ventana{Clave: clave, Etiqueta: etiqueta, Porcentaje: porcentaje, ReiniciaEn: f.Reinicios[clave], DuracionMin: minutos})
			}
			sort.Slice(l.estado.Ventanas, func(i, j int) bool { return l.estado.Ventanas[i].Clave < l.estado.Ventanas[j].Clave })
		}
	}
	return l
}

func (l *Lector) proxima() time.Time {
	p := l.inicio.Add(time.Minute)
	if !l.ultimo.IsZero() {
		p = l.ultimo.Add(10 * time.Minute)
	}
	if l.limitado.After(p) {
		p = l.limitado
	}
	return p
}
func (l *Lector) Toca() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return !l.Demo && l.Exe != "" && !l.leyendo && !l.Ahora().Before(l.proxima())
}

func (l *Lector) ProximaForzada() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	p := l.ultimo.Add(2 * time.Minute)
	if l.limitado.After(p) {
		p = l.limitado
	}
	if p.Before(l.Ahora()) {
		p = l.Ahora()
	}
	return p.Unix()
}

func (l *Lector) Estado(horas int) Estado {
	l.mu.Lock()
	defer l.mu.Unlock()
	ahora := l.Ahora()
	e := l.estado
	e.Ventanas = append([]Ventana{}, e.Ventanas...)
	e.Historial = []Punto{}
	if horas < 1 || horas > 720 {
		horas = 168
	}
	for _, f := range l.filas {
		if f.T >= ahora.Add(-time.Duration(horas)*time.Hour).Unix() && f.T <= ahora.Unix() {
			v := map[string]float64{}
			for k, n := range f.Ventanas {
				v[k] = n
			}
			r := map[string]int64{}
			for k, n := range f.Reinicios {
				r[k] = n
			}
			e.Historial = append(e.Historial, Punto{T: f.T, V: v, R: r})
		}
	}
	if e.TomadaEn > 0 {
		e.EdadS = max(0, ahora.Unix()-e.TomadaEn)
		for i := range e.Ventanas {
			e.Ventanas[i].Vencida = e.Ventanas[i].ReiniciaEn <= ahora.Unix()
		}
		if e.Estado == "ok" && e.EdadS > 1200 {
			e.Estado = "viejo"
			e.Detalle = "La última lectura tiene más de 20 minutos"
		}
	}
	if l.Demo {
		e.Estado = "sin-claude"
		e.Detalle = "La lectura de Claude está desactivada en demo"
	} else if l.Exe == "" {
		e.Estado = "sin-claude"
		e.Detalle = "No se encontró Claude Code"
	} else if l.leyendo && e.TomadaEn == 0 {
		e.Estado = "leyendo"
		e.Detalle = "Leyendo el uso de Claude"
	} else if e.Estado == "" {
		e.Estado = "leyendo"
		e.Detalle = "Esperando la primera lectura"
	}
	if l.limitado.After(ahora) {
		t := l.limitado.Unix()
		e.LimitadoHasta = &t
	} else if e.Estado == "limitado" && e.TomadaEn > 0 && e.EdadS > 1200 {
		e.Estado = "viejo"
		e.Detalle = "La última lectura tiene más de 20 minutos"
	}
	e.ProximaEn = l.proxima().Unix()
	return e
}

// Leer reserva la lectura antes de ejecutar, para que nunca se solapen.
func (l *Lector) Leer(ctx context.Context, forzada bool) bool {
	l.mu.Lock()
	ahora := l.Ahora()
	proxima := l.proxima()
	if forzada {
		proxima = l.ultimo.Add(2 * time.Minute)
		if l.limitado.After(proxima) {
			proxima = l.limitado
		}
	}
	if l.Demo || l.Exe == "" || l.leyendo || (!l.ultimo.IsZero() || !forzada) && ahora.Before(proxima) {
		l.mu.Unlock()
		return false
	}
	l.leyendo = true
	l.ultimo = ahora
	l.mu.Unlock()
	if l.Cambio != nil {
		l.Cambio()
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	dir, err := os.MkdirTemp("", "atalaya-uso-claude-*")
	var salida []byte
	if err == nil {
		defer os.RemoveAll(dir)
		salida, err = l.Correr(ctx, l.Exe, dir, []string{"-p", "/usage", "--output-format", "stream-json", "--verbose", "--setting-sources", "project", "--strict-mcp-config", "--mcp-config", `{"mcpServers":{}}`, "--tools", "", "--no-session-persistence"})
	}
	if err == nil {
		err = ctx.Err()
	}
	var r Resultado
	if err == nil {
		r, err = Analizar(salida, l.Ahora())
	}
	l.mu.Lock()
	l.leyendo = false
	switch {
	case err != nil:
		l.estado.Estado = "error"
		l.estado.Detalle = "No se pudo leer el uso de Claude"
	case r.Limitada:
		l.limitado = r.Hasta
		if l.limitado.IsZero() {
			l.limitado = l.Ahora().Add(30 * time.Minute)
		}
		l.estado.Estado = "limitado"
		l.estado.Detalle = "Claude limita temporalmente la lectura del uso"
	case r.SinSuscripcion:
		l.estado.Estado = "sin-suscripcion"
		l.estado.Detalle = "Claude no devolvió un informe de suscripción"
	default:
		l.limitado = time.Time{}
		l.estado.Estado = "ok"
		l.estado.Detalle = ""
		l.estado.TomadaEn = l.Ahora().Unix()
		l.estado.Fuente = "usage"
		l.estado.Ventanas = r.Ventanas
		l.estado.ExtraActivo = r.Extra
		f := fila{T: l.estado.TomadaEn, Ventanas: map[string]float64{}, Reinicios: map[string]int64{}}
		for _, v := range r.Ventanas {
			f.Ventanas[v.Clave] = v.Porcentaje
			f.Reinicios[v.Clave] = v.ReiniciaEn
		}
		l.filas = append(l.filas, f)
		var e error
		l.filas, e = guardar(l.ruta, l.filas, l.Ahora())
		if e != nil && !errors.Is(e, context.Canceled) {
			l.estado.Estado = "error"
			l.estado.Detalle = "No se pudo guardar el historial"
		}
	}
	l.mu.Unlock()
	if l.Cambio != nil {
		l.Cambio()
	}
	return true
}
