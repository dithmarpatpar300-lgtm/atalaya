// Package datapedia indexa la memoria de Claude Code y los documentos del centro de
// operaciones de cada proyecto en SQLite con FTS5, para buscarlos desde el panel.
//
// El índice es una copia reconstruible: si se borra, se rehace. Guarda el texto ya
// pasado por el filtro de secretos, así que lo que se muestra sale de aquí y no se
// vuelve a leer el archivo. Solo entra Markdown de carpetas conocidas: la memoria
// (~/.claude/projects/*/memory/*.md) y docs/**/*.md, AGENTS.md, CLAUDE.md y
// README.md de cada proyecto.
package datapedia

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"atalaya/internal/secretos"

	_ "modernc.org/sqlite" // SQLite sin cgo, con FTS5
)

// Proyecto cuyos documentos se indexan.
type Proyecto struct {
	Nombre string
	Raiz   string
}

// Documento: lo que se lista y se muestra.
type Documento struct {
	ID       int64  `json:"id"`
	Proyecto string `json:"proyecto"`
	Tipo     string `json:"tipo"`
	Titulo   string `json:"titulo"`
	Resumen  string `json:"resumen"`
	Ruta     string `json:"ruta"` // relativa a la raíz del proyecto, o memory/<archivo>.md
	Mtime    int64  `json:"mtime"`
	Tam      int64  `json:"tam"`
	// Fragmento: en una búsqueda, el trozo que coincide; las coincidencias van entre \x02 y \x03.
	Fragmento string `json:"fragmento,omitempty"`
	Cuerpo    string `json:"cuerpo,omitempty"`
}

// Cuenta de documentos por tipo, para los filtros.
type Cuenta struct {
	Tipo     string `json:"tipo"`
	Cantidad int    `json:"cantidad"`
}

// Indice abre la base y sabe qué carpetas recorrer.
type Indice struct {
	db        *sql.DB
	memoria   string // ~/.claude/projects
	proyectos []Proyecto
	mu        sync.Mutex // un reindexado a la vez
	// Virtuales: documentos que no son archivos (las trazas de los agentes). Se piden en cada reindexado.
	Virtuales func() []Virtual
	// Fuera indica proyectos que su dueño sacó de la datapedia: no se indexan sus documentos,
	// memoria ni trazas, y lo ya indexado sale en el siguiente reindexado. Nil no excluye ninguno.
	Fuera func(proyecto string) bool
}

// Virtual: un documento que arma el servidor, no un archivo del disco.
type Virtual struct {
	Clave    string // única y estable («traza:<agente>:<turno>»)
	Proyecto string
	Tipo     string
	Ruta     string
	Titulo   string
	Resumen  string
	Cuerpo   string
	Mtime    time.Time // si no cambia (ni el tamaño), no se vuelve a indexar
}

const esquema = `
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
CREATE TABLE IF NOT EXISTS documento (
	id       INTEGER PRIMARY KEY,
	archivo  TEXT    NOT NULL UNIQUE, -- ruta absoluta en el disco, o la clave de un documento virtual
	proyecto TEXT    NOT NULL,
	tipo     TEXT    NOT NULL,
	ruta     TEXT    NOT NULL,
	titulo   TEXT    NOT NULL,
	resumen  TEXT    NOT NULL,
	mtime    INTEGER NOT NULL,
	tam      INTEGER NOT NULL,
	sha256   TEXT    NOT NULL
);
CREATE INDEX IF NOT EXISTS documento_mtime ON documento (mtime DESC);
CREATE VIRTUAL TABLE IF NOT EXISTS texto USING fts5 (titulo, cuerpo, tokenize = 'unicode61 remove_diacritics 2');
`

// Abrir crea o abre el índice en `ruta`.
func Abrir(ruta, memoria string, proyectos []Proyecto) (*Indice, error) {
	if err := os.MkdirAll(filepath.Dir(ruta), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(ruta))
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1) // SQLite escribe de a uno; las lecturas son breves
	if _, err := db.Exec(esquema); err != nil {
		db.Close()
		return nil, fmt.Errorf("esquema de la datapedia: %w", err)
	}
	return &Indice{db: db, memoria: memoria, proyectos: append([]Proyecto(nil), proyectos...)}, nil
}

