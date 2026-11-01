package claude

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
)

// Edicion: un archivo que el agente editó o escribió, y cuándo. Ruta es la del hilo, para mostrar (relativa al
// proyecto cuando se puede, ver corta); Clave, la ruta completa normalizada, para comparar entre agentes sin que dos
// «docs/idea.md» de repositorios distintos parezcan el mismo archivo.
type Edicion struct {
	Ruta  string    `json:"ruta"`
	Clave string    `json:"-"`
	Hora  time.Time `json:"hora"`
}

// Huella: lo que las alertas necesitan de un agente sin copiar todos sus pasos: sus ediciones y el contexto de su
// última petición, con la ventana que se infiere igual que en Contexto.
type Huella struct {
	Ediciones []Edicion
	Contexto  int64
	Ventana   int64
}

// claveArchivo: la ruta completa del archivo que edita una herramienta, en minúsculas y con barras normales (en
// Windows, «D:\X» y «d:/x» son el mismo archivo).
func claveArchivo(input json.RawMessage) string {
	var m struct {
		FilePath     string `json:"file_path"`
		NotebookPath string `json:"notebook_path"`
	}
	if json.Unmarshal(input, &m) != nil {
		return ""
	}
	r := m.FilePath
	if r == "" {
		r = m.NotebookPath
	}
	if r == "" {
		return ""
	}
	return strings.ToLower(filepath.ToSlash(filepath.Clean(r)))
}

// Huellas devuelve la huella de cada agente con actividad desde `desde`, por id.
func (ix *Indice) Huellas(desde time.Time) map[string]Huella {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	out := map[string]Huella{}
	for _, a := range ix.arch {
		if a.agente.ID == "" || a.agente.Ultimo.Before(desde) {
			continue
		}
		h := Huella{Ventana: 200_000}
		for _, p := range a.pasos {
			if p.editado != "" && !p.Error {
				h.Ediciones = append(h.Ediciones, Edicion{Ruta: p.Hilo, Clave: p.editado, Hora: p.Hora})
			}
		}
		for _, p := range a.peticiones {
			if p.Contexto > 200_000 {
				h.Ventana = 1_000_000
			}
		}
		for _, k := range a.compactaciones {
			if k.Antes > 200_000 {
				h.Ventana = 1_000_000
			}
		}
		if n := len(a.peticiones); n > 0 {
			h.Contexto = a.peticiones[n-1].Contexto
		}
		out[a.agente.ID] = h
	}
	return out
}
