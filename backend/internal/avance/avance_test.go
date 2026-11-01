package avance

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestMarcasEnInglesYMezcladas(t *testing.T) {
	espanol := "Avance: 50 % — trabajando\nPlan: docs/plan.md\n\n" +
		"| Paquete | Qué | Horas | Estado | % | Evidencia |\n|---|---|---|---|---|---|\n" +
		"| A | uno | 1 | hecho | 100 | a.md |\n| B | dos | 1 | hecho | 100 | b.md |\n" +
		"| C | tres | 1 | hecho | 100 | c.md |\n| D | cuatro | 1 | en curso | 50 | d.md |\n" +
		"| E | cinco | 1 | en curso | 50 | e.md |\n| F | seis | 1 | sin empezar | 0 | f.md |\n" +
		"| G | siete | 1 | sin empezar | 0 | g.md |\n| H | ocho | 1 | sin empezar | 0 | h.md |\n" +
		"- 10:00 · trabajando\n"
	ingles := strings.NewReplacer("Avance:", "Progress:", "Paquete", "PACKAGE", "Qué", "What", "Horas", "Hours", "Estado", "Status", "Evidencia", "Evidence").Replace(espanol)
	mezclado := ingles
	for _, par := range [][2]string{{"hecho", "DONE"}, {"hecho", "Complete"}, {"hecho", "Completed"}, {"en curso", "In Progress"}, {"en curso", "Ongoing"}, {"sin empezar", "Not Started"}, {"sin empezar", "Pending"}, {"sin empezar", "Todo"}} {
		ingles = strings.Replace(ingles, par[0], par[1], 1)
	}
	quiere := Parsear(espanol, "F1", "x")
	for nombre, texto := range map[string]string{"inglés": ingles, "mezclado": mezclado} {
		f := Parsear(texto, "F1", "x")
		if !reflect.DeepEqual(f, quiere) {
			t.Errorf("%s: %+v; se esperaba %+v", nombre, f, quiere)
		}
	}
}

func TestEstadosInglesConPrefijo(t *testing.T) {
	for texto, quiere := range map[string]string{"done · listo": Hecho, "completed · listo": Hecho, "in progress · hoy": EnCurso, "ongoing · hoy": EnCurso, "not started · aún": SinEmpezar, "pending · aún": SinEmpezar, "todo": SinEmpezar, "to do": SinEmpezar, "todo hecho": "todo hecho"} {
		if got := normalizarEstado(texto); got != quiere {
			t.Errorf("%q: %q, se esperaba %q", texto, got, quiere)
		}
	}
}

// Una fase inventada de Tienda Demo con horas y porcentajes para comprobar el cálculo.
const fase5 = "Avance: 83 % — T1b: pruebas de inventario\r\n" +
	"Plan: docs/planes/fase-5.md §4\r\n\r\n" +
	"| Paquete | Qué | Est. (h) | Estado | % | Evidencia |\r\n" +
	"|---|---|---|---|---|---|\r\n" +
	"| **P1** | Catálogo inicial | 3 | Hecho | 100 | `F5/02-catalogo-inicial.md` |\r\n" +
	"| P2 | Registro de clientes | 4 | hecho | 100 | |\r\n" +
	"| P3 | Búsqueda de productos | 5 | hecho | 100 | |\r\n" +
	"| P4 | Carrito de compra | 4 | hecho | 100 | |\r\n" +
	"| P5 | Pago de pedidos | 3 + sesión | en curso | 50 | faltan dos casos |\r\n" +
	"| T1 | Inventario | 4 | en curso | 85 | prueba de inventario |\r\n" +
	"| P7 | Notificaciones | 8 | hecho | 100 | |\r\n" +
	"| P8 | Descuentos | 3 | hecho | 100 | |\r\n" +
	"| P9 | Devoluciones | 4 | hecho | 100 | |\r\n" +
	"| P10 | Validación de pedidos | 5 | hecho | 100 | |\r\n" +
	"| P11 | Informe de ventas | 3 | hecho | 100 | |\r\n" +
	"| P12 | Preparación de entrega | 3 | sin empezar | 0 | |\r\n" +
	"| P13 | Cierre de la tienda | 4 + sesión | sin empezar | | |\r\n" +
	"| **Total** | | ≈ 53 h | | | |\r\n\r\n" +
	"Otra tabla que no es de paquetes:\r\n\r\n| Riesgo | Qué |\r\n|---|---|\r\n| R1 | algo |\r\n\r\n" +
	"- 13:24 · T1b: prueba alternada de inventario\r\n" +
	"- 12:25 · T1a terminada\r\n"

