package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"atalaya/internal/secretos"
)

// El registro de las consultas a Atalaya rehecho desde los transcritos (servidor/mcp.go, «Consultas»): el registro en
// memoria se vacía cada vez que se reinicia atalaya.exe, pero cada llamada sigue en el transcrito del agente que la hizo
// como un paso «Atalaya» (`mcp__atalaya__<herramienta>`), con su entrada en el tool_use y la respuesta entera en el
// tool_result. Las llamadas de Codex en vivo no están aquí (codexvivo importa este paquete): las suma el servidor.

// PrefijoAtalaya: el nombre de las herramientas de Atalaya en Claude Code (la clave «atalaya» de .mcp.json).
const PrefijoAtalaya = "mcp__atalaya__"

// prefijoAnterior: la clave con que se instalaba antes («centro»). Los transcritos viejos y los .mcp.json ya
// instalados la siguen usando, y sus consultas también cuentan.
const prefijoAnterior = "mcp__centro__"

// HerramientaAtalaya: si herr es una herramienta de Atalaya, su nombre sin el prefijo («que_hizo»).
func HerramientaAtalaya(herr string) (string, bool) {
	for _, p := range []string{PrefijoAtalaya, prefijoAnterior} {
		if h, ok := strings.CutPrefix(herr, p); ok {
			return h, true
		}
	}
	return "", false
}

// EsServidorAtalaya: el nombre del servidor MCP en la traza de Codex («atalaya», o «centro» en las de antes).
func EsServidorAtalaya(nombre string) bool { return nombre == "atalaya" || nombre == "centro" }

// Topes de una llamada leída del transcrito: los mismos del registro en memoria.
const (
	MaxArgumentosCentro = 300  // runas
	MaxRespuestaCentro  = 1200 // runas
)

// LlamadaCentro: una llamada a una herramienta de Atalaya, tal como la guarda el transcrito.
type LlamadaCentro struct {
	Agente              Agente
	Paso                Paso   // el paso «Atalaya»: hora, error, número
	Herramienta         string // sin el prefijo: «que_hizo»
	Argumentos          string // la entrada en JSON compacto, con los secretos tapados
	ArgumentosConocidos bool   // distingue una entrada vacia de una que no se pudo leer
	Respuesta           string // el texto del tool_result, recortado; vacío si todavía no llegó
	Bytes               int    // el tamaño de la respuesta entera (≈ TokEntra × 4 si no se pudo releer)
	BytesConocidos      bool   // Bytes salió del tool_result, no de la estimación
	ID                  string // el tool_use id
	// Leidas: las rutas enteras que el mismo agente leyó (pasos «Lee») en sus VentanaPasos pasos siguientes: el
	// ahorro de una consulta no cuenta si después igual leyó el documento.
	Leidas []string
}

// VentanaPasos: cuántos pasos del mismo agente, antes o después de uno, se miran para el ahorro de las consultas.
const VentanaPasos = 10

// Lectura: un paso «Lee» con la ruta entera que pidió.
type Lectura struct {
	Agente Agente
	Paso   Paso
	Ruta   string
	// AtalayaAntes: las herramientas de Atalaya (sin prefijo) del mismo agente en sus VentanaPasos pasos anteriores.
	AtalayaAntes []string
}

// pasoEn: un paso con el transcrito del que se relee.
type pasoEn struct {
	archivo string
	p       Paso
}

// siguientesLee: los pasos «Lee» entre i+1 e i+VentanaPasos.
func siguientesLee(a *archivo, i int) []pasoEn {
	var out []pasoEn
	for j := i + 1; j < len(a.pasos) && j <= i+VentanaPasos; j++ {
		if a.pasos[j].Tipo == "Lee" {
			out = append(out, pasoEn{a.ruta, *a.pasos[j]})
		}
	}
	return out
}

// cacheRutas: la ruta que pidió cada paso «Lee» (archivo + tool_use id). Un tool_use no cambia nunca, así que no
// vence; solo se vacía al pasar del tope.
var cacheRutas = struct {
	sync.Mutex
	m map[string]string
}{m: map[string]string{}}

const maxCacheRutas = 50000

// rutasLeidas devuelve la ruta entera de cada paso «Lee» (vacía si no se pudo releer): Hilo la trae cortada
// («docs/…»), sin la raíz del proyecto, y el ahorro compara rutas enteras. Cada archivo se abre una vez.
func rutasLeidas(ps []pasoEn) []string {
	out := make([]string, len(ps))
	porArchivo := map[string][]int{}
	cacheRutas.Lock()
	for i, x := range ps {
		if r, ok := cacheRutas.m[x.archivo+"\x00"+x.p.idUso]; ok && x.p.idUso != "" {
			out[i] = r
			continue
		}
		porArchivo[x.archivo] = append(porArchivo[x.archivo], i)
	}
	cacheRutas.Unlock()
	nuevas := map[string]string{}
	for ruta, is := range porArchivo {
		f, err := os.Open(ruta)
		if err != nil {
			continue
		}
		for _, i := range is {
			out[i] = rutaDelUso(f, &ps[i].p)
			if out[i] != "" && ps[i].p.idUso != "" {
				nuevas[ruta+"\x00"+ps[i].p.idUso] = out[i]
			}
		}
		f.Close()
	}
	cacheRutas.Lock()
	if len(cacheRutas.m)+len(nuevas) > maxCacheRutas {
		clear(cacheRutas.m)
	}
	for k, v := range nuevas {
		cacheRutas.m[k] = v
	}
	cacheRutas.Unlock()
	return out
}

