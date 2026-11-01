package claude

import (
	"strings"
	"testing"
)

func TestBuscarPasos(t *testing.T) {
	raiz, _ := preparar(t)
	ix := NuevoIndice(raiz)
	if _, err := ix.Actualizar(); err != nil {
		t.Fatal(err)
	}
	buscar := func(f FiltroPasos) []string {
		hs, _ := ix.BuscarPasos(f)
		var out []string
		for _, h := range hs {
			out = append(out, h.Paso.Tipo)
		}
		return out
	}
	// En la orden (no en el resumen), sin importar mayúsculas.
	if got := buscar(FiltroPasos{Terminos: Terminos("CARGO test")}); strings.Join(got, ",") != "Ejecuta" {
		t.Errorf("orden: %v", got)
	}
	// Sin tildes: «auditoria» no está en los pasos; «pruebas» sí, en el resumen.
	if got := buscar(FiltroPasos{Terminos: Terminos("pruébas")}); len(got) != 1 {
		t.Errorf("tildes: %v", got)
	}
	// Por archivo, con barras de Windows en la búsqueda.
	if got := buscar(FiltroPasos{Archivo: Normalizar(`pedidos\src\pedidos.rs`)}); strings.Join(got, ",") != "Lee" {
		t.Errorf("archivo: %v", got)
	}
	// El secreto de la orden está tapado: no se encuentra.
	if got := buscar(FiltroPasos{Terminos: Terminos("abcdefghijklmnop")}); len(got) != 0 {
		t.Errorf("el secreto se encontró: %v", got)
	}
	// Por tipo y por proyecto; el máximo corta pero el total cuenta todo.
	if got := buscar(FiltroPasos{Tipo: "Lee", Proyecto: "Otro"}); len(got) != 0 {
		t.Errorf("proyecto: %v", got)
	}
	hs, total := ix.BuscarPasos(FiltroPasos{Max: 1})
	if len(hs) != 1 || total < 3 || hs[0].Agente.ID != "ses1~abc" {
		t.Errorf("máximo: %d de %d", len(hs), total)
	}
}
