package servidor

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const diasRegistroConsultas = 90

// Serializa el recorte y las escrituras, también entre servidores de una prueba.
var consultasArchivoMu sync.Mutex

// cargarConsultas se llama con s.mu tomado; nil distingue un registro aún no leído.
func (s *Servidor) cargarConsultas() {
	if s.consultas != nil {
		return
	}
	s.consultas = []Consulta{}
	if s.Datos == "" {
		return
	}
	ruta := filepath.Join(s.Datos, "consultas.jsonl")
	consultasArchivoMu.Lock()
	cs, _, err := leerRegistroConsultas(ruta, s.ahora())
	consultasArchivoMu.Unlock()
	if err != nil {
		log.Printf("cargar consultas: %v", err)
		return
	}
	sort.SliceStable(cs, func(i, j int) bool { return cs[i].Hora.Before(cs[j].Hora) })
	if len(cs) > MaxConsultas {
		cs = cs[len(cs)-MaxConsultas:]
	}
	s.consultas = cs
	go s.recortarConsultas()
}

func leerRegistroConsultas(ruta string, ahora time.Time) ([]Consulta, bool, error) {
	cs := []Consulta{}
	f, err := os.Open(ruta)
	if os.IsNotExist(err) {
		return cs, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	recortar := false
	scan := bufio.NewScanner(f)
	scan.Buffer(make([]byte, 4096), 4*1024*1024)
	for scan.Scan() {
		var c Consulta
		if err := json.Unmarshal(scan.Bytes(), &c); err != nil {
			return nil, false, err
		}
		if c.Hora.Before(ahora.AddDate(0, 0, -diasRegistroConsultas)) {
			recortar = true
		} else {
			cs = append(cs, c)
		}
	}
	return cs, recortar, scan.Err()
}

// El archivo temporal evita perder el registro si falla el recorte.
func recortarRegistroConsultas(ruta string, ahora time.Time) error {
	cs, recortar, err := leerRegistroConsultas(ruta, ahora)
	if err != nil || !recortar {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(ruta), "consultas-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	enc := json.NewEncoder(f)
	for _, c := range cs {
		if err = enc.Encode(c); err != nil {
			f.Close()
			return err
		}
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), ruta)
}

func (s *Servidor) recortarConsultas() {
	consultasArchivoMu.Lock()
	defer consultasArchivoMu.Unlock()
	if err := recortarRegistroConsultas(filepath.Join(s.Datos, "consultas.jsonl"), s.ahora()); err != nil {
		log.Printf("recortar consultas: %v", err)
	}
}

func (s *Servidor) guardarConsulta(c Consulta) {
	consultasArchivoMu.Lock()
	defer consultasArchivoMu.Unlock()
	ruta := filepath.Join(s.Datos, "consultas.jsonl")
	if err := os.MkdirAll(s.Datos, 0o755); err != nil {
		log.Printf("guardar consulta: %v", err)
		return
	}
	if err := recortarRegistroConsultas(ruta, s.ahora()); err != nil {
		log.Printf("recortar consultas: %v", err)
	}
	f, err := os.OpenFile(ruta, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err == nil {
		err = json.NewEncoder(f).Encode(c)
		if cerrar := f.Close(); err == nil {
			err = cerrar
		}
	}
	if err != nil {
		log.Printf("guardar consulta: %v", err)
	}
}