// rutaDelUso: el file_path (o notebook_path) del tool_use de un paso.
func rutaDelUso(f *os.File, p *Paso) string {
	b, err := lineaDe(f, p.ofsUso)
	if err != nil {
		return ""
	}
	var l linea
	if json.Unmarshal(b, &l) != nil || l.Message == nil {
		return ""
	}
	_, bs := bloques(l.Message.Content)
	if p.bloque < 0 || p.bloque >= len(bs) {
		return ""
	}
	var m struct {
		Archivo  string `json:"file_path"`
		Cuaderno string `json:"notebook_path"`
	}
	_ = json.Unmarshal(bs[p.bloque].Input, &m)
	if m.Archivo != "" {
		return m.Archivo
	}
	return m.Cuaderno
}

// Lecturas devuelve los pasos «Lee» desde `desde` cuyo Hilo acepta `quiere` (se filtra por el Hilo antes de
// releer la ruta del archivo), con las consultas a Atalaya de sus pasos anteriores. Como en ConsultasCentro, la
// copia de una sesión retomada no repite un paso.
func (ix *Indice) Lecturas(desde time.Time, quiere func(hilo string) bool) []Lectura {
	var ls []Lectura
	var ps []pasoEn
	visto := map[string]int{}
	ix.mu.RLock()
	for _, a := range ix.arch {
		if a.agente.ID == "" || (!desde.IsZero() && a.agente.Ultimo.Before(desde)) {
			continue
		}
		for i, p := range a.pasos {
			if p.Tipo != "Lee" || (!desde.IsZero() && p.Hora.Before(desde)) || !quiere(p.Hilo) {
				continue
			}
			l := Lectura{Agente: ix.conDelegacion(*a.agente), Paso: *p}
			for j := max(0, i-VentanaPasos); j < i; j++ {
				if h, ok := HerramientaAtalaya(a.pasos[j].Herramienta); ok {
					l.AtalayaAntes = append(l.AtalayaAntes, h)
				}
			}
			if p.idUso != "" {
				if k, ok := visto[p.idUso]; ok {
					otro := ls[k].Agente
					if a.agente.Inicio.Before(otro.Inicio) || (a.agente.Inicio.Equal(otro.Inicio) && a.agente.ID < otro.ID) {
						ls[k], ps[k] = l, pasoEn{a.ruta, *p}
					}
					continue
				}
				visto[p.idUso] = len(ls)
			}
			ls = append(ls, l)
			ps = append(ps, pasoEn{a.ruta, *p})
		}
	}
	ix.mu.RUnlock()
	for i, r := range rutasLeidas(ps) {
		ls[i].Ruta = r
	}
	sort.SliceStable(ls, func(i, j int) bool { return ls[i].Paso.Hora.After(ls[j].Paso.Hora) })
	return ls
}

// datosCentro: lo que se relee del archivo, guardado por tool_use id para no volver a leerlo en cada pedido.
type datosCentro struct {
	args, resp    string
	bytes         int
	conResultado  bool
	conArgumentos bool
}

var cacheCentro = struct {
	sync.Mutex
	m map[string]datosCentro
}{m: map[string]datosCentro{}}

