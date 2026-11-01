// Package configurar es la única puerta por la que Atalaya cambia algo fuera de lo suyo: el registro de proyectos, la
// skill o la conexión MCP de Claude Code y de Codex (docs/plan-configuracion.md §5). El panel es de solo lectura; esta
// puerta es la excepción, acotada:
//
//   - Lista cerrada: solo los archivos que el servidor declara permitidos, y solo los programas que tiene resueltos,
//     con argumentos que arma Atalaya (nunca un shell ni texto libre). Nunca un archivo dentro de un proyecto.
//   - Primero la vista previa (Proponer) y después Aplicar con su id: si el archivo cambió entre una y otra, no se aplica.
//   - Antes de escribir, una copia de lo que había; la escritura es atómica (un temporal que se renombra).
//   - Cada cambio queda en el historial (cambios.jsonl) con cómo deshacerlo, y Deshacer lo revierte.
//
// Quién puede llamarla lo decide el servidor (solo el panel: origen local y su cabecera; el MCP no la tiene).
package configurar

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Tipos de cambio.
const (
	Archivo = "archivo" // escribir un archivo de la lista cerrada
	Orden   = "orden"   // correr un programa permitido con argumentos fijos
)

// VidaPropuesta: cuánto vale una vista previa sin aplicar.
const VidaPropuesta = 10 * time.Minute

// Cambio: una de las cosas que hace una operación.
type Cambio struct {
	Tipo string `json:"tipo"`
	// Archivo: la ruta, lo que tiene ahora (Existia false si no existe) y lo que va a tener.
	Ruta    string `json:"ruta,omitempty"`
	Existia bool   `json:"existia,omitempty"`
	Antes   string `json:"antes,omitempty"`
	Despues string `json:"despues,omitempty"`
	// Orden: el programa (su nombre en la lista de permitidos), sus argumentos y los de la orden que lo deshace.
	Programa   string   `json:"programa,omitempty"`
	Argumentos []string `json:"argumentos,omitempty"`
	Deshacer   []string `json:"deshacer,omitempty"`
}

// Propuesta: la vista previa de una operación, esperando que el panel la aplique.
type Propuesta struct {
	ID        string    `json:"id"`
	Operacion string    `json:"operacion"`
	Resumen   string    `json:"resumen"`
	Cambios   []Cambio  `json:"cambios"`
	Creada    time.Time `json:"creada"`
}

// Entrada: una línea del historial. Copias: ruta del archivo → su copia de antes (vacía si no existía).
type Entrada struct {
	ID        string            `json:"id"`
	Hora      time.Time         `json:"hora"`
	Operacion string            `json:"operacion"`
	Resumen   string            `json:"resumen"`
	Cambios   []Cambio          `json:"cambios"`
	Copias    map[string]string `json:"copias,omitempty"`
	Salida    string            `json:"salida,omitempty"`
	Error     string            `json:"error,omitempty"`
	// Deshace: el id de la entrada que esta deshace; Deshecha: la que ya se deshizo.
	Deshace  string `json:"deshace,omitempty"`
	Deshecha bool   `json:"deshecha,omitempty"`
}

// Puerta: la lista cerrada, dónde guarda las copias y el historial, y cómo corre un programa.
type Puerta struct {
	// Dir: la carpeta propia de Atalaya; las copias van en Dir/copias/<id>/ y el historial en Dir/cambios.jsonl.
	Dir string
	// Permitido: si una ruta (ya limpia y absoluta) está en la lista cerrada de archivos.
	Permitido func(ruta string) bool
	// Programa: el ejecutable resuelto de un nombre de la lista cerrada («claude», «codex»), o false si no está (no
	// instalado, o no es de la lista). Se pregunta en cada uso: uno instalado después de arrancar también vale.
	Programa func(nombre string) (string, bool)
	// Correr: por omisión, el programa sin shell, con 60 s de límite; devuelve lo que escribió.
	Correr func(ctx context.Context, exe string, args []string) (string, error)
	Ahora  func() time.Time

	mu         sync.Mutex
	propuestas map[string]Propuesta
}

func (p *Puerta) programa(nombre string) (string, bool) {
	if p.Programa == nil {
		return "", false
	}
	return p.Programa(nombre)
}

func (p *Puerta) ahora() time.Time {
	if p.Ahora != nil {
		return p.Ahora()
	}
	return time.Now()
}

// Errores que el servidor muestra tal cual.
var (
	ErrNoPermitido  = errors.New("ese archivo no está entre los que Atalaya puede cambiar")
	ErrPrograma     = errors.New("ese programa no está entre los que Atalaya puede correr")
	ErrNoHay        = errors.New("no hay una vista previa con ese id (vence a los 10 min): vuelve a pedirla")
	ErrCambioFuera  = errors.New("el archivo cambió desde la vista previa: vuelve a pedirla")
	ErrNoEnHistoria = errors.New("ese cambio no está en el historial")
	ErrYaDeshecho   = errors.New("ese cambio ya se deshizo")
	// ErrCambiadoDespues: deshacer pisaría lo que alguien cambió después en el archivo.
	ErrCambiadoDespues = errors.New("el archivo cambió después de aplicarlo; deshacerlo pisaría esos cambios")
)

