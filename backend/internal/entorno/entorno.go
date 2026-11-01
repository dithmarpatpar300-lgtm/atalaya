// Package entorno detecta las herramientas y las marcas de un proyecto. Solo lee.
package entorno

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"
)

// Herramienta: Claude Code o Codex en este equipo.
type Herramienta struct {
	ID           string `json:"id"`
	Nombre       string `json:"nombre"`
	Instalada    bool   `json:"instalada"`
	Ruta         string `json:"ruta,omitempty"`
	Version      string `json:"version,omitempty"`
	Datos        string `json:"datos"`
	DatosExisten bool   `json:"datos_existen"`
	MCP          bool   `json:"mcp"`
}

// Opciones reúne lo que depende del equipo.
type Opciones struct {
	Casa      string
	CodexHome string
	Buscar    func(nombre string) (string, error)
	Version   func(ruta string) string
}

// Herramientas devuelve Claude Code y Codex, en ese orden.
func Herramientas(o Opciones) []Herramienta {
	if o.Buscar == nil {
		o.Buscar = exec.LookPath
	}
	if o.Version == nil {
		o.Version = versionEjecutable
	}
	codexHome := o.CodexHome
	if codexHome == "" {
		codexHome = filepath.Join(o.Casa, ".codex")
	}
	hs := []Herramienta{
		{ID: "claude", Nombre: "Claude Code", Datos: filepath.Join(o.Casa, ".claude"), MCP: mcpJSON(filepath.Join(o.Casa, ".claude.json"))},
		{ID: "codex", Nombre: "Codex", Datos: codexHome, MCP: mcpTOML(filepath.Join(codexHome, "config.toml"))},
	}
	for i := range hs {
		h := &hs[i]
		h.DatosExisten = esCarpeta(h.Datos)
		if ruta, err := o.Buscar(h.ID); err == nil {
			h.Instalada, h.Ruta = true, ruta
			h.Version = o.Version(ruta)
		}
	}
	return hs
}

var patronVersion = regexp.MustCompile(`\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?`)

func extraerVersion(s string) string { return patronVersion.FindString(s) }

func versionEjecutable(ruta string) string {
	ctx, cancelar := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancelar()
	cmd := exec.CommandContext(ctx, ruta, "--version")
	ocultarVentana(cmd)
	salida, err := cmd.Output()
	if err != nil || ctx.Err() != nil {
		return ""
	}
	return extraerVersion(string(salida))
}

func mcpJSON(ruta string) bool {
	f, err := os.Open(ruta)
	if err != nil {
		return false
	}
	defer f.Close()
	var config struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
	}
	dec := json.NewDecoder(f)
	if dec.Decode(&config) != nil || dec.Decode(new(any)) != io.EOF {
		return false
	}
	_, atalaya := config.MCPServers["atalaya"]
	_, centro := config.MCPServers["centro"]
	return atalaya || centro
}

var seccionMCP = regexp.MustCompile(`^\s*\[\s*mcp_servers\s*\.\s*(?:atalaya|centro|"atalaya"|"centro")\s*\]\s*(?:#.*)?$`)

func mcpTOML(ruta string) bool {
	f, err := os.Open(ruta)
	if err != nil {
		return false
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		if seccionMCP.MatchString(s.Text()) {
			return true
		}
	}
	return false
}

// Marcas: lo que un proyecto ya tiene para el asistente.
type Marcas struct {
	AgentsMD     bool `json:"agents_md"`
	ClaudeMD     bool `json:"claude_md"`
	MCPJSON      bool `json:"mcp_json"`
	MCPAtalaya   bool `json:"mcp_atalaya"`
	Trazabilidad bool `json:"trazabilidad"`
}

// MarcasDe revisa las marcas en la raíz de un proyecto.
func MarcasDe(raiz string) Marcas {
	mcp := filepath.Join(raiz, ".mcp.json")
	return Marcas{
		AgentsMD: existeArchivo(filepath.Join(raiz, "AGENTS.md")),
		ClaudeMD: existeArchivo(filepath.Join(raiz, "CLAUDE.md")),
		MCPJSON:  existeArchivo(mcp), MCPAtalaya: mcpJSON(mcp),
		Trazabilidad: esCarpeta(filepath.Join(raiz, "docs", "trazabilidad")),
	}
}

func existeArchivo(ruta string) bool {
	info, err := os.Stat(ruta)
	return err == nil && !info.IsDir()
}

func esCarpeta(ruta string) bool {
	info, err := os.Stat(ruta)
	return err == nil && info.IsDir()
}
