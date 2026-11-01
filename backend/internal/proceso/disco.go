package proceso

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"atalaya/internal/alertas"
)

// Dónde buscar los temporales: la raíz que fija AGENTS.md y las que mira el script que AGENTS.md nombra para
// listarlos. Si el script cambia, el panel lo sigue.

var (
	reLugarFijo = regexp.MustCompile(`'([A-Za-z]:\\[^']*-tmp)'`)
	reLugarEnv  = regexp.MustCompile(`\$env:(\w+)\s+'([^']*-tmp)'`)
)

// Lugares de los temporales del proyecto, sin repetir.
func Lugares(raiz string, r alertas.Reglas) []string {
	var out []string
	visto := map[string]bool{}
	sumar := func(l string) {
		l = filepath.Clean(l)
		if k := strings.ToLower(l); l != "." && !visto[k] {
			visto[k] = true
			out = append(out, l)
		}
	}
	if r.TmpRaiz != "" {
		sumar(r.TmpRaiz)
	}
	if r.Retomar != "" && raiz != "" {
		if b, err := os.ReadFile(filepath.Join(raiz, filepath.FromSlash(r.Retomar))); err == nil {
			t := string(b)
			for _, m := range reLugarFijo.FindAllStringSubmatch(t, -1) {
				sumar(m[1])
			}
			for _, m := range reLugarEnv.FindAllStringSubmatch(t, -1) {
				if base := os.Getenv(m[1]); base != "" {
					sumar(filepath.Join(base, m[2]))
				}
			}
		}
	}
	return out
}

// Firma de las carpetas de los lugares (nombre y hora), para avisar por SSE solo cuando cambian. No las mide.
func Firma(lugares []string) string {
	var b strings.Builder
	for _, l := range lugares {
		es, err := os.ReadDir(l)
		if err != nil {
			continue
		}
		for _, e := range es {
			if !e.IsDir() {
				continue
			}
			b.WriteString(l + `\` + e.Name())
			if fi, err := e.Info(); err == nil {
				b.WriteString("@" + strconv.FormatInt(fi.ModTime().UnixNano(), 10))
			}
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// maxEntradas: lo más que se recorre de una carpeta para medirla; una más grande se informa con lo contado.
const maxEntradas = 20000

// Escanear lista y mide las carpetas de los lugares, como retomar.ps1: archivos, bytes y .rgpart.
func Escanear(lugares []string) []Carpeta {
	var out []Carpeta
	for _, l := range lugares {
		es, err := os.ReadDir(l)
		if err != nil {
			continue
		}
		for _, e := range es {
			if !e.IsDir() {
				continue
			}
			c := Carpeta{Ruta: filepath.Join(l, e.Name()), Nombre: e.Name(), Lugar: l}
			if fi, err := e.Info(); err == nil {
				c.Mod = fi.ModTime()
			}
			n := 0
			_ = filepath.WalkDir(c.Ruta, func(_ string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if n++; n > maxEntradas {
					return fs.SkipAll
				}
				if d.IsDir() {
					return nil
				}
				c.Archivos++
				if strings.HasSuffix(strings.ToLower(d.Name()), ".rgpart") {
					c.Parciales++
				}
				if fi, err := d.Info(); err == nil {
					c.Bytes += fi.Size()
					if fi.ModTime().After(c.Mod) {
						c.Mod = fi.ModTime()
					}
				}
				return nil
			})
			out = append(out, c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Mod.After(out[j].Mod) })
	return out
}
