// Package pruebas tiene ayudas para las pruebas de los demás paquetes; el programa no lo usa.
package pruebas

import (
	"os"
	"regexp"
	"runtime"
	"strings"
)

// La unidad de una ruta de Windows ya con barras normales («C:/»), que no siga a una letra o cifra («https://» no).
var reUnidad = regexp.MustCompile(`(^|[^A-Za-z0-9])([A-Za-z]):/`)

// Las pruebas escriben sus rutas como en Windows (C:\Proyectos\Tienda Demo), donde Atalaya se probó primero. En Linux
// y macOS la barra invertida no separa, así que filepath.Base o filepath.Rel no las parten: se pasan a la forma del
// sistema, C:\Proyectos\Tienda Demo → /C/Proyectos/Tienda Demo. En Windows quedan igual, salvo con ATALAYA_RUTAS_UNIX=1,
// que las convierte también para probarlo sin Linux (Windows acepta las dos barras).
func convertir() bool {
	return runtime.GOOS != "windows" || os.Getenv("ATALAYA_RUTAS_UNIX") != ""
}

// Nativa convierte las rutas de un texto suelto: toda barra invertida es separador.
func Nativa(s string) string {
	if !convertir() {
		return s
	}
	return reUnidad.ReplaceAllString(strings.ReplaceAll(s, `\`, "/"), "$1/$2/")
}

// NativaJSON convierte las rutas dentro de un JSON, donde el separador va escapado (C:\\Proyectos): solo cambian las
// barras dobles, y los demás escapes («\n», «\"») quedan como están.
func NativaJSON(s string) string {
	if !convertir() {
		return s
	}
	return reUnidad.ReplaceAllString(strings.ReplaceAll(s, `\\`, "/"), "$1/$2/")
}
