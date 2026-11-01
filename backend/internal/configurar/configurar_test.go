package configurar

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func puerta(t *testing.T, permitidos ...string) *Puerta {
	t.Helper()
	ahora := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	return &Puerta{
		Dir: filepath.Join(t.TempDir(), "Atalaya"),
		Permitido: func(r string) bool {
			for _, p := range permitidos {
				if MismaRuta(r, p) {
					return true
				}
			}
			return false
		},
		Programa: func(n string) (string, bool) {
			if n == "codex" {
				return "/bin/codex", true
			}
			return "", false
		},
		Ahora: func() time.Time { return ahora },
	}
}

// Lo que la puerta tiene que rechazar: fuera de la lista, rutas relativas o con «..» que salen de ella, programas
// que no están, tipos inventados, y una operación vacía.
func TestLoQueSeRechaza(t *testing.T) {
	dir := t.TempDir()
	registro := filepath.Join(dir, "proyectos.json")
	p := puerta(t, registro)
	casos := []struct {
		nombre string
		c      Cambio
		err    error
	}{
		{"fuera de la lista", Cambio{Tipo: Archivo, Ruta: filepath.Join(dir, "AGENTS.md"), Despues: "x"}, ErrNoPermitido},
		{"relativa", Cambio{Tipo: Archivo, Ruta: "proyectos.json", Despues: "x"}, ErrNoPermitido},
		{"con ..", Cambio{Tipo: Archivo, Ruta: filepath.Join(dir, "sub", "..", "..", "proyectos.json"), Despues: "x"}, ErrNoPermitido},
		{"programa ajeno", Cambio{Tipo: Orden, Programa: "powershell", Argumentos: []string{"-c", "x"}}, ErrPrograma},
	}
	for _, c := range casos {
		if _, err := p.Proponer("x", "x", []Cambio{c.c}); !errors.Is(err, c.err) {
			t.Errorf("%s: %v, quería %v", c.nombre, err, c.err)
		}
	}
	if _, err := p.Proponer("x", "x", []Cambio{{Tipo: "borrar", Ruta: registro}}); err == nil {
		t.Error("un tipo inventado se aceptó")
	}
	if _, err := p.Proponer("x", "x", nil); err == nil {
		t.Error("una operación vacía se aceptó")
	}
	// Una ruta con «..» que vuelve a la permitida sí vale: lo que cuenta es adónde llega.
	if _, err := p.Proponer("x", "x", []Cambio{{Tipo: Archivo, Ruta: filepath.Join(dir, "sub", "..", "proyectos.json"), Despues: "[]"}}); err != nil {
		t.Errorf("la ruta limpia es la permitida: %v", err)
	}
	if _, err := p.Aplicar(context.Background(), "inventado"); !errors.Is(err, ErrNoHay) {
		t.Errorf("un id inventado: %v", err)
	}
}

