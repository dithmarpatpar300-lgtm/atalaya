// Package cuotas lee las cuotas de Claude y Codex que mide la herramienta Cuotas
// (una herramienta opcional). Solo lee: ejecuta `cuotas.exe una-vez --json` y, si no
// puede, arma lo que sabe con sus archivos `claude-statusline.json` e
// `historial.jsonl`. Nunca toca credenciales: eso lo resuelve Cuotas.
package cuotas

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Lectura es la salida de `cuotas una-vez --json`, con los mismos nombres.
type Lectura struct {
	Advice  *Consejo `json:"advice"`
	Panels  []Panel  `json:"panels"`
	TakenAt int64    `json:"taken_at"`
}

type Consejo struct {
	FirstChoice string             `json:"first_choice"`
	Providers   []PosturaProveedor `json:"providers"`
	Summary     string             `json:"summary"`
}

type PosturaProveedor struct {
	EvenPacePercentPerHour float64  `json:"even_pace_percent_per_hour"`
	HeadroomPercent        float64  `json:"headroom_percent"`
	Provider               string   `json:"provider"`
	Reasons                []string `json:"reasons"`
	Stance                 string   `json:"stance"`
}

type Panel struct {
	CapturedAt int64     `json:"captured_at"`
	Notice     *string   `json:"notice"`
	Plan       *string   `json:"plan"`
	Provider   string    `json:"provider"`
	Source     string    `json:"source"`
	Windows    []Ventana `json:"windows"`
}

type Ventana struct {
	DurationMins       int     `json:"duration_mins"`
	Label              string  `json:"label"`
	Pace               *Ritmo  `json:"pace"`
	ResetsAt           int64   `json:"resets_at"`
	Tokens             *Tokens `json:"tokens"`
	TokensLeftEstimate *int64  `json:"tokens_left_estimate"`
	UsedPercent        float64 `json:"used_percent"`
}

type Ritmo struct {
	FullAt         int64   `json:"full_at"`
	PercentPerHour float64 `json:"percent_per_hour"`
}

type Tokens struct {
	CacheRead  int64 `json:"cache_read"`
	CacheWrite int64 `json:"cache_write"`
	Fresh      int64 `json:"fresh"`
	Input      int64 `json:"input"`
	Messages   int64 `json:"messages"`
	Output     int64 `json:"output"`
}

// Estado es lo que el panel recibe: la lectura y de dónde salió.
type Estado struct {
	Lectura *Lectura `json:"lectura"`
	// Origen: "ejecutable" (cuotas.exe) o "archivos" (statusline e historial, sin consejo).
	Origen string `json:"origen"`
	// LeidaEn: cuándo la leyó este servidor (unix, s).
	LeidaEn int64  `json:"leida_en"`
	Error   string `json:"error,omitempty"`
	// Instalada: falso si no está cuotas.exe ni sus archivos. Cuotas es opcional: no es un error, y el panel no
	// muestra la franja.
	Instalada bool `json:"instalada"`
}

// Decodificar valida y convierte la salida JSON de Cuotas.
func Decodificar(b []byte) (*Lectura, error) {
	var l Lectura
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, fmt.Errorf("la salida de cuotas no es JSON válido: %w", err)
	}
	if len(l.Panels) == 0 {
		return nil, errors.New("la salida de cuotas no trae paneles")
	}
	return &l, nil
}

// Ejecutor corre el programa y devuelve su salida estándar. Se cambia en las pruebas.
type Ejecutor func(ctx context.Context, exe string, args ...string) ([]byte, error)

func ejecutar(ctx context.Context, exe string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, exe, args...)
	ocultarVentana(cmd)
	var errSal bytes.Buffer
	cmd.Stderr = &errSal
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("%s: %w %s", filepath.Base(exe), err, bytes.TrimSpace(errSal.Bytes()))
	}
	return out, nil
}

// Lector guarda la última lectura y decide cuándo volver a leer.
type Lector struct {
	Exe        string // cuotas.exe
	Dir        string // carpeta de sus archivos
	Ejecutar   Ejecutor
	Ahora      func() time.Time
	Intervalo  time.Duration // lectura periódica
	MinForzada time.Duration // mínimo entre dos lecturas forzadas

	mu        sync.Mutex
	estado    Estado
	ultimoRun time.Time
	modStatus time.Time
	leyendo   atomic.Bool // una lectura en curso (Leer)
}

