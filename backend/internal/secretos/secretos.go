// Package secretos tapa lo que parece una credencial antes de mostrar un texto.
// Es una red de seguridad, no una garantía: el panel no lee archivos de
// credenciales, y esto cubre lo que pudo colarse en la salida de una orden.
package secretos

import (
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Cada patrón lleva sus pistas: textos (en minúsculas) de los que al menos uno tiene que estar para que el patrón
// pueda encajar. Casi ningún texto trae un secreto, y buscar las pistas es mucho más barato que correr las 8
// expresiones sobre todo el texto: Atalaya tapa cada orden, salida, bitácora y AVANCE.md que muestra.
var patrones = []struct {
	re     *regexp.Regexp
	por    string
	pistas []string
}{
	{regexp.MustCompile(`sk-ant-[A-Za-z0-9_\-]{8,}`), "sk-ant-[tapado]", []string{"sk-ant-"}},
	{regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_\-]{20,}`), "sk-[tapado]", []string{"sk-"}},
	{regexp.MustCompile(`\b(?:ghp|gho|ghu|ghs|github_pat)_[A-Za-z0-9_]{20,}`), "gh_[tapado]", []string{"ghp_", "gho_", "ghu_", "ghs_", "github_pat_"}},
	{regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`), "AKIA[tapado]", []string{"akia"}},
	{regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._\-~+/]{16,}=*`), "Bearer [tapado]", []string{"bearer"}},
	{regexp.MustCompile(`(?i)("(?:access_token|refresh_token|id_token|api_key|apikey|client_secret|password)"\s*:\s*")[^"]{6,}(")`), "${1}[tapado]${2}",
		[]string{"token\"", "api_key\"", "apikey\"", "client_secret\"", "password\""}},
	{regexp.MustCompile(`(?i)([?&](?:sig|signature|token|key|X-Amz-Signature)=)[^&\s"']{6,}`), "${1}[tapado]", []string{"sig=", "signature=", "token=", "key="}},
	{regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}`), "[jwt tapado]", []string{"eyj"}},
}

// Tapar reemplaza lo que parece un secreto.
func Tapar(s string) string {
	if s == "" {
		return s
	}
	bajo := strings.ToLower(s)
	for _, p := range patrones {
		if !algunaPista(bajo, p.pistas) {
			continue
		}
		s = p.re.ReplaceAllString(s, p.por)
		bajo = strings.ToLower(s)
	}
	return s
}

func algunaPista(bajo string, pistas []string) bool {
	for _, x := range pistas {
		if strings.Contains(bajo, x) {
			return true
		}
	}
	return false
}

// leido: un archivo ya tapado, con lo que lo identifica en el disco.
type leido struct {
	tam   int64
	mod   time.Time
	texto string
}

var (
	cacheMu sync.Mutex
	cache   = map[string]leido{}
)

// MaxCache: cuántos archivos guarda LeerTapado; al llenarse, se vacía (los que se usan vuelven en la vuelta siguiente).
const MaxCache = 512

// LeerTapado lee un archivo y lo devuelve tapado. Lo guarda mientras el archivo no cambie (mismo tamaño y misma hora
// de modificación): las bitácoras y los AVANCE.md se leen para cada agente en cada pedido del panel, y casi nunca
// cambiaron desde la vez anterior.
func LeerTapado(ruta string) (string, error) {
	fi, err := os.Stat(ruta)
	if err != nil {
		return "", err
	}
	cacheMu.Lock()
	l, ok := cache[ruta]
	cacheMu.Unlock()
	if ok && l.tam == fi.Size() && l.mod.Equal(fi.ModTime()) {
		return l.texto, nil
	}
	b, err := os.ReadFile(ruta)
	if err != nil {
		return "", err
	}
	t := Tapar(string(b))
	cacheMu.Lock()
	if len(cache) >= MaxCache {
		clear(cache)
	}
	// Con el tamaño y la hora del Stat de antes de leer: si el archivo cambió mientras se leía, la próxima vez no
	// coincide y se vuelve a leer.
	cache[ruta] = leido{tam: fi.Size(), mod: fi.ModTime(), texto: t}
	cacheMu.Unlock()
	return t, nil
}
