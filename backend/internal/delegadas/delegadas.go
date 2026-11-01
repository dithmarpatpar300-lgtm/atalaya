// Package delegadas lee las fichas de scripts/claude.ps1: cuando una sesión de Codex le pasa un encargo a Claude Code
// sin pantalla, el script escribe %LOCALAPPDATA%\Atalaya\claude\<fecha>-<tema>.meta.json al empezar y la completa al
// terminar. El transcrito de esa sesión lo guarda Claude Code como cualquier otro; la ficha dice lo que el transcrito
// no sabe: de qué repositorio es (trabajó en una copia), quién la lanzó, el tema, la clase y el plazo. Solo lee.
package delegadas

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/codexvivo"
)

// Ficha: la que escribe claude.ps1 (los mismos campos que la de codex.ps1, más los de la sesión).
type Ficha struct {
	Tema       string `json:"tema"`
	Clase      string `json:"clase"`
	Modelo     string `json:"modelo"`
	Effort     string `json:"effort"`
	Fase       string `json:"fase"`
	Raiz       string `json:"raiz"`
	Base       string `json:"base"`
	MinutosMax int    `json:"minutos_max"`
	Inicio     string `json:"inicio"`
	Fin        string `json:"fin"`
	Salida     *int   `json:"salida"`
	Informe    string `json:"informe"`
	// SesionClaude: el --session-id con que corrió; es el nombre de su transcrito y el id de su agente.
	SesionClaude string `json:"sesion_claude"`
	// Sesion: quién la lanzó (CODEX_THREAD_ID o CLAUDE_CODE_SESSION_ID), y PadreHerramienta, cuál de las dos.
	Sesion           string `json:"sesion"`
	PadreHerramienta string `json:"padre_herramienta"`
}

// Lector sigue la carpeta de las fichas.
type Lector struct {
	Dir    string
	mu     sync.RWMutex
	fichas map[string]Ficha     // por SesionClaude
	mods   map[string]time.Time // por ruta, para releer solo lo que cambió
}

func NuevoLector(dir string) *Lector {
	return &Lector{Dir: dir, fichas: map[string]Ficha{}, mods: map[string]time.Time{}}
}

// Actualizar relee las fichas que cambiaron y devuelve los ids de las sesiones cuya ficha es nueva o cambió.
func (l *Lector) Actualizar() []string {
	if l == nil || l.Dir == "" {
		return nil
	}
	rutas, _ := filepath.Glob(filepath.Join(l.Dir, "*.meta.json"))
	var cambiadas []string
	vistas := map[string]bool{}
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, r := range rutas {
		vistas[r] = true
		fi, err := os.Stat(r)
		if err != nil || fi.ModTime().Equal(l.mods[r]) {
			continue
		}
		b, err := os.ReadFile(r)
		if err != nil {
			continue
		}
		var f Ficha
		// Una ficha a medio escribir no cuenta; se relee en la vuelta siguiente.
		if json.Unmarshal(bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")), &f) != nil || f.SesionClaude == "" {
			continue
		}
		l.mods[r] = fi.ModTime()
		l.fichas[f.SesionClaude] = f
		cambiadas = append(cambiadas, f.SesionClaude)
	}
	for r := range l.mods {
		if !vistas[r] {
			delete(l.mods, r) // claude.ps1 borra las de más de 7 días; su sesión vuelve a ser una más
		}
	}
	return cambiadas
}

// Delegacion: lo que la ficha de esa sesión cambia de su agente (claude.Indice.Delegadas).
func (l *Lector) Delegacion(id string) (claude.Delegacion, bool) {
	if l == nil {
		return claude.Delegacion{}, false
	}
	l.mu.RLock()
	f, ok := l.fichas[id]
	l.mu.RUnlock()
	if !ok {
		return claude.Delegacion{}, false
	}
	// El rol a secas, como las llamadas a Codex («Codex · <tema>»): la clase ya sale aparte.
	d := claude.Delegacion{Raiz: f.Raiz, Padre: Padre(f), Rol: "Claude", Tarea: f.Tema, Clase: f.Clase,
		Effort: f.Effort, Base: f.Base, Informe: f.Informe, Terminada: f.Fin != ""}
	if f.MinutosMax > 0 {
		d.Presupuesto = fmt.Sprintf("%d min", f.MinutosMax)
	}
	return d, true
}

// Padre: el id del agente que lanzó la sesión. Una sesión de Codex se llama como en codexvivo («codexvivo~sesion-<id>»);
// una de Claude Code, por su propio id.
func Padre(f Ficha) string {
	s := strings.TrimSpace(f.Sesion)
	switch {
	case s == "":
		return ""
	case f.PadreHerramienta == "codex":
		return codexvivo.Prefijo + codexvivo.PrefijoSesion + s
	default:
		return s
	}
}
