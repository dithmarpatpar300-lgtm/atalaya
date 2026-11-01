package claude

import (
	"encoding/json"
	"strings"
)

// La continuidad de la actividad: qué pasos siguen el mismo hilo, cuándo el agente se desvía a otra cosa y
// cuándo vuelve, qué llamadas salieron en paralelo de un mismo mensaje y en qué paso del plan va.
//
// Nada de esto se adivina del texto: sale de lo que el transcrito ya guarda.
//   - Turno: cada pedido de la persona abre uno.
//   - Etapa: el paso del plan en curso, de update_plan (la app de escritorio) o de TodoWrite.
//   - Lote: las líneas del transcrito con el mismo message.id son un solo mensaje de Claude; si trae varias
//     herramientas, salieron en paralelo (una bifurcación que se une cuando vuelven todos los resultados).
//   - Hilo: el archivo que la herramienta lee, busca, edita o escribe.
//   - Desvío y vuelta: si un hilo reaparece dentro del mismo turno tras pasos de otros hilos, esos pasos fueron un
//     desvío (se investigó o se probó otra cosa) y el paso que reaparece continúa al anterior del mismo hilo.

// planificar actualiza el plan con una llamada a update_plan o TodoWrite. Dice si la herramienta era de plan.
func (a *archivo) planificar(herr string, input json.RawMessage) bool {
	type item struct {
		Content string `json:"content"`
		Status  string `json:"status"`
	}
	switch {
	case strings.HasSuffix(herr, "update_plan"):
		var in struct {
			Items   []item `json:"items"`
			Advance bool   `json:"advance"`
			Done    []int  `json:"done"`
			Start   int    `json:"start"`
		}
		if json.Unmarshal(input, &in) != nil {
			return true
		}
		if len(in.Items) > 0 {
			a.plan, a.planActual = nil, -1
			for i, it := range in.Items {
				a.plan = append(a.plan, it.Content)
				if it.Status == "in_progress" && a.planActual < 0 {
					a.planActual = i
				}
			}
		} else if in.Start > 0 {
			a.planActual = in.Start - 1
		} else if in.Advance {
			a.planActual++
		}
		return true
	case herr == "TodoWrite":
		var in struct {
			Todos []item `json:"todos"`
		}
		if json.Unmarshal(input, &in) != nil {
			return true
		}
		a.plan, a.planActual = nil, -1
		for i, it := range in.Todos {
			a.plan = append(a.plan, it.Content)
			if it.Status == "in_progress" && a.planActual < 0 {
				a.planActual = i
			}
		}
		return true
	}
	return false
}

func (a *archivo) etapa() string {
	if a.planActual >= 0 && a.planActual < len(a.plan) {
		return a.plan[a.planActual]
	}
	return ""
}

// hiloDe: de qué archivo trata una llamada. Las órdenes de consola no tienen hilo propio: siguen al que esté en curso.
func hiloDe(herr string, input json.RawMessage) string {
	var m map[string]any
	if json.Unmarshal(input, &m) != nil {
		return ""
	}
	cad := func(k string) string { s, _ := m[k].(string); return s }
	var ruta string
	switch herr {
	case "Read", "Write", "Edit", "MultiEdit", "NotebookEdit", "NotebookRead":
		ruta = cad("file_path")
		if ruta == "" {
			ruta = cad("notebook_path")
		}
	case "Grep", "Glob":
		ruta = cad("path")
	}
	if ruta == "" {
		return ""
	}
	return corta(ruta)
}

// MaxDesvio: un hilo que vuelve tras más pasos que esto ya no se cuenta como desvío, sino como otra visita.
const MaxDesvio = 30

// enlazar calcula, sobre todos los pasos de un agente, el paralelo de cada lote, la profundidad de los desvíos y
// a qué paso continúa cada vuelta. Recibe y modifica copias.
func enlazar(pasos []Paso) {
	porLote := map[int]int{}
	for _, p := range pasos {
		if p.idUso != "" {
			porLote[p.Lote]++
		}
	}
	ultimo := map[string]int{} // hilo -> último índice en el turno
	turno := -1
	for i := range pasos {
		p := &pasos[i]
		p.Paralelo = max(1, porLote[p.Lote])
		if p.Turno != turno {
			turno = p.Turno
			clear(ultimo)
		}
		if p.Hilo == "" {
			continue
		}
		previo, hay := ultimo[p.Hilo]
		ultimo[p.Hilo] = i
		if !hay || previo == i-1 || i-previo > MaxDesvio {
			continue
		}
		// ¿Hubo otro hilo en medio? Si solo hubo pasos sin hilo (órdenes, respuestas), es el mismo camino.
		otro := false
		for k := previo + 1; k < i; k++ {
			if pasos[k].Hilo != "" && pasos[k].Hilo != p.Hilo {
				otro = true
				break
			}
		}
		if !otro {
			continue
		}
		p.Continua = pasos[previo].N
		for k := previo + 1; k < i; k++ {
			if pasos[k].Tipo != "Pide" {
				pasos[k].Profundidad = min(3, pasos[k].Profundidad+1)
			}
		}
	}
	// El contexto tras cada paso: el de la petición del lote siguiente, que ya lleva su resultado. El último lote
	// todavía no tiene petición siguiente: se estima con lo suyo más lo que su resultado metió.
	ctxLote := map[int]int64{}
	var orden []int
	for _, p := range pasos {
		if _, visto := ctxLote[p.Lote]; !visto {
			orden = append(orden, p.Lote)
		}
		if ctxLote[p.Lote] == 0 {
			ctxLote[p.Lote] = p.TokContexto
		}
	}
	siguiente := map[int]int64{}
	var prox int64
	for k := len(orden) - 1; k >= 0; k-- {
		siguiente[orden[k]] = prox
		if c := ctxLote[orden[k]]; c > 0 {
			prox = c
		}
	}
	for i := range pasos {
		p := &pasos[i]
		if s := siguiente[p.Lote]; s > 0 {
			p.TokTras = s
		} else if p.TokContexto > 0 {
			p.TokTras = p.TokContexto + p.TokSalida + p.TokEntra
		}
	}
	// Un desvío empieza donde la profundidad sube.
	for i := range pasos {
		antes := 0
		if i > 0 {
			antes = pasos[i-1].Profundidad
		}
		pasos[i].Desvio = pasos[i].Profundidad > antes
	}
}
