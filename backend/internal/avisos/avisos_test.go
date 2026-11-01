package avisos

import (
	"strings"
	"testing"
	"time"
)

func TestMirar(t *testing.T) {
	t0 := time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC)
	v := Nuevo(15 * time.Second)
	vieja := Cosa{Clave: "dec:vieja", Titulo: "Vieja", Seccion: "decisiones"}
	if _, ok := v.Mirar([]Cosa{vieja}, t0); ok {
		t.Fatal("al arrancar no se avisa de lo que ya estaba")
	}
	if _, ok := v.Mirar([]Cosa{vieja}, t0.Add(2*time.Second)); ok {
		t.Fatal("lo ya visto no se avisa")
	}

	// Una nueva: sale sola, con su título, texto y sección.
	nueva := Cosa{Clave: "dec:1", Titulo: "Te espera una decisión", Texto: "¿Seguimos?", Seccion: "decisiones"}
	e, ok := v.Mirar([]Cosa{vieja, nueva}, t0.Add(4*time.Second))
	if !ok || e.Titulo != nueva.Titulo || e.Texto != "¿Seguimos?" || e.Seccion != "decisiones" || len(e.Claves) != 1 {
		t.Fatalf("una nueva: %+v %v", e, ok)
	}

	// Dos más dentro de la pausa: esperan y salen juntas, con la sección de la decisión.
	alerta := Cosa{Clave: "alerta:tiempo:a", Titulo: "Pasó su tope", Seccion: "agentes"}
	otra := Cosa{Clave: "dec:2", Titulo: "Otra decisión", Seccion: "decisiones"}
	if _, ok := v.Mirar([]Cosa{vieja, nueva, alerta, otra}, t0.Add(10*time.Second)); ok {
		t.Fatal("dentro de la pausa no se avisa")
	}
	e, ok = v.Mirar([]Cosa{vieja, nueva, alerta, otra}, t0.Add(20*time.Second))
	if !ok || e.Titulo != "2 cosas te esperan" || e.Texto != "Pasó su tope\nOtra decisión" || e.Seccion != "decisiones" {
		t.Fatalf("juntas: %+v %v", e, ok)
	}

	// Una que se resuelve durante la pausa ya no se avisa.
	fugaz := Cosa{Clave: "dec:3", Titulo: "Fugaz", Seccion: "decisiones"}
	v.Mirar([]Cosa{fugaz}, t0.Add(25*time.Second))
	if e, ok := v.Mirar(nil, t0.Add(40*time.Second)); ok {
		t.Fatalf("resuelta antes de avisar: %+v", e)
	}

	// Ausente más de un día, se olvida: si vuelve, se avisa otra vez.
	if e, ok := v.Mirar([]Cosa{vieja}, t0.Add(25*time.Hour)); !ok || e.Claves[0] != "dec:vieja" {
		t.Fatalf("vuelta tras un día: %+v %v", e, ok)
	}

	env := v.Enviados()
	if len(env) != 3 || env[0].Claves[0] != "dec:vieja" {
		t.Fatalf("enviados, del más nuevo al más viejo: %+v", env)
	}
	v.Resultado(env[0].Hora, errPrueba("sin PowerShell"))
	if v.Enviados()[0].Error != "sin PowerShell" {
		t.Error("el error del notificador queda anotado")
	}
}

func TestJuntarMuchas(t *testing.T) {
	var cs []Cosa
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		cs = append(cs, Cosa{Clave: n, Titulo: n, Seccion: "agentes"})
	}
	e := juntar(cs, time.Time{})
	if e.Titulo != "5 cosas te esperan" || e.Texto != "a\nb\nc\ny 2 más" || e.Seccion != "agentes" {
		t.Errorf("%+v", e)
	}
}

func TestLimpiar(t *testing.T) {
	if got := limpiar("a\x00b\tc\nd", 80); got != "abc\nd" {
		t.Errorf("control: %q", got)
	}
	if got := limpiar(strings.Repeat("x", 10), 5); got != "xxxx…" {
		t.Errorf("corte: %q", got)
	}
}

type errPrueba string

func (e errPrueba) Error() string { return string(e) }
