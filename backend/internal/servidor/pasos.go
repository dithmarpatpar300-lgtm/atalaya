package servidor

import (
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"atalaya/internal/claude"
)

// Buscar en los pasos de todos los agentes, de Claude y de Codex en vivo. La misma búsqueda
// sirve a la sección del panel y a las herramientas MCP (mcp.go).

type hallado struct {
	agente AgenteVista
	paso   claude.Paso
}

// hallar busca con el filtro en los transcritos y en las trazas de Codex; devuelve lo más reciente primero (hasta
// f.Max) y cuántos había en total.
func (s *Servidor) hallar(f claude.FiltroPasos) ([]hallado, int) {
	hs, total := s.Claude.BuscarPasos(f)
	vistas := map[string]AgenteVista{}
	out := make([]hallado, 0, len(hs))
	for _, h := range hs {
		v, ok := vistas[h.Agente.ID]
		if !ok {
			v = s.vistaClaude(h.Agente, false)
			vistas[h.Agente.ID] = v
		}
		out = append(out, hallado{v, h.Paso})
	}
	if s.CodexVivo != nil {
		for _, c := range s.CodexVivo.Llamadas() {
			if f.Proyecto != "" && filepath.Base(c.Meta.Raiz) != f.Proyecto {
				continue
			}
			var v *AgenteVista
			for i := range c.Pasos {
				if !f.Pasa(&c.Pasos[i]) {
					continue
				}
				if v == nil {
					x := s.vistaCodexVivo(c, nil)
					v = &x
				}
				out = append(out, hallado{*v, c.Pasos[i]})
				total++
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].paso.Hora.After(out[j].paso.Hora) })
	if f.Max > 0 && len(out) > f.Max {
		out = out[:f.Max]
	}
	return out, total
}

type grupoPasos struct {
	Agente AgenteVista   `json:"agente"`
	Pasos  []claude.Paso `json:"pasos"`
}

type respuestaPasos struct {
	Total  int          `json:"total"`
	Grupos []grupoPasos `json:"grupos"`
}

// agrupar junta los hallazgos por agente, en el orden en que aparece cada uno (el de su paso más reciente).
func agrupar(hs []hallado) []grupoPasos {
	out := []grupoPasos{}
	pos := map[string]int{}
	for _, h := range hs {
		i, ok := pos[h.agente.ID]
		if !ok {
			i = len(out)
			pos[h.agente.ID] = i
			out = append(out, grupoPasos{Agente: h.agente})
		}
		out[i].Pasos = append(out[i].Pasos, h.paso)
	}
	return out
}

// MaxPasosBuscados: lo más que devuelve una búsqueda de pasos.
const MaxPasosBuscados = 150

func (s *Servidor) buscarPasos(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := claude.FiltroPasos{Terminos: claude.Terminos(q.Get("q")), Tipo: q.Get("tipo"), Proyecto: q.Get("proyecto"), Max: MaxPasosBuscados}
	if h, err := strconv.Atoi(q.Get("horas")); err == nil && h > 0 {
		f.Desde = s.ahora().Add(-time.Duration(h) * time.Hour)
	}
	hs, total := s.hallar(f)
	escribir(w, respuestaPasos{Total: total, Grupos: agrupar(hs)})
}
