// Package secretos tapa lo que parece una credencial antes de mostrar un texto.
// Es una red de seguridad, no una garantía: el panel no lee archivos de
// credenciales, y esto cubre lo que pudo colarse en la salida de una orden.
package secretos

import "regexp"

var patrones = []struct {
	re  *regexp.Regexp
	por string
}{
	{regexp.MustCompile(`sk-ant-[A-Za-z0-9_\-]{8,}`), "sk-ant-[tapado]"},
	{regexp.MustCompile(`\bsk-(?:proj-)?[A-Za-z0-9_\-]{20,}`), "sk-[tapado]"},
	{regexp.MustCompile(`\b(?:ghp|gho|ghu|ghs|github_pat)_[A-Za-z0-9_]{20,}`), "gh_[tapado]"},
	{regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`), "AKIA[tapado]"},
	{regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._\-~+/]{16,}=*`), "Bearer [tapado]"},
	{regexp.MustCompile(`(?i)("(?:access_token|refresh_token|id_token|api_key|apikey|client_secret|password)"\s*:\s*")[^"]{6,}(")`), "${1}[tapado]${2}"},
	{regexp.MustCompile(`(?i)([?&](?:sig|signature|token|key|X-Amz-Signature)=)[^&\s"']{6,}`), "${1}[tapado]"},
	{regexp.MustCompile(`eyJ[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}\.[A-Za-z0-9_\-]{10,}`), "[jwt tapado]"},
}

// Tapar reemplaza lo que parece un secreto.
func Tapar(s string) string {
	for _, p := range patrones {
		s = p.re.ReplaceAllString(s, p.por)
	}
	return s
}
