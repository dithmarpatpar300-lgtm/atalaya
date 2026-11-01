package servidor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"atalaya/internal/alertas"
	"atalaya/internal/codexvivo"
	"atalaya/internal/diario"
)

// Un subagente cuyo transcrito ya se borró: solo queda en el diario, y Estadísticas, sus semanas y la línea de su día
// lo siguen mostrando.
func TestLoQueSoloQuedaEnElDiario(t *testing.T) {
	loc := time.Local
	ahora := time.Date(2026, 11, 20, 12, 0, 0, 0, loc)
	inicio := time.Date(2026, 11, 3, 10, 0, 0, 0, loc) // un martes, hace 17 días
	s := nuevo(t)
	s.Ahora = func() time.Time { return ahora }
	s.Proyectos = []Proyecto{{Nombre: "P", Raiz: t.TempDir()}}
	s.Diario = t.TempDir()
	viejo := diario.Agente{ID: "ses~borrado", Nombre: "codificador-viejo", Proyecto: "P", Proveedor: "Claude", Tipo: "subagente",
		Clase: "I1", Modelo: "sonnet", Effort: "medium", Inicio: inicio, Fin: inicio.Add(12 * time.Minute), Activo: true, Minutos: 12,
		Nuevos: 5000, Padre: "ses", Corto: "sonnet-5-5/medium"}
	if _, err := diario.Guardar(s.Diario, diario.Dia{Fecha: diario.Fecha(inicio), Agentes: []diario.Agente{viejo}}); err != nil {
		t.Fatal(err)
	}

	inf := s.informeEstadisticas("P", PeriodoMes)
	if inf.Totales.Agentes != 1 || inf.Totales.Minutos != 12 || inf.DiasDiario != 1 {
		t.Errorf("el mes debe contar al del diario: %+v (días %d)", inf.Totales, inf.DiasDiario)
	}
	if semana := s.informeEstadisticas("P", PeriodoSemana); semana.Totales.Agentes != 0 {
		t.Errorf("la semana no lo cuenta (empezó hace 17 días): %+v", semana.Totales)
	}
	var enSuSemana int
	for _, w := range inf.Semanas {
		if w.Desde == "2026-11-02" {
			enSuSemana = w.Agentes
			if w.Mediana != 12 || w.Claude != 1 {
				t.Errorf("su semana: %+v", w)
			}
		}
	}
	if len(inf.Semanas) != semanasEstadisticas || enSuSemana != 1 {
		t.Errorf("semanas: %+v", inf.Semanas)
	}

	l := s.lineaDeTiempo("P", diario.Fecha(inicio))
	if len(l.Filas) != 1 || !l.Filas[0].Diario || l.Filas[0].Vivo || l.Filas[0].Padre != "ses" || l.Filas[0].Clase == "" {
		t.Fatalf("la línea de su día: %+v", l.Filas)
	}
	hay := false
	for _, d := range l.Dias {
		hay = hay || d == diario.Fecha(inicio)
	}
	if !hay {
		t.Errorf("su día debe estar entre los días: %v", l.Dias)
	}
}

func TestDiarioDeLaMemoria(t *testing.T) {
	vivo := t.TempDir()
	base := filepath.Join(vivo, "2026-10-31-e1-x")
	if err := os.WriteFile(base+".meta.json", []byte(`{"tema":"e1-x","clase":"E1","modelo":"gpt-6.1-terra","effort":"high","fase":"F2","raiz":"C:\\Proyectos\\Tienda Demo","inicio":"2026-10-31T10:00:00-05:00","fin":"2026-10-31T10:01:30-05:00","salida":0,"segundos":90}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(base+".jsonl", []byte(`{"type":"turn.completed","usage":{"input_tokens":1000,"cached_input_tokens":800,"output_tokens":50}}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := nuevo(t)
	s.Proyectos = []Proyecto{{Nombre: "Tienda Demo", Raiz: `C:\Proyectos\Tienda Demo`}}
	s.CodexVivo = codexvivo.NuevoLector(vivo)
	if _, err := s.CodexVivo.Actualizar(); err != nil {
		t.Fatal(err)
	}
	ahora := time.Date(2026, 10, 31, 12, 0, 0, 0, time.FixedZone("PE", -5*3600))
	s.alertas = []alertas.Alerta{
		{ID: "tiempo:a:pasado", Nivel: alertas.Error, Proyecto: "Tienda Demo", Regla: "tiempo", Agente: "a", Titulo: "Pasó su tope", Hora: ahora.Add(-time.Hour)},
		{ID: "contexto:a", Nivel: alertas.Aviso, Proyecto: "Tienda Demo", Regla: "contexto"},
	}
	dias := s.diasEnMemoria(ahora)
	d := dias[diario.Fecha(ahora)]
	if d == nil || len(d.Agentes) != 1 || len(d.Alertas) != 1 {
		t.Fatalf("el día: %+v", d)
	}
	a := d.Agentes[0]
	if a.Proyecto != "Tienda Demo" || a.Tipo != "codex" || a.Minutos != 1.5 || a.Clase != "E1" || a.Activo {
		t.Errorf("la llamada: %+v", a)
	}
	if d.Alertas[0].Regla != "tiempo" {
		t.Errorf("solo las alertas de error: %+v", d.Alertas)
	}
	dir := t.TempDir()
	if _, err := diario.Guardar(dir, *d); err != nil {
		t.Fatal(err)
	}
	if got := diario.Fechas(dir); len(got) != 1 || got[0] != d.Fecha {
		t.Errorf("fechas: %v", got)
	}
}