// ConsultasCentro devuelve las llamadas a Atalaya de los transcritos con actividad desde `desde`, de la más reciente
// a la más vieja, hasta `tope` (0: todas). Una sesión retomada copia la historia de la anterior en su archivo: la
// misma llamada (el mismo tool_use id) sale una sola vez.
func (ix *Indice) ConsultasCentro(desde time.Time, tope int) []LlamadaCentro {
	type hallada struct {
		ruta string
		l    LlamadaCentro
		lee  []pasoEn // sus pasos «Lee» siguientes
	}
	// Bajo el candado solo se copian los pasos; los archivos se leen después, como en Detalle.
	ix.mu.RLock()
	var hs []hallada
	visto := map[string]int{} // tool_use id -> índice en hs
	for _, a := range ix.arch {
		if a.agente.ID == "" || (!desde.IsZero() && a.agente.Ultimo.Before(desde)) {
			continue
		}
		for i, p := range a.pasos {
			herr, ok := HerramientaAtalaya(p.Herramienta)
			if !ok || (!desde.IsZero() && p.Hora.Before(desde)) {
				continue
			}
			h := hallada{a.ruta, LlamadaCentro{Agente: ix.conDelegacion(*a.agente), Paso: *p, Herramienta: herr, ID: p.idUso}, siguientesLee(a, i)}
			if p.idUso != "" {
				// La copia de una sesión retomada empieza después: la llamada es de la sesión que la hizo.
				if i, ok := visto[p.idUso]; ok {
					otro := hs[i].l.Agente
					if a.agente.Inicio.Before(otro.Inicio) || (a.agente.Inicio.Equal(otro.Inicio) && a.agente.ID < otro.ID) {
						hs[i] = h
					}
					continue
				}
				visto[p.idUso] = len(hs)
			}
			hs = append(hs, h)
		}
	}
	ix.mu.RUnlock()

	sort.SliceStable(hs, func(i, j int) bool { return hs[i].l.Paso.Hora.After(hs[j].l.Paso.Hora) })
	if tope > 0 && len(hs) > tope {
		hs = hs[:tope]
	}

	// Lo que ya se leyó en otro pedido; lo demás se lee abriendo cada archivo una vez.
	cacheCentro.Lock()
	viejo := cacheCentro.m
	cacheCentro.Unlock()
	nuevo := make(map[string]datosCentro, len(hs))
	porArchivo := map[string][]int{}
	for i, h := range hs {
		clave := h.ruta + "\x00" + h.l.ID
		if d, ok := viejo[clave]; ok && h.l.ID != "" && (d.conResultado || h.l.Paso.ofsRes < 0) {
			nuevo[clave] = d
			continue
		}
		porArchivo[h.ruta] = append(porArchivo[h.ruta], i)
	}
	for ruta, is := range porArchivo {
		f, err := os.Open(ruta)
		if err != nil {
			continue
		}
		for _, i := range is {
			nuevo[ruta+"\x00"+hs[i].l.ID] = releerCentro(f, &hs[i].l.Paso)
		}
		f.Close()
	}
	// La caché queda solo con lo de este pedido: no crece más allá del tope.
	cacheCentro.Lock()
	cacheCentro.m = nuevo
	cacheCentro.Unlock()

	// Las rutas de los pasos «Lee» siguientes de todas las llamadas, en una sola pasada por los archivos.
	var lee []pasoEn
	for _, h := range hs {
		lee = append(lee, h.lee...)
	}
	rutas := rutasLeidas(lee)

	out := make([]LlamadaCentro, len(hs))
	k := 0
	for i, h := range hs {
		l := h.l
		d := nuevo[h.ruta+"\x00"+l.ID]
		l.Argumentos, l.Respuesta, l.Bytes = d.args, d.resp, d.bytes
		l.ArgumentosConocidos, l.BytesConocidos = d.conArgumentos, d.conResultado
		for range h.lee {
			if rutas[k] != "" {
				l.Leidas = append(l.Leidas, rutas[k])
			}
			k++
		}
		if !d.conResultado && l.Bytes == 0 {
			l.Bytes = int(l.Paso.TokEntra * 4)
		}
		out[i] = l
	}
	return out
}

// releerCentro lee del archivo la entrada del tool_use y el texto de su tool_result.
func releerCentro(f *os.File, p *Paso) datosCentro {
	var d datosCentro
	if b, err := lineaDe(f, p.ofsUso); err == nil {
		var l linea
		if json.Unmarshal(b, &l) == nil && l.Message != nil {
			if _, bs := bloques(l.Message.Content); p.bloque >= 0 && p.bloque < len(bs) {
				var buf bytes.Buffer
				if json.Compact(&buf, bs[p.bloque].Input) == nil {
					d.args = recortar(secretos.Tapar(buf.String()), MaxArgumentosCentro)
					d.conArgumentos = true
				}
			}
		}
	}
	if d.args == "{}" {
		d.args = ""
	}
	if p.ofsRes < 0 {
		return d
	}
	b, err := lineaDe(f, p.ofsRes)
	if err != nil {
		return d
	}
	var lr struct {
		Message *mensaje `json:"message"`
	}
	if json.Unmarshal(b, &lr) != nil || lr.Message == nil {
		return d
	}
	_, rs := bloques(lr.Message.Content)
	for _, r := range rs {
		if r.Type == "tool_result" && r.ToolUseID == p.idUso {
			t := textoResultado(r.Content)
			d.bytes, d.resp, d.conResultado = len(t), recortar(secretos.Tapar(t), MaxRespuestaCentro), true
		}
	}
	return d
}

// lineaDe lee la línea que empieza en ofs de un archivo ya abierto.
func lineaDe(f *os.File, ofs int64) ([]byte, error) {
	if _, err := f.Seek(ofs, io.SeekStart); err != nil {
		return nil, err
	}
	b, err := bufio.NewReaderSize(f, 64*1024).ReadBytes('\n')
	if err != nil && len(b) == 0 {
		return nil, err
	}
	return b, nil
}