func (ix *Indice) Cerrar() error { return ix.db.Close() }

// CambiarProyectos sustituye la lista que se usa en el siguiente reindexado.
func (ix *Indice) CambiarProyectos(proyectos []Proyecto) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	ix.proyectos = append([]Proyecto(nil), proyectos...)
}

// candidato: lo que debería estar en el índice, un archivo o un documento virtual.
type candidato struct {
	archivo, proyecto, tipo, ruta string
	mtime, tam                    int64
	leer                          func() ([]byte, error)
	// Título y resumen dados (virtuales); si van vacíos, salen del texto.
	titulo, resumen string
}

// TamMaximo: un Markdown más grande no es documentación del centro, es un volcado.
const TamMaximo = 2 << 20

// Un nombre de archivo que anuncia credenciales no se indexa aunque sea Markdown («tokens.md», que cuenta tokens, sí).
var nombreProhibido = regexp.MustCompile(`(?i)(^|[._-])(env|secrets?|secretos?|credentials?|credenciales|passwords?)([._-]|$)`)

func (ix *Indice) candidatos() []candidato {
	var out []candidato
	agregar := func(archivo, proyecto, tipo, ruta string) {
		fi, err := os.Stat(archivo)
		if err != nil || !fi.Mode().IsRegular() || fi.Size() > TamMaximo || nombreProhibido.MatchString(fi.Name()) {
			return
		}
		out = append(out, candidato{archivo: archivo, proyecto: proyecto, tipo: tipo, ruta: ruta, mtime: fi.ModTime().UnixNano(), tam: fi.Size(),
			leer: func() ([]byte, error) { return os.ReadFile(archivo) }})
	}
	if ix.Virtuales != nil {
		for _, v := range ix.Virtuales() {
			cuerpo := []byte(v.Cuerpo)
			out = append(out, candidato{archivo: v.Clave, proyecto: v.Proyecto, tipo: v.Tipo, ruta: v.Ruta, mtime: v.Mtime.UnixNano(), tam: int64(len(cuerpo)),
				leer: func() ([]byte, error) { return cuerpo, nil }, titulo: v.Titulo, resumen: v.Resumen})
		}
	}
	// La memoria de cada proyecto de Claude Code.
	porCarpeta := map[string]string{}
	for _, p := range ix.proyectos {
		porCarpeta[carpetaClaude(p.Raiz)] = p.Nombre
	}
	mems, _ := filepath.Glob(filepath.Join(ix.memoria, "*", "memory", "*.md"))
	for _, m := range mems {
		carpeta := filepath.Base(filepath.Dir(filepath.Dir(m)))
		nombre := porCarpeta[carpeta]
		if nombre == "" {
			nombre = nombreDeCarpeta(carpeta)
		}
		agregar(m, nombre, "memoria", "memory/"+filepath.Base(m))
	}
	// Los documentos de cada proyecto.
	for _, p := range ix.proyectos {
		for _, f := range []string{"AGENTS.md", "CLAUDE.md", "README.md"} {
			agregar(filepath.Join(p.Raiz, f), p.Nombre, "regla", f)
		}
		docs := filepath.Join(p.Raiz, "docs")
		_ = filepath.WalkDir(docs, func(ruta string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				if n := d.Name(); n == "node_modules" || n == "target" || strings.HasPrefix(n, ".") {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.EqualFold(filepath.Ext(ruta), ".md") {
				return nil
			}
			rel, err := filepath.Rel(p.Raiz, ruta)
			if err != nil {
				return nil
			}
			rel = filepath.ToSlash(rel)
			agregar(ruta, p.Nombre, TipoDe(rel), rel)
			return nil
		})
	}
	if ix.Fuera != nil {
		filtrados := out[:0]
		for _, c := range out {
			if !ix.Fuera(c.proyecto) {
				filtrados = append(filtrados, c)
			}
		}
		out = filtrados
	}
	return out
}

// carpetaClaude: cómo nombra Claude Code la carpeta de un proyecto («C:\Proyectos\Tienda Demo» → «C--Proyectos-Tienda-Demo»).
func carpetaClaude(raiz string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return '-'
	}, raiz)
}

