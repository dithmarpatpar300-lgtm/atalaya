//go:build !windows

package usoclaude

import "os/exec"

func ocultar(*exec.Cmd) {}
