package cuotas

import (
	"os/exec"
	"syscall"
)

// ocultarVentana evita que cada lectura abra una consola en el escritorio.
func ocultarVentana(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
