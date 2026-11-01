package alertas

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atalaya/internal/claude"
)

func TestLeeLasReglasDeAgentsMd(t *testing.T) {
	raiz := t.TempDir()
	texto := "# Proyecto\n\n- Un agente solo si la tarea es grande. **Ningún agente pasa de 20 minutos**; se avisa antes.\n" +
		"- Agentes en paralelo: la frontera va en el encargo, y nunca hay dos sobre el mismo archivo.\n"
	_ = os.WriteFile(filepath.Join(raiz, "AGENTS.md"), []byte(texto), 0o644)
	r := Leer(raiz)
	if r.Tope != 20*time.Minute || r.Lista[0].Fuente != "AGENTS.md:3" || r.Lista[0].Omision {
		t.Errorf("tiempo: %v %+v", r.Tope, r.Lista[0])
	}
	if r.Lista[3].Fuente != "AGENTS.md:4" || !r.Lista[1].Omision || !r.Lista[2].Omision {
		t.Errorf("reglas: %+v", r.Lista)
	}
	// Sin AGENTS.md: todo por omisión, y lo dice.
	v := Leer(t.TempDir())
	if v.Tope != 30*time.Minute || !v.Lista[0].Omision || v.Lista[0].Fuente != "" {
		t.Errorf("omisión: %+v", v.Lista[0])
	}
	// El caché vuelve a leer cuando el archivo cambia.
	var l Lector
	if l.Reglas(raiz).Tope != 20*time.Minute {
		t.Fatal("primera lectura")
	}
	_ = os.WriteFile(filepath.Join(raiz, "AGENTS.md"), []byte("Ningún agente pasa de 45 minutos, nunca.\n"), 0o644)
	if got := l.Reglas(raiz).Tope; got != 45*time.Minute {
		t.Errorf("no releyó: %v", got)
	}
}

// El kit en inglés escribe las mismas reglas con sus frases en inglés.
func TestReglasEnIngles(t *testing.T) {
	raiz := t.TempDir()
	texto := "# Project\n\n- One agent only for a large task. **No agent runs longer than 25 minutes**; warn before.\n" +
		"- Parallel agents: the boundary goes in the brief, and never two on the same file.\n"
	_ = os.WriteFile(filepath.Join(raiz, "AGENTS.md"), []byte(texto), 0o644)
	r := Leer(raiz)
	if r.Tope != 25*time.Minute || r.Lista[0].Fuente != "AGENTS.md:3" || r.Lista[3].Fuente != "AGENTS.md:4" {
		t.Errorf("reglas en inglés: %v %+v", r.Tope, r.Lista)
	}
}

func TestTopeDe(t *testing.T) {
	r := Reglas{Tope: 30 * time.Minute}
	for in, want := range map[string]time.Duration{"30 min": 30 * time.Minute, "1 h": time.Hour, "45 minutos": 45 * time.Minute,
		"1,5 h": 90 * time.Minute, "2 horas": 2 * time.Hour, "20min": 20 * time.Minute, "": 30 * time.Minute, "sin decir": 30 * time.Minute} {
		if got := TopeDe(r, in); got != want {
			t.Errorf("TopeDe(%q) = %v, quería %v", in, got, want)
		}
	}
}

func TestEvaluar(t *testing.T) {
	ahora := time.Date(2026, 10, 31, 11, 0, 0, 0, time.UTC)
	min := func(m int) time.Time { return ahora.Add(time.Duration(-m) * time.Minute) }
	r := Leer("")
	ags := []Agente{
		// Aviso a los 26 de 30; pasado a los 32; el presupuesto del encargo manda (15 min).
		{ID: "a", Nombre: "Medidor T1", Proyecto: "RG", ConTope: true, Trabajando: true, Activo: true, Inicio: min(26), Ultimo: min(0)},
		{ID: "b", Nombre: "Codificador I2", Proyecto: "RG", ConTope: true, Trabajando: true, Activo: true, Inicio: min(32), Ultimo: min(1)},
		{ID: "c", Nombre: "Codex E1", Proyecto: "RG", ConTope: true, Trabajando: true, Activo: true, Presupuesto: "15 min", Inicio: min(12), Ultimo: min(0)},
		// La sesión no tiene tope de tiempo, pero sí contexto: 850 k de 1 M.
		{ID: "s", Nombre: "Sesión", Proyecto: "RG", Trabajando: true, Activo: true, Inicio: min(300), Ultimo: min(0), Contexto: 850_000, Ventana: 1_000_000},
		// Trabajando y quieto 7 min (una orden larga).
		{ID: "q", Nombre: "Arquitecto A", Proyecto: "RG", ConTope: true, Trabajando: true, Activo: true, Inicio: min(15), Ultimo: min(7)},
	}
	as := Evaluar(map[string]Reglas{"RG": r}, ags, ahora)
	ids := []string{}
	for _, a := range as {
		ids = append(ids, a.ID)
	}
	// Del más grave al más leve y, en cada nivel, lo más reciente primero.
	want := "tiempo:b:pasado,contexto:s,tiempo:a:aviso,tiempo:c:aviso,quieto:q"
	if strings.Join(ids, ",") != want {
		t.Fatalf("alertas:\n got %s\nwant %s", strings.Join(ids, ","), want)
	}
	if as[1].Chip != "contexto 85 %" || as[2].Chip != "26 de 30 min" || as[3].Chip != "12 de 15 min" || as[4].Chip != "quieto 7 min" {
		t.Errorf("chips: %q %q %q %q", as[1].Chip, as[2].Chip, as[3].Chip, as[4].Chip)
	}
	if Firma(as) == Firma(as[1:]) {
		t.Error("la firma no cambia")
	}
}

