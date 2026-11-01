// Package avisos decide cuándo Atalaya avisa: una decisión que te espera o una alerta de nivel error, una
// vez por cosa, varias juntas en un solo aviso y con una pausa entre dos. Cómo llega el aviso a la persona es cosa de
// un Notificador (la notificación de Windows, en windows.go).
package avisos

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// Cosa: algo que puede avisarse. Clave es estable mientras sea la misma cosa («dec:<id>», «alerta:<id>»).
type Cosa struct {
	Clave   string
	Titulo  string
	Texto   string
	Seccion string // la sección del panel que la muestra: decisiones, agentes
}

// Enviado: un aviso que salió, con las cosas que juntó y el error del notificador si lo hubo.
type Enviado struct {
	Hora    time.Time `json:"hora"`
	Titulo  string    `json:"titulo"`
	Texto   string    `json:"texto"`
	Seccion string    `json:"seccion"`
	Claves  []string  `json:"claves"`
	Error   string    `json:"error,omitempty"`
}

// Notificador hace llegar un aviso; url abre el panel en su sección.
type Notificador interface {
	Notificar(titulo, texto, url string) error
}

const (
	maxEnviados = 50
	// olvido: una cosa ausente más de un día se olvida; si vuelve, se avisa de nuevo.
	olvido = 24 * time.Hour
	// juntas: cuántos títulos van en el texto de un aviso que junta varias cosas.
	juntas = 3
)

// Vigia recuerda qué ya se avisó. El cero no sirve: se crea con Nuevo.
type Vigia struct {
	mu         sync.Mutex
	pausa      time.Duration
	vistos     map[string]time.Time // clave → la última vez que estaba
	arrancado  bool
	pendientes []Cosa
	ultimo     time.Time
	enviados   []Enviado
}

// Nuevo: un vigía con la pausa mínima entre dos avisos.
func Nuevo(pausa time.Duration) *Vigia {
	return &Vigia{pausa: pausa, vistos: map[string]time.Time{}}
}

// Mirar recibe todo lo que hoy podría avisarse y devuelve el aviso que toca mandar ahora, si toca. La primera vez
// solo aprende lo que ya estaba: al arrancar Atalaya no se avisa de lo viejo.
func (v *Vigia) Mirar(cosas []Cosa, ahora time.Time) (Enviado, bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for k, t := range v.vistos {
		if ahora.Sub(t) > olvido {
			delete(v.vistos, k)
		}
	}
	presentes := map[string]bool{}
	for _, c := range cosas {
		presentes[c.Clave] = true
		if _, visto := v.vistos[c.Clave]; !visto && v.arrancado {
			v.pendientes = append(v.pendientes, c)
		}
		v.vistos[c.Clave] = ahora
	}
	// Lo que se resolvió durante la pausa (la respondiste, el agente terminó) ya no se avisa.
	quedan := v.pendientes[:0]
	for _, c := range v.pendientes {
		if presentes[c.Clave] {
			quedan = append(quedan, c)
		}
	}
	v.pendientes = quedan
	if !v.arrancado {
		v.arrancado = true
		return Enviado{}, false
	}
	if len(v.pendientes) == 0 || ahora.Sub(v.ultimo) < v.pausa {
		return Enviado{}, false
	}
	e := juntar(v.pendientes, ahora)
	v.pendientes, v.ultimo = nil, ahora
	v.enviados = append(v.enviados, e)
	if len(v.enviados) > maxEnviados {
		v.enviados = v.enviados[len(v.enviados)-maxEnviados:]
	}
	return e, true
}

// juntar arma un solo aviso: el de la cosa, o «N cosas te esperan» con los primeros títulos. La sección es la de las
// decisiones si hay alguna: es lo que espera una respuesta.
func juntar(cs []Cosa, ahora time.Time) Enviado {
	e := Enviado{Hora: ahora, Seccion: cs[0].Seccion}
	for _, c := range cs {
		e.Claves = append(e.Claves, c.Clave)
		if c.Seccion == "decisiones" {
			e.Seccion = "decisiones"
		}
	}
	if len(cs) == 1 {
		e.Titulo, e.Texto = cs[0].Titulo, cs[0].Texto
		return e
	}
	e.Titulo = fmt.Sprintf("%d cosas te esperan", len(cs))
	var ts []string
	for _, c := range cs[:min(len(cs), juntas)] {
		ts = append(ts, c.Titulo)
	}
	if len(cs) > juntas {
		ts = append(ts, fmt.Sprintf("y %d más", len(cs)-juntas))
	}
	e.Texto = strings.Join(ts, "\n")
	return e
}

// Anotar guarda un aviso que no salió de Mirar (la prueba de la campana), para que también esté entre los enviados.
func (v *Vigia) Anotar(e Enviado) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.enviados = append(v.enviados, e)
	if len(v.enviados) > maxEnviados {
		v.enviados = v.enviados[len(v.enviados)-maxEnviados:]
	}
}

// Resultado anota cómo le fue al notificador con el aviso de esa hora.
func (v *Vigia) Resultado(hora time.Time, err error) {
	if err == nil {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	for i := range v.enviados {
		if v.enviados[i].Hora.Equal(hora) {
			v.enviados[i].Error = err.Error()
		}
	}
}

// Enviados: los últimos avisos, del más nuevo al más viejo.
func (v *Vigia) Enviados() []Enviado {
	v.mu.Lock()
	defer v.mu.Unlock()
	out := make([]Enviado, 0, len(v.enviados))
	for i := len(v.enviados) - 1; i >= 0; i-- {
		out = append(out, v.enviados[i])
	}
	return out
}
