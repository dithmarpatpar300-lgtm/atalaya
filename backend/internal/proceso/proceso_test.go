package proceso

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"atalaya/internal/alertas"
)

func reglasDe(t *testing.T, agents string) (string, alertas.Reglas) {
	t.Helper()
	raiz := t.TempDir()
	if err := os.WriteFile(filepath.Join(raiz, "AGENTS.md"), []byte(agents), 0o644); err != nil {
		t.Fatal(err)
	}
	return raiz, alertas.Leer(raiz)
}

const agentsMd = "# Proyecto\n\n" +
	"- Temporales en `X:\\Algo-tmp\\<fecha>-<rol>\\`, borrados al terminar.\n" +
	"- **Antes de lanzar un agente se mira la cuota, siempre:**\n" +
	"  - si el dato de Claude tiene más de 15 min, se importa.\n" +
	"- El effort se fija siempre explícito.\n" +
	"- El encargo se redacta sobre la plantilla y su primera línea es la asignación.\n" +
	"- **El acta es obligatoria** para toda carpeta.\n" +
	"- **Cada agente lleva una bitácora** `avance/<rol>.md`.\n" +
	"- se comprueba que no quedan temporales (`scripts/ops/retomar.ps1` los lista).\n" +
	"- Agentes en paralelo: nunca hay dos sobre el mismo archivo.\n"

func TestLasColumnasCitanAgentsMd(t *testing.T) {
	_, r := reglasDe(t, agentsMd)
	if r.Frescura != 15*time.Minute || r.TmpRaiz != `X:\Algo-tmp` || r.Retomar != "scripts/ops/retomar.ps1" {
		t.Fatalf("reglas: %v %q %q", r.Frescura, r.TmpRaiz, r.Retomar)
	}
	fuentes := map[string]string{}
	for _, c := range Columnas(r) {
		fuentes[c.ID] = c.Fuente
		if c.Omision {
			t.Errorf("%s por omisión", c.ID)
		}
	}
	want := map[string]string{"temporales": "AGENTS.md:3", "cuota": "AGENTS.md:4", "effort": "AGENTS.md:6", "asignacion": "AGENTS.md:7",
		"acta": "AGENTS.md:8", "bitacora": "AGENTS.md:9", "frontera": "AGENTS.md:11"}
	for k, v := range want {
		if fuentes[k] != v {
			t.Errorf("%s: %q, quería %q", k, fuentes[k], v)
		}
	}
	// Sin AGENTS.md: cada columna dice que va por omisión.
	for _, c := range Columnas(alertas.Leer(t.TempDir())) {
		if !c.Omision || c.Fuente != "" {
			t.Errorf("sin documento: %+v", c)
		}
	}
}

func TestLugaresDelScript(t *testing.T) {
	raiz, r := reglasDe(t, agentsMd)
	_ = os.MkdirAll(filepath.Join(raiz, "scripts", "ops"), 0o755)
	t.Setenv("ATALAYA_PRUEBA", `Y:\Local`)
	script := "$places = @(\n    'X:\\Algo-tmp',\n    (Join-Path $env:ATALAYA_PRUEBA 'Temp\\Algo-tmp'),\n    'F:\\Algo-tmp'\n)\n$exe = 'target\\release\\x.exe'\n"
	_ = os.WriteFile(filepath.Join(raiz, "scripts", "ops", "retomar.ps1"), []byte(script), 0o644)
	got := strings.Join(Lugares(raiz, r), "|")
	if got != `X:\Algo-tmp|F:\Algo-tmp|`+filepath.Join(`Y:\Local`, `Temp\Algo-tmp`) {
		t.Errorf("lugares: %s", got)
	}
}

func TestEscanearMideLasCarpetas(t *testing.T) {
	lugar := t.TempDir()
	c := filepath.Join(lugar, "2026-10-31-medidor-w6")
	_ = os.MkdirAll(filepath.Join(c, "sub"), 0o755)
	_ = os.WriteFile(filepath.Join(c, "a.bin"), make([]byte, 2048), 0o644)
	_ = os.WriteFile(filepath.Join(c, "sub", "b.rgpart"), make([]byte, 100), 0o644)
	_ = os.WriteFile(filepath.Join(lugar, "suelto.txt"), []byte("x"), 0o644)
	cs := Escanear([]string{lugar, filepath.Join(lugar, "no-existe")})
	if len(cs) != 1 || cs[0].Archivos != 2 || cs[0].Bytes != 2148 || cs[0].Parciales != 1 || cs[0].Nombre != "2026-10-31-medidor-w6" {
		t.Fatalf("carpetas: %+v", cs)
	}
	if Firma([]string{lugar}) == "" {
		t.Error("sin firma")
	}
	if Peso(2148) != "2,1 KB" || Peso(41*1024*1024) != "41,0 MB" {
		t.Errorf("peso: %s %s", Peso(2148), Peso(41*1024*1024))
	}
}

