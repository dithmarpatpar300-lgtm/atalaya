package claude

import (
	"errors"
	"sort"
	"time"
)

// El contexto de un agente, petición por petición, como lo cuenta la API: lo que ocupó cada petición (entrada sin
// caché + caché escrita + caché leída), su salida, y cada compactación con el antes y el después. El reparto interno
// de Claude Code (herramientas del sistema, MCP, memoria, skills) no queda en el transcrito: aquí no se inventa.

// Peticion: un mensaje de Claude con su usage.
type Peticion struct {
	Hora         time.Time `json:"hora"`
	Turno        int       `json:"turno"`
	Etapa        string    `json:"etapa"`
	Entrada      int64     `json:"entrada"`
	CacheEscrita int64     `json:"cache_escrita"`
	CacheLeida   int64     `json:"cache_leida"`
	Salida       int64     `json:"salida"`
	Contexto     int64     `json:"contexto"`
	// Resumen: el primer paso del mensaje, para reconocer la petición.
	Resumen string `json:"resumen"`
	lote    int
}

// Compactacion: Claude Code resumió la conversación para liberar contexto.
type Compactacion struct {
	Hora    time.Time `json:"hora"`
	Antes   int64     `json:"antes"`
	Despues int64     `json:"despues"`
	Motivo  string    `json:"motivo"` // auto | manual
	// Peticion: el índice de la primera petición después de la compactación.
	Peticion int `json:"peticion"`
}

// Llenado: cuánto metió al contexto un tipo de cosa.
type Llenado struct {
	Categoria string `json:"categoria"`
	Tokens    int64  `json:"tokens"`
}

// Pesado: uno de los resultados que más contexto ocupó.
type Pesado struct {
	N       int       `json:"n"`
	Hora    time.Time `json:"hora"`
	Tipo    string    `json:"tipo"`
	Resumen string    `json:"resumen"`
	Tokens  int64     `json:"tokens"`
}

// Contexto de un agente.
type Contexto struct {
	Ventana        int64          `json:"ventana"` // inferida: 1 M si alguna petición pasó de 200 k
	Peticiones     []Peticion     `json:"peticiones"`
	Compactaciones []Compactacion `json:"compactaciones"`
	Lleno          []Llenado      `json:"lleno"`
	Pesados        []Pesado       `json:"pesados"`
	Totales        Tokens         `json:"totales"`
}

// MaxPeticiones: lo más que se devuelve de la serie (las últimas). Una sesión larga pasa de mil.
const MaxPeticiones = 1500

func (a *archivo) anotarPeticion(id string, hora time.Time, u Tokens) {
	if a.porPeticion == nil {
		a.porPeticion = map[string]int{}
	}
	p := Peticion{Hora: hora, Turno: a.turno, Etapa: a.etapa(), Entrada: u.Entrada, CacheEscrita: u.CacheEscrita, CacheLeida: u.CacheLeida,
		Salida: u.Salida, Contexto: u.Entrada + u.CacheEscrita + u.CacheLeida, lote: a.lote}
	if i, ok := a.porPeticion[id]; ok {
		// El mismo mensaje en otra línea: el usage crece (la salida), la hora y el resumen se quedan.
		p.Hora, p.Resumen = a.peticiones[i].Hora, a.peticiones[i].Resumen
		a.peticiones[i] = p
		return
	}
	a.porPeticion[id] = len(a.peticiones)
	a.peticiones = append(a.peticiones, p)
}

// categoria agrupa los pasos por lo que meten al contexto.
func categoria(tipo string) string {
	switch tipo {
	case "Lee":
		return "Leer archivos"
	case "Ejecuta":
		return "Órdenes (Bash, PowerShell)"
	case "Busca":
		return "Buscar (Grep, Glob)"
	case "Edita", "Escribe":
		return "Editar y escribir"
	case "Delega":
		return "Informes de subagentes"
	case "Consulta":
		return "Consultar la web"
	}
	return "Otras herramientas"
}

// ErrSinAgente: no hay agente de Claude con ese id.
var ErrSinAgente = errors.New("no existe ese agente")

// Contexto arma el contexto del agente `id`.
func (ix *Indice) Contexto(id string) (Contexto, error) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	var a *archivo
	for _, x := range ix.arch {
		if x.agente.ID == id {
			a = x
			break
		}
	}
	if a == nil {
		return Contexto{}, ErrSinAgente
	}
	c := Contexto{Ventana: 200_000, Totales: a.agente.Tokens, Compactaciones: append([]Compactacion{}, a.compactaciones...)}

	// El resumen de cada petición: el primer paso de su lote (el mensaje que la API devolvió).
	primerPaso := map[int]string{}
	for _, p := range a.pasos {
		if _, ok := primerPaso[p.Lote]; !ok && p.Tipo != "Pide" {
			primerPaso[p.Lote] = p.Tipo + " · " + p.Resumen
		}
	}

	desde := max(0, len(a.peticiones)-MaxPeticiones)
	c.Peticiones = make([]Peticion, 0, len(a.peticiones)-desde)
	for _, p := range a.peticiones[desde:] {
		if p.Contexto > 200_000 {
			c.Ventana = 1_000_000
		}
		if p.Resumen == "" {
			p.Resumen = recortar(primerPaso[p.lote], 120)
		}
		c.Peticiones = append(c.Peticiones, p)
	}
	for i := range c.Compactaciones {
		c.Compactaciones[i].Peticion -= desde
		if c.Compactaciones[i].Antes > 200_000 {
			c.Ventana = 1_000_000
		}
	}

	// Qué llenó el contexto: los resultados de las herramientas por categoría, y lo que generó la IA (que entra
	// como contexto de la petición siguiente).
	por := map[string]int64{}
	var pesados []Pesado
	for _, p := range a.pasos {
		if p.TokEntra > 0 {
			por[categoria(p.Tipo)] += p.TokEntra
			pesados = append(pesados, Pesado{N: p.N, Hora: p.Hora, Tipo: p.Tipo, Resumen: p.Resumen, Tokens: p.TokEntra})
		}
	}
	por["Respuestas y llamadas de la IA"] = a.agente.Tokens.Salida
	for k, v := range por {
		if v > 0 {
			c.Lleno = append(c.Lleno, Llenado{Categoria: k, Tokens: v})
		}
	}
	sort.Slice(c.Lleno, func(i, j int) bool { return c.Lleno[i].Tokens > c.Lleno[j].Tokens })
	sort.SliceStable(pesados, func(i, j int) bool { return pesados[i].Tokens > pesados[j].Tokens })
	if len(pesados) > 5 {
		pesados = pesados[:5]
	}
	c.Pesados = pesados
	return c, nil
}
