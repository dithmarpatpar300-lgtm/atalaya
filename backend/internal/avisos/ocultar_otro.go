//go:build !windows

package avisos

import "os/exec"

// ocultar: fuera de Windows no hay ventana que ocultar (y tampoco notificación: powershell.exe no existe y Notificar
// devuelve el error).
func ocultar(*exec.Cmd) {}
