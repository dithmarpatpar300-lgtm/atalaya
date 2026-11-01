package avisos

import (
	"os/exec"
	"syscall"
)

// ocultar: PowerShell arranca sin ventana de consola (CREATE_NO_WINDOW); -WindowStyle Hidden llega tarde, con la
// ventana ya parpadeando.
func ocultar(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
}
