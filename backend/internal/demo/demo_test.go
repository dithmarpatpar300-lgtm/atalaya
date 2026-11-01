package demo

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"atalaya/internal/alertas"
	"atalaya/internal/avance"
	"atalaya/internal/claude"
	"atalaya/internal/codexvivo"
	"atalaya/internal/proyectos"
	"atalaya/internal/servidor"
)

func TestMundo(t *testing.T) {
	ahora := time.Now()
	dir := filepath.Join(t.TempDir(), "mundo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "viejo"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	m, err := Preparar(dir, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "viejo")); !os.IsNotExist(err) {
		t.Fatalf("no borró el mundo anterior: %v", err)
	}
	ps, err := proyectos.Leer(m.Registro)
	if err != nil || len(ps) != 3 {
		t.Fatalf("proyectos: %v %v", ps, err)
	}
	for _, p := range ps {
		fs := avance.Leer(p.Ruta)
		if len(fs) != 1 || fs[0].Declarado != int(math.Round(fs[0].Calculado)) || len(fs[0].Avisos) != 0 {
			t.Errorf("avance %s: %+v", p.Nombre, fs)
		}
	}
	ix := claude.NuevoIndice(m.Claude)
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	huella := ix.Huellas(ahora.Add(-time.Hour))
	var ses, subs int
	var ags []alertas.Agente
	estados := map[string]string{}
	for _, a := range ix.Agentes() {
		if a.Proyecto != "Tienda" {
			continue
		}
		if a.Tipo == "sesion" {
			ses++
		} else {
			subs++
		}
		estado := servidor.EstadoClaude(a, ahora)
		estados[a.Tarea] = estado
		ag := alertas.Agente{ID: a.ID, Nombre: a.Tarea, Proyecto: a.Proyecto, Padre: a.Padre, ConTope: a.Tipo == "subagente", Trabajando: estado == servidor.Trabajando, Activo: estado == servidor.Trabajando || estado == servidor.EnEspera, Presupuesto: a.Presupuesto, Inicio: a.Inicio, Ultimo: a.Ultimo, Ediciones: huella[a.ID].Ediciones}
		ags = append(ags, ag)
	}
	// Hoy, ayer y el lunes: la de hoy con sus cuatro subagentes; las otras dos, con tres cada una.
	if ses != 3 || subs != 10 {
		t.Errorf("Tienda: %d sesiones, %d subagentes", ses, subs)
	}
	for nombre, quiere := range map[string]string{"codificador-carrito": servidor.Entregado, "auditor-pagos": servidor.Trabajando, "codificador-cupones": servidor.Trabajando, "codificador-envios": servidor.Trabajando} {
		if estados[nombre] != quiere {
			t.Errorf("%s: estado %q, quiere %q", nombre, estados[nombre], quiere)
		}
	}
	// La de hoy espera; la de ayer ya se respondió con el mensaje siguiente.
	estadosDec := map[string]string{}
	for _, d := range ix.Decisiones("Tienda") {
		estadosDec[d.Agente] = d.Estado
	}
	if len(estadosDec) != 2 || estadosDec["sesion-tienda"] != claude.DecPendiente || estadosDec["sesion-tienda-ayer"] != claude.DecRespondida {
		t.Errorf("decisiones: %+v", estadosDec)
	}
	l := codexvivo.NuevoLector(m.Codex)
	if _, err := l.Actualizar(); err != nil {
		t.Fatal(err)
	}
	if cs := l.Llamadas(); len(cs) != 2 || !cs[0].Terminada || !cs[1].Terminada {
		t.Errorf("Codex: %+v", cs)
	}
	reglas := map[string]alertas.Reglas{"Tienda": alertas.Leer(ps[0].Ruta)}
	vistas := map[string]bool{}
	for _, a := range alertas.Evaluar(reglas, ags, ahora) {
		if a.Nivel == alertas.Error {
			vistas[a.Regla] = true
		}
	}
	if !vistas["tiempo"] || !vistas["frontera"] {
		t.Errorf("alertas de error: %v; agentes: %+v", vistas, ags)
	}
}
