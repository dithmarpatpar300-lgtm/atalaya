// Package diario guarda lo que Atalaya sabe de cada día: un JSON por día con cada agente, las
// decisiones y las alertas de error. Claude Code borra sus transcritos a los 30 días y codex.ps1 sus trazas a los 7;
// el diario no: lo que ya estaba en un día se conserva aunque desaparezca de la memoria (Fusionar).
//
// No va en atalaya.db: esa base es el índice de la datapedia y se rehace si se borra; el diario no se puede rehacer.
package diario

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Agente de un día: las mismas cifras que Estadísticas (servidor/estadisticas.go), más lo que hace falta para
// filtrar sin los transcritos (proyecto, tipo, fase).
type Agente struct {
	ID        string    `json:"id"`
	Nombre    string    `json:"nombre"`
	Proyecto  string    `json:"proyecto"`
	Proveedor string    `json:"proveedor"` // Claude | Codex
	Tipo      string    `json:"tipo"`      // sesion | subagente | codex
	Clase     string    `json:"clase,omitempty"`
	Modelo    string    `json:"modelo,omitempty"` // familia en Claude («sonnet»), nombre en Codex
	Effort    string    `json:"effort,omitempty"`
	Fase      string    `json:"fase,omitempty"`
	Inicio    time.Time `json:"inicio"`
	Fin       time.Time `json:"fin"`
	Activo    bool      `json:"activo,omitempty"` // seguía trabajando cuando se escribió
	Minutos   float64   `json:"minutos"`
	Salida    int64     `json:"salida"`
	Nuevos    int64     `json:"nuevos"` // entrada que no vino de la caché
	Leida     int64     `json:"leida"`  // entrada leída de la caché
	// Presupuesto: los minutos de su asignación; 0 si no lo dice.
	Presupuesto float64 `json:"presupuesto,omitempty"`
	// Para la línea de tiempo: quién lo lanzó y su modelo corto («sonnet-5-5/high»).
	Padre string `json:"padre,omitempty"`
	Corto string `json:"corto,omitempty"`
	// Registro: en una llamada a Codex, su clave en REGISTRO.md, para no contarla dos veces cuando su traza se borre.
	Registro string `json:"registro,omitempty"`
}

// Decision de un día: cuándo se preguntó y cómo quedó.
type Decision struct {
	ID       string    `json:"id"`
	Proyecto string    `json:"proyecto"`
	Sesion   string    `json:"sesion"`
	Hora     time.Time `json:"hora"`
	Estado   string    `json:"estado"`
}

// Alerta de nivel error, con la primera vez que se vio.
type Alerta struct {
	ID       string    `json:"id"`
	Proyecto string    `json:"proyecto"`
	Regla    string    `json:"regla"`
	Agente   string    `json:"agente"`
	Titulo   string    `json:"titulo"`
	Hora     time.Time `json:"hora"`
}

// Dia: todo lo de una fecha local («2026-10-31»).
type Dia struct {
	Fecha      string     `json:"fecha"`
	Agentes    []Agente   `json:"agentes"`
	Decisiones []Decision `json:"decisiones"`
	Alertas    []Alerta   `json:"alertas"`
}

var reFecha = regexp.MustCompile(`^\d{4}-\d\d-\d\d$`)

// Fecha local de un instante, la llave de su día.
func Fecha(t time.Time) string { return t.Local().Format("2006-01-02") }

func ruta(dir, fecha string) string { return filepath.Join(dir, fecha+".json") }

// Leer el día; vacío (sin error) si todavía no hay archivo.
func Leer(dir, fecha string) (Dia, error) {
	d := Dia{Fecha: fecha}
	if !reFecha.MatchString(fecha) {
		return d, errors.New("la fecha no es AAAA-MM-DD: " + fecha)
	}
	b, err := os.ReadFile(ruta(dir, fecha))
	if errors.Is(err, os.ErrNotExist) {
		return d, nil
	}
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(b, &d); err != nil {
		return Dia{Fecha: fecha}, err
	}
	return d, nil
}

