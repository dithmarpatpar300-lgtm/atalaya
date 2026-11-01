package entorno

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// En Linux (WSL), un claude que solo está en una carpeta de Windows montada (/mnt/c/…) no cuenta; uno de Linux, sí.
func TestBuscarSinLasCarpetasDeWindows(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("solo en Linux")
	}
	if !reMontajeWindows.MatchString("/mnt/c/Users/x/AppData/Roaming/npm") || reMontajeWindows.MatchString("/mnt/data/bin") {
		t.Fatal("el patrón de las carpetas de Windows")
	}
	propia := t.TempDir()
	t.Setenv("PATH", "/mnt/c/no-existe"+string(os.PathListSeparator)+propia)
	if _, err := buscarEnEsteSistema("claude"); err == nil {
		t.Fatal("encontró un claude que no hay")
	}
	exe := filepath.Join(propia, "claude")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if r, err := buscarEnEsteSistema("claude"); err != nil || r != exe {
		t.Fatalf("buscar = %q, %v", r, err)
	}
}
