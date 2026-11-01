package main

import (
	"path/filepath"
	"testing"
)

// La carpeta de datos nunca queda vacía: vacía, las rutas eran relativas y fuera de Windows Atalaya guardaba su base en
// la carpeta desde donde se lo lanzaba.
func TestCarpetaLocal(t *testing.T) {
	casa := filepath.Join("casa", "ana")
	t.Setenv("LOCALAPPDATA", "")
	t.Setenv("XDG_DATA_HOME", "")
	casos := map[string]string{
		"windows": filepath.Join(casa, "AppData", "Local"),
		"darwin":  filepath.Join(casa, "Library", "Application Support"),
		"linux":   filepath.Join(casa, ".local", "share"),
	}
	for sistema, quiere := range casos {
		if got := carpetaLocal(casa, sistema); got != quiere {
			t.Errorf("%s: %q, quería %q", sistema, got, quiere)
		}
	}
	t.Setenv("XDG_DATA_HOME", filepath.Join("datos", "ana"))
	if got := carpetaLocal(casa, "linux"); got != filepath.Join("datos", "ana") {
		t.Errorf("con XDG_DATA_HOME: %q", got)
	}
	t.Setenv("LOCALAPPDATA", filepath.Join("C", "Local"))
	if got := carpetaLocal(casa, "windows"); got != filepath.Join("C", "Local") {
		t.Errorf("con LOCALAPPDATA: %q", got)
	}
}
