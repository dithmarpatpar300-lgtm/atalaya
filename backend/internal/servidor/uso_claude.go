package servidor

import (
	"net/http"
	"strconv"

	"atalaya/internal/usoclaude"
)

// ConectarUsoClaude avisa al panel cuando cambia el estado de la lectura.
func (s *Servidor) ConectarUsoClaude() {
	if s.UsoClaude != nil {
		s.UsoClaude.Cambio = func() { s.difusion.enviar(Evento{Tipo: "cuotas"}) }
	}
}

func (s *Servidor) estadoUsoClaude(r *http.Request) usoclaude.Estado {
	horas, _ := strconv.Atoi(r.URL.Query().Get("horas"))
	if s.Demo {
		return usoclaude.Estado{Estado: "sin-claude", Detalle: "La lectura de Claude está desactivada en demo", Ventanas: []usoclaude.Ventana{}, Historial: []usoclaude.Punto{}}
	}
	if s.UsoClaude == nil {
		return usoclaude.Estado{Estado: "sin-claude", Detalle: "No se encontró Claude Code", Ventanas: []usoclaude.Ventana{}, Historial: []usoclaude.Punto{}}
	}
	return s.UsoClaude.Estado(horas)
}
func (s *Servidor) verUsoClaude(w http.ResponseWriter, r *http.Request) {
	escribir(w, s.estadoUsoClaude(r))
}
func (s *Servidor) leerUsoClaude(w http.ResponseWriter, r *http.Request) {
	if !delPanel(r) {
		fallo(w, http.StatusForbidden, "solo el panel puede solicitar la lectura")
		return
	}
	if !s.Demo && s.UsoClaude != nil && s.UsoClaude.Exe != "" && !s.UsoClaude.Leer(r.Context(), true) {
		e := s.estadoUsoClaude(r)
		e.ProximaEn = s.UsoClaude.ProximaForzada()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusTooManyRequests)
		escribir(w, e)
		return
	}
	escribir(w, s.estadoUsoClaude(r))
}
