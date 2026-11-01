package servidor

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"atalaya/internal/usoclaude"
)

// ConectarUsoClaude avisa al panel cuando cambia el estado de la lectura.
func (s *Servidor) ConectarUsoClaude() {
	if s.UsoClaude != nil {
		s.UsoClaude.Cambio = func() { s.difusion.enviar(Evento{Tipo: "cuotas"}) }
	}
	// La ventana de contexto que manda el mod llega por el mismo camino; el índice la usa en vez de inferirla.
	if s.Claude != nil {
		s.Claude.Ventanas = s.ventanaDe
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

// cabModUso: la cabecera con que el mod de Claude Code empuja la cuota. Un navegador no la manda sin un preflight que
// Atalaya no contesta, así que una página ajena no puede empujar cifras falsas.
const cabModUso = "X-Atalaya-Mod"

// empujarUsoClaude: POST /api/uso-claude/empujar. El mod (session.measure) manda {"rate_limits": [...]} cuando una
// ventana se mueve un punto; complementa la lectura de /usage, que pasa cada 10 minutos.
func (s *Servidor) empujarUsoClaude(w http.ResponseWriter, r *http.Request) {
	if !origenLocal(r) || r.Header.Get(cabModUso) != "uso-claude" || !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		fallo(w, http.StatusForbidden, "solo el mod de Claude Code puede empujar la cuota")
		return
	}
	if s.Demo {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	var cuerpo struct {
		Ventanas []usoclaude.Empujada `json:"rate_limits"`
		// La ventana de contexto de la sesión que mide (ventanas.go): la app la sabe, Atalaya solo la infiere.
		Contexto *struct {
			Sesion  string `json:"session"`
			Ventana int64  `json:"window"`
		} `json:"context"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&cuerpo); err != nil {
		fallo(w, http.StatusBadRequest, "cuerpo inválido")
		return
	}
	if c := cuerpo.Contexto; c != nil && c.Sesion != "" && c.Ventana > 0 {
		if s.anotarVentana(c.Sesion, c.Ventana) {
			s.difusion.enviar(Evento{Tipo: "agentes", IDs: []string{c.Sesion}})
		}
		if len(cuerpo.Ventanas) == 0 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	if s.UsoClaude == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if err := s.UsoClaude.Empujar(cuerpo.Ventanas); errors.Is(err, usoclaude.ErrEmpujadaInvalida) {
		fallo(w, http.StatusBadRequest, err.Error())
		return
	} else if err != nil {
		log.Printf("cuota empujada: %v", err)
	}
	w.WriteHeader(http.StatusNoContent)
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
