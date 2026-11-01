package servidor

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"atalaya/internal/secretos"
)

// Las preguntas que una sesión le hace al dueño del proyecto por Atalaya: la sesión llama a `preguntar` del
// MCP y espera; el dueño del proyecto responde en el panel por pasos, con texto libre e imágenes; Atalaya le devuelve todo junto.
// Es la única puerta por la que el panel envía algo: solo a la memoria de Atalaya, solo desde este equipo y solo con la
// cabecera del panel. Las imágenes viven en memoria hasta que la sesión las recibe; después se sueltan.

// Límites.
const (
	maxPreguntas      = 8
	maxOpciones       = 8
	maxImagenes       = 4
	maxBytesImagen    = 5 << 20
	maxCuerpoRespuest = 26 << 20
	maxGuardadas      = 50
	// esperaMCP: cuánto espera `preguntar` en una llamada. Menos de los 5 min sin actividad con que Claude Code corta un
	// servidor HTTP: si no hay respuesta, devuelve el id para volver a llamar.
	esperaMCP  = 4 * time.Minute
	latidoMCP  = 20 * time.Second
	cabPanel   = "X-Atalaya-Panel"
	prefijoDec = "atalaya~"
)

// Estados de una pregunta de Atalaya.
const (
	PregEsperando = "esperando"
	PregRespond   = "respondida"
	PregEntregada = "entregada" // la sesión ya la recibió
)

type opcionP struct {
	Etiqueta    string   `json:"etiqueta"`
	Nota        string   `json:"nota,omitempty"`
	Recomendada bool     `json:"recomendada,omitempty"`
	Abre        []string `json:"abre,omitempty"` // ids de preguntas que esta opción abre
}

type dependeP struct {
	Pregunta string `json:"pregunta"`
	Opcion   string `json:"opcion"`
}

type preguntaP struct {
	ID         string    `json:"id"`
	Texto      string    `json:"texto"`
	Encabezado string    `json:"encabezado,omitempty"`
	Multiple   bool      `json:"multiple,omitempty"`
	Opciones   []opcionP `json:"opciones"`
	// Depende: solo se pregunta si en esa pregunta se eligió esa opción.
	Depende *dependeP `json:"depende,omitempty"`
}

type imagenP struct {
	Nombre string `json:"nombre"`
	Tipo   string `json:"tipo"`
	Bytes  int    `json:"bytes"`
	datos  []byte
}

type respuestaP struct {
	Elecciones map[string][]string `json:"elecciones"`
	Textos     map[string]string   `json:"textos,omitempty"`
	Nota       string              `json:"nota,omitempty"`
	Imagenes   []imagenP           `json:"imagenes,omitempty"`
	Hora       time.Time           `json:"hora"`
}

// PreguntaCentro: lo que una sesión preguntó por el MCP y, si ya está, la respuesta.
type PreguntaCentro struct {
	ID        string      `json:"id"`
	Proyecto  string      `json:"proyecto"`
	Cliente   string      `json:"cliente"`
	Titulo    string      `json:"titulo"`
	Contexto  string      `json:"contexto,omitempty"`
	Preguntas []preguntaP `json:"preguntas"`
	Creada    time.Time   `json:"creada"`
	Estado    string      `json:"estado"`
	Respuesta *respuestaP `json:"respuesta,omitempty"`
	listo     chan struct{}
}

type almacenPreguntas struct {
	mu    sync.Mutex
	por   map[string]*PreguntaCentro
	orden []string
	n     int
}

func (al *almacenPreguntas) guardar(p *PreguntaCentro, ahora time.Time) {
	al.mu.Lock()
	defer al.mu.Unlock()
	if al.por == nil {
		al.por = map[string]*PreguntaCentro{}
	}
	al.n++
	p.ID = ahora.Format("150405") + "-" + strconv.Itoa(al.n)
	p.listo = make(chan struct{})
	al.por[p.ID] = p
	al.orden = append(al.orden, p.ID)
	// Se olvidan las más viejas ya entregadas.
	for len(al.orden) > maxGuardadas {
		viejo := al.por[al.orden[0]]
		if viejo != nil && viejo.Estado == PregEsperando {
			break
		}
		delete(al.por, al.orden[0])
		al.orden = al.orden[1:]
	}
}

