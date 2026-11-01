//go:build !windows

package configurar

import "os/exec"

// ocultar: fuera de Windows no hay ventana que ocultar.
func ocultar(*exec.Cmd) {}
