package avisos

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
	"unicode"
	"unicode/utf16"
)

// Windows: la notificación de Windows, con PowerShell 5.1 y las clases de WinRT (sin instalar nada). El texto viene de
// los transcritos, así que es dato: llega al script por variables de entorno, el script lo escapa como XML y nunca se
// pega en el código. La notificación sale a nombre de Windows PowerShell (su AppUserModelID registrado); un clic abre
// la url, que es el panel en su sección.
type Windows struct{}

const scriptAviso = `$ErrorActionPreference = 'Stop'
[void][Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime]
[void][Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime]
$e = [System.Security.SecurityElement]
$t = $e::Escape($env:ATALAYA_AVISO_TITULO)
$x = $e::Escape($env:ATALAYA_AVISO_TEXTO)
$u = $e::Escape($env:ATALAYA_AVISO_URL)
$xml = '<toast activationType="protocol" launch="' + $u + '"><visual><binding template="ToastGeneric"><text>' + $t + '</text><text>' + $x + '</text></binding></visual></toast>'
$d = New-Object Windows.Data.Xml.Dom.XmlDocument
$d.LoadXml($xml)
$n = [Windows.UI.Notifications.ToastNotification]::new($d)
$app = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($app).Show($n)
`

// scriptCodificado: el script para -EncodedCommand (UTF-16LE en base64), así ninguna comilla pasa por la consola.
var scriptCodificado = func() string {
	u := utf16.Encode([]rune(scriptAviso))
	b := make([]byte, 0, 2*len(u))
	for _, c := range u {
		b = append(b, byte(c), byte(c>>8))
	}
	return base64.StdEncoding.EncodeToString(b)
}()

// Notificar muestra el aviso; tarda lo que tarda PowerShell en arrancar (1 a 2 s), así que se llama aparte del bucle.
func (Windows) Notificar(titulo, texto, url string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass",
		"-WindowStyle", "Hidden", "-EncodedCommand", scriptCodificado)
	cmd.Env = append(os.Environ(),
		"ATALAYA_AVISO_TITULO="+limpiar(titulo, 80),
		"ATALAYA_AVISO_TEXTO="+limpiar(texto, 300),
		"ATALAYA_AVISO_URL="+limpiar(url, 200))
	ocultar(cmd)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("notificación de Windows: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// limpiar quita los caracteres de control (salvo el salto de línea) y corta en max runas: una variable de entorno no
// lleva NUL, y una notificación larga se corta sola de todos modos.
func limpiar(s string, max int) string {
	r := []rune(strings.Map(func(c rune) rune {
		if c == '\n' || !unicode.IsControl(c) {
			return c
		}
		return -1
	}, s))
	if len(r) > max {
		r = append(r[:max-1], '…')
	}
	return string(r)
}
