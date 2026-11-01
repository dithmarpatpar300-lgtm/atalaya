package claude

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"atalaya/internal/secretos"
)

// Traza: un turno de un agente contado en Markdown, para la datapedia. Lleva lo mismo que la línea de tiempo del
// panel (etapas, desvíos, vueltas, paralelos y delegaciones) en texto buscable: «¿cuándo se tocó segmented.rs?»,
// «¿qué turnos pasaron por la etapa de pruebas?», «¿a quién se delegó la auditoría?».
type Traza struct {
	Agente   string    // id del agente
	Proyecto string    //
	Turno    int       //
	Titulo   string    //
	Resumen  string    //
	Cuerpo   string    // Markdown, con los secretos tapados
	Ultimo   time.Time // el último paso del turno: si no cambia, la traza tampoco
}

// RutaTraza: dónde vive la traza dentro de la datapedia; el panel la usa para volver al agente y su turno.
func RutaTraza(agente string, turno int) string {
	return fmt.Sprintf("trazas/%s/turno-%d", agente, turno)
}

// Trazas arma las de todos los agentes con pasos.
func (ix *Indice) Trazas() []Traza {
	ix.mu.RLock()
	type copia struct {
		ag    Agente
		pasos []Paso
		hijos map[string]Agente // subagentes por la llamada que los lanzó
	}
	var todos []copia
	for _, a := range ix.arch {
		if len(a.pasos) == 0 || a.agente.Proyecto == "" {
			continue
		}
		c := copia{ag: ix.conDelegacion(*a.agente), pasos: make([]Paso, len(a.pasos)), hijos: map[string]Agente{}}
		for i, p := range a.pasos {
			c.pasos[i] = *p
		}
		for _, x := range ix.arch {
			if x.agente.Padre == a.agente.ID && x.agente.ToolUseID != "" {
				c.hijos[x.agente.ToolUseID] = *x.agente
			}
		}
		todos = append(todos, c)
	}
	ix.mu.RUnlock()

	var out []Traza
	for _, c := range todos {
		enlazar(c.pasos)
		// Los pasos de cada turno, en orden.
		turnos := map[int][]Paso{}
		var orden []int
		for _, p := range c.pasos {
			if _, ok := turnos[p.Turno]; !ok {
				orden = append(orden, p.Turno)
			}
			turnos[p.Turno] = append(turnos[p.Turno], p)
		}
		sort.Ints(orden)
		for _, t := range orden {
			out = append(out, armarTraza(c.ag, t, turnos[t], c.hijos))
		}
	}
	return out
}

