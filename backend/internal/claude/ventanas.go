package claude

// ventanaReal: la ventana que mandó la app para esta sesión, si la hay (Indice.Ventanas).
func (ix *Indice) ventanaReal(id string) (int64, bool) {
	if ix.Ventanas == nil || id == "" {
		return 0, false
	}
	w, ok := ix.Ventanas(id)
	return w, ok && w > 0
}
