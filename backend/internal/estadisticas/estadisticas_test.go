package estadisticas

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const agentesMd = "# Agentes\n\n" +
	"## 2. Clasificación\n\n" +
	"| Clase | Qué es | Ejemplos |\n| --- | --- | --- |\n" +
	"| **M · Mecánica** | Todo decidido | Subir la versión |\n" +
	"| **I2 · Implementación con decisiones** | Decidir cómo | El ledger |\n" +
	"| **R · Búsqueda e investigación** | R1, R2 y R3 | — |\n\n" +
	"## 3. Matriz: clase → rol → modelo y effort\n\n" +
	"| Clase | Rol | Modelo · effort | Revisión | Autonomía |\n| --- | --- | --- | --- | --- |\n" +
	"| M | `ejecutor` | Haiku · `medium` (`low` si es copiar) | Automática | Ninguna |\n" +
	"| I2 | `codificador` | Sonnet · `high` | Auditoría E2 | Media |\n" +
	"| R1 | `Explore` | Haiku · `low` | Se comprueba | — |\n\n" +
	"## 12. Medición\n\n" +
	"| Porcentaje de agentes en Opus | Mezcla de costo | Meta inicial sugerida: menos del 40 % sin que suban los defectos |\n\n" +
	"### 14.2 Qué va a Codex y qué no\n\n" +
	"| Clase | Con Codex | Por qué |\n| --- | --- | --- |\n" +
	"| R1 | **Codex `gpt-6-luna` · `medium`**, en lugar de `Explore` | Acertó |\n" +
	"| A o E2 irreversible | **Codex `gpt-6-astra` · `high`** | Otra familia |\n"

func matrizDePrueba(t *testing.T) Matriz {
	t.Helper()
	raiz := t.TempDir()
	_ = os.MkdirAll(filepath.Join(raiz, "docs", "operacion"), 0o755)
	_ = os.WriteFile(filepath.Join(raiz, "AGENTS.md"), []byte("| Qué agente | `docs/operacion/AGENTES.md`, `ROLES.md` |\n"), 0o644)
	_ = os.WriteFile(filepath.Join(raiz, "docs", "operacion", "AGENTES.md"), []byte(agentesMd), 0o644)
	return LeerMatriz(raiz)
}

func TestAgentesPorClaseYProveedor(t *testing.T) {
	base := time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)
	var ags []Agente
	for i := 0; i < 45; i++ {
		ags = append(ags, Agente{ID: fmt.Sprint(i), Nombre: "agente", Clase: "I1", Proveedor: "Codex", Modelo: "gpt-6.1-terra", Effort: "high", Minutos: 2, Presupuesto: 30, Fin: base.Add(time.Duration(i) * time.Minute)})
	}
	ags = append(ags, Agente{ID: "claude", Clase: "I1", Proveedor: "Claude", Modelo: "sonnet", Effort: "medium", Fin: base})
	ags = append(ags, Agente{ID: "otra", Clase: "M", Proveedor: "Codex", Fin: base})
	inf := Calcular(Matriz{}, 30, "tope", ags)
	grupos := map[string]Clase{}
	for _, c := range inf.Clases {
		grupos[c.Clase+"/"+c.Proveedor] = c
		b, err := json.Marshal(c)
		if err != nil || strings.Contains(string(b), `"agentes":null`) {
			t.Fatalf("agentes en JSON: %s, %v", b, err)
		}
	}
	c := grupos["I1/Codex"]
	if c.N != 45 || len(c.Agentes) != 40 {
		t.Fatalf("cantidad: N=%d, agentes=%d", c.N, len(c.Agentes))
	}
	for i, a := range c.Agentes {
		if a.ID != fmt.Sprint(44-i) || a.Nombre != "agente" || a.Proveedor != "Codex" || a.Modelo != "gpt-6.1-terra/high" || a.Minutos != 2 || a.Presupuesto != 30 || !a.Fin.Equal(base.Add(time.Duration(44-i)*time.Minute)) {
			t.Errorf("agente %d: %+v", i, a)
		}
	}
	if g := grupos["I1/Claude"].Agentes; len(g) != 1 || g[0].ID != "claude" || g[0].Modelo != "sonnet/medium" {
		t.Errorf("I1/Claude: %+v", g)
	}
	if g := grupos["M/Codex"].Agentes; len(g) != 1 || g[0].ID != "otra" {
		t.Errorf("M/Codex: %+v", g)
	}
}

