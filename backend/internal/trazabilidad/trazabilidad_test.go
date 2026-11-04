package trazabilidad

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParsearAvance(t *testing.T) {
	av := ParsearAvance("Avance: 85 % — check en verde\r\n\r\nDesglose: …\r\n- 21:32 · 5 % · leído el ADR\n- 22:08 · 100 % · informe hecho\n", "x.md")
	// El último hito (100 %) va más adelante que la primera línea (85 %): vale el hito (TestAvanceDesfasado).
	if av.Porcentaje != 100 || av.Desfase == "" || av.Texto != "check en verde" || len(av.Hitos) != 2 || av.Hitos[1].Hora != "22:08" {
		t.Errorf("%+v", av)
	}
	if av := ParsearAvance("sin avance", "y.md"); av.Porcentaje != -1 {
		t.Errorf("sin dato debería ser -1: %+v", av)
	}
}

// La primera línea puede copiar el % con decimales, como lo calcula Atalaya.
func TestParsearConDecimales(t *testing.T) {
	for texto, quiere := range map[string]int{"Avance: 88,2 % — sigue": 88, "Progress: 88.6 % — sigue": 89, "Avance: 100,0 % — fin": 100} {
		av := ParsearAvance(texto+"\n", "x.md")
		if av.Porcentaje != quiere || av.Texto == "" {
			t.Errorf("%q: %d %%, texto %q; se esperaba %d %%", texto, av.Porcentaje, av.Texto, quiere)
		}
	}
}

func TestParsearProgress(t *testing.T) {
	for _, guion := range []string{"—", "–", "-"} {
		av := ParsearAvance("Progress: 42 % "+guion+" reading\n- 10:00 · 40 % · started\n", "x.md")
		if av.Porcentaje != 42 || av.Texto != "reading" || len(av.Hitos) != 1 || av.Hitos[0].Texto != "40 % · started" {
			t.Errorf("guion %q: %+v", guion, av)
		}
	}
}

// La primera línea quedó en 5 % y el último hito dice 75 %: vale el hito, y el desfase se cuenta. Al revés (la
// cabecera más adelante que los hitos) manda la cabecera.
func TestAvanceDesfasado(t *testing.T) {
	av := ParsearAvance("Avance: 5 % — leyendo\n\n- 10:51 · 5 % · leído\n- 11:07 · 75 % · corpus escrito\n- 11:09 · sin % aquí\n", "x.md")
	if av.Porcentaje != 75 || av.Desfase != "la primera línea dice 5 %; el último hito (11:07), 75 %" {
		t.Fatalf("desfasado: %d %q", av.Porcentaje, av.Desfase)
	}
	av = ParsearAvance("Avance: 100 % — entregado\n\n- 11:11 · 85 % · lint\n", "x.md")
	if av.Porcentaje != 100 || av.Desfase != "" {
		t.Fatalf("cabecera al día: %d %q", av.Porcentaje, av.Desfase)
	}
}

func TestParsearFilaCodex(t *testing.T) {
	fila := "- 2026-10-30 · e2-c3max-sol · E2 · gpt-6.1-sol/high · codex-cli 0.166.0 · commit a8b0b0d + cambios sin commit (19 archivos nuevos) · 107 s · salida 0 · 377272 entrada (319872 en cache) / 3968 salida · contraste: —"
	ll, ok := ParsearFilaCodex(fila)
	if !ok {
		t.Fatal("no la leyó")
	}
	if ll.Tema != "e2-c3max-sol" || ll.Clase != "E2" || ll.Modelo != "gpt-6.1-sol" || ll.Effort != "high" || ll.Segundos != 107 || ll.Salida != 0 {
		t.Errorf("%+v", ll)
	}
	if ll.Entrada != 377272-319872 || ll.EnCache != 319872 || ll.TokSalida != 3968 || ll.Contraste != "—" {
		t.Errorf("tokens o contraste: %+v", ll)
	}
	if ll.Base != "commit a8b0b0d + cambios sin commit (19 archivos nuevos)" {
		t.Errorf("base %q", ll.Base)
	}
	// El contraste puede traer « · » propio.
	ll, ok = ParsearFilaCodex("- 2026-10-30 · r1 · R1 · luna/medium · cli · commit f · 34 s · salida 0 · 10 entrada (4 en cache) / 2 salida · contraste: bien · 0 falsos")
	if !ok || ll.Contraste != "bien · 0 falsos" {
		t.Errorf("%+v %v", ll, ok)
	}
	if _, ok := ParsearFilaCodex("Una linea por llamada"); ok {
		t.Error("el encabezado no es una fila")
	}
}

