package servidor

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"atalaya/internal/claude"
	"atalaya/internal/codexvivo"
	"atalaya/internal/secretos"
	"atalaya/internal/trazabilidad"
)

// Exportar un agente como Markdown: un documento para adjuntar a la
// trazabilidad, con las secciones que se pidan. El servidor solo lo arma y lo devuelve; copiarlo o descargarlo lo
// hace el navegador, y nada se escribe en el disco. Los secretos se tapan con la misma regla del panel.

type opcionesExportar struct {
	Encargo, Resumen, Archivos, Pasos, Diffs, Tokens bool
	Salidas                                          string // recortadas | completas | ninguna
}

type respuestaExportar struct {
	Nombre   string `json:"nombre"`
	Archivo  string `json:"archivo"`
	Markdown string `json:"markdown"`
}

// Lo más que se lee para exportar: un agente de miles de pasos no debe colgar el panel.
const (
	maxPasosExportar = 3000
	lineasRecortadas = 40
)

func leerOpciones(q map[string][]string) opcionesExportar {
	si := func(k string, porOmision bool) bool {
		v, ok := q[k]
		if !ok || len(v) == 0 {
			return porOmision
		}
		return v[0] == "1"
	}
	o := opcionesExportar{Encargo: si("encargo", true), Resumen: si("resumen", true), Archivos: si("archivos", true), Pasos: si("pasos", true),
		Diffs: si("diffs", false), Tokens: si("tokens", true), Salidas: "recortadas"}
	if v := q["salidas"]; len(v) > 0 && (v[0] == "completas" || v[0] == "ninguna") {
		o.Salidas = v[0]
	}
	return o
}

func (s *Servidor) exportar(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var v AgenteVista
	var pasos []claude.Paso
	var raiz, informe string
	var detalle func(int) (claude.DetallePaso, bool)
	switch {
	case strings.HasPrefix(id, codexvivo.Prefijo) && s.CodexVivo != nil:
		c, ok := s.CodexVivo.Llamada(id)
		if !ok {
			fallo(w, http.StatusNotFound, "no existe esa llamada")
			return
		}
		_, reg := s.codexEnVivo()
		v = s.vistaCodexVivo(c, reg[claveRegistro(c)])
		raiz, pasos = c.Meta.Raiz, c.Pasos
		if len(pasos) > maxPasosExportar {
			pasos = pasos[len(pasos)-maxPasosExportar:]
		}
		v.Encargo = c.Meta.Tema
		detalle = func(n int) (claude.DetallePaso, bool) { d, err := s.CodexVivo.Detalle(id, n); return d, err == nil }
	case strings.HasPrefix(id, "codex~"):
		c, ok := s.codexPorID(id)
		if !ok {
			fallo(w, http.StatusNotFound, "no existe esa llamada")
			return
		}
		v = vistaCodex(c)
		v.Encargo, raiz = c.Tema, s.raizDe(c.Proyecto)
		detalle = func(int) (claude.DetallePaso, bool) { return claude.DetallePaso{}, false }
	default:
		a, ps, ok := s.Claude.Agente(id, maxPasosExportar)
		if !ok {
			fallo(w, http.StatusNotFound, "no existe ese agente")
			return
		}
		v, raiz, pasos = s.vistaClaude(a, true), a.Raiz, ps
		detalle = func(n int) (claude.DetallePaso, bool) { d, err := s.Claude.Detalle(id, n); return d, err == nil }
	}
	if v.Tipo == "codex" && v.Informe != "" && raiz != "" {
		informe, _ = trazabilidad.LeerDoc(raiz, v.Informe)
	}
	o := leerOpciones(r.URL.Query())
	if v.Tipo == "codex" && v.Tokens == (claude.Tokens{}) {
		o.Tokens = false
	}
	nombre := nombreExportar(v)
	md := armarMarkdown(nombre, v, raiz, pasos, o, detalle, s.ahora().Location())
	if informe != "" {
		md = strings.Replace(md, pieExportar, "## Informe\n\n"+strings.TrimSpace(informe)+"\n\n"+pieExportar, 1)
	}
	escribir(w, respuestaExportar{Nombre: nombre, Archivo: archivoSeguro(nombre) + ".md", Markdown: secretos.Tapar(md)})
}

// nombreExportar: el nombre del agente, el mismo de todo el panel (nombreVista).
func nombreExportar(v AgenteVista) string { return v.Nombre }

func archivoSeguro(nombre string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(nombre) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		case strings.ContainsRune("áàäâ", r):
			b.WriteRune('a')
		case strings.ContainsRune("éèëê", r):
			b.WriteRune('e')
		case strings.ContainsRune("íìïî", r):
			b.WriteRune('i')
		case strings.ContainsRune("óòöô", r):
			b.WriteRune('o')
		case strings.ContainsRune("úùüû", r):
			b.WriteRune('u')
		case r == 'ñ':
			b.WriteRune('n')
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-.")
	for strings.Contains(out, "--") {
		out = strings.ReplaceAll(out, "--", "-")
	}
	if len(out) > 60 {
		out = strings.TrimRight(out[:60], "-.")
	}
	if out == "" {
		out = "agente"
	}
	return out
}

