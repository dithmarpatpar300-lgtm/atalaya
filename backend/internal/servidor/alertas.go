package servidor

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"atalaya/internal/alertas"
)

// Las alertas de las reglas: se evalúan en cada vuelta, porque el tiempo corre aunque
// ningún archivo cambie, y se avisa por SSE solo cuando el conjunto cambia.

// VentanaAlertas: lo más viejo que se mira. Un agente sin actividad en ese rato no puede estar en una alerta.
const VentanaAlertas = 3 * time.Hour

func (s *Servidor) evaluarAlertas() []alertas.Alerta {
	ahora := s.ahora()
	desde := ahora.Add(-VentanaAlertas)
	reglas := map[string]alertas.Reglas{}
	for _, p := range s.listaProyectos() {
		reglas[p.Nombre] = s.reglas.Reglas(p.Raiz)
	}
	huellas := s.Claude.Huellas(desde)
	equipo := s.equipoTrabajando()
	var ags []alertas.Agente
	for _, a := range s.Claude.Agentes() {
		if a.Ultimo.Before(desde) {
			break // vienen del más reciente al más viejo
		}
		if _, ok := reglas[a.Proyecto]; !ok && a.Raiz != "" {
			reglas[a.Proyecto] = s.reglas.Reglas(a.Raiz)
		}
		est := s.estadoClaude(a, equipo)
		h := huellas[a.ID]
		nombre := s.nombreDe(a)
		ags = append(ags, alertas.Agente{ID: a.ID, Nombre: nombre, Proyecto: a.Proyecto, ConTope: a.Tipo == "subagente",
			Trabajando: est == Trabajando, Activo: activo(est), Presupuesto: a.Presupuesto, Inicio: a.Inicio, Ultimo: a.Ultimo,
			Contexto: h.Contexto, Ventana: h.Ventana, Ediciones: h.Ediciones, Padre: a.Padre})
	}
	if s.CodexVivo != nil {
		for _, c := range s.CodexVivo.Llamadas() {
			if !c.Activa(ahora) {
				continue
			}
			nombre := nombreVista(AgenteVista{Tipo: "codex", Tarea: c.Meta.Tema}, docsAgente{})
			pres := ""
			if c.Meta.MinutosMax > 0 {
				pres = fmt.Sprintf("%d min", c.Meta.MinutosMax)
			}
			ags = append(ags, alertas.Agente{ID: c.ID, Nombre: nombre, Proyecto: filepath.Base(c.Meta.Raiz), ConTope: true, Trabajando: true,
				Activo: true, Presupuesto: pres, Inicio: c.Inicio, Ultimo: c.Ultimo})
		}
	}
	return alertas.Evaluar(reglas, ags, ahora)
}

// actualizarAlertas guarda las alertas nuevas y dice si cambiaron.
func (s *Servidor) actualizarAlertas() bool {
	as := s.evaluarAlertas()
	firma := alertas.Firma(as)
	s.mu.Lock()
	defer s.mu.Unlock()
	cambio := firma != s.firmaAlertas
	s.alertas, s.firmaAlertas = as, firma
	return cambio
}

type respuestaAlertas struct {
	Reglas  []alertas.Regla  `json:"reglas"`
	Alertas []alertas.Alerta `json:"alertas"`
}

func (s *Servidor) verAlertas(w http.ResponseWriter, r *http.Request) {
	proyecto := r.URL.Query().Get("proyecto")
	out := respuestaAlertas{Reglas: s.reglas.Reglas(s.raizDe(proyecto)).Lista, Alertas: []alertas.Alerta{}}
	s.mu.RLock()
	for _, a := range s.alertas {
		if proyecto == "" || a.Proyecto == proyecto {
			out.Alertas = append(out.Alertas, a)
		}
	}
	s.mu.RUnlock()
	escribir(w, out)
}
