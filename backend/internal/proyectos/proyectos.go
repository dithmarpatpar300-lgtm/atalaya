// Package proyectos lee el registro de los proyectos: proyectos.json, junto a atalaya.exe (cada equipo tiene el suyo y
// no va al repositorio; el ejemplo es proyectos.ejemplo.json). Es la lista que sigue Atalaya: sus transcritos, su
// trazabilidad, sus alertas y la portada «Tus proyectos».
//
//	[
//	  {"nombre": "Mi-Proyecto", "ruta": "C:\\Proyectos\\Mi-Proyecto", "sigla": "MP", "nivel": 1}
//	]
//
// El nombre es el de la carpeta del proyecto: es el que los transcritos de Claude Code y las llamadas a Codex le dan
// (la carpeta raíz del repositorio), y así el panel junta lo de los dos lados. Solo lee.
package proyectos

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// Archivo: el nombre del registro.
const Archivo = "proyectos.json"

// Proyecto registrado.
type Proyecto struct {
	Nombre string `json:"nombre"`
	Ruta   string `json:"ruta"`
	// Sigla: dos o tres letras para la casilla de la tarjeta; si falta, sale del nombre.
	Sigla string `json:"sigla,omitempty"`
	// Nivel del contrato de trazabilidad que el proyecto se compromete a cumplir: 0 (sin contrato) a 3.
	Nivel int `json:"nivel"`
}

// Leer el registro. Un archivo que no existe devuelve un error que cumple errors.Is(err, os.ErrNotExist).
func Leer(ruta string) ([]Proyecto, error) {
	b, err := os.ReadFile(ruta)
	if err != nil {
		return nil, err
	}
	return Parsear(b)
}

// Parsear valida la lista: nombre y ruta no vacíos, nombres sin repetir y nivel de 0 a 3.
func Parsear(b []byte) ([]Proyecto, error) {
	var ps []Proyecto
	if err := json.Unmarshal(b, &ps); err != nil {
		return nil, fmt.Errorf("%s no es una lista de proyectos: %w", Archivo, err)
	}
	vistos := map[string]bool{}
	for i := range ps {
		p := &ps[i]
		p.Nombre, p.Ruta, p.Sigla = strings.TrimSpace(p.Nombre), strings.TrimSpace(p.Ruta), strings.TrimSpace(p.Sigla)
		if p.Nombre == "" || p.Ruta == "" {
			return nil, fmt.Errorf("%s: el proyecto %d no trae nombre o ruta", Archivo, i+1)
		}
		if vistos[strings.ToLower(p.Nombre)] {
			return nil, fmt.Errorf("%s: %q está dos veces", Archivo, p.Nombre)
		}
		vistos[strings.ToLower(p.Nombre)] = true
		if p.Nivel < 0 || p.Nivel > 3 {
			return nil, fmt.Errorf("%s: %q tiene nivel %d; va de 0 a 3", Archivo, p.Nombre, p.Nivel)
		}
		if p.Sigla == "" {
			p.Sigla = Sigla(p.Nombre)
		}
	}
	return ps, nil
}

// Sigla: las iniciales de las dos primeras palabras («Tienda Demo» → TD, «Mi-Proyecto» → MP) o, con una sola
// palabra, sus dos primeras letras («Cuotas» → CU).
func Sigla(nombre string) string {
	palabras := strings.FieldsFunc(nombre, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) })
	var out []rune
	switch len(palabras) {
	case 0:
		return "?"
	case 1:
		rs := []rune(palabras[0])
		out = rs[:min(2, len(rs))]
	default:
		out = []rune{[]rune(palabras[0])[0], []rune(palabras[1])[0]}
	}
	return strings.ToUpper(string(out))
}

// NombreDistinto: el nombre no es el de la carpeta. Los transcritos nombran el proyecto por su carpeta, así que sus
// agentes no se juntarían con él.
func NombreDistinto(p Proyecto) bool {
	return !strings.EqualFold(p.Nombre, filepath.Base(filepath.Clean(p.Ruta)))
}
