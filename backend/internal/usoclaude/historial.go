package usoclaude

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type fila struct {
	T         int64              `json:"t"`
	Ventanas  map[string]float64 `json:"ventanas"`
	Reinicios map[string]int64   `json:"reinicios"`
}
type Punto struct {
	T int64              `json:"t"`
	V map[string]float64 `json:"v"`
	R map[string]int64   `json:"r"`
}

func cargar(ruta string) ([]fila, error) {
	if ruta == "" {
		return nil, nil
	}
	f, err := os.Open(ruta)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var filas []fila
	s := bufio.NewScanner(f)
	s.Buffer(make([]byte, 4096), 16<<20)
	for s.Scan() {
		var v fila
		if json.Unmarshal(s.Bytes(), &v) == nil && v.T > 0 && len(v.Ventanas) > 0 {
			filas = append(filas, v)
		}
	}
	return filas, s.Err()
}

func guardar(ruta string, filas []fila, ahora time.Time) ([]fila, error) {
	var vigentes []fila
	for _, v := range filas {
		if v.T >= ahora.Add(-30*24*time.Hour).Unix() {
			vigentes = append(vigentes, v)
		}
	}
	if ruta == "" {
		return vigentes, nil
	}
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return filas, err
	}
	f, err := os.CreateTemp(filepath.Dir(ruta), "uso-claude-*.tmp")
	if err != nil {
		return filas, err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	enc := json.NewEncoder(f)
	for _, v := range vigentes {
		if err = enc.Encode(v); err != nil {
			break
		}
	}
	if e := f.Close(); err == nil {
		err = e
	}
	if err == nil {
		err = os.Rename(tmp, ruta)
	}
	if err != nil {
		return filas, err
	}
	return vigentes, nil
}
