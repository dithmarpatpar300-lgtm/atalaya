// Package proceso revisa a cada agente contra las reglas del centro de operaciones del proyecto: la asignación en la primera línea del encargo, el effort explícito, la cuota
// mirada antes de lanzar, la bitácora al 100 %, el acta cerrada y los temporales borrados.
//
// Solo lee: no corrige nada, dice dónde mirar. Cada falta lleva la ruta que la prueba, y lo que los archivos no
// dicen queda «sin dato», no se supone. Las reglas se citan de AGENTS.md (internal/alertas las lee); si el
// documento no escribe una, la regla lo dice.
package proceso

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"atalaya/internal/alertas"
)

// Marcas de una celda.
const (
	Cumple   = "cumple"
	Falta    = "falta"
	Medias   = "medias"
	NoAplica = "no_aplica"
	SinDato  = "sin_dato"
)

// Las columnas, en orden.
var columnas = []struct{ id, nombre, nota string }{
	{"asignacion", "Asignación", "La primera línea del encargo es «Asignación:» con nivel, clase, rol, modelo/effort, presupuesto y base."},
	{"effort", "Effort", "El effort se fija siempre explícito: en la asignación y en la llamada que lanza al agente."},
	{"cuota", "Cuota mirada", "Antes de lanzar, la sesión lee la cuota (Cuotas o get_usage) y la asignación lleva la postura."},
	{"bitacora", "Bitácora", "Cada agente lleva su bitácora y termina en «Avance: 100 %»."},
	{"acta", "Acta", "Toda carpeta temporal de un agente tiene su acta, y se cierra al borrarla."},
	{"temporales", "Temporales", "La carpeta temporal se borra al terminar."},
	{"frontera", "Frontera", "Nunca hay dos agentes sobre el mismo archivo a la vez (ni el coordinador sobre la frontera de su subagente)."},
}

// Columnas: las reglas revisadas, con su fuente en AGENTS.md.
func Columnas(r alertas.Reglas) []alertas.Regla {
	out := make([]alertas.Regla, 0, len(columnas))
	for _, c := range columnas {
		g := alertas.Regla{ID: c.id, Nombre: c.nombre, Nota: c.nota, Fuente: r.Fuente(c.id)}
		switch c.id {
		case "cuota":
			g.Umbral = fmt.Sprintf("%d min antes", int(r.Frescura.Minutes()))
			if r.Fuente("frescura") == "" {
				g.Nota += " La lectura vale los minutos del umbral (por omisión)."
			}
		case "temporales":
			g.Umbral = r.TmpRaiz
		}
		g.Omision = g.Fuente == ""
		out = append(out, g)
	}
	return out
}

// Agente: lo que la revisión necesita de uno, ya leído de sus archivos.
type Agente struct {
	ID        string
	Nombre    string // «medidor-w6» (la bitácora o el acta lo nombran mejor que el tipo de agente)
	Proveedor string // Claude | Codex
	Clase     string // «T1 · sonnet/medium»
	Tarea     string
	Estado    string
	Activo    bool
	Inicio    time.Time
	Ultimo    time.Time
	Fase      string

	// La asignación: en qué línea va (1 = la primera; 0 = no la trae) y sus campos. Prueba: el encargo.
	LineaAsignacion int
	Nivel           string
	ClaseAsignada   string
	Rol             string
	Modelo          string // «sonnet/medium»: modelo y effort de la asignación
	Presupuesto     string
	Base            string
	CuotaAsignada   string // la postura que lleva la asignación
	Encargo         string // dónde está el encargo (el transcrito o la ficha de Codex)

	// EffortLlamada: el effort con el que corrió (el .meta.json del subagente o la ficha de Codex).
	EffortLlamada string

	// El lanzamiento: Lanzado es cero si no se encontró quién lo lanzó. LanzadoEn y MiradaEn son la ruta del
	// transcrito de la sesión; el paso va aparte, en PasoLanzado y PasoMirada, y Sesion es el id de esa sesión.
	Lanzado     time.Time
	LanzadoEn   string
	PasoLanzado int
	Mirada      time.Time
	MiradaEn    string
	PasoMirada  int
	MiradaOrden string
	Sesion      string

	Bitacora string
	Avance   int // -1 si la bitácora no dice cuánto
	// BitacoraAjena: la bitácora de su nombre existe, pero se escribió después de que entregó (la retomó otro
	// agente): no se sabe en cuánto la dejó.
	BitacoraAjena string

	Acta       string
	ActaEstado string // abierta | cerrada

	// Temporales: las carpetas que usó (por sus pasos, su acta o el nombre fijo de codex.ps1).
	Temporales []string

	// Cruces: los archivos que editó a la vez que otro agente vivo (alertas.Cruces).
	Cruces []Cruce
}

