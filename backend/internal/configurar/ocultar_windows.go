//go:build windows

package configurar

import (
	"os/exec"
	"syscall"
)

// ocultar: un programa de consola lanzado por Atalaya no abre una ventana negra.
func ocultar(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
