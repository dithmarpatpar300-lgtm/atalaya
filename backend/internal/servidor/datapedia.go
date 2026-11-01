package servidor

import (
	"errors"
	"net/http"
	"strconv"

	"atalaya/internal/claude"
	"atalaya/internal/datapedia"
)

// Trazas da a la datapedia un documento por cada turno de cada agente (claude/trazas.go), con tipo «traza».
func Trazas(ix *claude.Indice) func() []datapedia.Virtual {
	return func() []datapedia.Virtual {
		ts := ix.Trazas()
		out := make([]datapedia.Virtual, 0, len(ts))
		for _, t := range ts {
			out = append(out, datapedia.Virtual{
				Clave:    "traza:" + t.Agente + ":" + strconv.Itoa(t.Turno),
				Proyecto: t.Proyecto,
				Tipo:     "traza",
				Ruta:     claude.RutaTraza(t.Agente, t.Turno),
				Titulo:   t.Titulo,
				Resumen:  t.Resumen,
				Cuerpo:   t.Cuerpo,
				Mtime:    t.Ultimo,
			})
		}
		return out
	}
}

type respuestaBusqueda struct {
	Resultados []datapedia.Documento `json:"resultados"`
	Cuentas    []datapedia.Cuenta    `json:"cuentas"`
}

// GET /api/datapedia?q=&tipo=&proyecto=&limite=
func (s *Servidor) buscarDatapedia(w http.ResponseWriter, r *http.Request) {
	if s.Datapedia == nil {
		fallo(w, http.StatusServiceUnavailable, "la datapedia no está disponible: el índice no se pudo abrir")
		return
	}
	q := r.URL.Query()
	limite, _ := strconv.Atoi(q.Get("limite"))
	res, err := s.Datapedia.Buscar(datapedia.Filtro{Texto: q.Get("q"), Tipo: q.Get("tipo"), Proyecto: q.Get("proyecto"), Limite: limite})
	if err != nil {
		fallo(w, http.StatusInternalServerError, err.Error())
		return
	}
	cuentas, err := s.Datapedia.Cuentas(q.Get("q"), q.Get("proyecto"))
	if err != nil {
		fallo(w, http.StatusInternalServerError, err.Error())
		return
	}
	escribir(w, respuestaBusqueda{Resultados: res, Cuentas: cuentas})
}

// GET /api/datapedia/{id}; también /api/datapedia/ruta?proyecto=&ruta= para abrir la bitácora o el informe de un agente.
func (s *Servidor) documentoDatapedia(w http.ResponseWriter, r *http.Request) {
	if s.Datapedia == nil {
		fallo(w, http.StatusServiceUnavailable, "la datapedia no está disponible")
		return
	}
	var id int64
	var err error
	if r.PathValue("id") == "ruta" {
		q := r.URL.Query()
		id, err = s.Datapedia.PorRuta(q.Get("proyecto"), q.Get("ruta"))
	} else {
		id, err = strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			fallo(w, http.StatusBadRequest, "id no válido")
			return
		}
	}
	var d datapedia.Documento
	if err == nil {
		d, err = s.Datapedia.Documento(id)
	}
	if errors.Is(err, datapedia.ErrNoExiste) {
		fallo(w, http.StatusNotFound, err.Error())
		return
	}
	if err != nil {
		fallo(w, http.StatusInternalServerError, err.Error())
		return
	}
	escribir(w, d)
}