func marcas(f Fila) string {
	var b strings.Builder
	for _, c := range f.Celdas {
		b.WriteString(map[string]string{Cumple: "✓", Falta: "✕", Medias: "!", NoAplica: "—", SinDato: "?"}[c.Marca])
	}
	return b.String()
}

func TestEvaluar(t *testing.T) {
	_, r := reglasDe(t, agentsMd)
	lanzado := time.Date(2026, 10, 31, 10, 30, 0, 0, time.Local)
	bien := Agente{ID: "a", Nombre: "codificador-w6a", Proveedor: "Claude", Estado: "Entregado", Inicio: lanzado,
		LineaAsignacion: 1, Nivel: "enfocado", ClaseAsignada: "I2", Rol: "codificador", Modelo: "sonnet/high", Presupuesto: "30 min",
		Base: "main", CuotaAsignada: "Codex normal", EffortLlamada: "high", Lanzado: lanzado, Mirada: lanzado.Add(-5 * time.Minute),
		MiradaEn: "~/ses.jsonl", PasoMirada: 3, Sesion: "ses", Bitacora: "docs/trazabilidad/F5/avance/codificador-w6a.md", Avance: 100,
		Acta: "docs/trazabilidad/F5/temporales/2026-10-31-codificador-w6a.md", ActaEstado: "cerrada", Temporales: []string{"2026-10-31-codificador-w6a"}}
	mal := Agente{ID: "b", Nombre: "medidor-w6", Proveedor: "Claude", Estado: "Entregado", Inicio: lanzado.Add(time.Minute),
		LineaAsignacion: 2, Nivel: "enfocado", ClaseAsignada: "T1", Rol: "medidor", Modelo: "sonnet", Presupuesto: "30 min",
		Lanzado: lanzado, Mirada: lanzado.Add(-70 * time.Minute), LanzadoEn: "~/ses.jsonl", PasoLanzado: 9, Sesion: "ses", Bitacora: "docs/x.md", Avance: 60,
		Acta: "docs/trazabilidad/F5/temporales/2026-10-31-medidor-w6.md", ActaEstado: "abierta", Temporales: []string{"2026-10-31-medidor-w6"}}
	vivo := mal
	vivo.ID, vivo.Activo, vivo.Inicio, vivo.Temporales = "c", true, lanzado.Add(2*time.Minute), []string{"2026-10-31-medidor-w6x"}
	sinLanzador := Agente{ID: "d", Proveedor: "Claude", Inicio: lanzado.Add(-time.Hour), LineaAsignacion: 0, Avance: -1, CuotaAsignada: "normal"}
	codex := Agente{ID: "e", Proveedor: "Codex", Inicio: lanzado.Add(-2 * time.Hour), ClaseAsignada: "E1", Modelo: "gpt/high", Base: "main",
		EffortLlamada: "high", Lanzado: lanzado, Avance: -1, Temporales: []string{"2026-10-31-codex-e1"}}
	carpetas := []Carpeta{{Ruta: `X:\Algo-tmp\2026-10-31-medidor-w6`, Nombre: "2026-10-31-medidor-w6", Archivos: 3, Bytes: 41 * 1024 * 1024},
		{Ruta: `X:\Algo-tmp\2026-10-31-medidor-w6x`, Nombre: "2026-10-31-medidor-w6x", Archivos: 1, Bytes: 10}}
	filas, cs := Evaluar(r, []Agente{bien, mal, vivo, sinLanzador, codex}, carpetas)
	got := map[string]string{}
	for _, f := range filas {
		got[f.ID] = marcas(f)
	}
	want := map[string]string{
		"a": "✓✓✓✓✓✓✓",
		// Asignación en la segunda línea y sin base; sin effort; cuota de hace 70 min con postura vacía; 60 % al entregar.
		"b": "!✕✕✕✕✕✓",
		"c": "!✕✕!!!✓",
		"d": "✕✕?✕——✓",
		"e": "✓✓✕——✓—",
	}
	for id, w := range want {
		if got[id] != w {
			t.Errorf("%s: %s, quería %s", id, got[id], w)
		}
	}
	if filas[0].ID != "c" {
		t.Errorf("orden: primero %s", filas[0].ID)
	}
	if len(cs) != 2 || cs[0].Agente != "b" || cs[0].Estado != "agente entregado" || cs[1].Agente != "c" || cs[1].Estado != "en uso" {
		t.Errorf("carpeta: %+v", cs)
	}
	// La ruta y el paso van aparte: la cuota apunta al paso de la sesión; las demás celdas, a un documento sin paso.
	for _, f := range filas {
		quiere := map[string]int{"a": 3, "b": 9, "c": 9}
		for i, c := range f.Celdas {
			switch {
			case i == 2 && quiere[f.ID] > 0:
				if c.Paso != quiere[f.ID] || c.Agente != "ses" || c.Ruta != "~/ses.jsonl" {
					t.Errorf("%s: la cuota no apunta al paso: %+v", f.ID, c)
				}
			case c.Paso != -1 || c.Agente != "" || strings.Contains(c.Ruta, "paso"):
				t.Errorf("%s: celda %s con paso: %+v", f.ID, c.Regla, c)
			}
		}
	}
	for _, f := range filas {
		if f.ID != "b" {
			continue
		}
		if f.Faltas != 5 || f.Medias != 1 {
			t.Errorf("cuentas: %d %d", f.Faltas, f.Medias)
		}
		c := f.Celdas[2]
		if !strings.Contains(c.Texto, "70 min") || c.Ruta == "" {
			t.Errorf("cuota: %+v", c)
		}
		if f.Celdas[5].Ruta != carpetas[0].Ruta || !strings.Contains(f.Celdas[5].Texto, "41,0 MB") {
			t.Errorf("temporal: %+v", f.Celdas[5])
		}
	}
}

