package servidor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// El paso «Conectar» del asistente se pone verde cuando Atalaya ve llegar a cada herramienta por su MCP, no cuando su
// configuración dice que debería: se anota cuándo se conectó (initialize) y cuándo consultó (tools/call) por última vez,
// y se guarda en la carpeta de Atalaya para que siga en verde después de reiniciar.

// ClienteMCP: lo último que hizo una herramienta con el MCP de Atalaya.
type ClienteMCP struct {
	Conexion time.Time `json:"conexion,omitzero"`
	Consulta time.Time `json:"consulta,omitzero"`
}

type clientesMCP struct {
	mu      sync.Mutex
	m       map[string]ClienteMCP
	cargado bool
	escrito time.Time
}

// herramientaDeCliente: «claude» o «codex» por el User-Agent («claude-code/2.1.316 (…)», «codex_cli_rs/0.166.0 …»);
// vacío si es otro (un curl de prueba).
func herramientaDeCliente(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "codex"):
		return "codex"
	case strings.Contains(ua, "claude"):
		return "claude"
	}
	return ""
}

func (s *Servidor) rutaClientesMCP() string {
	if s.Datos == "" {
		return ""
	}
	return filepath.Join(s.Datos, "mcp-clientes.json")
}

// cargarClientes: la primera vez, lo guardado. Con s.clientes.mu tomado.
func (s *Servidor) cargarClientes() {
	if s.clientes.cargado {
		return
	}
	s.clientes.cargado = true
	s.clientes.m = map[string]ClienteMCP{}
	if r := s.rutaClientesMCP(); r != "" {
		if b, err := os.ReadFile(r); err == nil {
			_ = json.Unmarshal(b, &s.clientes.m)
		}
	}
}

// anotarCliente: lo que hizo una herramienta conocida. Escribe el archivo cuando algo pasa de no visto a visto, y si no,
// como mucho una vez por minuto.
func (s *Servidor) anotarCliente(ua, metodo string) {
	h := herramientaDeCliente(ua)
	if h == "" || (metodo != "initialize" && metodo != "tools/call") {
		return
	}
	ahora := s.ahora()
	s.clientes.mu.Lock()
	defer s.clientes.mu.Unlock()
	s.cargarClientes()
	c := s.clientes.m[h]
	nuevo := false
	if metodo == "initialize" {
		nuevo = c.Conexion.IsZero()
		c.Conexion = ahora
	} else {
		nuevo = c.Consulta.IsZero()
		c.Consulta = ahora
	}
	s.clientes.m[h] = c
	r := s.rutaClientesMCP()
	if r == "" || (!nuevo && ahora.Sub(s.clientes.escrito) < time.Minute) {
		return
	}
	if b, err := json.Marshal(s.clientes.m); err == nil && os.MkdirAll(filepath.Dir(r), 0o755) == nil {
		tmp := r + ".tmp"
		if os.WriteFile(tmp, b, 0o644) == nil && os.Rename(tmp, r) == nil {
			s.clientes.escrito = ahora
		}
	}
}

// clientesVistos: una copia de lo anotado, por herramienta.
func (s *Servidor) clientesVistos() map[string]ClienteMCP {
	s.clientes.mu.Lock()
	defer s.clientes.mu.Unlock()
	s.cargarClientes()
	out := make(map[string]ClienteMCP, len(s.clientes.m))
	for k, v := range s.clientes.m {
		out[k] = v
	}
	return out
}
