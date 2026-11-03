package claude

import (
	"math"
	"time"
)

const MapaCeldas = 100

type Mapa struct {
	Total    int         `json:"total"`
	PorCelda int         `json:"por_celda"`
	Celdas   []CeldaMapa `json:"celdas"`
}

type CeldaMapa struct {
	N       int       `json:"n"`
	Hora    time.Time `json:"hora"`
	Pasos   int       `json:"pasos"`
	Edita   float64   `json:"edita"`
	Errores int       `json:"errores"`
	Pedidos int       `json:"pedidos"`
}

// Mapa resume todos los pasos guardados, sin volver a leer el transcrito.
func (ix *Indice) Mapa(id string) (Mapa, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	for _, a := range ix.arch {
		if a.agente.ID != id {
			continue
		}
		total := len(a.pasos)
		porCelda := total / MapaCeldas
		if total%MapaCeldas != 0 {
			porCelda++
		}
		if porCelda < 1 {
			porCelda = 1
		}
		m := Mapa{Total: total, PorCelda: porCelda, Celdas: []CeldaMapa{}}
		for inicio := 0; inicio < total; inicio += porCelda {
			fin := min(inicio+porCelda, total)
			c := CeldaMapa{N: a.pasos[inicio].N, Hora: a.pasos[inicio].Hora, Pasos: fin - inicio}
			ediciones := 0
			for _, p := range a.pasos[inicio:fin] {
				if p.Tipo == "Edita" || p.Tipo == "Escribe" {
					ediciones++
				}
				if p.Error {
					c.Errores++
				}
				if p.Tipo == "Pide" {
					c.Pedidos++
				}
			}
			c.Edita = math.Round(float64(ediciones)*100/float64(c.Pasos)) / 100
			m.Celdas = append(m.Celdas, c)
		}
		return m, true
	}
	return Mapa{}, false
}