func TestLaCuentaDeLaFase5(t *testing.T) {
	f := Parsear(fase5, "F5", "docs/trazabilidad/F5/AVANCE.md")
	if len(f.Paquetes) != 13 {
		t.Fatalf("paquetes: %d", len(f.Paquetes))
	}
	// 43,9 de 53 h = 82,8 %: el valor declarado (83 %) está redondeado.
	if f.HorasTot != 53 || f.Calculado != 82.8 || f.Declarado != 83 {
		t.Errorf("cuenta: %v h, %v %%, declarado %d", f.HorasTot, f.Calculado, f.Declarado)
	}
	if f.HorasHech != 39 {
		t.Errorf("horas hechas: %v", f.HorasHech)
	}
	if len(f.Avisos) != 0 {
		t.Errorf("avisos: %v", f.Avisos)
	}
	// Si la primera línea se queda atrás de la tabla, se avisa.
	g := Parsear(strings.Replace(fase5, "Avance: 83 %", "Avance: 75 %", 1), "F5", "x")
	if len(g.Avisos) != 1 || !strings.Contains(g.Avisos[0], "dice 75 % y los paquetes dan 82,8 %") {
		t.Errorf("declarado viejo: %v", g.Avisos)
	}
	w1 := f.Paquetes[0]
	if w1.ID != "P1" || w1.Estado != Hecho || w1.Evidencia != "F5/02-catalogo-inicial.md" {
		t.Errorf("P1: %+v", w1)
	}
	if f.Paquetes[12].Pct != 0 {
		t.Errorf("P13 sin empezar y sin %%: %+v", f.Paquetes[12])
	}
	if f.Plan != "docs/planes/fase-5.md §4" || f.Ahora != "T1b: pruebas de inventario" || len(f.Hitos) != 2 {
		t.Errorf("cabecera: %q %q %v", f.Plan, f.Ahora, f.Hitos)
	}
	if p := f.EnCursoAhora(); p == nil || p.ID != "P5" {
		t.Errorf("en curso: %+v", p)
	}
}

func TestAvisosDeUnaTablaMal(t *testing.T) {
	f := Parsear("Avance: 50 % — x\n\n| Paquete | Horas | Estado | % |\n|---|---|---|---|\n| A | 2 | hecho | 90 |\n| B | 2 | sin empezar | 10 |\n| C | | en curso | 40 |\n| D | 2 | en curso | |\n", "F1", "x")
	quiere := []string{"A está hecho pero dice 90 %", "B está sin empezar pero dice 10 %", "C no dice sus horas", "D no dice su %"}
	todo := strings.Join(f.Avisos, " | ")
	for _, q := range quiere {
		if !strings.Contains(todo, q) {
			t.Errorf("falta el aviso %q en %s", q, todo)
		}
	}
}

func TestLeerLasFases(t *testing.T) {
	raiz := t.TempDir()
	for _, f := range []string{"F4", "F5", "F10", "otros"} {
		_ = os.MkdirAll(filepath.Join(raiz, "docs", "trazabilidad", f), 0o755)
	}
	_ = os.WriteFile(filepath.Join(raiz, "docs", "trazabilidad", "F4", "AVANCE.md"), []byte("Avance: 100 % — cerrada\n"), 0o644)
	_ = os.WriteFile(filepath.Join(raiz, "docs", "trazabilidad", "F10", "AVANCE.md"), []byte("Avance: 5 % — empieza\n"), 0o644)
	_ = os.WriteFile(filepath.Join(raiz, "docs", "trazabilidad", "otros", "AVANCE.md"), []byte("Avance: 1 %\n"), 0o644)
	fs := Leer(raiz)
	if len(fs) != 2 || fs[0].Fase != "F10" || fs[1].Fase != "F4" || fs[0].Ruta != "docs/trazabilidad/F10/AVANCE.md" {
		t.Fatalf("fases: %+v", fs)
	}
	if a := Actual(raiz); a == nil || a.Fase != "F10" {
		t.Errorf("actual: %+v", a)
	}
	if Actual(t.TempDir()) != nil || Leer("") != nil {
		t.Error("sin avance")
	}
}