// Fechas con diario, de la más nueva a la más vieja.
func Fechas(dir string) []string {
	rutas, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	var out []string
	for _, r := range rutas {
		if f := filepath.Base(r)[:len(filepath.Base(r))-len(".json")]; reFecha.MatchString(f) {
			out = append(out, f)
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out)))
	return out
}

// Recortar aplica los días que el dueño elige en Configuración; el servidor lo llama una vez al día.
func Recortar(dir string, dias int, ahora time.Time) (int, error) {
	if dias <= 0 {
		return 0, nil
	}
	entradas, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	limite := ahora.Local().AddDate(0, 0, -dias).Format("2006-01-02")
	borrados := 0
	for _, entrada := range entradas {
		nombre := entrada.Name()
		if entrada.IsDir() || !strings.HasSuffix(nombre, ".json") {
			continue
		}
		fecha := strings.TrimSuffix(nombre, ".json")
		if !reFecha.MatchString(fecha) || fecha >= limite {
			continue
		}
		if _, err := time.Parse("2006-01-02", fecha); err != nil {
			continue
		}
		if err := os.Remove(filepath.Join(dir, nombre)); err != nil {
			return borrados, err
		}
		borrados++
	}
	return borrados, nil
}

// Fusionar junta lo que había en el archivo con lo que hay ahora en memoria. Lo del archivo que la memoria ya no tiene
// se queda (el transcrito se borró); lo que está en los dos, gana la memoria, salvo la primera vez que se vio una
// alerta, que es la del archivo.
func Fusionar(viejo, nuevo Dia) Dia {
	out := Dia{Fecha: nuevo.Fecha}
	if out.Fecha == "" {
		out.Fecha = viejo.Fecha
	}
	out.Agentes = unir(viejo.Agentes, nuevo.Agentes, func(a Agente) string { return a.ID }, nil)
	out.Decisiones = unir(viejo.Decisiones, nuevo.Decisiones, func(d Decision) string { return d.ID }, nil)
	out.Alertas = unir(viejo.Alertas, nuevo.Alertas, func(a Alerta) string { return a.ID }, func(v, n Alerta) Alerta {
		if !v.Hora.IsZero() && v.Hora.Before(n.Hora) {
			n.Hora = v.Hora
		}
		return n
	})
	sort.SliceStable(out.Agentes, func(i, j int) bool { return out.Agentes[i].Inicio.Before(out.Agentes[j].Inicio) })
	sort.SliceStable(out.Decisiones, func(i, j int) bool { return out.Decisiones[i].Hora.Before(out.Decisiones[j].Hora) })
	sort.SliceStable(out.Alertas, func(i, j int) bool { return out.Alertas[i].Hora.Before(out.Alertas[j].Hora) })
	return out
}

func unir[T any](viejo, nuevo []T, clave func(T) string, mezclar func(v, n T) T) []T {
	por := map[string]int{}
	out := make([]T, 0, len(viejo)+len(nuevo))
	for _, x := range viejo {
		por[clave(x)] = len(out)
		out = append(out, x)
	}
	for _, x := range nuevo {
		i, ok := por[clave(x)]
		switch {
		case !ok:
			por[clave(x)] = len(out)
			out = append(out, x)
		case mezclar != nil:
			out[i] = mezclar(out[i], x)
		default:
			out[i] = x
		}
	}
	return out
}

// Guardar fusiona d con lo que ya hay en su archivo y escribe solo si algo cambió. Escribe aparte y cambia el nombre:
// nunca queda un día a medias. Devuelve si escribió.
func Guardar(dir string, d Dia) (bool, error) {
	viejo, err := Leer(dir, d.Fecha)
	if err != nil {
		return false, err
	}
	f := Fusionar(viejo, d)
	b, err := json.MarshalIndent(f, "", " ")
	if err != nil {
		return false, err
	}
	if antes, err := os.ReadFile(ruta(dir, d.Fecha)); err == nil && bytes.Equal(antes, b) {
		return false, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, err
	}
	tmp := ruta(dir, d.Fecha) + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return false, err
	}
	return true, os.Rename(tmp, ruta(dir, d.Fecha))
}