// Cruce de un agente con otro sobre un archivo.
type Cruce struct {
	Otro        string    // el nombre del otro («Tienda Demo», la sesión; «Codificador I2»)
	OtroID      string    // su id
	Ruta        string    // el archivo
	Hora        time.Time // la última edición de este agente en el cruce
	HoraOtro    time.Time
	Coordinador bool // el otro es la sesión que lo lanzó
	Abierto     bool // los dos siguen trabajando
}

// Carpeta temporal en el disco.
type Carpeta struct {
	Ruta      string    `json:"ruta"`
	Nombre    string    `json:"nombre"`
	Lugar     string    `json:"lugar"`
	Bytes     int64     `json:"bytes"`
	Archivos  int       `json:"archivos"`
	Parciales int       `json:"parciales"` // .rgpart dentro
	Mod       time.Time `json:"mod"`
	// Lo que se sabe de ella: el acta con el mismo nombre y el agente que la usó.
	Acta         string `json:"acta,omitempty"`
	ActaEstado   string `json:"acta_estado,omitempty"`
	Agente       string `json:"agente,omitempty"`
	AgenteNombre string `json:"agente_nombre,omitempty"`
	Estado       string `json:"estado"` // en uso | agente entregado | sin agente
}

// Celda: una regla para un agente. Ruta es la prueba (un documento relativo a la raíz del proyecto, o un transcrito
// con «~»); si la prueba es un paso de un transcrito, Paso es su número y Agente el id de quien lo dio (-1 y vacío si no).
type Celda struct {
	Regla  string `json:"regla"`
	Marca  string `json:"marca"`
	Titulo string `json:"titulo"`
	Texto  string `json:"texto"`
	Ruta   string `json:"ruta,omitempty"`
	Paso   int    `json:"paso"`
	Agente string `json:"agente,omitempty"`
}

// enPaso: la celda apunta al paso n de la sesión que lanzó al agente.
func (c *Celda) enPaso(ruta string, n int, sesion string) {
	c.Ruta, c.Paso, c.Agente = ruta, n, ""
	if ruta != "" && sesion != "" {
		c.Agente = sesion
	} else {
		c.Paso = -1
	}
}

// Fila de la tabla.
type Fila struct {
	ID        string    `json:"id"`
	Nombre    string    `json:"nombre"`
	Proveedor string    `json:"proveedor"`
	Clase     string    `json:"clase"`
	Tarea     string    `json:"tarea"`
	Estado    string    `json:"estado"`
	Activo    bool      `json:"activo"`
	Inicio    time.Time `json:"inicio"`
	Ultimo    time.Time `json:"ultimo"`
	Fase      string    `json:"fase,omitempty"`
	Celdas    []Celda   `json:"celdas"`
	Faltas    int       `json:"faltas"`
	Medias    int       `json:"medias"`
}

// Informe de un proyecto.
type Informe struct {
	Reglas     []alertas.Regla `json:"reglas"`
	Lugares    []string        `json:"lugares"`
	Filas      []Fila          `json:"filas"`
	Temporales []Carpeta       `json:"temporales"`
}

func hhmm(t time.Time) string { return t.Local().Format("15:04") }