func TestLeeLaMatrizDelDocumento(t *testing.T) {
	m := matrizDePrueba(t)
	if m.Ruta != "docs/operacion/AGENTES.md" || strings.Join(m.Orden, ",") != "M,I2,R1" {
		t.Fatalf("matriz: %q %v", m.Ruta, m.Orden)
	}
	if c := m.Claude["M"]; c.Texto() != "haiku/medium" || c.Fuente != "AGENTES.md:15" {
		t.Errorf("M: %+v", c)
	}
	if c := m.Claude["I2"]; c.Texto() != "sonnet/high" {
		t.Errorf("I2: %+v", c)
	}
	if c := m.Codex["R1"]; c.Texto() != "gpt-6-luna/medium" {
		t.Errorf("Codex R1: %+v", c)
	}
	if len(m.Codex) != 1 {
		t.Errorf("una fila con varias clases no entra: %v", m.Codex)
	}
	if m.NombreDe("I2") != "Implementación con decisiones" || m.NombreDe("R1") != "Búsqueda e investigación" || m.NombreDe("X9") != "" {
		t.Errorf("nombres: %v", m.Nombres)
	}
	if math.Abs(m.MetaOpus-0.40) > 1e-9 || m.FuenteOpus != "AGENTES.md:21" {
		t.Errorf("meta de Opus: %v %q", m.MetaOpus, m.FuenteOpus)
	}
	// Sin AGENTS.md que nombre el documento, la matriz va vacía.
	if v := LeerMatriz(t.TempDir()); v.Ruta != "" || len(v.Claude) != 0 {
		t.Errorf("sin documento: %+v", v)
	}
}

func TestPercentil(t *testing.T) {
	v := []float64{10, 2, 7, 4}
	if percentil(v, 0.5) != 5.5 || percentil(v, 0.9) != 10 || percentil([]float64{3}, 0.9) != 3 || percentil(nil, 0.5) != 0 {
		t.Errorf("%v %v", percentil(v, 0.5), percentil(v, 0.9))
	}
	if percentil([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 0.9) != 9 {
		t.Error("p90 de 10 datos")
	}
}

func TestCalcular(t *testing.T) {
	m := matrizDePrueba(t)
	ags := []Agente{
		{Clase: "I2", Proveedor: "Claude", Modelo: "sonnet", Effort: "high", Minutos: 20, Salida: 40_000, Nuevos: 100, Leida: 900, Presupuesto: 30},
		{Clase: "I2", Proveedor: "Claude", Modelo: "sonnet", Effort: "medium", Minutos: 30, Salida: 60_000, Nuevos: 100, Leida: 900, Presupuesto: 45},
		{Clase: "I2", Proveedor: "Claude", Modelo: "opus", Effort: "high", Minutos: 40, Salida: 80_000, Nuevos: 100, Leida: 900},
		{Clase: "R1", Proveedor: "Codex", Modelo: "gpt-6-luna", Effort: "medium", Minutos: 1, Salida: 2_000},
		{Clase: "M", Proveedor: "Claude", Modelo: "haiku", Effort: "medium", Minutos: 4},
		{Proveedor: "Claude", Modelo: "sonnet", Minutos: 9},
	}
	inf := Calcular(m, 30, "el tope de AGENTS.md:25", ags)
	tt := inf.Totales
	if tt.Agentes != 6 || tt.Claude != 5 || tt.Codex != 1 || tt.Minutos != 104 || tt.Opus != 1 || tt.OpusPct != 20 || tt.CachePct != 90 {
		t.Fatalf("totales: %+v", tt)
	}
	var orden []string
	for _, c := range inf.Clases {
		orden = append(orden, c.Clase+"/"+c.Proveedor)
	}
	// El orden de la matriz y, al final, lo que no tiene clase.
	if strings.Join(orden, ",") != "M/Claude,I2/Claude,R1/Codex,—/Claude" {
		t.Fatalf("orden: %v", orden)
	}
	i2 := inf.Clases[1]
	if i2.N != 3 || i2.Minutos.Mediana != 30 || i2.Minutos.P90 != 40 || i2.Salida.Mediana != 60_000 {
		t.Errorf("I2: %+v", i2)
	}
	// El presupuesto: la mediana de lo que dicen las asignaciones (30 y 45); el que no lo dice no cuenta.
	if i2.Presupuesto != 37.5 || i2.PresupuestoFuente != "las asignaciones de la clase" {
		t.Errorf("presupuesto: %v %q", i2.Presupuesto, i2.PresupuestoFuente)
	}
	// Solo uno corrió con sonnet/high: el de medium y el de opus no siguen la matriz.
	if i2.Matriz != "sonnet/high" || i2.EnMatriz != 1 {
		t.Errorf("matriz: %q %d", i2.Matriz, i2.EnMatriz)
	}
	if r1 := inf.Clases[2]; r1.Matriz != "gpt-6-luna/medium" || r1.EnMatriz != 1 || r1.Presupuesto != 30 || r1.PresupuestoFuente != "el tope de AGENTS.md:25" {
		t.Errorf("R1: %+v", r1)
	}
	if strings.Join(inf.Modelos, ",") != "haiku,sonnet,opus,gpt-6-luna" {
		t.Errorf("modelos: %v", inf.Modelos)
	}
	enMatriz := map[string]bool{}
	for _, c := range inf.Celdas {
		enMatriz[c.Clase+"/"+c.Modelo] = c.EnMatriz
	}
	if !enMatriz["I2/sonnet"] || enMatriz["I2/opus"] || !enMatriz["R1/gpt-6-luna"] {
		t.Errorf("celdas: %v", enMatriz)
	}
}
