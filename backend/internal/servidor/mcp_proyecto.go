package servidor

import (
	"sort"
	"strings"
	"unicode"

	"atalaya/internal/claude"
)

// Un agente escribe el proyecto como lo leyó: el nombre del panel («Atalaya»), el del README o la sigla, no
// siempre el de la carpeta, que es como Atalaya junta los agentes. proyectoMCP lo traduce; un nombre que no es de
// ninguno se dice, en vez de filtrar todo y responder «ninguno».

// vaciasProyecto: palabras que no distinguen un nombre de otro («Mi Proyecto» = «mi-proyecto»).
var vaciasProyecto = map[string]bool{"de": true, "del": true, "la": true, "las": true, "el": true, "los": true, "y": true}

// claveProyecto: el nombre sin mayúsculas, tildes, separadores ni palabras vacías.
func claveProyecto(t string) string {
	palabras := strings.FieldsFunc(claude.Normalizar(t), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	var b strings.Builder
	for _, p := range palabras {
		if !vaciasProyecto[p] {
			b.WriteString(p)
		}
	}
	return b.String()
}

// nombresProyectos: los del registro y los de los agentes leídos (un proyecto sin registrar también tiene agentes).
func (s *Servidor) nombresProyectos() []string {
	vistos := map[string]bool{}
	var out []string
	agregar := func(n string) {
		if n != "" && !vistos[n] {
			vistos[n] = true
			out = append(out, n)
		}
	}
	for _, p := range s.listaProyectos() {
		agregar(p.Nombre)
	}
	for _, a := range s.Claude.Agentes() {
		agregar(a.Proyecto)
	}
	sort.Strings(out)
	return out
}

// proyectoMCP: el nombre con que Atalaya conoce al proyecto que escribió el agente: igual, con la misma clave o por
// su sigla. false si no es de ninguno.
func (s *Servidor) proyectoMCP(q string) (string, bool) {
	q = strings.TrimSpace(q)
	for _, n := range s.nombresProyectos() {
		if n == q {
			return n, true
		}
	}
	k := claveProyecto(q)
	for _, p := range s.listaProyectos() {
		if (k != "" && claveProyecto(p.Nombre) == k) || (p.Sigla != "" && strings.EqualFold(p.Sigla, q)) {
			return p.Nombre, true
		}
	}
	for _, n := range s.nombresProyectos() {
		if k != "" && claveProyecto(n) == k {
			return n, true
		}
	}
	return "", false
}