// Evaluar arma las filas y la lista de temporales. Las carpetas llegan medidas; se les añade el agente que las usó.
func Evaluar(r alertas.Reglas, ags []Agente, carpetas []Carpeta) ([]Fila, []Carpeta) {
	enDisco := map[string]int{}
	for i, c := range carpetas {
		enDisco[strings.ToLower(c.Nombre)] = i
	}
	// Una carpeta puede ser de dos agentes (un relanzamiento retoma la del que se cortó): manda el que sigue vivo,
	// y si ninguno, el primero que la nombra (los agentes llegan del más reciente al más viejo).
	for _, vivos := range []bool{true, false} {
		for _, a := range ags {
			if a.Activo != vivos {
				continue
			}
			for _, t := range a.Temporales {
				if i, ok := enDisco[strings.ToLower(t)]; ok && carpetas[i].Agente == "" {
					carpetas[i].Agente, carpetas[i].AgenteNombre = a.ID, a.Nombre
					carpetas[i].Estado = "agente entregado"
					if a.Activo {
						carpetas[i].Estado = "en uso"
					}
				}
			}
		}
	}
	filas := make([]Fila, 0, len(ags))
	for _, a := range ags {
		var cs []Celda
		if a.Proveedor == "Codex" {
			cs = []Celda{asignacionCodex(a), effort(a), cuota(r, a), noAplica("bitacora", "Codex no lleva bitácora: su informe y REGISTRO.md hacen ese papel."),
				noAplica("acta", "Codex no lleva acta: la línea de REGISTRO.md la reemplaza (codex.ps1 crea, usa y borra la carpeta)."), temporales(a, carpetas, enDisco),
				noAplica("frontera", "Codex trabaja en una copia aparte, de solo lectura: no edita los archivos del proyecto.")}
		} else {
			cs = []Celda{asignacion(a), effort(a), cuota(r, a), bitacora(a), acta(a), temporales(a, carpetas, enDisco), frontera(a)}
		}
		// Una regla de proceso que el AGENTS.md del proyecto no escribe no se le exige (las reglas se leen, no se
		// copian). La frontera sí, siempre, como su alerta: pisarse el trabajo es un riesgo técnico, no una costumbre.
		for i, c := range cs {
			if c.Regla != "frontera" && c.Marca != NoAplica && r.Fuente(c.Regla) == "" {
				cs[i] = noAplica(c.Regla, "El AGENTS.md del proyecto no escribe esta regla: no se le exige.")
			}
			// Solo la cuota prueba con un paso (enPaso); las demás celdas, con un documento o ninguno.
			if cs[i].Agente == "" {
				cs[i].Paso = -1
			}
		}
		f := Fila{ID: a.ID, Nombre: a.Nombre, Proveedor: a.Proveedor, Clase: a.Clase, Tarea: a.Tarea, Estado: a.Estado, Activo: a.Activo,
			Inicio: a.Inicio, Ultimo: a.Ultimo, Fase: a.Fase, Celdas: cs}
		for _, c := range cs {
			switch c.Marca {
			case Falta:
				f.Faltas++
			case Medias:
				f.Medias++
			}
		}
		filas = append(filas, f)
	}
	for i := range carpetas {
		if carpetas[i].Estado == "" {
			carpetas[i].Estado = "sin agente"
		}
	}
	sort.SliceStable(filas, func(i, j int) bool { return filas[i].Inicio.After(filas[j].Inicio) })
	return filas, carpetas
}

func noAplica(regla, texto string) Celda {
	return Celda{Regla: regla, Marca: NoAplica, Titulo: "No aplica", Texto: texto}
}

func asignacion(a Agente) Celda {
	c := Celda{Regla: "asignacion", Ruta: a.Encargo}
	if a.LineaAsignacion == 0 {
		c.Marca, c.Titulo, c.Texto = Falta, "Sin asignación", "El encargo no trae la línea «Asignación:»."
		return c
	}
	var faltan []string
	for _, x := range []struct{ nombre, valor string }{{"nivel", a.Nivel}, {"clase", a.ClaseAsignada}, {"rol", a.Rol},
		{"modelo/effort", a.Modelo}, {"presupuesto", a.Presupuesto}, {"base", a.Base}} {
		if strings.TrimSpace(x.valor) == "" {
			faltan = append(faltan, x.nombre)
		}
	}
	switch {
	case a.LineaAsignacion > 1 && len(faltan) > 0:
		c.Marca, c.Titulo = Medias, "Asignación incompleta y fuera de lugar"
		c.Texto = fmt.Sprintf("Va en la línea %d, no en la primera, y le falta: %s.", a.LineaAsignacion, strings.Join(faltan, ", "))
	case a.LineaAsignacion > 1:
		c.Marca, c.Titulo = Medias, "Asignación fuera de lugar"
		c.Texto = fmt.Sprintf("Va en la línea %d del encargo, no en la primera.", a.LineaAsignacion)
	case len(faltan) > 0:
		c.Marca, c.Titulo = Medias, "Asignación incompleta"
		c.Texto = "Le falta: " + strings.Join(faltan, ", ") + "."
	default:
		c.Marca, c.Titulo, c.Texto = Cumple, "Asignación completa", "En la primera línea, con todos sus campos."
	}
	return c
}

