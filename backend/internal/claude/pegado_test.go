package claude

import (
	"testing"
	"time"
)

func TestUnPegadoEsUnPedido(t *testing.T) {
	a := nuevoArchivo("C:/x/proyecto/s.jsonl")
	a.pedido("  <pasted_content id=\"a382\">\nRetomas Tienda Demo en una sesión nueva.\n</pasted_content id=\"a382\">", time.Now(), 0)
	a.pedido("<command-name>/clear</command-name>", time.Now(), 0)
	if len(a.pasos) != 1 || a.pasos[0].Tipo != "Pide" || a.pasos[0].Resumen != "Retomas Tienda Demo en una sesión nueva." {
		t.Fatalf("pasos: %+v", a.pasos)
	}
}
