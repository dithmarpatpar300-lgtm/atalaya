package servidor

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// La ventana de contexto real de cada sesión de Claude Code, que manda el mod con la cuota (empujarUsoClaude). Sin ella
// Atalaya la infiere de las peticiones: 200 k hasta que una pasa de ahí, y una sesión de 1 M salía al 94 % con 188 k.
// Se guarda en la carpeta de datos para que un reinicio de Atalaya no la pierda hasta la próxima medición.

// MaxVentanas: cuántas sesiones se recuerdan; las más viejas se sueltan.
const MaxVentanas = 500

type ventanaSesion struct {
	Ventana int64     `json:"ventana"`
	Visto   time.Time `json:"visto"`
}

type ventanasContexto struct {
	mu     sync.Mutex
	m      map[string]ventanaSesion
	cargue bool
}

func (s *Servidor) rutaVentanas() string {
	if s.Datos == "" {
		return ""
	}
	return filepath.Join(s.Datos, "ventanas-contexto.json")
}

// cargarVentanas lee el archivo una vez; con el candado tomado.
func (s *Servidor) cargarVentanas() {
	v := &s.ventanas
	if v.cargue {
		return
	}
	v.cargue, v.m = true, map[string]ventanaSesion{}
	if ruta := s.rutaVentanas(); ruta != "" {
		if b, err := os.ReadFile(ruta); err == nil {
			_ = json.Unmarshal(b, &v.m)
		}
	}
}

// ventanaDe: lo que usa el índice (claude.Indice.Ventanas).
func (s *Servidor) ventanaDe(id string) (int64, bool) {
	s.ventanas.mu.Lock()
	defer s.ventanas.mu.Unlock()
	s.cargarVentanas()
	w, ok := s.ventanas.m[id]
	return w.Ventana, ok
}

// anotarVentana guarda la ventana de una sesión; solo escribe el archivo, y dice que cambió, si es nueva o distinta.
func (s *Servidor) anotarVentana(id string, ventana int64) bool {
	if id == "" || ventana <= 0 {
		return false
	}
	v := &s.ventanas
	v.mu.Lock()
	defer v.mu.Unlock()
	s.cargarVentanas()
	antes, estaba := v.m[id]
	v.m[id] = ventanaSesion{Ventana: ventana, Visto: s.ahora()}
	if estaba && antes.Ventana == ventana {
		return false
	}
	if len(v.m) > MaxVentanas {
		var viejo string
		for k, x := range v.m {
			if viejo == "" || x.Visto.Before(v.m[viejo].Visto) {
				viejo = k
			}
		}
		delete(v.m, viejo)
	}
	if ruta := s.rutaVentanas(); ruta != "" {
		b, _ := json.Marshal(v.m)
		if err := os.WriteFile(ruta, b, 0o600); err != nil {
			log.Printf("ventanas de contexto: %v", err)
		}
	}
	return true
}
