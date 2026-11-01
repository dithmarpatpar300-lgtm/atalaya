//go:build !windows

package entorno

import "os/exec"

func ocultarVentana(*exec.Cmd) {}
