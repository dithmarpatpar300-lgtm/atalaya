package servidor

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"

	"atalaya/internal/alertas"
	"atalaya/internal/avisos"
	"atalaya/internal/claude"
)

// Los avisos: lo que te espera llega como notificación aunque el panel esté cerrado. Qué cuenta,
// aquí; cuándo se avisa (una vez por cosa, juntas, con pausa), en internal/avisos. Desde la campana del panel se
// encienden y apagan por tipo y se prueban: es la segunda escritura del panel, con la misma puerta que responder una
// pregunta (origen local y la cabecera X-Atalaya-Panel).

// cadaAvisos: cada cuántas vueltas se mira (3 vueltas, 6 s): una decisión no necesita llegar antes.
const cadaAvisos = 3

// configAvisos: la de avisos.json, leída una vez; PorOmision si no hay ruta.
func (s *Servidor) configAvisos() avisos.Config {
	s.confAvisos.mu.Lock()
	defer s.confAvisos.mu.Unlock()
	if !s.confAvisos.leida {
		s.confAvisos.c, s.confAvisos.leida = avisos.PorOmision, true
		if s.RutaAvisos != "" {
			s.confAvisos.c = avisos.LeerConfig(s.RutaAvisos)
		}
	}
	return s.confAvisos.c
}

// cosasParaAvisar: las decisiones pendientes que no son viejas y las alertas de nivel error, según los tipos
// encendidos.
func (s *Servidor) cosasParaAvisar() []avisos.Cosa {
	conf := s.configAvisos()
	var out []avisos.Cosa
	if conf.Decisiones {
		for _, d := range s.decisiones("").Decisiones {
			if d.Estado != claude.DecPendiente || d.Vieja {
				continue
			}
			texto := d.Contexto
			if len(d.Preguntas) > 0 {
				texto = d.Preguntas[0].Texto
			}
			out = append(out, avisos.Cosa{Clave: "dec:" + d.ID, Titulo: "Te espera una decisión · " + d.Sesion,
				Texto: strings.TrimSpace(d.Proyecto + ": " + texto), Seccion: "decisiones"})
		}
	}
	if conf.Alertas {
		s.mu.RLock()
		for _, a := range s.alertas {
			if a.Nivel == alertas.Error {
				out = append(out, avisos.Cosa{Clave: "alerta:" + a.ID, Titulo: a.Titulo, Texto: a.Proyecto + ": " + a.Texto,
					Seccion: "agentes"})
			}
		}
		s.mu.RUnlock()
	}
	return out
}

// mirarAvisos manda el aviso que toque. La notificación tarda lo que tarda PowerShell en arrancar: va aparte, sin
// frenar la vuelta.
func (s *Servidor) mirarAvisos() {
	if s.Avisos == nil || s.Notificador == nil {
		return
	}
	e, ok := s.Avisos.Mirar(s.cosasParaAvisar(), s.ahora())
	if !ok {
		return
	}
	go func() {
		err := s.Notificador.Notificar(e.Titulo, e.Texto, s.urlDe(e.Seccion))
		if err != nil {
			log.Printf("aviso: %v", err)
		}
		s.Avisos.Resultado(e.Hora, err)
		s.difusion.enviar(Evento{Tipo: "avisos"})
	}()
}

func (s *Servidor) urlDe(seccion string) string {
	return strings.TrimSuffix(s.Base, "/") + "/#" + seccion
}

type respuestaAvisos struct {
	// Activos: Atalaya arrancó con avisos (-avisos); si no, la campana solo lo dice.
	Activos bool `json:"activos"`
	avisos.Config
	Enviados []avisos.Enviado `json:"enviados"`
}

func (s *Servidor) estadoAvisos() respuestaAvisos {
	out := respuestaAvisos{Activos: s.Avisos != nil && s.Notificador != nil, Config: s.configAvisos(), Enviados: []avisos.Enviado{}}
	if s.Avisos != nil {
		out.Enviados = s.Avisos.Enviados()
	}
	return out
}

func (s *Servidor) verAvisos(w http.ResponseWriter, _ *http.Request) {
	escribir(w, s.estadoAvisos())
}

// delPanel: la puerta de las escrituras del panel (la misma de responderPregunta).
func delPanel(r *http.Request) bool {
	return origenLocal(r) && r.Header.Get(cabPanel) == "1"
}

// configurarAvisos enciende o apaga un tipo; lo que no viene en el cuerpo no cambia.
func (s *Servidor) configurarAvisos(w http.ResponseWriter, r *http.Request) {
	if !delPanel(r) {
		fallo(w, http.StatusForbidden, "solo el panel de Atalaya cambia los avisos")
		return
	}
	var c struct {
		Decisiones *bool `json:"decisiones"`
		Alertas    *bool `json:"alertas"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1024)).Decode(&c); err != nil {
		fallo(w, http.StatusBadRequest, "el cuerpo no es un JSON válido")
		return
	}
	conf := s.configAvisos()
	if c.Decisiones != nil {
		conf.Decisiones = *c.Decisiones
	}
	if c.Alertas != nil {
		conf.Alertas = *c.Alertas
	}
	if s.RutaAvisos != "" {
		if err := avisos.GuardarConfig(s.RutaAvisos, conf); err != nil {
			fallo(w, http.StatusInternalServerError, "no se pudo guardar avisos.json: "+err.Error())
			return
		}
	}
	s.confAvisos.mu.Lock()
	s.confAvisos.c = conf
	s.confAvisos.mu.Unlock()
	s.difusion.enviar(Evento{Tipo: "avisos"})
	escribir(w, s.estadoAvisos())
}

// probarAviso manda una notificación de prueba y espera a que salga (1 a 2 s), para decir si funcionó.
func (s *Servidor) probarAviso(w http.ResponseWriter, r *http.Request) {
	if !delPanel(r) {
		fallo(w, http.StatusForbidden, "solo el panel de Atalaya prueba los avisos")
		return
	}
	if s.Avisos == nil || s.Notificador == nil {
		fallo(w, http.StatusConflict, "los avisos están apagados: Atalaya arrancó con -avisos=false")
		return
	}
	e := avisos.Enviado{Hora: s.ahora(), Titulo: "Atalaya · prueba", Texto: "Así llega un aviso. Un clic abre la bandeja de decisiones.",
		Seccion: "decisiones", Claves: []string{"prueba"}}
	err := s.Notificador.Notificar(e.Titulo, e.Texto, s.urlDe(e.Seccion))
	if err != nil {
		e.Error = err.Error()
	}
	s.Avisos.Anotar(e)
	s.difusion.enviar(Evento{Tipo: "avisos"})
	if err != nil {
		fallo(w, http.StatusBadGateway, e.Error)
		return
	}
	escribir(w, s.estadoAvisos())
}