// pieExportar: el cierre de todo Markdown exportado; el informe de una llamada a Codex va justo antes.
const pieExportar = "---\nExportado de Atalaya. Secretos tapados con la regla del panel.\n"

func armarMarkdown(nombre string, v AgenteVista, raiz string, pasos []claude.Paso, o opcionesExportar, detalle func(int) (claude.DetallePaso, bool), loc *time.Location) string {
	hh := func(t time.Time) string { return t.In(loc).Format("15:04") }
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", nombre)
	cabeza := []string{}
	if c := claseDeVista(v); c != "" {
		cabeza = append(cabeza, c)
	}
	cabeza = append(cabeza, fmt.Sprintf("%s – %s (%s)", hh(v.Inicio), hh(v.Ultimo), duracionMD(v.Ultimo.Sub(v.Inicio))))
	if v.Base != "" {
		cabeza = append(cabeza, "base "+v.Base)
	}
	fmt.Fprintf(&b, "> %s · %s · %s\n\n", v.Proyecto, v.Inicio.In(loc).Format("2006-01-02"), strings.Join(cabeza, " · "))

	if o.Encargo && (v.Asignacion != "" || v.Encargo != "") {
		b.WriteString("## Encargo\n\n")
		if v.Asignacion != "" {
			fmt.Fprintf(&b, "**Asignación:** %s\n\n", v.Asignacion)
		}
		// La asignación ya va arriba en negrita: en el texto citado no se repite.
		if t := sinAsignacion(v.Encargo, v.Asignacion != ""); t != "" {
			b.WriteString(cita(t))
			b.WriteString("\n\n")
		}
	}

	if o.Resumen {
		b.WriteString("## Resumen\n\n")
		fmt.Fprintf(&b, "- Estado: %s\n", strings.ToLower(v.Estado))
		if v.Tarea != "" && v.Tipo != "sesion" {
			fmt.Fprintf(&b, "- Tarea: %s\n", v.Tarea)
		}
		if v.Bitacora != "" {
			pct := ""
			if v.Avance != nil {
				pct = fmt.Sprintf(" (%d %%)", v.Avance.Porcentaje)
			}
			fmt.Fprintf(&b, "- Bitácora: %s%s\n", v.Bitacora, pct)
		}
		if v.Informe != "" {
			fmt.Fprintf(&b, "- Informe: %s\n", v.Informe)
		}
		if v.Acta != "" {
			fmt.Fprintf(&b, "- Acta: %s (%s)\n", v.Acta, v.ActaEstado)
		}
		fmt.Fprintf(&b, "- Pasos: %d\n\n", v.Pasos)
	}

	// Los detalles de las ediciones y las órdenes, leídos una sola vez.
	dets := map[int]claude.DetallePaso{}
	leer := func(n int) (claude.DetallePaso, bool) {
		if d, ok := dets[n]; ok {
			return d, true
		}
		d, ok := detalle(n)
		if ok {
			dets[n] = d
		}
		return d, ok
	}
	edita := func(p claude.Paso) bool { return p.Tipo == "Edita" || p.Tipo == "Escribe" }

	if o.Archivos {
		type cuenta struct{ mas, menos int }
		orden := []string{}
		por := map[string]*cuenta{}
		for _, p := range pasos {
			if !edita(p) {
				continue
			}
			d, ok := leer(p.N)
			if !ok || d.Archivo == "" {
				continue
			}
			c := por[d.Archivo]
			if c == nil {
				c = &cuenta{}
				por[d.Archivo] = c
				orden = append(orden, d.Archivo)
			}
			for _, t := range d.Diff {
				for _, l := range t.Lineas {
					switch {
					case strings.HasPrefix(l, "+"):
						c.mas++
					case strings.HasPrefix(l, "-"):
						c.menos++
					}
				}
			}
		}
		if len(orden) > 0 {
			b.WriteString("## Archivos\n\n| Archivo | + | − |\n| --- | ---: | ---: |\n")
			for _, f := range orden {
				fmt.Fprintf(&b, "| %s | %d | %d |\n", relativa(f, raiz), por[f].mas, por[f].menos)
			}
			b.WriteString("\n")
		}
	}

	if o.Pasos && len(pasos) > 0 {
		fmt.Fprintf(&b, "## Pasos (%d)\n\n", len(pasos))
		turno := -1
		varios := pasos[len(pasos)-1].Turno > pasos[0].Turno
		for _, p := range pasos {
			if varios && p.Turno != turno {
				turno = p.Turno
				fmt.Fprintf(&b, "\n### Turno %d\n\n", turno)
			}
			linea := fmt.Sprintf("- %s %s · %s", hh(p.Hora), p.Tipo, strings.ReplaceAll(p.Resumen, "\n", " "))
			if p.Meta != "" {
				linea += " (" + p.Meta + ")"
			}
			if p.Error {
				linea += " · error"
			}
			b.WriteString(linea + "\n")
			if p.Tipo == "Ejecuta" && o.Salidas != "ninguna" {
				if d, ok := leer(p.N); ok {
					b.WriteString(salidaMD(d, o.Salidas == "recortadas"))
				}
			}
		}
		b.WriteString("\n")
	}

	if o.Diffs {
		hay := false
		for _, p := range pasos {
			if !edita(p) {
				continue
			}
			d, ok := leer(p.N)
			if !ok || len(d.Diff) == 0 {
				continue
			}
			if !hay {
				b.WriteString("## Diffs\n\n")
				hay = true
			}
			fmt.Fprintf(&b, "%s · %s\n\n```diff\n", hh(p.Hora), relativa(d.Archivo, raiz))
			for _, t := range d.Diff {
				fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", t.ViejoDesde, t.ViejoLineas, t.NuevoDesde, t.NuevoLineas)
				for _, l := range t.Lineas {
					b.WriteString(strings.ReplaceAll(l, "```", "ˋˋˋ") + "\n")
				}
			}
			b.WriteString("```\n\n")
		}
	}

	if o.Tokens {
		t := v.Tokens
		b.WriteString("## Tokens\n\n| Entrada | Caché escrita | Caché leída | Salida |\n| ---: | ---: | ---: | ---: |\n")
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n\n", milesMD(t.Entrada), milesMD(t.CacheEscrita), milesMD(t.CacheLeida), milesMD(t.Salida))
		for i := len(pasos) - 1; i >= 0; i-- {
			if c := pasos[i].TokContexto; c > 0 {
				fmt.Fprintf(&b, "Contexto al final: %s tokens.\n\n", milesMD(c))
				break
			}
		}
	}
	b.WriteString(pieExportar)
	return b.String()
}