// El ciclo de un archivo: vista previa con lo que hay, aplicar con copia, historial, y deshacer que lo devuelve.
func TestArchivoAplicarYDeshacer(t *testing.T) {
	dir := t.TempDir()
	registro := filepath.Join(dir, "proyectos.json")
	_ = os.WriteFile(registro, []byte("[viejo]"), 0o644)
	p := puerta(t, registro)
	prop, err := p.Proponer("proyectos", "Registrar Tienda", []Cambio{{Tipo: Archivo, Ruta: registro, Despues: "[nuevo]"}})
	if err != nil || !prop.Cambios[0].Existia || prop.Cambios[0].Antes != "[viejo]" {
		t.Fatalf("vista previa: %+v %v", prop, err)
	}
	if b, _ := os.ReadFile(registro); string(b) != "[viejo]" {
		t.Fatal("la vista previa escribió")
	}
	e, err := p.Aplicar(context.Background(), prop.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(registro); string(b) != "[nuevo]" {
		t.Fatalf("aplicado: %q", b)
	}
	if b, _ := os.ReadFile(e.Copias[MismaRutaClave(e.Copias, registro)]); string(b) != "[viejo]" {
		t.Errorf("la copia: %q", b)
	}
	if _, err := p.Aplicar(context.Background(), prop.ID); !errors.Is(err, ErrNoHay) {
		t.Errorf("una vista previa se aplica una sola vez: %v", err)
	}
	if _, err := os.Stat(registro + ".atalaya-tmp"); !errors.Is(err, os.ErrNotExist) {
		t.Error("quedó el temporal")
	}

	d, err := p.Deshacer(context.Background(), prop.ID)
	if err != nil || d.Deshace != prop.ID {
		t.Fatalf("deshacer: %+v %v", d, err)
	}
	if b, _ := os.ReadFile(registro); string(b) != "[viejo]" {
		t.Fatalf("deshecho: %q", b)
	}
	if _, err := p.Deshacer(context.Background(), prop.ID); !errors.Is(err, ErrYaDeshecho) {
		t.Errorf("dos veces: %v", err)
	}
	h, _ := p.Historial(0)
	if len(h) != 2 || h[0].Deshace != prop.ID || h[1].ID != prop.ID || !h[1].Deshecha {
		t.Errorf("historial: %+v", h)
	}
}

// MismaRutaClave: la clave del mapa de copias que corresponde a una ruta (en Windows pueden diferir en mayúsculas).
func MismaRutaClave(m map[string]string, ruta string) string {
	for k := range m {
		if MismaRuta(k, ruta) {
			return k
		}
	}
	return ruta
}

// Un archivo que no existía: deshacer lo borra. Y si alguien lo tocó entre la vista previa y aplicar, no se aplica;
// si lo tocó después de aplicar, deshacer no pisa su trabajo.
func TestArchivoNuevoYCambiosAjenos(t *testing.T) {
	dir := t.TempDir()
	registro := filepath.Join(dir, "proyectos.json")
	p := puerta(t, registro)
	prop, _ := p.Proponer("proyectos", "Primer registro", []Cambio{{Tipo: Archivo, Ruta: registro, Despues: "[a]"}})
	if prop.Cambios[0].Existia {
		t.Fatal("no existía")
	}
	if _, err := p.Aplicar(context.Background(), prop.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Deshacer(context.Background(), prop.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(registro); !errors.Is(err, os.ErrNotExist) {
		t.Error("deshacer un archivo nuevo lo borra")
	}

	_ = os.WriteFile(registro, []byte("[a]"), 0o644)
	prop, _ = p.Proponer("proyectos", "Cambiar", []Cambio{{Tipo: Archivo, Ruta: registro, Despues: "[b]"}})
	_ = os.WriteFile(registro, []byte("[a, editado a mano]"), 0o644)
	if _, err := p.Aplicar(context.Background(), prop.ID); !errors.Is(err, ErrCambioFuera) {
		t.Errorf("cambiado entre la vista y aplicar: %v", err)
	}
	if b, _ := os.ReadFile(registro); string(b) != "[a, editado a mano]" {
		t.Error("aplicó encima de un cambio ajeno")
	}

	prop, _ = p.Proponer("proyectos", "Cambiar", []Cambio{{Tipo: Archivo, Ruta: registro, Despues: "[c]"}})
	if _, err := p.Aplicar(context.Background(), prop.ID); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(registro, []byte("[c, y algo más]"), 0o644)
	if _, err := p.Deshacer(context.Background(), prop.ID); !errors.Is(err, ErrCambiadoDespues) {
		t.Errorf("deshacer sobre un cambio posterior: %v", err)
	}
	if b, _ := os.ReadFile(registro); string(b) != "[c, y algo más]" {
		t.Error("deshacer pisó un cambio posterior")
	}
}

// Una orden: corre el ejecutable resuelto con sus argumentos tal cual, y deshacer corre la inversa.
func TestOrdenYSuInversa(t *testing.T) {
	p := puerta(t)
	var corridas []string
	p.Correr = func(_ context.Context, exe string, args []string) (string, error) {
		corridas = append(corridas, exe+" "+strings.Join(args, " "))
		return "ok", nil
	}
	prop, err := p.Proponer("mcp", "Conectar Codex", []Cambio{{Tipo: Orden, Programa: "codex",
		Argumentos: []string{"mcp", "add", "atalaya", "--url", "http://127.0.0.1:7420/mcp"}, Deshacer: []string{"mcp", "remove", "atalaya"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(corridas) != 0 {
		t.Fatal("la vista previa corrió algo")
	}
	e, err := p.Aplicar(context.Background(), prop.ID)
	if err != nil || e.Salida != "ok" {
		t.Fatalf("%+v %v", e, err)
	}
	if _, err := p.Deshacer(context.Background(), prop.ID); err != nil {
		t.Fatal(err)
	}
	quiere := []string{"/bin/codex mcp add atalaya --url http://127.0.0.1:7420/mcp", "/bin/codex mcp remove atalaya"}
	if strings.Join(corridas, "|") != strings.Join(quiere, "|") {
		t.Errorf("corridas: %q", corridas)
	}
}

// Aplicar que se corta a medias (el archivo se escribió, la orden falló): deshacer borra el archivo y no corre la
// inversa de la orden que nunca se hizo. Y un deshacer que falló se puede reintentar (el arranque de Linux en una WSL
// sin systemd, que encontró la prueba de S8).
func TestFallaAMediasYDeshacer(t *testing.T) {
	unidad := filepath.Join(t.TempDir(), "atalaya.service")
	p := puerta(t, unidad)
	var corridas []string
	falla := true
	p.Correr = func(_ context.Context, _ string, args []string) (string, error) {
		corridas = append(corridas, strings.Join(args, " "))
		if falla {
			return "sin systemd", errors.New("salida 1")
		}
		return "", nil
	}
	prop, err := p.Proponer("arranque", "Arrancar", []Cambio{
		{Tipo: Archivo, Ruta: unidad, Despues: "[Unit]\n"},
		{Tipo: Orden, Programa: "codex", Argumentos: []string{"enable"}, Deshacer: []string{"disable"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	e, err := p.Aplicar(context.Background(), prop.ID)
	if err == nil || e.hechos() != 1 {
		t.Fatalf("a medias: hechos %d, err %v", e.hechos(), err)
	}
	if _, err := p.Deshacer(context.Background(), prop.ID); err != nil {
		t.Fatalf("deshacer: %v", err)
	}
	if _, err := os.Stat(unidad); !os.IsNotExist(err) {
		t.Error("la unidad sigue ahí")
	}
	if strings.Join(corridas, "|") != "enable" {
		t.Errorf("corrió la inversa de lo que nunca se hizo: %q", corridas)
	}

	// Un deshacer que falla (la inversa de una orden que sí corrió) se puede reintentar, y lo ya revertido se salta.
	falla = false
	prop, _ = p.Proponer("arranque", "Arrancar", []Cambio{
		{Tipo: Archivo, Ruta: unidad, Despues: "[Unit]\n"},
		{Tipo: Orden, Programa: "codex", Argumentos: []string{"enable"}, Deshacer: []string{"disable"}},
	})
	if _, err := p.Aplicar(context.Background(), prop.ID); err != nil {
		t.Fatal(err)
	}
	falla = true
	if _, err := p.Deshacer(context.Background(), prop.ID); err == nil {
		t.Fatal("el disable falló y no lo dijo")
	}
	falla = false
	if _, err := p.Deshacer(context.Background(), prop.ID); err != nil {
		t.Fatalf("reintento: %v", err)
	}
	if _, err := os.Stat(unidad); !os.IsNotExist(err) {
		t.Error("tras reintentar, la unidad sigue ahí")
	}
	if _, err := p.Deshacer(context.Background(), prop.ID); !errors.Is(err, ErrYaDeshecho) {
		t.Errorf("tercera vez: %v", err)
	}
}

// Una orden que falla queda anotada con su error, y la vista previa vence a los 10 min.
func TestFallaYVencimiento(t *testing.T) {
	p := puerta(t)
	p.Correr = func(context.Context, string, []string) (string, error) { return "no hay red", errors.New("salida 1") }
	prop, _ := p.Proponer("mcp", "Conectar", []Cambio{{Tipo: Orden, Programa: "codex", Argumentos: []string{"mcp", "add"}}})
	if _, err := p.Aplicar(context.Background(), prop.ID); err == nil {
		t.Fatal("la falla no llegó")
	}
	h, _ := p.Historial(1)
	if len(h) != 1 || h[0].Error == "" || h[0].Salida != "no hay red" {
		t.Errorf("anotada: %+v", h)
	}
	ahora := time.Date(2026, 10, 31, 23, 0, 0, 0, time.UTC)
	p.Ahora = func() time.Time { return ahora }
	prop, _ = p.Proponer("mcp", "Conectar", []Cambio{{Tipo: Orden, Programa: "codex", Argumentos: []string{"mcp", "add"}}})
	ahora = ahora.Add(VidaPropuesta + time.Second)
	if _, err := p.Aplicar(context.Background(), prop.ID); !errors.Is(err, ErrNoHay) {
		t.Errorf("vencida: %v", err)
	}
}