// limpia: absoluta y sin «..»; en Windows, sin distinguir mayúsculas al comparar (eso lo hace Permitido).
func limpia(ruta string) (string, bool) {
	if ruta == "" || !filepath.IsAbs(ruta) {
		return "", false
	}
	return filepath.Clean(ruta), true
}

// Proponer arma la vista previa: valida cada cambio contra la lista cerrada y anota lo que hay ahora en cada archivo.
func (p *Puerta) Proponer(operacion, resumen string, cambios []Cambio) (Propuesta, error) {
	if len(cambios) == 0 {
		return Propuesta{}, errors.New("la operación no cambia nada")
	}
	out := Propuesta{ID: nuevoID(), Operacion: operacion, Resumen: resumen, Creada: p.ahora()}
	for _, c := range cambios {
		switch c.Tipo {
		case Archivo:
			r, ok := limpia(c.Ruta)
			if !ok || p.Permitido == nil || !p.Permitido(r) {
				return Propuesta{}, fmt.Errorf("%w: %s", ErrNoPermitido, c.Ruta)
			}
			c.Ruta = r
			b, err := os.ReadFile(r)
			switch {
			case err == nil:
				c.Existia, c.Antes = true, string(b)
			case errors.Is(err, os.ErrNotExist):
				c.Existia, c.Antes = false, ""
			default:
				return Propuesta{}, fmt.Errorf("no se pudo leer %s: %w", r, err)
			}
		case Orden:
			if _, ok := p.programa(c.Programa); !ok {
				return Propuesta{}, fmt.Errorf("%w: %s", ErrPrograma, c.Programa)
			}
		default:
			return Propuesta{}, fmt.Errorf("tipo de cambio desconocido: %q", c.Tipo)
		}
		out.Cambios = append(out.Cambios, c)
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.propuestas == nil {
		p.propuestas = map[string]Propuesta{}
	}
	for id, v := range p.propuestas {
		if p.ahora().Sub(v.Creada) > VidaPropuesta {
			delete(p.propuestas, id)
		}
	}
	p.propuestas[out.ID] = out
	return out, nil
}

// Aplicar hace lo que dijo la vista previa `id`, en orden, y lo anota en el historial. Si un paso falla, los anteriores
// quedan hechos y anotados (con su copia), y el error dice dónde paró.
func (p *Puerta) Aplicar(ctx context.Context, id string) (Entrada, error) {
	p.mu.Lock()
	prop, ok := p.propuestas[id]
	if ok && p.ahora().Sub(prop.Creada) > VidaPropuesta {
		ok = false
	}
	delete(p.propuestas, id)
	p.mu.Unlock()
	if !ok {
		return Entrada{}, ErrNoHay
	}
	// Antes de tocar nada: que ningún archivo haya cambiado desde la vista previa.
	for _, c := range prop.Cambios {
		if c.Tipo != Archivo {
			continue
		}
		b, err := os.ReadFile(c.Ruta)
		existe := err == nil
		if existe != c.Existia || (existe && string(b) != c.Antes) {
			return Entrada{}, fmt.Errorf("%w (%s)", ErrCambioFuera, c.Ruta)
		}
	}
	e := Entrada{ID: prop.ID, Hora: p.ahora(), Operacion: prop.Operacion, Resumen: prop.Resumen, Cambios: prop.Cambios, Copias: map[string]string{}}
	err := p.hacer(ctx, &e)
	if err != nil {
		e.Error = err.Error()
	}
	if errH := p.anotar(e); errH != nil && err == nil {
		err = errH
	}
	return e, err
}

func (p *Puerta) hacer(ctx context.Context, e *Entrada) error {
	var salidas []string
	defer func() { e.Salida = strings.Join(salidas, "\n") }()
	for _, c := range e.Cambios {
		switch c.Tipo {
		case Archivo:
			copia := ""
			if c.Existia {
				copia = filepath.Join(p.Dir, "copias", e.ID, fmt.Sprintf("%d-%s", len(e.Copias), filepath.Base(c.Ruta)))
				if err := escribir(copia, []byte(c.Antes)); err != nil {
					return fmt.Errorf("no se pudo guardar la copia de %s: %w", c.Ruta, err)
				}
			}
			e.Copias[c.Ruta] = copia
			if err := escribir(c.Ruta, []byte(c.Despues)); err != nil {
				return fmt.Errorf("no se pudo escribir %s: %w", c.Ruta, err)
			}
		case Orden:
			out, err := p.correr(ctx, c.Programa, c.Argumentos)
			if out != "" {
				salidas = append(salidas, out)
			}
			if err != nil {
				return fmt.Errorf("%s %s: %w", c.Programa, strings.Join(c.Argumentos, " "), err)
			}
		}
	}
	return nil
}

func (p *Puerta) correr(ctx context.Context, programa string, args []string) (string, error) {
	exe, ok := p.programa(programa)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrPrograma, programa)
	}
	if p.Correr != nil {
		return p.Correr(ctx, exe, args)
	}
	return CorrerSinShell(ctx, exe, args)
}