// salidaMD: la salida de una orden como bloque sangrado bajo su paso, recortada o entera.
func salidaMD(d claude.DetallePaso, recortar bool) string {
	s := strings.TrimRight(d.Salida, "\n")
	if d.Stdout != "" || d.Stderr != "" {
		s = strings.TrimRight(strings.TrimRight(d.Stdout, "\n")+"\n"+d.Stderr, "\n")
	}
	if strings.TrimSpace(s) == "" {
		return ""
	}
	lineas := strings.Split(s, "\n")
	extra := 0
	if recortar && len(lineas) > lineasRecortadas {
		extra = len(lineas) - lineasRecortadas
		lineas = lineas[:lineasRecortadas]
	}
	var b strings.Builder
	b.WriteString("  ```\n")
	for _, l := range lineas {
		b.WriteString("  " + strings.ReplaceAll(l, "```", "ˋˋˋ") + "\n")
	}
	if extra > 0 {
		fmt.Fprintf(&b, "  … (%d líneas más)\n", extra)
	}
	b.WriteString("  ```\n")
	return b.String()
}

func sinAsignacion(encargo string, quitar bool) string {
	if !quitar {
		return strings.TrimSpace(encargo)
	}
	lineas := strings.Split(encargo, "\n")
	for i, l := range lineas {
		if strings.HasPrefix(strings.TrimLeft(l, "* "), "Asignación:") {
			lineas = append(lineas[:i:i], lineas[i+1:]...)
			break
		}
	}
	return strings.TrimSpace(strings.Join(lineas, "\n"))
}

func cita(t string) string {
	lineas := strings.Split(strings.TrimRight(t, "\n"), "\n")
	for i, l := range lineas {
		lineas[i] = strings.TrimRight("> "+l, " ")
	}
	return strings.Join(lineas, "\n")
}

// relativa: la ruta dentro del repositorio del agente, con barras normales; fuera de él, con la casa en «~».
func relativa(ruta, raiz string) string {
	if raiz != "" {
		if r, err := filepath.Rel(raiz, ruta); err == nil && !strings.HasPrefix(r, "..") {
			return filepath.ToSlash(r)
		}
	}
	return casa(ruta)
}

func duracionMD(d time.Duration) string {
	m := int(d.Round(time.Minute).Minutes())
	if m < 60 {
		return strconv.Itoa(m) + " min"
	}
	return fmt.Sprintf("%d h %02d min", m/60, m%60)
}

// milesMD: 41100 → «41,1 k»; 1187389 → «1,19 M».
func milesMD(n int64) string {
	switch {
	case n >= 1_000_000:
		return strings.Replace(strconv.FormatFloat(float64(n)/1e6, 'f', 2, 64), ".", ",", 1) + " M"
	case n >= 1_000:
		return strings.Replace(strconv.FormatFloat(float64(n)/1e3, 'f', 1, 64), ".", ",", 1) + " k"
	}
	return strconv.FormatInt(n, 10)
}
