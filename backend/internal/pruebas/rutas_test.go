package pruebas

import "testing"

func TestNativa(t *testing.T) {
	t.Setenv("ATALAYA_RUTAS_UNIX", "1")
	if g := Nativa(`C:\Proyectos\Tienda Demo\`); g != "/C/Proyectos/Tienda Demo/" {
		t.Errorf("Nativa: %q", g)
	}
	if g := NativaJSON(`{"cwd":"C:\\Proyectos\\Tienda Demo","texto":"a\nb \"c\"","url":"https://x/y"}`); g != `{"cwd":"/C/Proyectos/Tienda Demo","texto":"a\nb \"c\"","url":"https://x/y"}` {
		t.Errorf("NativaJSON: %q", g)
	}
}
