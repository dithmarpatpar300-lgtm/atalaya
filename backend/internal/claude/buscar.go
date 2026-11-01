package claude

import (
	"sort"
	"strings"
	"time"
)

// Buscar en los pasos de todos los agentes. Los pasos ya están en memoria: recorrerlos
// tarda milisegundos, así que no hace falta un índice aparte. Se busca en el resumen, el archivo, la herramienta y
// la orden de los «Ejecuta»; no en las salidas, que no se guardan en memoria.

var sinTildes = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n", "\\", "/")

// Normalizar: minúsculas, sin tildes y con barras normales, para que «bitacora» encuentre «Bitácora» y
// «crates\x» encuentre «crates/x».
func Normalizar(s string) string { return sinTildes.Replace(strings.ToLower(s)) }

// Terminos parte una búsqueda en palabras normalizadas.
func Terminos(q string) []string { return strings.Fields(Normalizar(q)) }

// Coincide: todas las palabras están en el resumen, el archivo, la herramienta o la orden del paso.
func Coincide(p *Paso, ts []string) bool {
	if len(ts) == 0 {
		return true
	}
	texto := Normalizar(p.Resumen + "\n" + p.Hilo + "\n" + p.Herramienta + "\n" + p.orden)
	for _, t := range ts {
		if !strings.Contains(texto, t) {
			return false
		}
	}
	return true
}

// FiltroPasos de una búsqueda. Lo vacío no filtra.
type FiltroPasos struct {
	Terminos    []string
	Tipo        string
	Proyecto    string
	Archivo     string // parte de la ruta del archivo del paso (normalizada)
	Herramienta string // exacta
	Desde       time.Time
	Hasta       time.Time
	Max         int
}

// Pasa: el paso cumple el filtro (sin mirar el proyecto, que es del agente).
func (f FiltroPasos) Pasa(p *Paso) bool {
	if p.Tipo == "Pide" || (f.Tipo != "" && p.Tipo != f.Tipo) || (f.Herramienta != "" && p.Herramienta != f.Herramienta) {
		return false
	}
	if (!f.Desde.IsZero() && p.Hora.Before(f.Desde)) || (!f.Hasta.IsZero() && p.Hora.After(f.Hasta)) {
		return false
	}
	if f.Archivo != "" && (p.Hilo == "" || !strings.Contains(Normalizar(p.Hilo), f.Archivo)) {
		return false
	}
	return Coincide(p, f.Terminos)
}

// Hallazgo: un paso y el agente que lo dio.
type Hallazgo struct {
	Agente Agente
	Paso   Paso
}

// BuscarPasos devuelve los pasos que cumplen el filtro, del más reciente al más viejo, y cuántos había en total.
func (ix *Indice) BuscarPasos(f FiltroPasos) ([]Hallazgo, int) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	// Primero punteros: una búsqueda amplia encuentra miles de pasos y solo se copian los que se devuelven.
	type par struct {
		a *Agente
		p *Paso
	}
	var hay []par
	for _, a := range ix.arch {
		if a.agente.ID == "" || (f.Proyecto != "" && a.agente.Proyecto != f.Proyecto) || (!f.Desde.IsZero() && a.agente.Ultimo.Before(f.Desde)) {
			continue
		}
		for _, p := range a.pasos {
			if f.Pasa(p) {
				hay = append(hay, par{a.agente, p})
			}
		}
	}
	sort.Slice(hay, func(i, j int) bool { return hay[i].p.Hora.After(hay[j].p.Hora) })
	total := len(hay)
	if f.Max > 0 && len(hay) > f.Max {
		hay = hay[:f.Max]
	}
	out := make([]Hallazgo, len(hay))
	for i, h := range hay {
		out[i] = Hallazgo{Agente: *h.a, Paso: *h.p}
	}
	return out, total
}