// CorrerSinShell corre un ejecutable con sus argumentos tal cual (sin intérprete de órdenes), con 60 s de límite y,
// en Windows, sin ventana.
func CorrerSinShell(ctx context.Context, exe string, args []string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, exe, args...)
	ocultar(cmd)
	var buf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &buf, &buf
	err := cmd.Run()
	return strings.TrimSpace(buf.String()), err
}

// Deshacer revierte el cambio `id` del historial: cada archivo vuelve a su copia (o se borra si no existía), y cada
// orden corre su inversa, en orden contrario. Queda anotado como una entrada nueva.
func (p *Puerta) Deshacer(ctx context.Context, id string) (Entrada, error) {
	hist, err := p.Historial(0)
	if err != nil {
		return Entrada{}, err
	}
	var orig *Entrada
	for i := range hist {
		if hist[i].ID == id {
			orig = &hist[i]
		}
		if hist[i].Deshace == id {
			return Entrada{}, ErrYaDeshecho
		}
	}
	if orig == nil || orig.Deshace != "" {
		return Entrada{}, ErrNoEnHistoria
	}
	e := Entrada{ID: nuevoID(), Hora: p.ahora(), Operacion: orig.Operacion, Resumen: "Deshacer: " + orig.Resumen, Deshace: id, Copias: map[string]string{}}
	var salidas []string
	var errD error
	for i := len(orig.Cambios) - 1; i >= 0 && errD == nil; i-- {
		c := orig.Cambios[i]
		switch c.Tipo {
		case Archivo:
			copia, hecha := orig.Copias[c.Ruta]
			if !hecha {
				continue // no llegó a escribirse
			}
			// Si alguien lo cambió después, deshacer pisaría ese trabajo: no se hace.
			if b, err := os.ReadFile(c.Ruta); err != nil || string(b) != c.Despues {
				errD = fmt.Errorf("%w (%s)", ErrCambiadoDespues, c.Ruta)
				break
			}
			inv := Cambio{Tipo: Archivo, Ruta: c.Ruta, Antes: c.Despues, Existia: true}
			if copia == "" {
				errD = os.Remove(c.Ruta)
				if errors.Is(errD, os.ErrNotExist) {
					errD = nil
				}
			} else {
				b, err := os.ReadFile(copia)
				if err != nil {
					errD = fmt.Errorf("no está la copia de %s: %w", c.Ruta, err)
					break
				}
				inv.Despues = string(b)
				errD = escribir(c.Ruta, b)
			}
			e.Cambios = append(e.Cambios, inv)
		case Orden:
			if len(c.Deshacer) == 0 {
				continue
			}
			out, err := p.correr(ctx, c.Programa, c.Deshacer)
			if out != "" {
				salidas = append(salidas, out)
			}
			e.Cambios = append(e.Cambios, Cambio{Tipo: Orden, Programa: c.Programa, Argumentos: c.Deshacer})
			errD = err
		}
	}
	e.Salida = strings.Join(salidas, "\n")
	if errD != nil {
		e.Error = errD.Error()
	}
	if err := p.anotar(e); err != nil && errD == nil {
		errD = err
	}
	return e, errD
}

// Historial: las entradas, la más nueva primero (todas con n = 0). Una que ya se deshizo lleva Deshecha.
func (p *Puerta) Historial(n int) ([]Entrada, error) {
	f, err := os.Open(filepath.Join(p.Dir, "cambios.jsonl"))
	if errors.Is(err, os.ErrNotExist) {
		return []Entrada{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var todas []Entrada
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64<<10), 16<<20)
	for sc.Scan() {
		var e Entrada
		if json.Unmarshal(sc.Bytes(), &e) == nil && e.ID != "" {
			todas = append(todas, e)
		}
	}
	deshechas := map[string]bool{}
	for _, e := range todas {
		if e.Deshace != "" && e.Error == "" {
			deshechas[e.Deshace] = true
		}
	}
	out := make([]Entrada, 0, len(todas))
	for i := len(todas) - 1; i >= 0; i-- {
		e := todas[i]
		e.Deshecha = deshechas[e.ID]
		out = append(out, e)
		if n > 0 && len(out) == n {
			break
		}
	}
	return out, sc.Err()
}

func (p *Puerta) anotar(e Entrada) error {
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(p.Dir, "cambios.jsonl"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

// escribir: un temporal en la misma carpeta que se renombra, para que nadie lea un archivo a medias.
func escribir(ruta string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return err
	}
	tmp := ruta + ".atalaya-tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, ruta); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func nuevoID() string {
	var b [6]byte
	_, _ = rand.Read(b[:])
	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(b[:])
}

// MismaRuta compara dos rutas limpias como las compara el sistema: sin distinguir mayúsculas en Windows y macOS.
func MismaRuta(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		return strings.EqualFold(a, b)
	}
	return a == b
}
