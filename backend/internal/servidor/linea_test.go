package servidor

import (
	"testing"
	"time"

	"atalaya/internal/cuotas"
)

func TestArmarLinea(t *testing.T) {
	loc := time.FixedZone("PE", -5*3600)
	ahora := time.Date(2026, 10, 31, 16, 0, 0, 0, loc)
	h := func(d, hh, mm int) time.Time { return time.Date(2026, 10, d, hh, mm, 0, 0, loc) }
	todas := []filaLinea{
		{ID: "ses", Tipo: "sesion", Inicio: h(31, 9, 0), Fin: h(31, 15, 50), Vivo: true},
		{ID: "largo", Tipo: "subagente", Inicio: h(31, 10, 0), Fin: h(31, 10, 45)},
		{ID: "corto", Tipo: "subagente", Inicio: h(31, 11, 0), Fin: h(31, 11, 20)},
		{ID: "ayer", Tipo: "codex", Inicio: h(30, 22, 0), Fin: h(30, 22, 10)},
		{ID: "sinhora", Tipo: "subagente"},
	}
	var pedido [2]time.Time
	cuota := func(d, hh time.Time) map[string][]cuotas.Punto { pedido = [2]time.Time{d, hh}; return nil }

	r := armarLinea("RG", "", ahora, 30*time.Minute, todas, cuota)
	if r.Dia != "2026-10-31" || len(r.Filas) != 3 {
		t.Fatalf("hoy: día %s, filas %+v", r.Dia, r.Filas)
	}
	if !r.Filas[0].Fin.Equal(ahora) {
		t.Errorf("la sesión viva llega hasta ahora: %v", r.Filas[0].Fin)
	}
	if r.Filas[0].Excede || !r.Filas[1].Excede || r.Filas[2].Excede {
		t.Errorf("excede: %v %v %v", r.Filas[0].Excede, r.Filas[1].Excede, r.Filas[2].Excede)
	}
	if len(r.Dias) != 2 || r.Dias[0] != "2026-10-31" || r.Dias[1] != "2026-10-30" {
		t.Errorf("días: %v", r.Dias)
	}
	if !pedido[0].Equal(h(31, 0, 0)) || !pedido[1].Equal(h(31, 0, 0).AddDate(0, 0, 1)) || r.TopeMin != 30 {
		t.Errorf("cuota pedida %v, tope %d", pedido, r.TopeMin)
	}

	r = armarLinea("RG", "2026-10-30", ahora, 30*time.Minute, todas, cuota)
	if len(r.Filas) != 1 || r.Filas[0].ID != "ayer" {
		t.Errorf("ayer: %+v", r.Filas)
	}
}
