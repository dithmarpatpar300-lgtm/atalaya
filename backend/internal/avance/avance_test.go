package avance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
