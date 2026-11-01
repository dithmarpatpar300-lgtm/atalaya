package claude

import (
	"regexp"
	"strings"
	"time"
)

// La vista de la sesión principal. Una sesión no tiene encargo ni bitácora: conversa con
// la persona, decide y lanza al equipo. Lo que se resume de ella: cada pedido de la persona y lo que siguió (a
// quién lanzó, cuántas veces preguntó a Atalaya, si miró la cuota, si comiteó y por qué etapas pasó), los
// documentos de estado que tocó, la cuota y los commits.

// PedidoSesion: un mensaje de la persona y lo que la sesión hizo hasta el siguiente.
type PedidoSesion struct {
	N          int       `json:"n"` // el paso del pedido
	Hora       time.Time `json:"hora"`
	Tipo       string    `json:"tipo"` // pedido | permiso
	Texto      string    `json:"texto"`
	Caracteres string    `json:"caracteres"`
	Lanzados   []string  `json:"lanzados"` // ids de los subagentes que lanzó
	Consultas  int       `json:"consultas"`
	Cuota      int       `json:"cuota"`
	Commits    int       `json:"commits"`
	Pasos      int       `json:"pasos"`
	Etapas     []string  `json:"etapas"`
	Respuesta  string    `json:"respuesta,omitempty"` // la primera línea de su última respuesta
}

// DocEstado: un documento de estado que la sesión editó.
type DocEstado struct {
	// ID: la ruta completa normalizada, la misma con que se agrupa. Dos documentos de proyectos o copias distintas
	// pueden dar la misma ruta corta: el panel pagina y dibuja por el ID, y la ruta solo se muestra.
	ID        string    `json:"id"`
	Ruta      string    `json:"ruta"`
	Ediciones int       `json:"ediciones"`
	Ultima    time.Time `json:"ultima"`
}

// CommitSesion: una orden git commit que corrió la sesión.
type CommitSesion struct {
	Hora  time.Time `json:"hora"`
	Paso  int       `json:"paso"`
	Orden string    `json:"orden"`
}

// ResumenSesion de una sesión.
type ResumenSesion struct {
	Pedidos     []PedidoSesion `json:"pedidos"`
	Documentos  []DocEstado    `json:"documentos"`
	Commits     []CommitSesion `json:"commits"`
	Consultas   int            `json:"consultas"`
	Lecturas    int            `json:"lecturas_cuota"`
	UltimaCuota time.Time      `json:"ultima_cuota,omitempty"`
	Mensajes    int            `json:"mensajes_sistema"` // avisos y entregas de agentes que no son pedidos
}

var (
	// Sin \b: en Go solo entiende ASCII y no corta después de la «í» de «Sí».
	rePermiso = regexp.MustCompile(`(?i)^(s[ií]|no|ok|dale|listo|procede|adelante|hazlo|perfecto|de acuerdo|vale)([\s,.;:!]|$)`)
	// Mensajes que llegan como «pedido» pero no los escribió la persona.
	sistema = []string{"another claude session sent a message", "[subagent hand-back]", "this session is being continued", "caveat:", "<"}
	// git y, en la misma orden, commit: también «git -C 'D:\x' commit -F msg» y «cmd /c "git commit"».
	reGitCommit = regexp.MustCompile(`\bgit\b[^;&|\n]*\bcommit\b`)
	// Los documentos de estado que el coordinador mantiene.
	reDocEstado = regexp.MustCompile(`(?i)(estado-para-retomar|estado-del-proyecto|estado-historia|/avance\.md$|docs/planes/|herramientas\.md$|prompt-continuidad|/memory/)`)
)

func esDeSistema(t string) bool {
	l := strings.ToLower(strings.TrimSpace(t))
	for _, s := range sistema {
		if strings.HasPrefix(l, s) {
			return true
		}
	}
	return false
}

// tipoPedido: un «sí, comitéalo» es un permiso; lo demás, un pedido.
func tipoPedido(t string) string {
	t = strings.TrimSpace(t)
	if len([]rune(t)) <= 80 && rePermiso.MatchString(t) {
		return "permiso"
	}
	return "pedido"
}

// Sesion resume una sesión por su id; false si no existe o no es una sesión.
func (ix *Indice) Sesion(id string) (ResumenSesion, bool) {
	ix.mu.RLock()
	defer ix.mu.RUnlock()
	var a *archivo
	for _, x := range ix.arch {
		if x.agente.ID == id {
			a = x
			break
		}
	}
	if a == nil || a.agente.Tipo != "sesion" {
		return ResumenSesion{}, false
	}
	// Los subagentes que lanzó cada «Delega», por el id de la llamada.
	lanzados := map[string]string{}
	for _, x := range ix.arch {
		if x.agente.Padre == id && x.agente.ToolUseID != "" {
			lanzados[x.agente.ToolUseID] = x.agente.ID
		}
	}
	r := ResumenSesion{Pedidos: []PedidoSesion{}, Documentos: []DocEstado{}, Commits: []CommitSesion{}}
	docs := map[string]*DocEstado{}
	var orden []string
	var actual *PedidoSesion
	etapas := map[string]bool{}
	for _, p := range a.pasos {
		if p.Tipo == "Pide" {
			if esDeSistema(p.Resumen) {
				r.Mensajes++
				continue
			}
			r.Pedidos = append(r.Pedidos, PedidoSesion{N: p.N, Hora: p.Hora, Tipo: tipoPedido(p.Resumen), Texto: p.Resumen, Caracteres: p.Meta,
				Lanzados: []string{}, Etapas: []string{}})
			actual = &r.Pedidos[len(r.Pedidos)-1]
			etapas = map[string]bool{}
			continue
		}
		_, consulta := HerramientaAtalaya(p.Herramienta)
		cuota := miraCuota(p)
		commit := p.Tipo == "Ejecuta" && reGitCommit.MatchString(p.normalizada())
		if consulta {
			r.Consultas++
		}
		if cuota {
			r.Lecturas++
			r.UltimaCuota = p.Hora
		}
		if commit {
			r.Commits = append(r.Commits, CommitSesion{Hora: p.Hora, Paso: p.N, Orden: recortar(p.orden, 200)})
		}
		if p.editado != "" && !p.Error && reDocEstado.MatchString(p.editado) {
			d := docs[p.editado]
			if d == nil {
				d = &DocEstado{ID: p.editado, Ruta: p.Hilo}
				docs[p.editado] = d
				orden = append(orden, p.editado)
			}
			d.Ediciones++
			d.Ultima = p.Hora
		}
		if actual == nil {
			continue
		}
		actual.Pasos++
		if consulta {
			actual.Consultas++
		}
		if cuota {
			actual.Cuota++
		}
		if commit {
			actual.Commits++
		}
		if sub, ok := lanzados[p.idUso]; ok {
			actual.Lanzados = append(actual.Lanzados, sub)
		}
		if p.Etapa != "" && !etapas[p.Etapa] {
			etapas[p.Etapa] = true
			actual.Etapas = append(actual.Etapas, p.Etapa)
		}
		if p.Tipo == "Responde" {
			actual.Respuesta = p.Resumen
		}
	}
	for _, k := range orden {
		r.Documentos = append(r.Documentos, *docs[k])
	}
	return r, true
}
