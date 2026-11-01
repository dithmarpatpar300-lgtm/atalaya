package claude

import (
	"encoding/json"
	"strings"
	"time"
)

// Sesiones que lanza otra sesión (start_session de la app de escritorio, «crea una nueva sesión a partir de aquí»):
// son sesiones principales, autónomas, con su propio equipo, pero nacen de la que las lanzó. La lanzada no guarda quién
// la lanzó; lo que sí trae es su primer mensaje, «Background (from the session that started this one): … Task: <el
// encargo>», y la que la lanzó guarda la llamada con ese mismo encargo y su título. Se enlazan por ese texto.

// cabeceraLanzada: cómo empieza el primer mensaje de una sesión lanzada por otra.
const cabeceraLanzada = "Background (from the session that started this one):"

// Lanzada: una llamada a start_session de una sesión.
type lanzada struct {
	titulo string
	huella string
	hora   time.Time
}

// huellaEncargo: el comienzo del encargo, sin espacios de más, para comparar los dos lados.
func huellaEncargo(t string) string {
	return recortar(strings.Join(strings.Fields(t), " "), 300)
}

// anotarLanzada: una llamada a start_session (con o sin el prefijo del servidor MCP) deja su título y su encargo.
func (a *archivo) anotarLanzada(nombre string, input json.RawMessage, hora time.Time) {
	if !strings.HasSuffix(nombre, "start_session") {
		return
	}
	var in struct {
		Titulo string `json:"title"`
		Prompt string `json:"prompt"`
	}
	if json.Unmarshal(input, &in) != nil || in.Prompt == "" {
		return
	}
	a.lanzadas = append(a.lanzadas, lanzada{titulo: limpiarTitulo(in.Titulo), huella: huellaEncargo(in.Prompt), hora: hora})
}

// encargoDeLanzada: si el primer mensaje es el de una sesión lanzada, su encargo (lo que va después de «Task:»).
func encargoDeLanzada(t string) (string, bool) {
	if !strings.HasPrefix(t, cabeceraLanzada) {
		return "", false
	}
	_, tarea, ok := strings.Cut(t, "\nTask:\n")
	if !ok {
		return "", false
	}
	return strings.TrimSpace(tarea), true
}

// VentanaLanzada: cuánto antes de su primer mensaje puede estar la llamada que la lanzó.
const VentanaLanzada = 10 * time.Minute

// enlazarLanzadas da a cada sesión lanzada su origen y su título, con el candado de escritura tomado. Solo mira las que
// todavía no lo tienen: una vez enlazada, no se busca más.
func (ix *Indice) enlazarLanzadas() {
	var hijas []*archivo
	for _, a := range ix.arch {
		if a.huellaLanzada != "" && a.agente.LanzadaPor == "" {
			hijas = append(hijas, a)
		}
	}
	if len(hijas) == 0 {
		return
	}
	for _, h := range hijas {
		for _, p := range ix.arch {
			if p == h || p.agente.Tipo != "sesion" {
				continue
			}
			for _, l := range p.lanzadas {
				dt := h.agente.Inicio.Sub(l.hora)
				if l.huella == h.huellaLanzada && dt >= -time.Minute && dt <= VentanaLanzada {
					h.agente.LanzadaPor = p.agente.ID
					// El título que le dio la que la lanzó: el último recurso de tituloDe (uno propio manda).
					h.tituloLanzada = l.titulo
					h.agente.Titulo = h.tituloDe()
				}
			}
		}
	}
}
