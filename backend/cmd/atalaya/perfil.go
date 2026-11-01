//go:build perfil

// Solo para medir: con -tags perfil, pprof en 127.0.0.1:7439 (lejos del panel y del MCP).
package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

func init() {
	go func() { log.Println(http.ListenAndServe("127.0.0.1:7439", nil)) }()
}
