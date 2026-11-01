package servidor

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"atalaya/internal/avance"
	"atalaya/internal/claude"
	"atalaya/internal/secretos"
	"atalaya/internal/trazabilidad"
)

// El avance por fase: lo que escribe el coordinador en docs/trazabilidad/F<n>/AVANCE.md,
// leído y calculado por internal/avance. Lo usan la sección Avance, la tarjeta de la sesión principal, la portada
// de proyectos y las herramientas MCP estado y retomar.

type respuestaAvance struct {
	Proyecto string        `json:"proyecto"`
	Fases    []avance.Fase `json:"fases"`
	// Sin: por qué no hay avance (el proyecto no tiene AVANCE.md); vacío si lo hay.
	Sin string `json:"sin,omitempty"`
}

func (s *Servidor) verAvance(w http.ResponseWriter, r *http.Request) {
	proyecto := s.proyectoOMayor(r.URL.Query().Get("proyecto"))
	out := respuestaAvance{Proyecto: proyecto, Fases: avance.Leer(s.raizDe(proyecto))}
	if out.Fases == nil {
		out.Fases = []avance.Fase{}
		out.Sin = "El proyecto no tiene docs/trazabilidad/F<n>/AVANCE.md: su avance todavía no lo puede leer Atalaya."
	}
	escribir(w, out)
}

func pct1(v float64) string { return strings.Replace(strconv.FormatFloat(v, 'f', 1, 64), ".", ",", 1) }

func horas(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	return pct1(v)
}

// estadoMCP: la fase en curso, sus paquetes y lo que falta.
func (s *Servidor) estadoMCP(proyecto string) string {
	proyecto = s.proyectoOMayor(proyecto)
	f := avance.Actual(s.raizDe(proyecto))
	if f == nil {
		return "Proyecto " + proyecto + ": no tiene docs/trazabilidad/F<n>/AVANCE.md, así que Atalaya no sabe su avance. " +
			"Léelo de su plan y de ESTADO-para-retomar.md, y no calcules un % a mano sin decir cómo.\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Proyecto %s, fase %s: %s %% (%s de %s h, pesado por horas; lo calcula Atalaya desde %s).\n",
		proyecto, f.Fase, pct1(f.Calculado), horas(f.HorasTot*f.Calculado/100), horas(f.HorasTot), f.Ruta)
	if f.Ahora != "" {
		b.WriteString("Ahora: " + f.Ahora + "\n")
	}
	if f.Plan != "" {
		b.WriteString("Plan: " + f.Plan + "\n")
	}
	b.WriteString("Paquetes:\n")
	for _, p := range f.Paquetes {
		fmt.Fprintf(&b, "  %s · %s · %s h · %s · %d %%", p.ID, p.Que, horas(p.Horas), p.Estado, p.Pct)
		if p.Evidencia != "" && p.Estado != avance.Hecho {
			b.WriteString(" · " + p.Evidencia)
		}
		b.WriteByte('\n')
	}
	var falta float64
	for _, p := range f.Paquetes {
		falta += p.Horas * float64(100-p.Pct) / 100
	}
	fmt.Fprintf(&b, "Falta: unas %s h de las estimadas.\n", horas(falta))
	if len(f.Hitos) > 0 {
		b.WriteString("Últimos hitos:\n")
		for i, h := range f.Hitos {
			if i == 5 {
				break
			}
			fmt.Fprintf(&b, "  %s · %s\n", h.Hora, h.Texto)
		}
	}
	if len(f.Avisos) > 0 {
		b.WriteString("Avisos de la tabla: " + strings.Join(f.Avisos, "; ") + ".\n")
	}
	return b.String()
}

// rutaEstado: el ESTADO-para-retomar.md del proyecto; el de nombre exacto gana a uno con prefijo («00-ESTADO-…»),
// igual que en el contrato del portafolio.
func rutaEstado(raiz string) string {
	rutas, _ := filepath.Glob(filepath.Join(raiz, "docs", "trazabilidad", "*ESTADO-para-retomar.md"))
	for _, r := range rutas {
		if filepath.Base(r) == "ESTADO-para-retomar.md" {
			return r
		}
	}
	if len(rutas) > 0 {
		return rutas[0]
	}
	return ""
}

// seccionMd: la sección «## titulo» de un Markdown hasta la siguiente «## », recortada a max bytes en un salto de línea.
func seccionMd(t, titulo string, max int) string {
	i := strings.Index(t, "## "+titulo)
	if i < 0 {
		return ""
	}
	t = t[i:]
	if j := strings.Index(t[3:], "\n## "); j >= 0 {
		t = t[:j+3]
	}
	return recortarEnLinea(t, max)
}

// recortarEnLinea: a lo sumo max bytes, cortando en un salto de línea si queda uno en la segunda mitad.
func recortarEnLinea(t string, max int) string {
	if len(t) > max {
		corte := strings.LastIndex(t[:max], "\n")
		if corte < max/2 {
			corte = max
		}
		t = t[:corte] + "\n…"
	}
	return strings.TrimSpace(t)
}

