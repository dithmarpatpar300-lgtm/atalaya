package usoclaude

import (
	"testing"
	"time"
)

const salidaBuena = "{\"type\":\"assistant\",\"message\":{\"usage_report\":{\"rate_limits\":{\"limits\":[{\"kind\":\"session\",\"percent\":14,\"resets_at\":\"2026-11-01T19:10:00Z\",\"severity\":\"normal\",\"is_active\":true},{\"kind\":\"weekly_all\",\"percent\":49,\"resets_at\":\"2026-11-03T11:00:00Z\"},{\"kind\":\"weekly_scoped\",\"percent\":0,\"resets_at\":\"2026-11-03T11:00:00Z\",\"scope\":{\"model\":{\"display_name\":\"Fable\"}}}]},\"extra_usage\":{\"is_enabled\":false}}}}\n{\"type\":\"result\",\"result\":\"ok\"}"

func TestAnalizar(t *testing.T) {
	ahora := time.Date(2026, 11, 1, 14, 0, 0, 0, time.UTC)
	r, e := Analizar([]byte(salidaBuena), ahora)
	if e != nil || len(r.Ventanas) != 3 || r.Ventanas[2].Clave != "weekly_scoped:Fable" || r.Ventanas[0].Porcentaje != 14 {
		t.Fatalf("%+v %v", r, e)
	}
	r, e = Analizar([]byte(`{"type":"result","result":"rate limited until 3:10pm"}`), ahora)
	if e != nil || !r.Limitada || r.Hasta.Hour() != 15 || r.Hasta.Minute() != 10 {
		t.Fatalf("%+v %v", r, e)
	}
	r, e = Analizar([]byte(`{"type":"result","result":"ok"}`), ahora)
	if e != nil || !r.SinSuscripcion {
		t.Fatal(r, e)
	}
	if _, e = Analizar([]byte(`{rota`), ahora); e == nil {
		t.Fatal("línea rota aceptada")
	}
}

func TestAnalizarAlternativa(t *testing.T) {
	salida := `{"usage_report":{"rate_limits":{"five_hour":{"utilization":13,"resets_at":"2026-11-01T19:10:00Z"},"seven_day":{"utilization":49,"resets_at":"2026-11-03T11:00:00Z"}}}}`
	r, err := Analizar([]byte(salida), time.Now())
	if err != nil || len(r.Ventanas) != 2 || r.Ventanas[0].Porcentaje != 13 {
		t.Fatal(r, err)
	}
}
