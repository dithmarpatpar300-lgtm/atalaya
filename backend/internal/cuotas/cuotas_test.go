package cuotas

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const salida = `{"advice":{"first_choice":"Codex","providers":[{"even_pace_percent_per_hour":1.18,"headroom_percent":94.0,"provider":"Claude Code","reasons":[],"stance":"normal"}],"summary":"Codex normal, Claude normal."},
"panels":[{"captured_at":1793417945,"notice":null,"plan":null,"provider":"Claude Code","source":"statusLine","windows":[{"duration_mins":300,"label":"5 h","pace":null,"resets_at":1793412000,"tokens":null,"tokens_left_estimate":null,"used_percent":0.0}]},
{"captured_at":1793417952,"notice":null,"plan":"plus","provider":"Codex","source":"codex app-server","windows":[{"duration_mins":300,"label":"5 h","pace":{"full_at":1793455845,"percent_per_hour":9.3},"resets_at":1793435430,"tokens":null,"tokens_left_estimate":null,"used_percent":2.0}]}],"taken_at":1793417952}`

func TestLeerPorElEjecutable(t *testing.T) {
	l := NuevoLector("cuotas.exe", t.TempDir())
	l.Ejecutar = func(context.Context, string, ...string) ([]byte, error) { return []byte(salida), nil }
	est := l.Leer(context.Background())
	if est.Origen != "ejecutable" || est.Lectura.Advice.FirstChoice != "Codex" || len(est.Lectura.Panels) != 2 {
		t.Fatalf("%+v", est)
	}
	if p := est.Lectura.Panels[1].Windows[0].Pace; p == nil || p.PercentPerHour != 9.3 {
		t.Errorf("ritmo %+v", p)
	}
}

// Mientras una lectura corre, otra no lanza un segundo cuotas.exe y Toca dice que no.
func TestLecturasNoSeEnciman(t *testing.T) {
	l := NuevoLector("cuotas.exe", t.TempDir())
	entro, suelta := make(chan struct{}), make(chan struct{})
	corridas := 0
	l.Ejecutar = func(context.Context, string, ...string) ([]byte, error) {
		corridas++
		close(entro)
		<-suelta
		return []byte(salida), nil
	}
	hecho := make(chan Estado)
	go func() { hecho <- l.Leer(context.Background()) }()
	<-entro
	if l.Toca() {
		t.Error("Toca con una lectura en curso")
	}
	if est := l.Leer(context.Background()); est.Lectura != nil {
		t.Errorf("la segunda lectura debía devolver la guardada (vacía): %+v", est)
	}
	close(suelta)
	if est := <-hecho; est.Origen != "ejecutable" || corridas != 1 {
		t.Errorf("origen %q, %d corridas", est.Origen, corridas)
	}
}

func TestSinEjecutableUsaLosArchivos(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "claude-statusline.json"), []byte(`{"captured_at":100,"rate_limits":{"five_hour":{"resets_at":200,"used_percentage":48.0},"seven_day":{"resets_at":900,"used_percentage":6.0}},"version":"2.1.314"}`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "historial.jsonl"), []byte("{\"p\":\"codex\",\"r\":5,\"t\":90,\"u\":1.0,\"w\":\"5 h\"}\n{\"p\":\"codex\",\"r\":7,\"t\":110,\"u\":2.0,\"w\":\"5 h\"}\n{\"p\":\"codex\",\"r\":9,\"t\":110,\"u\":18.0,\"w\":\"Semanal\"}\n"), 0o644)
	l := NuevoLector("no-existe.exe", dir)
	l.Ejecutar = func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("no está") }
	est := l.Leer(context.Background())
	if est.Origen != "archivos" || est.Lectura == nil || len(est.Lectura.Panels) != 2 || est.Lectura.Advice != nil {
		t.Fatalf("%+v", est)
	}
	cl, cx := est.Lectura.Panels[0], est.Lectura.Panels[1]
	if cl.Windows[0].UsedPercent != 48 || cl.Windows[1].Label != "Semanal" {
		t.Errorf("claude %+v", cl)
	}
	if len(cx.Windows) != 2 || cx.Windows[0].UsedPercent != 2.0 || cx.Windows[1].UsedPercent != 18.0 || est.Lectura.TakenAt != 110 {
		t.Errorf("codex %+v", cx)
	}
}

func TestForzarRespetaElMinimo(t *testing.T) {
	ahora := time.Unix(1000, 0)
	veces := 0
	l := NuevoLector("c", t.TempDir())
	l.Ahora = func() time.Time { return ahora }
	l.Ejecutar = func(context.Context, string, ...string) ([]byte, error) { veces++; return []byte(salida), nil }
	l.Leer(context.Background())
	l.Forzar(context.Background())
	if veces != 1 {
		t.Errorf("a los 0 s no debía leer otra vez: %d", veces)
	}
	ahora = ahora.Add(31 * time.Second)
	l.Forzar(context.Background())
	if veces != 2 {
		t.Errorf("a los 31 s debía leer: %d", veces)
	}
	// Un fallo no borra la última lectura buena.
	l.Ejecutar = func(context.Context, string, ...string) ([]byte, error) { return nil, errors.New("caído") }
	est := l.Leer(context.Background())
	if est.Lectura == nil || est.Error == "" {
		t.Errorf("%+v", est)
	}
}

func TestDecodificarRechazaVacio(t *testing.T) {
	if _, err := Decodificar([]byte(`{"panels":[]}`)); err == nil {
		t.Error("sin paneles debería fallar")
	}
	if _, err := Decodificar([]byte(`no`)); err == nil {
		t.Error("no JSON debería fallar")
	}
}

func TestHistorialPorTramos(t *testing.T) {
	dir := t.TempDir()
	filas := `{"p":"claude","w":"5 h","u":10,"r":0,"t":1000}
{"p":"claude","w":"5 h","u":12,"r":0,"t":1100}
{"p":"claude","w":"Semanal","u":50,"r":0,"t":1100}
{"p":"codex","w":"5 h","u":3,"r":0,"t":1400}
{"p":"claude","w":"5 h","u":20,"r":0,"t":1400}
{"p":"claude","w":"5 h","u":30,"r":0,"t":9000}
`
	if err := os.WriteFile(filepath.Join(dir, "historial.jsonl"), []byte(filas), 0o600); err != nil {
		t.Fatal(err)
	}
	h := NuevoLector("", dir).Historial(time.Unix(1000, 0), time.Unix(2000, 0), 5*time.Minute)
	if c := h["claude"]; len(c) != 2 || c[0].U != 12 || c[1].U != 20 {
		t.Errorf("claude: %+v", c)
	}
	if x := h["codex"]; len(x) != 1 || x[0].U != 3 {
		t.Errorf("codex: %+v", x)
	}
}
