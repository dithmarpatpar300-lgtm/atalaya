package usoclaude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
)

type Ventana struct {
	Clave       string  `json:"clave"`
	Etiqueta    string  `json:"etiqueta"`
	Porcentaje  float64 `json:"porcentaje"`
	ReiniciaEn  int64   `json:"reinicia_en"`
	DuracionMin int     `json:"duracion_min"`
	Severidad   string  `json:"severidad"`
	Activa      bool    `json:"activa"`
	Vencida     bool    `json:"vencida"`
}

type informe struct {
	RateLimits struct {
		Limits []struct {
			Kind     string    `json:"kind"`
			Percent  float64   `json:"percent"`
			ResetsAt time.Time `json:"resets_at"`
			Severity string    `json:"severity"`
			Active   bool      `json:"is_active"`
			Scope    struct {
				Model struct {
					Name string `json:"display_name"`
				} `json:"model"`
			} `json:"scope"`
		} `json:"limits"`
		Five *struct {
			Utilization float64   `json:"utilization"`
			ResetsAt    time.Time `json:"resets_at"`
		} `json:"five_hour"`
		Seven *struct {
			Utilization float64   `json:"utilization"`
			ResetsAt    time.Time `json:"resets_at"`
		} `json:"seven_day"`
	} `json:"rate_limits"`
	Extra struct {
		Enabled bool `json:"is_enabled"`
	} `json:"extra_usage"`
}

type Resultado struct {
	Ventanas       []Ventana
	Extra          bool
	Limitada       bool
	Hasta          time.Time
	SinSuscripcion bool
}

var reLimite = regexp.MustCompile(`(?i)rate limited until\s+([^\s.,;]+)`)

// Analizar interpreta stream-json sin consultar el reloj ni acceder a archivos.
func Analizar(b []byte, ahora time.Time) (Resultado, error) {
	var out Resultado
	var reporte *informe
	s := bufio.NewScanner(bytes.NewReader(b))
	s.Buffer(make([]byte, 4096), 16<<20)
	for s.Scan() {
		var m struct {
			Type    string `json:"type"`
			Result  string `json:"result"`
			Message struct {
				Usage *informe `json:"usage_report"`
			} `json:"message"`
			Usage *informe `json:"usage_report"`
		}
		if err := json.Unmarshal(s.Bytes(), &m); err != nil {
			return out, err
		}
		if m.Usage != nil {
			reporte = m.Usage
		}
		if m.Message.Usage != nil {
			reporte = m.Message.Usage
		}
		if m.Type == "result" && strings.Contains(strings.ToLower(m.Result), "rate limited until") {
			out.Limitada = true
			if partes := reLimite.FindStringSubmatch(m.Result); len(partes) > 1 {
				for _, formato := range []string{"3:04pm", "3pm", "15:04"} {
					if t, err := time.ParseInLocation(formato, strings.ToLower(partes[1]), ahora.Location()); err == nil {
						out.Hasta = time.Date(ahora.Year(), ahora.Month(), ahora.Day(), t.Hour(), t.Minute(), 0, 0, ahora.Location())
						if !out.Hasta.After(ahora) {
							out.Hasta = out.Hasta.AddDate(0, 0, 1)
						}
						break
					}
				}
			}
		}
	}
	if err := s.Err(); err != nil {
		return out, err
	}
	if out.Limitada {
		return out, nil
	}
	if reporte == nil {
		out.SinSuscripcion = true
		return out, nil
	}
	for _, v := range reporte.RateLimits.Limits {
		clave, etiqueta, mins := v.Kind, "", 10080
		switch v.Kind {
		case "session":
			etiqueta, mins = "5 horas", 300
		case "weekly_all":
			etiqueta = "Semana · todos los modelos"
		case "weekly_scoped":
			if v.Scope.Model.Name == "" {
				continue
			}
			clave += ":" + v.Scope.Model.Name
			etiqueta = "Semana · " + v.Scope.Model.Name
		default:
			continue
		}
		if v.ResetsAt.IsZero() || v.Percent < 0 || v.Percent > 100 {
			return out, errors.New("ventana de uso inválida")
		}
		out.Ventanas = append(out.Ventanas, Ventana{Clave: clave, Etiqueta: etiqueta, Porcentaje: v.Percent, ReiniciaEn: v.ResetsAt.Unix(), DuracionMin: mins, Severidad: v.Severity, Activa: v.Active})
	}
	if len(reporte.RateLimits.Limits) == 0 {
		if v := reporte.RateLimits.Five; v != nil && !v.ResetsAt.IsZero() && v.Utilization >= 0 && v.Utilization <= 100 {
			out.Ventanas = append(out.Ventanas, Ventana{Clave: "session", Etiqueta: "5 horas", Porcentaje: v.Utilization, ReiniciaEn: v.ResetsAt.Unix(), DuracionMin: 300})
		}
		if v := reporte.RateLimits.Seven; v != nil && !v.ResetsAt.IsZero() && v.Utilization >= 0 && v.Utilization <= 100 {
			out.Ventanas = append(out.Ventanas, Ventana{Clave: "weekly_all", Etiqueta: "Semana · todos los modelos", Porcentaje: v.Utilization, ReiniciaEn: v.ResetsAt.Unix(), DuracionMin: 10080})
		}
	}
	if len(out.Ventanas) == 0 {
		return out, errors.New("el informe no trae ventanas válidas")
	}
	out.Extra = reporte.Extra.Enabled
	return out, nil
}