func TestFrontera(t *testing.T) {
	ahora := time.Date(2026, 10, 31, 11, 0, 0, 0, time.UTC)
	min := func(m int) time.Time { return ahora.Add(time.Duration(-m) * time.Minute) }
	ed := func(ruta string, m int) claude.Edicion {
		return claude.Edicion{Ruta: ruta, Clave: "d:/rg/" + ruta, Hora: min(m)}
	}
	ags := []Agente{
		// Dos subagentes en paralelo que editan controller.rs: choque.
		{ID: "w6a", Nombre: "Codificador I2", Proyecto: "RG", Activo: true, Inicio: min(40), Ultimo: min(10), Ediciones: []claude.Edicion{ed("controller.rs", 18), ed("a.rs", 30)}},
		{ID: "w6b", Nombre: "Codificador I2", Proyecto: "RG", Activo: true, Inicio: min(35), Ultimo: min(1), Ediciones: []claude.Edicion{ed("controller.rs", 16), ed("b.rs", 5)}},
		// La sesión editó a.rs antes de que w6a empezara: trabajo en serie, no choca.
		{ID: "ses", Nombre: "Sesión", Proyecto: "RG", Activo: true, Inicio: min(300), Ultimo: min(0), Ediciones: []claude.Edicion{ed("a.rs", 45)}},
		// Otro proyecto con el mismo nombre relativo: no choca (la clave es la ruta completa).
		{ID: "otro", Nombre: "Sesión", Proyecto: "CA", Activo: true, Inicio: min(60), Ultimo: min(0), Ediciones: []claude.Edicion{{Ruta: "controller.rs", Clave: "d:/ca/controller.rs", Hora: min(17)}}},
	}
	as := Evaluar(map[string]Reglas{"RG": Leer("")}, ags, ahora)
	var fr []Alerta
	for _, a := range as {
		if a.Regla == "frontera" {
			fr = append(fr, a)
		}
	}
	if len(fr) != 1 || fr[0].ID != "frontera:controller.rs:w6a+w6b" || fr[0].Nivel != Error || fr[0].Archivo != "controller.rs" {
		t.Fatalf("frontera: %+v", fr)
	}
	if !strings.Contains(fr[0].Texto, "controller.rs") {
		t.Errorf("texto: %q", fr[0].Texto)
	}
	// Cuando uno de los dos deja de trabajar, el cruce se cierra solo: queda un aviso informativo, sin chip, durante
	// `Disipa` (w6a terminó hace 10 min), y después nada.
	ags[0].Activo = false
	fr = nil
	for _, a := range Evaluar(map[string]Reglas{"RG": Leer("")}, ags[:2], ahora) {
		if a.Regla == "frontera" {
			fr = append(fr, a)
		}
	}
	if len(fr) != 1 || fr[0].ID != "frontera:controller.rs:w6a+w6b:cerrado" || fr[0].Nivel != Info || fr[0].Chip != "" ||
		!strings.Contains(fr[0].Texto, "Se cerró a las") || !fr[0].Hora.Equal(min(10)) {
		t.Fatalf("cerrado: %+v", fr)
	}
	for _, a := range Evaluar(map[string]Reglas{"RG": Leer("")}, ags[:2], ahora.Add(25*time.Minute)) {
		if a.Regla == "frontera" {
			t.Errorf("se disipa a los 30 min: %+v", a)
		}
	}
}

func TestFronteraConElCoordinador(t *testing.T) {
	ahora := time.Date(2026, 10, 31, 11, 0, 0, 0, time.UTC)
	min := func(m int) time.Time { return ahora.Add(time.Duration(-m) * time.Minute) }
	ed := claude.Edicion{Ruta: "api.ts", Clave: "d:/ca/api.ts", Hora: min(20)}
	ags := []Agente{
		{ID: "ses", Nombre: "Atalaya", Proyecto: "CA", Activo: true, Inicio: min(300), Ultimo: min(0), Ediciones: []claude.Edicion{ed}},
		{ID: "ses~sub", Nombre: "General-purpose", Proyecto: "CA", Padre: "ses", Activo: true, Inicio: min(40), Ultimo: min(1),
			Ediciones: []claude.Edicion{{Ruta: "api.ts", Clave: "d:/ca/api.ts", Hora: min(17)}}},
	}
	cs := Cruces(ags)
	if len(cs) != 1 || !cs[0].Coordinador || !cs[0].Fin.IsZero() {
		t.Fatalf("cruces: %+v", cs)
	}
	as := Evaluar(map[string]Reglas{"CA": Leer("")}, ags, ahora)
	if len(as) != 1 || as[0].Nivel != Aviso || as[0].Chip != "mismo archivo" {
		t.Fatalf("con el coordinador es aviso: %+v", as)
	}
	// El subagente entrega; la sesión sigue activa. Antes la alerta quedaba abierta mientras la sesión viviera.
	ags[1].Activo = false
	as = Evaluar(map[string]Reglas{"CA": Leer("")}, ags, ahora)
	if len(as) != 1 || as[0].Nivel != Info || as[0].Chip != "" || !strings.Contains(as[0].Texto, "General-purpose dejó de trabajar") {
		t.Fatalf("al entregar se cierra: %+v", as)
	}
}
