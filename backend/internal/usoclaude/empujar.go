package usoclaude

import (
	"errors"
	"math"
	"time"
)

// Empujada: una ventana tal como la empuja el mod de Claude Code (session.measure): five_hour o seven_day, su % y su
// reinicio. Llega cuando la app mueve un punto entero, entre una lectura de /usage y la siguiente.
type Empujada struct {
	Tipo       string    `json:"kind"`
	Porcentaje float64   `json:"percent_used"`
	ReiniciaEn time.Time `json:"resets_at"`
}

// ErrEmpujadaInvalida: ninguna ventana conocida, o un % o un reinicio imposibles.
var ErrEmpujadaInvalida = errors.New("lectura empujada inválida")

// ventanaEmpujada: la clave y la etiqueta de /usage que corresponden a lo que nombra el mod.
func ventanaEmpujada(tipo string) (clave, etiqueta string, mins int, ok bool) {
	switch tipo {
	case "five_hour":
		return "session", "5 horas", 300, true
	case "seven_day":
		return "weekly_all", "Semana · todos los modelos", 10080, true
	}
	return "", "", 0, false
}

// Empujar pone al día las ventanas que trae el mod y deja las demás (la semanal por modelo solo la da /usage). Cuenta
// como una lectura: la frescura y el historial la incluyen, con la fuente «app».
func (l *Lector) Empujar(es []Empujada) error {
	nuevas := map[string]Ventana{}
	for _, e := range es {
		clave, etiqueta, mins, ok := ventanaEmpujada(e.Tipo)
		if !ok {
			continue
		}
		if e.Porcentaje < 0 || e.Porcentaje > 100 || e.ReiniciaEn.IsZero() {
			return ErrEmpujadaInvalida
		}
		// Entero, como lo da /usage: el mod trae un decimal y, sin redondear, las dos fuentes discrepan en un punto.
		nuevas[clave] = Ventana{Clave: clave, Etiqueta: etiqueta, Porcentaje: math.Round(e.Porcentaje), ReiniciaEn: e.ReiniciaEn.Unix(), DuracionMin: mins}
	}
	if len(nuevas) == 0 {
		return ErrEmpujadaInvalida
	}
	l.mu.Lock()
	if l.Demo {
		l.mu.Unlock()
		return nil
	}
	ventanas := append([]Ventana{}, l.estado.Ventanas...)
	for i, v := range ventanas {
		if n, ok := nuevas[v.Clave]; ok {
			// La severidad y si está activa vienen de /usage; el mod no las trae.
			n.Severidad, n.Activa = v.Severidad, v.Activa
			ventanas[i] = n
			delete(nuevas, v.Clave)
		}
	}
	for _, clave := range []string{"session", "weekly_all"} {
		if n, ok := nuevas[clave]; ok {
			ventanas = append(ventanas, n)
		}
	}
	l.estado.Ventanas = ventanas
	l.estado.TomadaEn = l.Ahora().Unix()
	l.estado.Fuente = "app"
	if l.estado.Estado != "limitado" {
		l.estado.Estado, l.estado.Detalle = "ok", ""
	}
	f := fila{T: l.estado.TomadaEn, Ventanas: map[string]float64{}, Reinicios: map[string]int64{}}
	for _, v := range ventanas {
		f.Ventanas[v.Clave] = v.Porcentaje
		f.Reinicios[v.Clave] = v.ReiniciaEn
	}
	l.filas = append(l.filas, f)
	var err error
	l.filas, err = guardar(l.ruta, l.filas, l.Ahora())
	l.mu.Unlock()
	if l.Cambio != nil {
		l.Cambio()
	}
	return err
}