func (al *almacenPreguntas) buscar(id string) *PreguntaCentro {
	al.mu.Lock()
	defer al.mu.Unlock()
	return al.por[id]
}

// copias de todas, sin los bytes de las imágenes, de la más nueva a la más vieja.
func (al *almacenPreguntas) todas() []PreguntaCentro {
	al.mu.Lock()
	defer al.mu.Unlock()
	out := make([]PreguntaCentro, 0, len(al.por))
	for _, p := range al.por {
		c := *p
		c.listo = nil
		if p.Respuesta != nil {
			r := *p.Respuesta
			r.Imagenes = append([]imagenP(nil), p.Respuesta.Imagenes...)
			for i := range r.Imagenes {
				r.Imagenes[i].datos = nil
			}
			c.Respuesta = &r
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Creada.After(out[j].Creada) })
	return out
}

// argsPreguntar: lo que manda la sesión. Con `id`, sigue esperando una pregunta ya hecha.
type argsPreguntar struct {
	ID        string `json:"id"`
	Proyecto  string `json:"proyecto"`
	Titulo    string `json:"titulo"`
	Contexto  string `json:"contexto"`
	Preguntas []struct {
		ID         string    `json:"id"`
		Texto      string    `json:"texto"`
		Encabezado string    `json:"encabezado"`
		Multiple   bool      `json:"multiple"`
		Opciones   []opcionP `json:"opciones"`
	} `json:"preguntas"`
}

// armarPregunta valida lo que manda la sesión y convierte el «abre» de cada opción en el «depende» de la pregunta
// abierta. Una opción solo puede abrir preguntas que vienen después: así no hay ciclos.
func armarPregunta(a argsPreguntar) (*PreguntaCentro, error) {
	if strings.TrimSpace(a.Titulo) == "" {
		return nil, errors.New("falta «titulo»: una línea con lo que se decide")
	}
	if len(a.Preguntas) == 0 || len(a.Preguntas) > maxPreguntas {
		return nil, fmt.Errorf("«preguntas» lleva de 1 a %d preguntas", maxPreguntas)
	}
	p := &PreguntaCentro{Proyecto: a.Proyecto, Titulo: tapado(a.Titulo, 200), Contexto: tapado(a.Contexto, 2000), Estado: PregEsperando}
	pos := map[string]int{}
	for i, q := range a.Preguntas {
		id := strings.TrimSpace(q.ID)
		if id == "" {
			id = "p" + strconv.Itoa(i+1)
		}
		if _, ya := pos[id]; ya {
			return nil, fmt.Errorf("el id «%s» está dos veces", id)
		}
		if strings.TrimSpace(q.Texto) == "" {
			return nil, fmt.Errorf("la pregunta «%s» no tiene texto", id)
		}
		if len(q.Opciones) > maxOpciones {
			return nil, fmt.Errorf("la pregunta «%s» tiene más de %d opciones", id, maxOpciones)
		}
		pos[id] = i
		nq := preguntaP{ID: id, Texto: tapado(q.Texto, 600), Encabezado: tapado(q.Encabezado, 40), Multiple: q.Multiple}
		for _, o := range q.Opciones {
			o.Etiqueta = tapado(o.Etiqueta, 120)
			if o.Etiqueta == "" {
				return nil, fmt.Errorf("la pregunta «%s» tiene una opción vacía", id)
			}
			for _, vista := range nq.Opciones {
				if strings.EqualFold(strings.TrimSpace(vista.Etiqueta), strings.TrimSpace(o.Etiqueta)) {
					return nil, fmt.Errorf("la pregunta «%s» tiene la etiqueta repetida «%s»", id, o.Etiqueta)
				}
			}
			o.Nota = tapado(o.Nota, 300)
			if !o.Recomendada && esRecomendadaTexto(o.Etiqueta) {
				o.Recomendada = true
			}
			nq.Opciones = append(nq.Opciones, o)
		}
		p.Preguntas = append(p.Preguntas, nq)
	}
	for i, q := range p.Preguntas {
		for _, o := range q.Opciones {
			for _, abre := range o.Abre {
				j, ok := pos[abre]
				if !ok || j <= i {
					return nil, fmt.Errorf("la opción «%s» abre «%s», que no es una pregunta posterior", o.Etiqueta, abre)
				}
				if p.Preguntas[j].Depende != nil {
					return nil, fmt.Errorf("la pregunta «%s» la abren dos opciones; solo puede depender de una", abre)
				}
				p.Preguntas[j].Depende = &dependeP{Pregunta: q.ID, Opcion: o.Etiqueta}
			}
		}
	}
	return p, nil
}