// estadoParaRetomar: «Dónde estamos» y «Lo que falta» de ESTADO-para-retomar.md (AGENTS.md pide decir qué falta al
// retomar), cada una con su tope; sin esas secciones, el comienzo del archivo.
func estadoParaRetomar(raiz string) (string, string) {
	ruta := rutaEstado(raiz)
	if ruta == "" {
		return "", ""
	}
	rel, err := filepath.Rel(raiz, ruta)
	if err != nil {
		return "", ""
	}
	rel = filepath.ToSlash(rel)
	t, err := trazabilidad.LeerDoc(raiz, rel)
	if err != nil {
		return "", ""
	}
	t = secretos.Tapar(t)
	partes := []string{}
	for _, s := range []struct {
		titulo string
		max    int
	}{{"Dónde estamos", 1600}, {"Lo que falta", 1000}} {
		if p := seccionMd(t, s.titulo, s.max); p != "" {
			partes = append(partes, p)
		}
	}
	if len(partes) == 0 {
		partes = append(partes, recortarEnLinea(t, 1600))
	}
	return rel, strings.Join(partes, "\n\n") + "\n(completo en " + rel + ")"
}

// decisionesMCP: las preguntas que las sesiones del proyecto te hicieron y siguen sin respuesta, con la recomendada.
func (s *Servidor) decisionesMCP(proyecto string) string {
	var b strings.Builder
	n := 0
	for _, d := range s.decisiones(proyecto).Decisiones {
		if d.Estado != claude.DecPendiente || d.Vieja {
			continue
		}
		n++
		for _, q := range d.Preguntas {
			rec := ""
			for _, o := range q.Opciones {
				if o.Recomendada {
					rec = " (recomendada: «" + o.Etiqueta + "»)"
					break
				}
			}
			fmt.Fprintf(&b, "  %s · %s · %s%s\n", d.Hora.Local().Format("15:04"), d.Sesion, q.Texto, rec)
		}
	}
	return fmt.Sprintf("Decisiones que esperan al dueño del proyecto: %d\n", n) + b.String()
}

// retomarMCP: lo que hace falta para retomar el proyecto y su ESTADO, en una sola respuesta.
func (s *Servidor) retomarMCP(proyecto string) string {
	proyecto = s.proyectoOMayor(proyecto)
	raiz := s.raizDe(proyecto)
	ahora := s.ahora()
	var b strings.Builder
	fmt.Fprintf(&b, "Retomar %s · %s\n\n", proyecto, ahora.Local().Format("2006-01-02 15:04"))

	if f := avance.Actual(raiz); f != nil {
		fmt.Fprintf(&b, "Avance: fase %s al %s %%", f.Fase, pct1(f.Calculado))
		if f.Ahora != "" {
			b.WriteString(" · ahora: " + f.Ahora)
		}
		b.WriteString(" (detalle: herramienta estado)\n")
		if len(f.Avisos) > 0 {
			b.WriteString("Avisos del avance: " + strings.Join(f.Avisos, "; ") + ".\n")
		}
	} else {
		b.WriteString("Avance: sin AVANCE.md; Atalaya no lo sabe.\n")
	}

	var vivos []string
	for _, a := range s.todos() {
		if a.Proyecto != proyecto || !activo(a.Estado) {
			continue
		}
		vivos = append(vivos, fmt.Sprintf("  %s · %s · %s · hace %s", nombreAgente(a), a.Tarea, strings.ToLower(a.Estado), corto(ahora.Sub(a.Ultimo))))
	}
	fmt.Fprintf(&b, "Agentes vivos: %d\n", len(vivos))
	for _, v := range vivos {
		b.WriteString(v + "\n")
	}

	s.mu.RLock()
	var abiertas []string
	for _, a := range s.alertas {
		if a.Proyecto == proyecto && a.Chip != "" {
			abiertas = append(abiertas, "  ["+a.Nivel+"] "+a.Titulo+": "+a.Texto)
		}
	}
	s.mu.RUnlock()
	fmt.Fprintf(&b, "Alertas abiertas: %d\n", len(abiertas))
	for _, a := range abiertas {
		b.WriteString(a + "\n")
	}
	b.WriteString(s.decisionesMCP(proyecto))

	if est := s.Cuotas.Estado(); est.Lectura != nil && est.Lectura.Advice != nil && est.Lectura.Advice.Summary != "" {
		fmt.Fprintf(&b, "Cuota (leída hace %s): %s\n", corto(ahora.Sub(time.Unix(est.LeidaEn, 0))), est.Lectura.Advice.Summary)
	} else {
		b.WriteString("Cuota: sin lectura con consejo; corre cuotas.exe una-vez.\n")
	}

	b.WriteString("\n" + s.procesoMCP(proyecto))

	if rel, t := estadoParaRetomar(raiz); t != "" {
		b.WriteString("\nDe " + rel + ":\n" + t + "\n")
	}
	b.WriteString("\nEl árbol de git y los procesos de laboratorio no los ve Atalaya: míralos con git status.\n")
	return b.String()
}

// corto: «3 min», «2 h 5 min», «1 d».
func corto(d time.Duration) string {
	switch {
	case d < time.Minute:
		return strconv.Itoa(int(d.Seconds())) + " s"
	case d < time.Hour:
		return strconv.Itoa(int(d.Minutes())) + " min"
	case d < 24*time.Hour:
		return fmt.Sprintf("%d h %d min", int(d.Hours()), int(d.Minutes())%60)
	}
	return strconv.Itoa(int(d.Hours()/24)) + " d"
}