// nombreDeCarpeta: el nombre de un proyecto que no está en el registro, desde su carpeta de Claude Code
// («C--Proyectos-Mi-Proyecto» → «Mi-Proyecto»): sin la unidad ni la carpeta de proyectos, y sin cambiar guiones por
// espacios. Uno registrado toma su nombre exacto del registro (porCarpeta).
func nombreDeCarpeta(c string) string {
	if len(c) >= 3 && ((c[0] >= 'A' && c[0] <= 'Z') || (c[0] >= 'a' && c[0] <= 'z')) && c[1:3] == "--" {
		c = c[3:]
	}
	for _, marca := range []string{"Projects-", "Proyectos-"} {
		if strings.HasPrefix(c, marca) && len(c) > len(marca) {
			return c[len(marca):]
		}
	}
	return c
}

// TipoDe clasifica un documento por su carpeta.
func TipoDe(rel string) string {
	r := strings.ToLower(rel)
	switch {
	case strings.HasPrefix(r, "docs/archivo/"):
		return "archivo"
	case strings.HasPrefix(r, "docs/adr/"):
		return "adr"
	case strings.HasPrefix(r, "docs/operacion/"):
		return "regla"
	case strings.HasPrefix(r, "docs/planes/"):
		return "plan"
	case strings.HasPrefix(r, "docs/trazabilidad/"):
		switch {
		case strings.Contains(r, "/avance/"):
			return "bitacora"
		case strings.Contains(r, "/temporales/"):
			return "acta"
		case strings.Contains(r, "/codex/"):
			return "codex"
		}
		return "informe"
	case strings.HasPrefix(r, "docs/auditoria/"):
		return "informe"
	case strings.HasPrefix(r, "docs/estudios/"):
		return "estudio"
	}
	return "documento"
}

var (
	reFrontmatter = regexp.MustCompile(`(?s)^---\r?\n(.*?)\r?\n---\r?\n?`)
	reCampo       = regexp.MustCompile(`(?m)^\s*(name|description|title):\s*(.+?)\s*$`)
	reTitulo      = regexp.MustCompile(`(?m)^#\s+(.+?)\s*$`)
)

// Encabezado saca título y resumen: el frontmatter (memoria) o el primer «# » y el primer párrafo.
func Encabezado(texto, archivo string) (titulo, resumen string) {
	cuerpo := texto
	if m := reFrontmatter.FindStringSubmatch(texto); m != nil {
		cuerpo = texto[len(m[0]):]
		for _, c := range reCampo.FindAllStringSubmatch(m[1], -1) {
			switch c[1] {
			case "name", "title":
				if titulo == "" {
					titulo = strings.Trim(c[2], `"'`)
				}
			case "description":
				resumen = strings.Trim(c[2], `"'`)
			}
		}
	}
	if t := reTitulo.FindStringSubmatch(cuerpo); t != nil {
		titulo = strings.TrimSpace(t[1])
	}
	if titulo == "" {
		titulo = strings.TrimSuffix(filepath.Base(archivo), filepath.Ext(archivo))
	}
	if resumen == "" {
		for _, l := range strings.Split(strings.ReplaceAll(cuerpo, "\r\n", "\n"), "\n") {
			l = strings.TrimSpace(l)
			if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "|") || strings.HasPrefix(l, "```") || strings.HasPrefix(l, "---") || strings.HasPrefix(l, ">") {
				continue
			}
			resumen = strings.TrimLeft(l, "-*0123456789. ")
			break
		}
	}
	return recortar(titulo, 160), recortar(resumen, 240)
}

