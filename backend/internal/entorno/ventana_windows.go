//go:build windows

package entorno

import (
	"os/exec"
	"syscall"
)

func ocultarVentana(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