// El último hito no puede ser posterior a la hora en que se guardó el archivo (se vio «22:55» en un AVANCE.md guardado
// a las 21:30). Un hito de la víspera, 20 h «adelante», no cuenta.
func TestHitoConHoraInventada(t *testing.T) {
	guardado := time.Date(2026, 10, 31, 21, 30, 35, 0, time.Local)
	for _, c := range []struct {
		hora  string
		avisa bool
	}{{"22:55", true}, {"21:30", false}, {"21:31", false}, {"20:10", false}, {"17:45", false}} {
		f := Parsear("Avance: 0 % — x\n\n- "+c.hora+" · algo\n- 09:00 · antes\n", "F6", "x")
		f.horaDelUltimoHito(guardado)
		if got := strings.Contains(strings.Join(f.Avisos, "|"), "posterior"); got != c.avisa {
			t.Errorf("%s guardado 21:30: avisa %v, quería %v (%q)", c.hora, got, c.avisa, f.Avisos)
		}
	}
	// Guardado a las 00:10 con el último hito de anoche a las 23:50: es de la víspera.
	f := Parsear("Avance: 0 % — x\n\n- 23:50 · anoche\n", "F6", "x")
	f.horaDelUltimoHito(time.Date(2026, 11, 1, 0, 10, 0, 0, time.Local))
	if len(f.Avisos) != 0 {
		t.Errorf("un hito de la víspera no es inventado: %q", f.Avisos)
	}
}

// Una fase recién abierta: la tabla de AVANCE.md tiene solo un marcador y los paquetes están en el plan. El marcador no
// es un paquete (antes se pintaba como uno de nombre largo, sin horas, y rompía la vista) y los paquetes salen del plan.
func TestUnaFaseConLosPaquetesEnElPlan(t *testing.T) {
	raiz := t.TempDir()
	fase := filepath.Join(raiz, "docs", "trazabilidad", "F6")
	_ = os.MkdirAll(fase, 0o755)
	_ = os.MkdirAll(filepath.Join(raiz, "docs", "planes"), 0o755)
	avance := "Avance: 0 % — fase 6 en diseño\nPlan: docs/planes/fase-6.md §4\n\n" +
		"| Paquete | Qué | Horas | Estado | % | Evidencia |\n|---|---|---|---|---|---|\n| (W1 a W12, al aprobarse el plan) | | | | | |\n"
	_ = os.WriteFile(filepath.Join(fase, "AVANCE.md"), []byte(avance), 0o644)
	plan := "# Plan\n\n## 4. Resumen\n\n| Paquete | Qué | Clase · rol | Est. (h) |\n|---|---|---|---|\n" +
		"| **W1** | Hilos | I2 | 6 |\n| **W12** | Cierre | V | 4 + sesión |\n| **Total** | | | **≈ 10 h** |\n\n" +
		"## 8. Riesgos\n\n| Paquete | Riesgo |\n|---|---|\n| W1 | rompe algo sutil |\n"
	_ = os.WriteFile(filepath.Join(raiz, "docs", "planes", "fase-6.md"), []byte(plan), 0o644)

	if ps := Parsear(avance, "F6", "x").Paquetes; len(ps) != 0 {
		t.Fatalf("el marcador se leyó como paquete: %+v", ps)
	}
	f := Leer(raiz)[0]
	if len(f.Paquetes) != 2 || f.Paquetes[0].ID != "W1" || f.Paquetes[1].ID != "W12" || f.HorasTot != 10 || f.Calculado != 0 {
		t.Fatalf("paquetes del plan: %+v (horas %v)", f.Paquetes, f.HorasTot)
	}
	if f.Paquetes[0].Estado != SinEmpezar || len(f.Avisos) != 1 || !strings.Contains(f.Avisos[0], "salen del plan (docs/planes/fase-6.md)") {
		t.Errorf("estado o avisos: %+v %q", f.Paquetes[0], f.Avisos)
	}

	// Un plan fuera del proyecto no se lee.
	_ = os.WriteFile(filepath.Join(fase, "AVANCE.md"), []byte("Avance: 0 % — x\nPlan: ../fuera.md\n"), 0o644)
	_ = os.WriteFile(filepath.Join(filepath.Dir(raiz), "fuera.md"), []byte("| Paquete | Horas |\n|---|---|\n| X | 3 |\n"), 0o644)
	OlvidarMemoria() // el archivo cambió recién: sin esperar la vuelta
	if ps := Leer(raiz)[0].Paquetes; len(ps) != 0 {
		t.Errorf("leyó un plan fuera del proyecto: %+v", ps)
	}
}