func recortar(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

// Resultado de un reindexado.
type Resultado struct {
	Nuevos, Cambiados, Borrados, Total int
	Duracion                           time.Duration
}

func (r Resultado) Cambio() bool { return r.Nuevos+r.Cambiados+r.Borrados > 0 }

type fila struct {
	id         int64
	mtime, tam int64
	sha        string
	proyecto   string
}

// Reindexar recorre las carpetas y pone el índice al día: lo que no cambia de fecha, tamaño ni proyecto no se lee.
func (ix *Indice) Reindexar() (Resultado, error) {
	ix.mu.Lock()
	defer ix.mu.Unlock()
	inicio := time.Now()
	var res Resultado

	existentes := map[string]fila{}
	filas, err := ix.db.Query(`SELECT id, archivo, mtime, tam, sha256, proyecto FROM documento`)
	if err != nil {
		return res, err
	}
	for filas.Next() {
		var f fila
		var archivo string
		if err := filas.Scan(&f.id, &archivo, &f.mtime, &f.tam, &f.sha, &f.proyecto); err != nil {
			filas.Close()
			return res, err
		}
		existentes[archivo] = f
	}
	filas.Close()

	tx, err := ix.db.Begin()
	if err != nil {
		return res, err
	}
	defer tx.Rollback()
	vistos := map[string]bool{}
	for _, c := range ix.candidatos() {
		vistos[c.archivo] = true
		mtime, tam := c.mtime, c.tam
		viejo, hay := existentes[c.archivo]
		if hay && viejo.mtime == mtime && viejo.tam == tam && viejo.proyecto == c.proyecto {
			continue
		}
		b, err := c.leer()
		if err != nil {
			continue
		}
		suma := sha256.Sum256(b)
		sha := hex.EncodeToString(suma[:])
		if hay && viejo.sha == sha && viejo.proyecto == c.proyecto {
			if _, err := tx.Exec(`UPDATE documento SET mtime = ?, tam = ? WHERE id = ?`, mtime, tam, viejo.id); err != nil {
				return res, err
			}
			continue
		}
		texto := secretos.Tapar(strings.ToValidUTF8(string(b), "�"))
		titulo, resumen := Encabezado(texto, c.archivo)
		if c.titulo != "" {
			titulo, resumen = c.titulo, c.resumen
		}
		if hay {
			if _, err := tx.Exec(`UPDATE documento SET proyecto=?, tipo=?, ruta=?, titulo=?, resumen=?, mtime=?, tam=?, sha256=? WHERE id=?`,
				c.proyecto, c.tipo, c.ruta, titulo, resumen, mtime, tam, sha, viejo.id); err != nil {
				return res, err
			}
			if _, err := tx.Exec(`DELETE FROM texto WHERE rowid = ?`, viejo.id); err != nil {
				return res, err
			}
			if _, err := tx.Exec(`INSERT INTO texto (rowid, titulo, cuerpo) VALUES (?, ?, ?)`, viejo.id, titulo, texto); err != nil {
				return res, err
			}
			res.Cambiados++
			continue
		}
		r, err := tx.Exec(`INSERT INTO documento (archivo, proyecto, tipo, ruta, titulo, resumen, mtime, tam, sha256) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			c.archivo, c.proyecto, c.tipo, c.ruta, titulo, resumen, mtime, tam, sha)
		if err != nil {
			return res, err
		}
		id, _ := r.LastInsertId()
		if _, err := tx.Exec(`INSERT INTO texto (rowid, titulo, cuerpo) VALUES (?, ?, ?)`, id, titulo, texto); err != nil {
			return res, err
		}
		res.Nuevos++
	}
	for archivo, f := range existentes {
		if vistos[archivo] {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM documento WHERE id = ?`, f.id); err != nil {
			return res, err
		}
		if _, err := tx.Exec(`DELETE FROM texto WHERE rowid = ?`, f.id); err != nil {
			return res, err
		}
		res.Borrados++
	}
	if err := tx.Commit(); err != nil {
		return res, err
	}
	res.Total = len(vistos)
	res.Duracion = time.Since(inicio)
	return res, nil
}

// Consulta convierte lo que escribe la persona en una consulta FTS5 segura: cada palabra entre comillas
// (así «AND», «-» o «"» no son operadores), todas obligatorias, y la última como prefijo mientras se escribe.
func Consulta(q string) string {
	palabras := strings.FieldsFunc(q, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' })
	if len(palabras) == 0 {
		return ""
	}
	partes := make([]string, len(palabras))
	for i, p := range palabras {
		partes[i] = `"` + p + `"`
	}
	partes[len(partes)-1] += "*"
	return strings.Join(partes, " ")
}

// Filtro de una búsqueda.
type Filtro struct {
	Texto    string
	Tipo     string
	Proyecto string
	Limite   int
}

// Buscar devuelve los documentos que coinciden, los mejores primero; sin texto, los más recientes.
func (ix *Indice) Buscar(f Filtro) ([]Documento, error) {
	if f.Limite <= 0 || f.Limite > 200 {
		f.Limite = 50
	}
	var (
		args  []any
		donde []string
		q     string
	)
	fts := Consulta(f.Texto)
	if fts != "" {
		// bm25 con más peso al título; el fragmento sale del cuerpo (columna 1).
		q = `SELECT d.id, d.proyecto, d.tipo, d.titulo, d.resumen, d.ruta, d.mtime, d.tam,
		            snippet(texto, 1, char(2), char(3), ' … ', 14)
		     FROM texto JOIN documento d ON d.id = texto.rowid
		     WHERE texto MATCH ?`
		args = append(args, fts)
	} else {
		q = `SELECT d.id, d.proyecto, d.tipo, d.titulo, d.resumen, d.ruta, d.mtime, d.tam, '' FROM documento d WHERE 1 = 1`
	}
	if f.Tipo != "" {
		donde = append(donde, "d.tipo = ?")
		args = append(args, f.Tipo)
	}
	if f.Proyecto != "" {
		donde = append(donde, "d.proyecto = ?")
		args = append(args, f.Proyecto)
	}
	for _, d := range donde {
		q += " AND " + d
	}
	if fts != "" {
		q += ` ORDER BY bm25(texto, 5.0, 1.0)`
	} else {
		q += ` ORDER BY d.mtime DESC`
	}
	q += ` LIMIT ?`
	args = append(args, f.Limite)
	filas, err := ix.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := []Documento{}
	for filas.Next() {
		var d Documento
		if err := filas.Scan(&d.ID, &d.Proyecto, &d.Tipo, &d.Titulo, &d.Resumen, &d.Ruta, &d.Mtime, &d.Tam, &d.Fragmento); err != nil {
			return nil, err
		}
		d.Mtime /= int64(time.Second)
		out = append(out, d)
	}
	return out, filas.Err()
}

// Cuentas por tipo (con el mismo texto y proyecto de la búsqueda, sin el filtro de tipo).
func (ix *Indice) Cuentas(texto, proyecto string) ([]Cuenta, error) {
	var (
		q    string
		args []any
	)
	if fts := Consulta(texto); fts != "" {
		q = `SELECT d.tipo, count(*) FROM texto JOIN documento d ON d.id = texto.rowid WHERE texto MATCH ?`
		args = append(args, fts)
	} else {
		q = `SELECT d.tipo, count(*) FROM documento d WHERE 1 = 1`
	}
	if proyecto != "" {
		q += ` AND d.proyecto = ?`
		args = append(args, proyecto)
	}
	q += ` GROUP BY d.tipo`
	filas, err := ix.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer filas.Close()
	out := []Cuenta{}
	for filas.Next() {
		var c Cuenta
		if err := filas.Scan(&c.Tipo, &c.Cantidad); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Cantidad > out[j].Cantidad })
	return out, filas.Err()
}

// ErrNoExiste: no hay documento con ese id.
var ErrNoExiste = errors.New("no existe ese documento")

// Documento devuelve uno con su texto (ya filtrado).
func (ix *Indice) Documento(id int64) (Documento, error) {
	var d Documento
	err := ix.db.QueryRow(`SELECT d.id, d.proyecto, d.tipo, d.titulo, d.resumen, d.ruta, d.mtime, d.tam, t.cuerpo
	                       FROM documento d JOIN texto t ON t.rowid = d.id WHERE d.id = ?`, id).
		Scan(&d.ID, &d.Proyecto, &d.Tipo, &d.Titulo, &d.Resumen, &d.Ruta, &d.Mtime, &d.Tam, &d.Cuerpo)
	if errors.Is(err, sql.ErrNoRows) {
		return d, ErrNoExiste
	}
	d.Mtime /= int64(time.Second)
	return d, err
}

// PorRuta busca el id de un documento por proyecto y ruta (para abrir una bitácora o un informe desde un agente).
func (ix *Indice) PorRuta(proyecto, ruta string) (int64, error) {
	var id int64
	err := ix.db.QueryRow(`SELECT id FROM documento WHERE proyecto = ? AND ruta = ?`, proyecto, ruta).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNoExiste
	}
	return id, err
}
