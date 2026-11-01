package entorno

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func escribir(t *testing.T, ruta, contenido string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(ruta), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(ruta, []byte(contenido), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestExtraerVersion(t *testing.T) {
	for entrada, espera := range map[string]string{
		"2.1.309 (Claude Code)":       "2.1.309",
		"codex-cli 0.166.0-alpha.5.1": "0.166.0-alpha.5.1",
		"basura":                      "",
	} {
		if got := extraerVersion(entrada); got != espera {
			t.Errorf("extraerVersion(%q) = %q, quería %q", entrada, got, espera)
		}
	}
}

func TestHerramientas(t *testing.T) {
	casa := t.TempDir()
	codex := filepath.Join(casa, "datos-codex")
	escribir(t, filepath.Join(casa, ".claude.json"), `{"otro":1,"oauthAccount":{"secreto":"no devolver"},"mcpServers":{"atalaya":{}},"final":true}`)
	escribir(t, filepath.Join(casa, ".claude", "dato"), "")
	escribir(t, filepath.Join(codex, "config.toml"), "[mcp_servers.\"centro\"]\nvariable = 'secreto'\n")
	llamadas := 0
	hs := Herramientas(Opciones{Casa: casa, CodexHome: codex,
		Buscar: func(nombre string) (string, error) {
			if nombre == "claude" {
				return "ruta-claude", nil
			}
			return "", errors.New("no encontrado")
		},
		Version: func(ruta string) string {
			llamadas++
			if ruta != "ruta-claude" {
				t.Errorf("ruta inesperada: %q", ruta)
			}
			return "2.1.309"
		},
	})
	espera := []Herramienta{
		{ID: "claude", Nombre: "Claude Code", Instalada: true, Ruta: "ruta-claude", Version: "2.1.309", Datos: filepath.Join(casa, ".claude"), DatosExisten: true, MCP: true},
		{ID: "codex", Nombre: "Codex", Datos: codex, DatosExisten: true, MCP: true},
	}
	if !reflect.DeepEqual(hs, espera) || llamadas != 1 {
		t.Errorf("herramientas: %+v, llamadas: %d", hs, llamadas)
	}
	b, err := json.Marshal(hs)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "oauthAccount") || strings.Contains(string(b), "secreto") {
		t.Error("se devolvieron datos ajenos")
	}
}

func TestMCP(t *testing.T) {
	casa := t.TempDir()
	claude := filepath.Join(casa, ".claude.json")
	toml := filepath.Join(casa, ".codex", "config.toml")
	buscar := func(string) (string, error) { return "", errors.New("ausente") }
	for _, caso := range []struct {
		nombre, json, toml string
		claude, codex      bool
	}{
		{"atalaya", `{"mcpServers":{"atalaya":{}}}`, "[mcp_servers.atalaya]", true, true},
		{"centro", `{"mcpServers":{"centro":{}}}`, "[mcp_servers.centro]", true, true},
		{"comillas", `{"mcpServers":{"otro":{}}}`, "[mcp_servers.\"atalaya\"] # comentario", false, true},
		{"otro", `{"mcpServers":{"otro":{}}}`, "[mcp_servers.otro]", false, false},
		{"roto", `{"mcpServers":{"atalaya":{}}`, "valor = '[mcp_servers.atalaya]'", false, false},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			escribir(t, claude, caso.json)
			escribir(t, toml, caso.toml)
			hs := Herramientas(Opciones{Casa: casa, Buscar: buscar})
			if hs[0].MCP != caso.claude || hs[1].MCP != caso.codex {
				t.Errorf("MCP: %+v", hs)
			}
		})
	}
	if got := Herramientas(Opciones{Casa: casa, Buscar: buscar})[1].Datos; got != filepath.Join(casa, ".codex") {
		t.Errorf("datos: %q", got)
	}
}

func TestMarcasDe(t *testing.T) {
	raiz := t.TempDir()
	if got := MarcasDe(raiz); got != (Marcas{}) {
		t.Fatalf("vacío: %+v", got)
	}
	escribir(t, filepath.Join(raiz, "AGENTS.md"), "")
	escribir(t, filepath.Join(raiz, "CLAUDE.md"), "")
	escribir(t, filepath.Join(raiz, ".mcp.json"), `{"mcpServers":{"centro":{}}}`)
	if err := os.MkdirAll(filepath.Join(raiz, "docs", "trazabilidad"), 0755); err != nil {
		t.Fatal(err)
	}
	if got := MarcasDe(raiz); got != (Marcas{true, true, true, true, true}) {
		t.Errorf("marcas: %+v", got)
	}
	escribir(t, filepath.Join(raiz, ".mcp.json"), `{"mcpServers":{"atalaya":{}}`)
	if got := MarcasDe(raiz); !got.MCPJSON || got.MCPAtalaya {
		t.Errorf("JSON roto: %+v", got)
	}
}