func TestTemporalRetomadoPorUnRelanzamiento(t *testing.T) {
	_, r := reglasDe(t, agentsMd)
	viejo := Agente{ID: "viejo", Nombre: "auditor-w6b", Proveedor: "Claude", Avance: -1, Temporales: []string{"2026-10-31-auditor-w6b"}}
	nuevo := Agente{ID: "nuevo", Nombre: "auditor-w6b", Proveedor: "Claude", Activo: true, Avance: -1, Temporales: []string{"2026-10-31-auditor-w6b"}}
	carpetas := []Carpeta{{Ruta: `X:\Algo-tmp\2026-10-31-auditor-w6b`, Nombre: "2026-10-31-auditor-w6b"}}
	// El que se cortó llega primero (más reciente en la lista no importa): la carpeta es del que sigue vivo.
	filas, cs := Evaluar(r, []Agente{viejo, nuevo}, carpetas)
	if cs[0].Agente != "nuevo" || cs[0].Estado != "en uso" {
		t.Errorf("carpeta: %+v", cs[0])
	}
	for _, f := range filas {
		if c := f.Celdas[5]; c.Marca != Medias {
			t.Errorf("%s: %+v", f.ID, c)
		}
	}
}

func TestFronteraEnLaSalud(t *testing.T) {
	_, r := reglasDe(t, agentsMd)
	h := time.Date(2026, 10, 31, 12, 40, 0, 0, time.Local)
	conCoordinador := Agente{ID: "x", Proveedor: "Claude", Avance: -1,
		Cruces: []Cruce{{Otro: "Atalaya (la sesión)", Ruta: "frontend/src/lib/api.ts", Hora: h, HoraOtro: h.Add(-3 * time.Minute), Coordinador: true}}}
	enParalelo := Agente{ID: "y", Proveedor: "Claude", Avance: -1,
		Cruces: []Cruce{{Otro: "Codificador I2", Ruta: "controller.rs", Hora: h, HoraOtro: h, Abierto: true}}}
	filas, _ := Evaluar(r, []Agente{conCoordinador, enParalelo}, nil)
	for _, f := range filas {
		c := f.Celdas[6]
		switch f.ID {
		case "x":
			if c.Marca != Medias || c.Ruta != "frontend/src/lib/api.ts" || !strings.Contains(c.Texto, "12:40 y 12:37, cerrado") {
				t.Errorf("con el coordinador: %+v", c)
			}
		case "y":
			if c.Marca != Falta || !strings.Contains(c.Texto, "abierto") {
				t.Errorf("en paralelo: %+v", c)
			}
		}
	}
}

func TestSinAgentsMdSoloLaFrontera(t *testing.T) {
	r := alertas.Leer(t.TempDir())
	a := Agente{ID: "x", Proveedor: "Claude", Avance: -1, Temporales: []string{"2026-10-31-x"},
		Cruces: []Cruce{{Otro: "Codificador I2", Ruta: "a.go"}}}
	filas, _ := Evaluar(r, []Agente{a}, nil)
	if got := marcas(filas[0]); got != "——————✕" {
		t.Errorf("sin reglas escritas: %s", got)
	}
}

func TestBitacoraRetomada(t *testing.T) {
	_, r := reglasDe(t, agentsMd)
	a := Agente{ID: "x", Proveedor: "Claude", Avance: -1, BitacoraAjena: "docs/trazabilidad/F5/avance/auditor-w6b.md"}
	filas, _ := Evaluar(r, []Agente{a}, nil)
	if c := filas[0].Celdas[3]; c.Marca != SinDato || c.Ruta != a.BitacoraAjena {
		t.Errorf("bitácora: %+v", c)
	}
}