func asignacionCodex(a Agente) Celda {
	c := Celda{Regla: "asignacion", Ruta: a.Encargo}
	var faltan []string
	for _, x := range []struct{ nombre, valor string }{{"clase", a.ClaseAsignada}, {"modelo/effort", a.Modelo}, {"base", a.Base}} {
		if strings.TrimSpace(x.valor) == "" {
			faltan = append(faltan, x.nombre)
		}
	}
	if len(faltan) > 0 {
		c.Marca, c.Titulo, c.Texto = Medias, "Ficha incompleta", "La ficha de la llamada no dice: "+strings.Join(faltan, ", ")+"."
		return c
	}
	c.Marca, c.Titulo, c.Texto = Cumple, "Ficha completa", "codex.ps1 escribe clase, modelo, effort y base en la ficha de la llamada."
	return c
}

// effortDe: lo que va tras la barra en «sonnet/medium».
func effortDe(modelo string) string {
	if _, e, ok := strings.Cut(modelo, "/"); ok {
		return strings.TrimSpace(e)
	}
	return ""
}

func effort(a Agente) Celda {
	c := Celda{Regla: "effort", Ruta: a.Encargo}
	asig, llamada := effortDe(a.Modelo), strings.TrimSpace(a.EffortLlamada)
	switch {
	case asig == "" && llamada == "":
		c.Marca, c.Titulo, c.Texto = Falta, "Effort sin fijar", "Ni la asignación ni la llamada que lo lanzó fijan el effort: corrió con el de omisión."
	case asig != "" && llamada == "":
		c.Marca, c.Titulo = Medias, "Effort solo en el papel"
		c.Texto = "La asignación dice «" + asig + "», pero la llamada no registró effort: pudo correr con el de omisión."
	case asig != "" && !strings.EqualFold(asig, llamada):
		c.Marca, c.Titulo = Medias, "Effort distinto del asignado"
		c.Texto = "La asignación dice «" + asig + "» y corrió con «" + llamada + "»."
	default:
		c.Marca, c.Titulo, c.Texto = Cumple, "Effort explícito", "Corrió con «"+llamada+"»."
	}
	return c
}

