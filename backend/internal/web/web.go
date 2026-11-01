// Package web embebe el frontend exportado (frontend/out, copiado a dist/ por scripts/compilar.ps1).
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var archivos embed.FS

// FS devuelve el frontend, o nil si todavía no se compiló.
func FS() fs.FS {
	sub, err := fs.Sub(archivos, "dist")
	if err != nil {
		return nil
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil
	}
	return sub
}