func armarTraza(ag Agente, turno int, pasos []Paso, hijos map[string]Agente) Traza {
	var b strings.Builder
	pedido := ""
	for _, p := range pasos {
		if p.Tipo == "Pide" {
			pedido = p.Resumen
			break
		}
	}
	// En el primer turno de un subagente, el pedido es su encargo.
	if turno <= 1 && ag.Tarea != "" && (pedido == "" || ag.Tipo == "subagente") {
		pedido = ag.Tarea
	}
	inicio, fin := pasos[0].Hora, pasos[len(pasos)-1].Hora
	nombre := ag.Rol
	if ag.Tipo == "sesion" && ag.Proyecto != "" {
		nombre = ag.Proyecto // una sesión se nombra por su proyecto, como en la lista
	}
	titulo := fmt.Sprintf("%s · turno %d", nombre, turno)
	if pedido != "" {
		titulo += " · " + recortar(pedido, 90)
	}

	fmt.Fprintf(&b, "# %s\n\n", titulo)
	modelo := ag.Modelo
	if ag.Effort != "" {
		modelo += ", effort " + ag.Effort
	}
	fmt.Fprintf(&b, "**Agente:** %s (%s, %s) · **Proyecto:** %s · **Turno %d**, de %s a %s, %d pasos.\n\n",
		ag.Rol, ag.Tipo, modelo, ag.Proyecto, turno, hhmm(inicio), hhmm(fin), len(pasos))
	if pedido != "" {
		fmt.Fprintf(&b, "**Pedido:** %s\n\n", pedido)
	}
	if turno <= 1 && ag.Asignacion != "" {
		fmt.Fprintf(&b, "**Asignación:** %s\n\n", ag.Asignacion)
	}

	// Archivos: cuántas veces se leyeron, editaron o escribieron.
	type uso struct{ lee, edita, escribe, busca int }
	usos := map[string]*uso{}
	var archivos []string
	etapas := []string{}
	desvios, vueltas, delegados := 0, 0, []string{}

	etapa := ""
	fmt.Fprintf(&b, "## Pasos\n\n")
	for _, p := range pasos {
		if p.Etapa != etapa && p.Etapa != "" {
			etapa = p.Etapa
			etapas = append(etapas, etapa)
			fmt.Fprintf(&b, "\n**Etapa: %s**\n\n", etapa)
		}
		var marcas []string
		if p.Desvio {
			marcas = append(marcas, "↳ desvío")
			desvios++
		}
		if p.Continua >= 0 && p.Hilo != "" {
			marcas = append(marcas, fmt.Sprintf("↩ vuelve a `%s`", p.Hilo))
			vueltas++
		}
		if p.Paralelo > 1 && p.idUso != "" {
			marcas = append(marcas, fmt.Sprintf("∥ %d en paralelo", p.Paralelo))
		}
		linea := fmt.Sprintf("%s · **%s** · %s", hhmm(p.Hora), p.Tipo, limpiar(p.Resumen))
		if len(marcas) > 0 {
			linea += " — " + strings.Join(marcas, ", ")
		}
		if h, ok := hijos[p.idUso]; ok {
			linea += fmt.Sprintf(" → subagente **%s** (%s)", h.Rol, h.Tarea)
			delegados = append(delegados, h.Rol+": "+h.Tarea)
		}
		if p.Error {
			linea += " — error"
		}
		fmt.Fprintf(&b, "%s- %s\n", strings.Repeat("  ", p.Profundidad), linea)

		if p.Hilo != "" {
			u := usos[p.Hilo]
			if u == nil {
				u = &uso{}
				usos[p.Hilo] = u
				archivos = append(archivos, p.Hilo)
			}
			switch p.Tipo {
			case "Lee":
				u.lee++
			case "Edita":
				u.edita++
			case "Escribe":
				u.escribe++
			case "Busca":
				u.busca++
			}
		}
	}
	if len(archivos) > 0 {
		fmt.Fprintf(&b, "\n## Archivos\n\n")
		for _, f := range archivos {
			u := usos[f]
			var partes []string
			for _, x := range []struct {
				n    int
				verb string
			}{{u.lee, "leído"}, {u.busca, "buscado"}, {u.edita, "editado"}, {u.escribe, "escrito"}} {
				if x.n > 0 {
					partes = append(partes, fmt.Sprintf("%s %d", x.verb, x.n))
				}
			}
			fmt.Fprintf(&b, "- `%s`: %s\n", f, strings.Join(partes, ", "))
		}
	}

	resumen := []string{fmt.Sprintf("%d pasos de %s a %s", len(pasos), hhmm(inicio), hhmm(fin))}
	if len(etapas) > 0 {
		resumen = append(resumen, "etapas: "+strings.Join(etapas, "; "))
	}
	if desvios > 0 {
		resumen = append(resumen, plural(desvios, "desvío", "desvíos")+", "+plural(vueltas, "vuelta", "vueltas"))
	}
	if len(delegados) > 0 {
		resumen = append(resumen, "delega a "+strings.Join(delegados, "; "))
	}
	return Traza{
		Agente:   ag.ID,
		Proyecto: ag.Proyecto,
		Turno:    turno,
		Titulo:   secretos.Tapar(recortar(titulo, 160)),
		Resumen:  secretos.Tapar(recortar(strings.Join(resumen, " · "), 240)),
		Cuerpo:   secretos.Tapar(b.String()),
		Ultimo:   fin,
	}
}

func plural(n int, uno, varios string) string {
	if n == 1 {
		return "1 " + uno
	}
	return fmt.Sprintf("%d %s", n, varios)
}

func hhmm(t time.Time) string { return t.Local().Format("15:04") }

// limpiar evita que un resumen rompa la lista de Markdown (saltos de línea, asteriscos sueltos).
func limpiar(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