func cuota(r alertas.Reglas, a Agente) Celda {
	c := Celda{Regla: "cuota"}
	codex := a.Proveedor == "Codex"
	postura := codex || a.CuotaAsignada != "" // codex.ps1 no lleva postura: basta con la lectura
	if a.Lanzado.IsZero() {
		c.Marca, c.Titulo, c.Ruta = SinDato, "Sin dato", a.Encargo
		c.Texto = "No está el transcrito de la sesión que lo lanzó: no se puede ver si miró la cuota antes."
		if a.CuotaAsignada != "" {
			c.Texto = "La asignación trae la postura («" + a.CuotaAsignada + "»), pero no está el transcrito de la sesión que lo lanzó para ver cuándo se leyó."
		}
		return c
	}
	reciente := !a.Mirada.IsZero() && !a.Mirada.After(a.Lanzado) && a.Lanzado.Sub(a.Mirada) <= r.Frescura
	tope := int(r.Frescura.Minutes())
	switch {
	case reciente && postura:
		c.Marca, c.Titulo = Cumple, "Cuota mirada"
		c.enPaso(a.MiradaEn, a.PasoMirada, a.Sesion)
		c.Texto = fmt.Sprintf("Leída a las %s, %s antes de lanzarlo (%s).", hhmm(a.Mirada), minutos(a.Lanzado.Sub(a.Mirada)), hhmm(a.Lanzado))
	case reciente:
		c.Marca, c.Titulo = Medias, "Postura sin anotar"
		c.enPaso(a.MiradaEn, a.PasoMirada, a.Sesion)
		c.Texto = fmt.Sprintf("La sesión leyó la cuota a las %s, pero la asignación no lleva la postura.", hhmm(a.Mirada))
	case a.Mirada.IsZero():
		c.enPaso(a.LanzadoEn, a.PasoLanzado, a.Sesion)
		if postura && !codex {
			c.Marca, c.Titulo = Medias, "Cuota sin leer en la sesión"
			c.Texto = "La asignación trae la postura, pero la sesión no corrió Cuotas ni get_usage antes de lanzarlo (" + hhmm(a.Lanzado) + ")."
		} else {
			c.Marca, c.Titulo = Falta, "Cuota sin mirar"
			c.Texto = "La sesión no corrió Cuotas ni get_usage antes de lanzarlo (" + hhmm(a.Lanzado) + ")."
		}
	default:
		c.enPaso(a.MiradaEn, a.PasoMirada, a.Sesion)
		if c.Ruta == "" {
			c.enPaso(a.LanzadoEn, a.PasoLanzado, a.Sesion)
		}
		vieja := fmt.Sprintf("Entre la última lectura (%s) y el lanzamiento (%s) pasaron %s; la regla pide leerla en los %d min antes.",
			hhmm(a.Mirada), hhmm(a.Lanzado), minutos(a.Lanzado.Sub(a.Mirada)), tope)
		if postura && !codex {
			c.Marca, c.Titulo, c.Texto = Medias, "Cuota vieja al lanzar", vieja
		} else {
			c.Marca, c.Titulo, c.Texto = Falta, "Cuota sin mirar", vieja
		}
	}
	return c
}

func minutos(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d s", int(d.Seconds()))
	}
	return fmt.Sprintf("%d min", int(d.Minutes()))
}

func bitacora(a Agente) Celda {
	c := Celda{Regla: "bitacora", Ruta: a.Bitacora}
	switch {
	case a.Bitacora == "" && a.BitacoraAjena != "":
		c.Marca, c.Titulo, c.Ruta = SinDato, "Sin dato", a.BitacoraAjena
		c.Texto = "Su bitácora se volvió a escribir después de que entregó (la retomó otro agente): no se sabe en cuánto la dejó."
	case a.Bitacora == "":
		c.Marca, c.Titulo, c.Ruta = Falta, "Sin bitácora", a.Encargo
		c.Texto = "Ni el encargo nombra una bitácora ni hay una de su rol escrita mientras trabajaba."
	case a.Avance == 100:
		c.Marca, c.Titulo, c.Texto = Cumple, "Bitácora al 100 %", "Termina en «Avance: 100 %»."
	case a.Avance < 0:
		c.Marca, c.Titulo, c.Texto = Falta, "Bitácora sin avance", "No empieza con «Avance: NN %»."
	case a.Activo:
		c.Marca, c.Titulo, c.Texto = Medias, fmt.Sprintf("Bitácora en %d %%", a.Avance), "Sigue trabajando: se revisa otra vez cuando entregue."
	default:
		c.Marca, c.Titulo, c.Texto = Falta, fmt.Sprintf("Bitácora en %d %%", a.Avance), fmt.Sprintf("Entregó con la bitácora en %d %%, no en 100 %%.", a.Avance)
	}
	return c
}

func acta(a Agente) Celda {
	c := Celda{Regla: "acta", Ruta: a.Acta}
	switch {
	case a.Acta == "" && len(a.Temporales) > 0:
		c.Marca, c.Titulo, c.Ruta = Falta, "Sin acta", a.Encargo
		c.Texto = "Usó " + strings.Join(a.Temporales, ", ") + " y no tiene acta en temporales/."
	case a.Acta == "":
		return noAplica("acta", "No usó carpetas temporales (ninguno de sus pasos las nombra) ni tiene acta.")
	case a.ActaEstado == "cerrada":
		c.Marca, c.Titulo, c.Texto = Cumple, "Acta cerrada", "Ninguna sección queda «(en curso)» ni «(pendiente)»."
	case a.Activo:
		c.Marca, c.Titulo, c.Texto = Medias, "Acta abierta", "Normal mientras trabaja: se revisa otra vez cuando entregue."
	default:
		c.Marca, c.Titulo, c.Texto = Falta, "Acta sin cerrar", "Entregó y el acta sigue con secciones «(en curso)»."
	}
	return c
}