func TestParsearFilaCodexIdiomas(t *testing.T) {
	// La base y el contraste contienen separadores propios.
	es := "- 2026-10-30 · tema · E2 · modelo/high · cli 1 · commit a · cambios · 34 s · salida -1 · 10 entrada (4 en cache) / 2 salida · contraste: bien · detalle"
	en := "- 2026-10-30 · tema · E2 · modelo/high · cli 1 · commit a · cambios · 34 s · exit -1 · 10 in (4 cached) / 2 out · review: bien · detalle"
	mezcla := "- 2026-10-30 · tema · E2 · modelo/high · cli 1 · commit a · cambios · 34 s · exit -1 · 10 entrada (4 cached) / 2 salida · review: bien · detalle"
	esperada := LlamadaCodex{Fecha: "2026-10-30", Tema: "tema", Clase: "E2", Modelo: "modelo", Effort: "high", Version: "cli 1", Base: "commit a · cambios", Segundos: 34, Salida: -1, Entrada: 6, EnCache: 4, TokSalida: 2, Contraste: "bien · detalle"}
	for _, fila := range []string{es, en, mezcla} {
		ll, ok := ParsearFilaCodex(fila)
		if !ok || ll != esperada {
			t.Errorf("fila %q: %+v, ok=%v; esperada %+v", fila, ll, ok, esperada)
		}
	}
}

func TestLeerDocLimitaLaRuta(t *testing.T) {
	raiz := t.TempDir()
	_ = os.MkdirAll(filepath.Join(raiz, "docs"), 0o755)
	_ = os.WriteFile(filepath.Join(raiz, "docs", "a.md"), []byte("hola ghp_abcdefghijklmnopqrstuvwxyz0123"), 0o644)
	_ = os.WriteFile(filepath.Join(raiz, "secreto.md"), []byte("no"), 0o644)
	if s, err := LeerDoc(raiz, "docs/a.md"); err != nil || s != "hola gh_[tapado]" {
		t.Errorf("%q %v", s, err)
	}
	for _, mala := range []string{"secreto.md", "../x/docs/a.md", "docs/../secreto.md", "docs/a.txt", `C:\docs\a.md`, "docsx/a.md"} {
		if _, err := LeerDoc(raiz, mala); err == nil {
			t.Errorf("%q debería rechazarse", mala)
		}
	}
}

// LeerAvance guarda lo parseado mientras el archivo no cambie: una bitácora que cambia se vuelve a leer, y tocar lo
// devuelto no cambia lo guardado.
func TestLeerAvanceReleeAlCambiar(t *testing.T) {
	raiz := t.TempDir()
	ruta := filepath.Join(raiz, "avance.md")
	if err := os.WriteFile(ruta, []byte("Avance: 10 % — empezando\n\n- 10:00 · leído\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	av, err := LeerAvance(raiz, "avance.md")
	if err != nil || av.Porcentaje != 10 || len(av.Hitos) != 1 {
		t.Fatalf("primera lectura: %+v %v", av, err)
	}
	av.Hitos[0].Texto, av.Porcentaje = "tocado", 99
	if av, _ := LeerAvance(raiz, "avance.md"); av.Porcentaje != 10 || av.Hitos[0].Texto != "leído" {
		t.Fatalf("lo guardado cambió con la copia: %+v", av)
	}
	// Otro tamaño y otra hora: LeerTapado lo relee y el texto nuevo se parsea de nuevo.
	if err := os.WriteFile(ruta, []byte("Avance: 60 % — a medio camino\n\n- 10:00 · leído\n- 10:30 · probado\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(ruta, time.Now().Add(time.Minute), time.Now().Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	// Dentro de la memoria corta todavía sale la de antes; OlvidarAvance (o que pase VidaAvance) la hace mirar.
	if av, _ := LeerAvance(raiz, "avance.md"); av.Porcentaje != 10 {
		t.Fatalf("dentro de VidaAvance debía salir la guardada: %+v", av)
	}
	OlvidarAvance()
	if av, _ := LeerAvance(raiz, "avance.md"); av.Porcentaje != 60 || len(av.Hitos) != 2 || av.Texto != "a medio camino" {
		t.Fatalf("no releyó la bitácora que cambió: %+v", av)
	}
	// Y pasada la vida, sin olvidar: otro cambio se ve.
	vida := VidaAvance
	VidaAvance = 0
	defer func() { VidaAvance = vida }()
	if err := os.WriteFile(ruta, []byte("Avance: 100 % — entregado\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(ruta, time.Now().Add(2*time.Minute), time.Now().Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if av, _ := LeerAvance(raiz, "avance.md"); av.Porcentaje != 100 || len(av.Hitos) != 0 {
		t.Fatalf("pasada la vida no releyó: %+v", av)
	}
}