func NuevoLector(exe, dir string) *Lector {
	return &Lector{Exe: exe, Dir: dir, Ejecutar: ejecutar, Ahora: time.Now, Intervalo: 5 * time.Minute, MinForzada: 30 * time.Second}
}

// Estado devuelve la última lectura guardada.
func (l *Lector) Estado() Estado {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.estado
}

// Leer lee ahora (por el ejecutable, o por los archivos si falla) y guarda el resultado.
func (l *Lector) Leer(ctx context.Context) Estado {
	// Un cuotas.exe lento no se encima con otro: mientras una lectura corre, las demás devuelven la guardada.
	if !l.leyendo.CompareAndSwap(false, true) {
		return l.Estado()
	}
	defer l.leyendo.Store(false)
	ahora := l.Ahora()
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	est := Estado{LeidaEn: ahora.Unix(), Instalada: true}
	out, err := l.Ejecutar(ctx, l.Exe, "una-vez", "--json")
	noEsta := errors.Is(err, fs.ErrNotExist) || errors.Is(err, exec.ErrNotFound)
	if err == nil {
		var lec *Lectura
		if lec, err = Decodificar(out); err == nil {
			est.Lectura, est.Origen = lec, "ejecutable"
		}
	}
	if err != nil {
		lec, err2 := DesdeArchivos(l.Dir)
		switch {
		case err2 != nil && noEsta:
			est.Instalada = false // ni el ejecutable ni sus archivos: Cuotas no está, y no es un error
		case err2 != nil:
			est.Error = fmt.Sprintf("%v; y por los archivos: %v", err, err2)
		default:
			est.Lectura, est.Origen, est.Error = lec, "archivos", err.Error()
		}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	l.ultimoRun = ahora
	// Una lectura fallida no borra la última buena.
	if est.Lectura == nil && l.estado.Lectura != nil {
		l.estado.Error = est.Error
		return l.estado
	}
	l.estado = est
	return est
}

// Forzar lee si pasó el mínimo desde la última lectura; si no, devuelve la guardada.
func (l *Lector) Forzar(ctx context.Context) Estado {
	l.mu.Lock()
	reciente := !l.ultimoRun.IsZero() && l.Ahora().Sub(l.ultimoRun) < l.MinForzada
	l.mu.Unlock()
	if reciente {
		return l.Estado()
	}
	return l.Leer(ctx)
}

// Toca dice si corresponde leer: pasó el intervalo, o cambió la statusLine y la última lectura tiene más de 60 s.
func (l *Lector) Toca() bool {
	if l.leyendo.Load() {
		return false
	}
	ahora := l.Ahora()
	var mod time.Time
	if fi, err := os.Stat(filepath.Join(l.Dir, "claude-statusline.json")); err == nil {
		mod = fi.ModTime()
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.ultimoRun.IsZero() || ahora.Sub(l.ultimoRun) >= l.Intervalo {
		l.modStatus = mod
		return true
	}
	if !mod.Equal(l.modStatus) && ahora.Sub(l.ultimoRun) >= 60*time.Second {
		l.modStatus = mod
		return true
	}
	return false
}

// statusline es lo que Cuotas guarda de la statusLine de Claude Code.
type statusline struct {
	CapturedAt int64  `json:"captured_at"`
	Version    string `json:"version"`
	RateLimits map[string]struct {
		CapturedAt     int64   `json:"captured_at"`
		ResetsAt       int64   `json:"resets_at"`
		UsedPercentage float64 `json:"used_percentage"`
	} `json:"rate_limits"`
}

// filaHistorial es una línea de historial.jsonl: proveedor, ventana, uso, reinicio y hora.
type filaHistorial struct {
	P string  `json:"p"`
	W string  `json:"w"`
	U float64 `json:"u"`
	R int64   `json:"r"`
	T int64   `json:"t"`
}

// DesdeArchivos arma una lectura sin consejo con los archivos de Cuotas.
func DesdeArchivos(dir string) (*Lectura, error) {
	lec := &Lectura{}
	if b, err := os.ReadFile(filepath.Join(dir, "claude-statusline.json")); err == nil {
		var s statusline
		if err := json.Unmarshal(b, &s); err == nil {
			p := Panel{CapturedAt: s.CapturedAt, Provider: "Claude Code", Source: "statusLine de Claude Code " + s.Version}
			for _, k := range []struct {
				clave, etiqueta string
				mins            int
			}{{"five_hour", "5 h", 300}, {"seven_day", "Semanal", 10080}} {
				if v, ok := s.RateLimits[k.clave]; ok {
					p.Windows = append(p.Windows, Ventana{DurationMins: k.mins, Label: k.etiqueta, ResetsAt: v.ResetsAt, UsedPercent: v.UsedPercentage})
				}
			}
			if len(p.Windows) > 0 {
				lec.Panels = append(lec.Panels, p)
			}
		}
	}
	if f, err := os.Open(filepath.Join(dir, "historial.jsonl")); err == nil {
		defer f.Close()
		ultimas := map[string]filaHistorial{}
		sc := bufio.NewScanner(f)
		for sc.Scan() {
			var h filaHistorial
			if json.Unmarshal(sc.Bytes(), &h) == nil && h.P != "" {
				ultimas[h.P+"\x00"+h.W] = h
			}
		}
		paneles := map[string]*Panel{}
		claves := make([]string, 0, len(ultimas))
		for k := range ultimas {
			claves = append(claves, k)
		}
		sort.Strings(claves)
		for _, k := range claves {
			h := ultimas[k]
			if h.P != "codex" {
				continue // Claude ya vino por la statusLine, que es más fresca.
			}
			p := paneles[h.P]
			if p == nil {
				p = &Panel{Provider: "Codex", Source: "historial de Cuotas"}
				paneles[h.P] = p
			}
			mins := 10080
			if h.W == "5 h" {
				mins = 300
			}
			if h.T > p.CapturedAt {
				p.CapturedAt = h.T
			}
			p.Windows = append(p.Windows, Ventana{DurationMins: mins, Label: h.W, ResetsAt: h.R, UsedPercent: h.U})
		}
		for _, p := range paneles {
			sort.Slice(p.Windows, func(i, j int) bool { return p.Windows[i].DurationMins < p.Windows[j].DurationMins })
			lec.Panels = append(lec.Panels, *p)
		}
	}
	if len(lec.Panels) == 0 {
		return nil, errors.New("no hay statusline ni historial legibles en " + dir)
	}
	for _, p := range lec.Panels {
		if p.CapturedAt > lec.TakenAt {
			lec.TakenAt = p.CapturedAt
		}
	}
	return lec, nil
}

// Punto: el uso de una ventana a una hora (segundos Unix).
type Punto struct {
	T int64   `json:"t"`
	U float64 `json:"u"`
}

// Historial: el uso de la ventana de 5 h de cada proveedor («claude», «codex») entre desde y hasta, según
// historial.jsonl, a lo sumo un punto cada paso (el último de cada tramo) para que la curva no pese.
func (l *Lector) Historial(desde, hasta time.Time, paso time.Duration) map[string][]Punto {
	out := map[string][]Punto{}
	f, err := os.Open(filepath.Join(l.Dir, "historial.jsonl"))
	if err != nil {
		return out
	}
	defer f.Close()
	d, h, p := desde.Unix(), hasta.Unix(), int64(paso.Seconds())
	if p < 1 {
		p = 1
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var fila filaHistorial
		if json.Unmarshal(sc.Bytes(), &fila) != nil || fila.W != "5 h" || fila.T < d || fila.T > h {
			continue
		}
		pts := out[fila.P]
		pt := Punto{T: fila.T, U: fila.U}
		if n := len(pts); n > 0 && (pts[n-1].T-d)/p == (fila.T-d)/p {
			pts[n-1] = pt
		} else {
			pts = append(pts, pt)
		}
		out[fila.P] = pts
	}
	for k := range out {
		sort.Slice(out[k], func(i, j int) bool { return out[k][i].T < out[k][j].T })
	}
	return out
}