func esRecomendadaTexto(e string) bool {
	e = strings.ToLower(e)
	return strings.Contains(e, "(recomendado)") || strings.Contains(e, "(recomendada)")
}

func tapado(s string, max int) string {
	s = strings.TrimSpace(s)
	if len([]rune(s)) > max {
		s = string([]rune(s)[:max])
	}
	return secretos.Tapar(s)
}

// visibles: las preguntas que se hacen con estas elecciones (las que dependen de una opción no elegida, no).
func visibles(qs []preguntaP, elec map[string][]string) []preguntaP {
	ver := map[string]bool{}
	var out []preguntaP
	for _, q := range qs {
		if q.Depende != nil {
			if !ver[q.Depende.Pregunta] || !contiene(elec[q.Depende.Pregunta], q.Depende.Opcion) {
				continue
			}
		}
		ver[q.ID] = true
		out = append(out, q)
	}
	return out
}

func contiene(l []string, x string) bool {
	for _, y := range l {
		if y == x {
			return true
		}
	}
	return false
}

// cuerpoRespuesta: lo que manda el panel. Las imágenes van en base64.
type cuerpoRespuesta struct {
	Elecciones map[string][]string `json:"elecciones"`
	Textos     map[string]string   `json:"textos"`
	Nota       string              `json:"nota"`
	Imagenes   []struct {
		Nombre string `json:"nombre"`
		Base64 string `json:"base64"`
	} `json:"imagenes"`
}

// validarRespuesta: cada pregunta que se ve tiene una opción suya (una sola si no es múltiple) o un texto propio; las
// que no se ven se descartan. Las imágenes: hasta 4, de 5 MB, y de verdad PNG, JPEG, GIF o WebP.
func validarRespuesta(p *PreguntaCentro, c cuerpoRespuesta, ahora time.Time) (*respuestaP, error) {
	r := &respuestaP{Elecciones: map[string][]string{}, Textos: map[string]string{}, Nota: tapado(c.Nota, 4000), Hora: ahora}
	for _, q := range visibles(p.Preguntas, c.Elecciones) {
		el := c.Elecciones[q.ID]
		texto := tapado(c.Textos[q.ID], 4000)
		if len(el) == 0 && texto == "" {
			return nil, fmt.Errorf("falta responder «%s»", q.Texto)
		}
		if len(el) > 1 && !q.Multiple {
			return nil, fmt.Errorf("«%s» lleva una sola opción", q.Texto)
		}
		for _, e := range el {
			ok := false
			for _, o := range q.Opciones {
				ok = ok || o.Etiqueta == e
			}
			if !ok {
				return nil, fmt.Errorf("«%s» no es una opción de «%s»", e, q.Texto)
			}
		}
		if len(el) > 0 {
			r.Elecciones[q.ID] = el
		}
		if texto != "" {
			r.Textos[q.ID] = texto
		}
	}
	if len(c.Imagenes) > maxImagenes {
		return nil, fmt.Errorf("van hasta %d imágenes", maxImagenes)
	}
	for i, im := range c.Imagenes {
		b, err := base64.StdEncoding.DecodeString(im.Base64)
		if err != nil {
			return nil, fmt.Errorf("la imagen %d no viene en base64", i+1)
		}
		if len(b) > maxBytesImagen {
			return nil, fmt.Errorf("la imagen %d pasa de 5 MB", i+1)
		}
		tipo := http.DetectContentType(b)
		switch tipo {
		case "image/png", "image/jpeg", "image/gif", "image/webp":
		default:
			return nil, fmt.Errorf("la imagen %d es %s; van PNG, JPEG, GIF o WebP", i+1, tipo)
		}
		nombre := tapado(im.Nombre, 80)
		if nombre == "" {
			nombre = "imagen-" + strconv.Itoa(i+1)
		}
		r.Imagenes = append(r.Imagenes, imagenP{Nombre: nombre, Tipo: tipo, Bytes: len(b), datos: b})
	}
	return r, nil
}

