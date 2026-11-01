//go:build !windows

package cuotas

import "os/exec"

func ocultarVentana(*exec.Cmd) {}
