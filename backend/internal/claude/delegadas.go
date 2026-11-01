package claude

import "path/filepath"

// Delegacion: lo que la ficha de scripts/claude.ps1 sabe de una sesión de Claude Code y su transcrito no: la corrió
// otra sesión (de Codex, casi siempre) en una copia del repositorio, con un tema, una clase y un plazo
// (internal/delegadas la lee).
type Delegacion struct {
	Raiz        string // la del repositorio, no la de la copia donde trabajó
	Padre       string // el id del agente que la lanzó; vacío si no se sabe
	Rol         string
	Tarea       string
	Clase       string
	Effort      string
	Presupuesto string
	Base        string
	Informe     string
	Terminada   bool
}

// conDelegacion: el agente con lo de su ficha encima, si la tiene. Con padre se ve como un subagente de esa sesión;
// sin padre sigue siendo una sesión, pero del proyecto que corresponde y cerrada cuando la ficha dice que terminó.
func (ix *Indice) conDelegacion(a Agente) Agente {
	if ix.Delegadas == nil || a.Tipo != "sesion" {
		return a
	}
	d, ok := ix.Delegadas(a.ID)
	if !ok {
		return a
	}
	if d.Padre != "" {
		a.Tipo, a.Padre = "subagente", d.Padre
	}
	if d.Raiz != "" {
		a.Raiz, a.Proyecto = d.Raiz, filepath.Base(d.Raiz)
	}
	if d.Rol != "" {
		a.Rol = d.Rol
	}
	if d.Tarea != "" {
		a.Tarea = d.Tarea
	}
	if d.Effort != "" {
		a.Effort = d.Effort
	}
	if d.Informe != "" {
		a.Informe = d.Informe
	}
	a.Clase, a.Presupuesto, a.Base = d.Clase, d.Presupuesto, d.Base
	a.Cerrado = a.Cerrado || d.Terminada
	return a
}