// textoDeRespuesta: lo que la sesión lee, en el orden de las preguntas que se vieron.
func textoDeRespuesta(p *PreguntaCentro) string {
	r := p.Respuesta
	var b strings.Builder
	fmt.Fprintf(&b, "El dueño del proyecto respondió «%s» en Atalaya (%s):\n", p.Titulo, r.Hora.Local().Format("15:04"))
	for _, q := range visibles(p.Preguntas, r.Elecciones) {
		nombre := q.Encabezado
		if nombre == "" {
			nombre = q.Texto
		}
		partes := append([]string(nil), r.Elecciones[q.ID]...)
		if t := r.Textos[q.ID]; t != "" {
			partes = append(partes, "con sus palabras: "+t)
		}
		fmt.Fprintf(&b, "- %s: %s\n", nombre, strings.Join(partes, "; "))
	}
	if r.Nota != "" {
		b.WriteString("Nota: " + r.Nota + "\n")
	}
	if len(r.Imagenes) > 0 {
		var ns []string
		for _, im := range r.Imagenes {
			ns = append(ns, im.Nombre)
		}
		fmt.Fprintf(&b, "Adjuntó %d imagen(es), que van a continuación: %s.\n", len(r.Imagenes), strings.Join(ns, ", "))
	}
	return b.String()
}

// contenidoMCP: el texto y cada imagen como bloque de imagen; después de entregarla, las imágenes se sueltan.
func (al *almacenPreguntas) entregar(p *PreguntaCentro) []map[string]any {
	al.mu.Lock()
	defer al.mu.Unlock()
	out := []map[string]any{{"type": "text", "text": "Atalaya · respuesta del dueño del proyecto.\n\n" + textoDeRespuesta(p)}}
	for i, im := range p.Respuesta.Imagenes {
		if im.datos != nil {
			out = append(out, map[string]any{"type": "image", "mimeType": im.Tipo, "data": base64.StdEncoding.EncodeToString(im.datos)})
			p.Respuesta.Imagenes[i].datos = nil
		}
	}
	p.Estado = PregEntregada
	return out
}

// responder: el panel deja la respuesta; la sesión que espera la recibe al instante.
func (al *almacenPreguntas) responder(id string, c cuerpoRespuesta, ahora time.Time) (*PreguntaCentro, error) {
	al.mu.Lock()
	defer al.mu.Unlock()
	p := al.por[id]
	if p == nil {
		return nil, errors.New("no existe esa pregunta (Atalaya se reinició o ya se olvidó)")
	}
	if p.Estado != PregEsperando {
		return nil, errors.New("esa pregunta ya se respondió")
	}
	r, err := validarRespuesta(p, c, ahora)
	if err != nil {
		return nil, err
	}
	p.Respuesta, p.Estado = r, PregRespond
	close(p.listo)
	return p, nil
}