func temporales(a Agente, carpetas []Carpeta, enDisco map[string]int) Celda {
	c := Celda{Regla: "temporales"}
	for _, t := range a.Temporales {
		i, ok := enDisco[strings.ToLower(t)]
		if !ok {
			continue
		}
		d := carpetas[i]
		c.Ruta = d.Ruta
		peso := fmt.Sprintf("%s, %s", plural(d.Archivos, "archivo", "archivos"), Peso(d.Bytes))
		switch {
		case a.Activo:
			c.Marca, c.Titulo, c.Texto = Medias, "Temporal en uso", "Sigue trabajando ("+peso+")."
		case d.Estado == "en uso" && d.Agente != a.ID:
			c.Marca, c.Titulo = Medias, "Temporal retomado"
			c.Texto = "La carpeta sigue en el disco, pero la retomó otro agente (" + d.AgenteNombre + ") que sigue trabajando (" + peso + "): se revisa otra vez cuando entregue."
		default:
			c.Marca, c.Titulo, c.Texto = Falta, "Temporal en el disco", "La carpeta sigue ahí después de entregar ("+peso+")."
		}
		return c
	}
	if len(a.Temporales) == 0 {
		return noAplica("temporales", "No usó carpetas temporales.")
	}
	c.Marca, c.Titulo, c.Ruta = Cumple, "Temporal borrado", a.Acta
	c.Texto = strings.Join(a.Temporales, ", ") + " ya no está en el disco."
	if c.Ruta == "" {
		c.Ruta = a.Encargo
	}
	return c
}

// frontera: falta si se cruzó con otro agente en paralelo; a medias si solo con su coordinador, que puede verlo y
// decidir. La alerta del cruce se disipa; esta marca queda como registro.
func frontera(a Agente) Celda {
	c := Celda{Regla: "frontera"}
	if len(a.Cruces) == 0 {
		c.Marca, c.Titulo, c.Texto = Cumple, "Sin cruces", "Ningún archivo que editó lo editó otro agente a la vez."
		return c
	}
	conOtro := false
	var partes []string
	for _, x := range a.Cruces {
		if !x.Coordinador {
			conOtro = true
		}
		estado := "cerrado"
		if x.Abierto {
			estado = "abierto"
		}
		partes = append(partes, fmt.Sprintf("%s con %s (%s y %s, %s)", x.Ruta, x.Otro, hhmm(x.Hora), hhmm(x.HoraOtro), estado))
	}
	c.Ruta = a.Cruces[0].Ruta
	c.Texto = "Editó a la vez que otro: " + strings.Join(partes, "; ") + "."
	if conOtro {
		c.Marca, c.Titulo = Falta, "Archivo compartido con otro agente"
	} else {
		c.Marca, c.Titulo = Medias, "Archivo compartido con su coordinador"
		c.Texto += " El coordinador no debe editar la frontera de su subagente mientras trabaja."
	}
	return c
}

func plural(n int, uno, varios string) string {
	if n == 1 {
		return "1 " + uno
	}
	return fmt.Sprintf("%d %s", n, varios)
}

// Peso con coma decimal: «41,0 MB».
func Peso(b int64) string {
	switch {
	case b < 1024:
		return fmt.Sprintf("%d B", b)
	case b < 1024*1024:
		return strings.Replace(fmt.Sprintf("%.1f KB", float64(b)/1024), ".", ",", 1)
	case b < 1024*1024*1024:
		return strings.Replace(fmt.Sprintf("%.1f MB", float64(b)/(1024*1024)), ".", ",", 1)
	}
	return strings.Replace(fmt.Sprintf("%.2f GB", float64(b)/(1024*1024*1024)), ".", ",", 1)
}
