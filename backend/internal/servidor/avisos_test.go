package servidor

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atalaya/internal/alertas"
	"atalaya/internal/avisos"
)

type notificadorPrueba struct{ llegan chan [3]string }

func (n notificadorPrueba) Notificar(titulo, texto, url string) error {
	n.llegan <- [3]string{titulo, texto, url}
	return nil
}

func TestMirarAvisos(t *testing.T) {
	s := nuevo(t)
	n := notificadorPrueba{llegan: make(chan [3]string, 4)}
	s.Avisos, s.Notificador, s.Base = avisos.Nuevo(0), n, "http://127.0.0.1:7420"
	s.alertas = []alertas.Alerta{{ID: "tiempo:viejo:pasado", Nivel: alertas.Error, Proyecto: "P", Titulo: "Ya estaba"}}
	s.mirarAvisos() // aprende lo que había al arrancar
	s.mu.Lock()
	s.alertas = append(s.alertas,
		alertas.Alerta{ID: "tiempo:a:pasado", Nivel: alertas.Error, Proyecto: "Tienda Demo", Titulo: "codificador pasó su tope", Texto: "lleva 34 min de 30"},
		alertas.Alerta{ID: "contexto:a", Nivel: alertas.Aviso, Proyecto: "Tienda Demo", Titulo: "contexto alto"})
	s.mu.Unlock()
	s.mirarAvisos()
	select {
	case got := <-n.llegan:
		if got != [3]string{"codificador pasó su tope", "Tienda Demo: lleva 34 min de 30", "http://127.0.0.1:7420/#agentes"} {
			t.Errorf("aviso: %q", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no llegó el aviso de la alerta de error")
	}
	s.mirarAvisos()
	select {
	case got := <-n.llegan:
		t.Errorf("una alerta de nivel aviso, o una repetida, no avisa: %q", got)
	case <-time.After(200 * time.Millisecond):
	}
}

// Una notificación de verdad, a mano: ATALAYA_PROBAR_AVISO=1 go test ./internal/servidor -run TestAvisoDeVerdad
func TestAvisoDeVerdad(t *testing.T) {
	if os.Getenv("ATALAYA_PROBAR_AVISO") == "" {
		t.Skip("solo a mano")
	}
	if err := (avisos.Windows{}).Notificar("Atalaya · prueba", "Los avisos funcionan: un clic abre la bandeja de decisiones.", "http://127.0.0.1:7420/#decisiones"); err != nil {
		t.Fatal(err)
	}
}

func TestConfigurarYProbarAvisos(t *testing.T) {
	s := nuevo(t)
	n := notificadorPrueba{llegan: make(chan [3]string, 4)}
	s.Avisos, s.Notificador, s.Base = avisos.Nuevo(0), n, "http://127.0.0.1:7420"
	s.RutaAvisos = filepath.Join(t.TempDir(), "avisos.json")
	s.alertas = []alertas.Alerta{{ID: "tiempo:a:pasado", Nivel: alertas.Error, Proyecto: "P", Titulo: "Pasó su tope"}}
	pedir := func(ruta, cuerpo string, panel bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", ruta, strings.NewReader(cuerpo))
		r.Host = "localhost"
		if panel {
			r.Header.Set(cabPanel, "1")
		}
		w := httptest.NewRecorder()
		s.Handler().ServeHTTP(w, r)
		return w
	}
	if w := pedir("/api/avisos", `{"alertas":false}`, false); w.Code != http.StatusForbidden {
		t.Fatalf("sin la cabecera del panel no se cambia nada: %d", w.Code)
	}
	if w := pedir("/api/avisos", `{"alertas":false}`, true); w.Code != 200 || !strings.Contains(w.Body.String(), `"alertas":false`) || !strings.Contains(w.Body.String(), `"decisiones":true`) {
		t.Fatalf("apagar las alertas: %d %s", w.Code, w.Body.String())
	}
	if len(s.cosasParaAvisar()) != 0 {
		t.Error("con las alertas apagadas, una alerta de error no se avisa")
	}
	if c := avisos.LeerConfig(s.RutaAvisos); c.Alertas || !c.Decisiones {
		t.Errorf("avisos.json: %+v", c)
	}
	if w := pedir("/api/avisos/probar", ``, true); w.Code != 200 || !strings.Contains(w.Body.String(), `"claves":["prueba"]`) {
		t.Fatalf("probar: %d %s", w.Code, w.Body.String())
	}
	if got := <-n.llegan; got[0] != "Atalaya · prueba" || got[2] != "http://127.0.0.1:7420/#decisiones" {
		t.Errorf("la prueba: %q", got)
	}
}
