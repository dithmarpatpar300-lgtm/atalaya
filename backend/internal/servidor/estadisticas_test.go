package servidor

import "testing"

func TestFamiliaYPresupuesto(t *testing.T) {
	for in, want := range map[string]string{"claude-sonnet-5-5": "sonnet", "opus/high": "opus", "claude-haiku-5-5": "haiku", "gpt-6.1-terra": ""} {
		if got := familia(in); got != want {
			t.Errorf("familia(%q) = %q", in, got)
		}
	}
	for in, want := range map[string]float64{"30 min": 30, "1 h": 60, "45min": 45, "sin decir": 0} {
		if got := minutosDe(in); got != want {
			t.Errorf("minutosDe(%q) = %v", in, got)
		}
	}
	if effortAsignado("sonnet/high") != "high" || effortAsignado("sonnet") != "" {
		t.Error("effort de la asignación")
	}
}
