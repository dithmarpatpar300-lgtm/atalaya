package servidor

import (
	"fmt"
	"log"
	"os"

	"atalaya/internal/datapedia"
)

// El registro proyectos.json se relee en vivo: un proyecto que la skill atalaya-proyecto registra aparece en «Tus
// proyectos» sin reiniciar Atalaya. La lista cambia de golpe (un puntero atómico), así nadie la lee a medias.

// listaProyectos: la lista vigente; la del arranque (Proyectos) hasta que el registro cambie.
func (s *Servidor) listaProyectos() []Proyecto {
	l := s.Proyectos
	if p := s.lista.Load(); p != nil {
		l = *p
	}
	if l == nil {
		return []Proyecto{} // sin proyectos, una lista vacía: el JSON dice [] y no null
	}
	return l
}

// releerRegistro mira si proyectos.json cambió (tamaño y hora) y, si cambió, lo vuelve a leer con Registro. Un
// registro roto se anota y se ignora: la lista anterior sigue.
func (s *Servidor) releerRegistro() bool {
	if s.Registro == nil || s.RutaRegistro == "" {
		return false
	}
	firma := ""
	if fi, err := os.Stat(s.RutaRegistro); err == nil {
		firma = fmt.Sprintf("%d·%d", fi.Size(), fi.ModTime().UnixNano())
	}
	// La primera vuelta solo toma la firma (la lista del arranque ya está leída). Una bandera aparte, no la firma
	// vacía: un registro que no existía al arrancar, o que se borra y se vuelve a escribir, también es un cambio.
	if !s.registroVisto {
		s.registroVisto, s.firmaRegistro = true, firma
		return false
	}
	if firma == s.firmaRegistro {
		return false
	}
	s.firmaRegistro = firma
	if firma == "" {
		return false // borrado: sigue la lista anterior hasta que vuelva
	}
	ps, err := s.Registro()
	if err != nil {
		log.Printf("registro de proyectos: %v (sigue la lista anterior)", err)
		return false
	}
	if s.Datapedia != nil {
		proyectos := make([]datapedia.Proyecto, 0, len(ps))
		for _, p := range ps {
			proyectos = append(proyectos, datapedia.Proyecto{Nombre: p.Nombre, Raiz: p.Raiz})
		}
		s.Datapedia.CambiarProyectos(proyectos)
	}
	s.lista.Store(&ps)
	log.Printf("registro de proyectos: %d proyectos", len(ps))
	return true
}
