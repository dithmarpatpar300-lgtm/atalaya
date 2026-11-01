package servidor

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"atalaya/internal/alertas"
	"atalaya/internal/avance"
	"atalaya/internal/claude"
	"atalaya/internal/proceso"
	"atalaya/internal/trazabilidad"
)

// La salud del proceso: cada subagente y cada llamada a Codex contra las reglas de
// AGENTS.md (internal/proceso), con la ruta que prueba cada falta, y los temporales que siguen en el disco. Se arma
// al pedirla; en cada vuelta solo se mira si cambiaron las carpetas temporales, para avisar por SSE.

var (
	reFaseRuta = regexp.MustCompile(`docs/trazabilidad/(F\d+)/`)
	reFechaRol = regexp.MustCompile(`^\d{4}-\d\d-\d\d-`)
)

// Periodos.
const (
	PeriodoHoy  = "hoy"
	PeriodoFase = "fase"
	PeriodoTodo = "todo"
)

type respuestaProceso struct {
	proceso.Informe
	Periodo string `json:"periodo"`
	Fase    string `json:"fase,omitempty"` // la fase en curso del proyecto
	Hora    int64  `json:"hora"`
}

func (s *Servidor) verProceso(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	periodo := q.Get("periodo")
	if periodo != PeriodoFase && periodo != PeriodoTodo {
		periodo = PeriodoHoy
	}
	escribir(w, s.informeProceso(q.Get("proyecto"), periodo))
}

func (s *Servidor) proyectoOMayor(proyecto string) string {
	if proyecto == "" && len(s.listaProyectos()) > 0 {
		return s.listaProyectos()[0].Nombre
	}
	return proyecto
}

func (s *Servidor) informeProceso(proyecto, periodo string) respuestaProceso {
	proyecto = s.proyectoOMayor(proyecto)
	raiz := s.raizDe(proyecto)
	reglas := s.reglas.Reglas(raiz)
	lugares := proceso.Lugares(raiz, reglas)
	ahora := s.ahora()

	ags := append(s.subagentesProceso(proyecto, ahora), s.codexProceso(proyecto, raiz, ahora)...)
	carpetas := proceso.Escanear(lugares)
	for i, c := range carpetas {
		if actas, _ := filepath.Glob(filepath.Join(raiz, "docs", "trazabilidad", "F*", "temporales", c.Nombre+".md")); len(actas) > 0 {
			if rel, err := filepath.Rel(raiz, actas[0]); err == nil {
				carpetas[i].Acta = filepath.ToSlash(rel)
				carpetas[i].ActaEstado = estadoActa(raiz, carpetas[i].Acta)
			}
		}
	}
	// Se evalúan todos, para que una carpeta de ayer encuentre a su agente; el periodo filtra las filas después.
	filas, carpetas := proceso.Evaluar(reglas, ags, carpetas)
	fase := faseActual(raiz)
	hoy := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, ahora.Location())
	out := respuestaProceso{Periodo: periodo, Fase: fase, Hora: ahora.Unix(),
		Informe: proceso.Informe{Reglas: proceso.Columnas(reglas), Lugares: lugares, Filas: []proceso.Fila{}, Temporales: carpetas}}
	if out.Lugares == nil {
		out.Lugares = []string{}
	}
	if out.Temporales == nil {
		out.Temporales = []proceso.Carpeta{}
	}
	for _, f := range filas {
		switch {
		case periodo == PeriodoHoy && !f.Activo && f.Inicio.Before(hoy) && f.Ultimo.Before(hoy):
			continue
		case periodo == PeriodoFase && f.Fase != fase:
			continue
		}
		out.Filas = append(out.Filas, f)
	}
	return out
}

// esDelProyecto: la raíz de una llamada a Codex es la del proyecto (por su nombre de carpeta o por la ruta entera).
func esDelProyecto(raizLlamada, proyecto, raiz string) bool {
	return filepath.Base(raizLlamada) == proyecto || strings.EqualFold(filepath.Clean(raizLlamada), filepath.Clean(raiz))
}

// faseActual: la fase en curso: la de AVANCE.md más alta (avance.Actual), la misma que dice `estado`; sin
// AVANCE.md, la carpeta docs/trazabilidad/F<n> con el número más alto.
func faseActual(raiz string) string {
	if f := avance.Actual(raiz); f != nil {
		return f.Fase
	}
	rutas, _ := filepath.Glob(filepath.Join(raiz, "docs", "trazabilidad", "F*"))
	mejor, n := "", -1
	for _, r := range rutas {
		b := filepath.Base(r)
		if v, err := strconv.Atoi(strings.TrimPrefix(b, "F")); err == nil && v > n {
			mejor, n = b, v
		}
	}
	return mejor
}

// corta la carpeta de la persona en «~», para que las rutas de los transcritos quepan.
func casa(ruta string) string {
	if h, err := os.UserHomeDir(); err == nil && h != "" && strings.HasPrefix(strings.ToLower(ruta), strings.ToLower(h)) {
		return "~" + ruta[len(h):]
	}
	return ruta
}

// idDeTranscrito: el id de la sesión cuyo transcrito es esa ruta (su nombre sin «.jsonl»).
func idDeTranscrito(ruta string) string {
	if ruta == "" {
		return ""
	}
	return strings.TrimSuffix(filepath.Base(ruta), ".jsonl")
}

// iniciosFase: cuándo empezó cada fase (raíz y fase → la hora del archivo más viejo de su carpeta). No cambia.
var iniciosFase sync.Map

// faseDeAgente: la fase de un agente por sus documentos (docs/trazabilidad/F<n>/…); sin ninguno, la fase en curso
// si el agente empezó después de que ella empezara. Así «periodo: fase» no queda vacío en un proyecto cuyos agentes no
// escriben documentos (paquete P4, hecho por Codex con -Escribe y revisado por el coordinador).

func faseDeAgente(raiz string, inicio time.Time, rutas ...string) string {
	for _, r := range rutas {
		if m := reFaseRuta.FindStringSubmatch(r); m != nil {
			return m[1]
		}
	}
	fase := faseActual(raiz)
	if fase == "" {
		return ""
	}
	clave := struct{ raiz, fase string }{raiz, fase}
	valor, ok := iniciosFase.Load(clave)
	if !ok {
		var primero time.Time
		carpeta := filepath.Join(raiz, "docs", "trazabilidad", fase)
		archivos, err := os.ReadDir(carpeta)
		if err != nil {
			return ""
		}
		for _, archivo := range archivos {
			if archivo.IsDir() {
				continue
			}
			if info, err := archivo.Info(); err == nil && (primero.IsZero() || info.ModTime().Before(primero)) {
				primero = info.ModTime()
			}
		}
		if primero.IsZero() {
			return ""
		}
		valor, _ = iniciosFase.LoadOrStore(clave, primero)
	}
	if inicio.After(valor.(time.Time)) {
		return fase
	}
	return ""
}

// rutaFase: la fase de una llamada a Codex («F5») como ruta, para pasarla por faseDeAgente.
func rutaFase(fase string) string {
	if fase == "" {
		return ""
	}
	return "docs/trazabilidad/" + fase + "/"
}

func claseCorta(clase, modelo string) string {
	switch {
	case clase != "" && modelo != "":
		return clase + " · " + modelo
	case modelo != "":
		return modelo
	}
	return clase
}

// crucesPorAgente: los cruces de archivos (alertas.Cruces) de los agentes del proyecto, con todo lo leído, por id.
func (s *Servidor) crucesPorAgente(proyecto string, ahora time.Time) map[string][]proceso.Cruce {
	huellas := s.Claude.Huellas(time.Time{})
	var ags []alertas.Agente
	nombre := map[string]string{}
	for _, a := range s.Claude.Agentes() {
		if a.Proyecto != proyecto {
			continue
		}
		n := s.nombreDe(a)
		nombre[a.ID] = n
		ags = append(ags, alertas.Agente{ID: a.ID, Nombre: n, Proyecto: a.Proyecto, Activo: activo(EstadoClaude(a, ahora)), Inicio: a.Inicio,
			Ultimo: a.Ultimo, Ediciones: huellas[a.ID].Ediciones, Padre: a.Padre})
	}
	out := map[string][]proceso.Cruce{}
	for _, c := range alertas.Cruces(ags) {
		abierto := c.Fin.IsZero()
		out[c.A.ID] = append(out[c.A.ID], proceso.Cruce{Otro: nombre[c.B.ID], OtroID: c.B.ID, Ruta: c.Ruta, Hora: c.HoraA, HoraOtro: c.HoraB, Coordinador: c.Coordinador, Abierto: abierto})
		out[c.B.ID] = append(out[c.B.ID], proceso.Cruce{Otro: nombre[c.A.ID], OtroID: c.A.ID, Ruta: c.Ruta, Hora: c.HoraB, HoraOtro: c.HoraA, Coordinador: c.Coordinador, Abierto: abierto})
	}
	return out
}

func (s *Servidor) subagentesProceso(proyecto string, ahora time.Time) []proceso.Agente {
	revs := s.Claude.Revisiones(time.Time{})
	cruces := s.crucesPorAgente(proyecto, ahora)
	var out []proceso.Agente
	for _, a := range s.Claude.Agentes() {
		if a.Tipo != "subagente" || a.Proyecto != proyecto {
			continue
		}
		// La vista ya trae el nombre, la bitácora (la del encargo, la de su rol o la de su carpeta temporal), su
		// avance y el acta (buscarDocs); aquí solo se suma lo que pide la salud.
		v := s.vistaClaude(a, true)
		d := s.docsDe(a)
		rev := revs[a.ID]
		sesion := a.Padre
		if sesion == "" {
			sesion = idDeTranscrito(rev.Sesion)
		}
		p := proceso.Agente{ID: a.ID, Nombre: v.Nombre, Proveedor: "Claude", Clase: claseDeVista(v), Tarea: a.Tarea,
			Estado: v.Estado, Activo: activo(v.Estado), Inicio: a.Inicio, Ultimo: a.Ultimo, Fase: faseDeAgente(s.raizDe(proyecto), a.Inicio, v.Bitacora, v.Acta, v.Informe, a.Encargo),
			LineaAsignacion: rev.LineaAsignacion, Nivel: a.Nivel, ClaseAsignada: a.ClaseDetalle, Rol: a.RolAsignado, Modelo: a.ModeloAsignado,
			Presupuesto: a.Presupuesto, Base: a.Base, CuotaAsignada: a.Cuota, Encargo: casa(rev.Transcrito), EffortLlamada: a.Effort,
			Lanzado: rev.Lanzado, LanzadoEn: casa(rev.Sesion), PasoLanzado: rev.PasoLanzado, Mirada: rev.Mirada.Hora, MiradaOrden: rev.Mirada.Orden,
			Sesion: sesion, Bitacora: v.Bitacora, BitacoraAjena: d.Ajena, Avance: -1, Acta: v.Acta, ActaEstado: v.ActaEstado,
			Temporales: sinCortadas(a.Temporales), Cruces: cruces[a.ID]}
		if !rev.Mirada.Hora.IsZero() {
			p.MiradaEn, p.PasoMirada = casa(rev.Sesion), rev.Mirada.Paso
		}
		if v.Avance != nil {
			p.Avance = v.Avance.Porcentaje
		}
		// La carpeta de un acta lleva su mismo nombre (2026-10-31-medidor-w6.md → …\2026-10-31-medidor-w6\).
		if p.Acta != "" {
			p.Temporales = sumarUna(p.Temporales, strings.TrimSuffix(filepath.Base(filepath.FromSlash(p.Acta)), ".md"))
		}
		out = append(out, p)
	}
	return out
}

func primeroConTexto(xs ...string) string {
	for _, x := range xs {
		if x != "" {
			return x
		}
	}
	return ""
}

// sinCortadas quita los nombres que son el comienzo de otro: la orden guardada se recorta y deja «2026-10-31-med».
func sinCortadas(l []string) []string {
	var out []string
	for _, x := range l {
		cortada := false
		for _, y := range l {
			if len(y) > len(x) && strings.HasPrefix(strings.ToLower(y), strings.ToLower(x)) {
				cortada = true
				break
			}
		}
		if !cortada {
			out = append(out, x)
		}
	}
	return out
}

// documentoPorNombre: docs/trazabilidad/F*/<carpeta>/<nombre>.md si existe; enVentana exige, como documentoPorRol,
// que se escribiera mientras el agente trabajaba (o hasta 10 min después).
func documentoPorNombre(a claude.Agente, carpeta, nombre string, enVentana bool) string {
	if nombre == "" {
		return ""
	}
	rutas, _ := filepath.Glob(filepath.Join(a.Raiz, "docs", "trazabilidad", "F*", carpeta, nombre+".md"))
	for _, r := range rutas {
		fi, err := os.Stat(r)
		if err != nil || (enVentana && (fi.ModTime().Before(a.Inicio.Add(-time.Minute)) || fi.ModTime().After(a.Ultimo.Add(10*time.Minute)))) {
			continue
		}
		if rel, err := filepath.Rel(a.Raiz, r); err == nil {
			return filepath.ToSlash(rel)
		}
	}
	return ""
}

func sumarUna(l []string, x string) []string {
	for _, y := range l {
		if strings.EqualFold(y, x) {
			return l
		}
	}
	return append(l, x)
}

// lanzamientoDe: la llamada a codex.ps1 que corrió la sesión para este tema, la más cercana antes de su inicio.
func lanzamientoDe(ls []claude.LanzamientoCodex, proyecto, tema string, inicio time.Time) (claude.LanzamientoCodex, bool) {
	var mejor claude.LanzamientoCodex
	ok := false
	t := claude.Normalizar(tema)
	if t == "" {
		return mejor, false
	}
	// El tema como palabra entera: «e2-x» no debe enlazar con la orden de «e2-x-sol».
	re := regexp.MustCompile(`(^|[^\w.-])` + regexp.QuoteMeta(t) + `($|[^\w.-])`)
	for _, l := range ls {
		if l.Proyecto != proyecto || !re.MatchString(claude.Normalizar(l.Orden)) || l.Hora.After(inicio.Add(2*time.Minute)) {
			continue
		}
		if !ok || l.Hora.After(mejor.Hora) {
			mejor, ok = l, true
		}
	}
	return mejor, ok
}

func (s *Servidor) codexProceso(proyecto, raiz string, ahora time.Time) []proceso.Agente {
	ls := s.Claude.LanzamientosCodex(time.Time{})
	// sesion: la que anotó codex.ps1 en la meta, si la hay; entonces el lanzamiento se busca solo entre los suyos.
	armar := func(id, tema, fecha, clase, modelo, effort, base, fase, ficha, sesion string, inicio, ultimo time.Time, vivo bool) proceso.Agente {
		estado := Entregado
		if vivo {
			estado = Trabajando
		}
		me := modeloCorto(modelo, effort)
		p := proceso.Agente{ID: id, Nombre: nombreVista(AgenteVista{Tipo: "codex", Tarea: tema}, docsAgente{}), Proveedor: "Codex",
			Clase: claseCorta(clase, me), Tarea: tema, Estado: estado, Activo: vivo, Inicio: inicio, Ultimo: ultimo, Fase: faseDeAgente(raiz, inicio, rutaFase(fase)),
			ClaseAsignada: clase, Modelo: me, Base: base, Encargo: ficha, EffortLlamada: effort, Avance: -1,
			Temporales: []string{fecha + "-codex-" + tema}}
		deLaSesion := ls
		if sesion != "" {
			p.Sesion, deLaSesion = sesion, nil
			for _, l := range ls {
				if idDeTranscrito(l.Sesion) == sesion {
					deLaSesion = append(deLaSesion, l)
				}
			}
		}
		if l, ok := lanzamientoDe(deLaSesion, proyecto, tema, inicio); ok {
			p.Lanzado, p.LanzadoEn, p.PasoLanzado, p.Sesion = l.Hora, casa(l.Sesion), l.Paso, idDeTranscrito(l.Sesion)
			if !l.Mirada.Hora.IsZero() {
				p.Mirada, p.MiradaEn, p.PasoMirada, p.MiradaOrden = l.Mirada.Hora, casa(l.Sesion), l.Mirada.Paso, l.Mirada.Orden
			}
		}
		return p
	}
	var out []proceso.Agente
	cubiertas := map[string]bool{}
	if s.CodexVivo != nil {
		for _, c := range s.CodexVivo.Llamadas() {
			if !esDelProyecto(c.Meta.Raiz, proyecto, raiz) {
				continue
			}
			cubiertas[claveRegistro(c)] = true
			fecha := c.Nombre
			if len(fecha) >= 10 {
				fecha = fecha[:10]
			}
			ficha := casa(filepath.Join(s.CodexVivo.Dir, c.Nombre+".meta.json"))
			if c.Interactiva {
				fecha, ficha = c.Inicio.Format("2006-01-02"), casa(c.Ruta)
			}
			out = append(out, armar(c.ID, c.Meta.Tema, fecha, c.Meta.Clase, c.Meta.Modelo, c.Meta.Effort, c.Meta.Base, c.Meta.Fase, ficha, c.Meta.Sesion,
				c.Inicio, c.Ultimo, c.Activa(ahora)))
		}
	}
	s.mu.RLock()
	registro := append([]trazabilidad.LlamadaCodex(nil), s.codex[proyecto]...)
	s.mu.RUnlock()
	for _, c := range registro {
		if cubiertas[c.ID] {
			continue
		}
		ficha := "docs/trazabilidad/" + c.Fase + "/codex/REGISTRO.md"
		inicio := c.Terminada.Add(-time.Duration(c.Segundos) * time.Second)
		out = append(out, armar(c.ID, c.Tema, c.Fecha, c.Clase, c.Modelo, c.Effort, c.Base, c.Fase, ficha, "", inicio, c.Terminada, false))
	}
	return out
}

// firmaTemporales de todos los proyectos: cambia cuando aparece, desaparece o se toca una carpeta temporal.
func (s *Servidor) firmaTemporales() string {
	var b strings.Builder
	for _, p := range s.listaProyectos() {
		b.WriteString(proceso.Firma(proceso.Lugares(p.Raiz, s.reglas.Reglas(p.Raiz))))
	}
	return b.String()
}

// actualizarTemporales guarda la firma nueva y dice si cambió.
func (s *Servidor) actualizarTemporales() bool {
	f := s.firmaTemporales()
	s.mu.Lock()
	defer s.mu.Unlock()
	cambio := f != s.firmaTmp
	s.firmaTmp = f
	return cambio
}

// procesoMCP: el resumen de la salud del proceso de hoy para la herramienta salud.
func (s *Servidor) procesoMCP(proyecto string) string {
	inf := s.informeProceso(proyecto, PeriodoHoy)
	var b strings.Builder
	enRegla := 0
	for _, f := range inf.Filas {
		if f.Faltas == 0 && f.Medias == 0 {
			enRegla++
		}
	}
	fmt.Fprintf(&b, "Proceso de hoy (%s): %d de %d agentes en regla.\n", s.proyectoOMayor(proyecto), enRegla, len(inf.Filas))
	nombres := map[string]string{}
	for _, r := range inf.Reglas {
		nombres[r.ID] = r.Nombre
	}
	for _, f := range inf.Filas {
		for _, c := range f.Celdas {
			if c.Marca != proceso.Falta && c.Marca != proceso.Medias {
				continue
			}
			marca := "falta"
			if c.Marca == proceso.Medias {
				marca = "a medias"
			}
			fmt.Fprintf(&b, "  [%s] %s · %s: %s %s", marca, f.Nombre, nombres[c.Regla], c.Titulo+".", c.Texto)
			if c.Ruta != "" {
				b.WriteString(" (" + c.Ruta)
				if c.Paso >= 0 {
					b.WriteString(" · paso " + strconv.Itoa(c.Paso))
				}
				b.WriteString(")")
			}
			b.WriteByte('\n')
		}
	}
	if len(inf.Temporales) == 0 {
		b.WriteString("Ningún temporal en el disco.\n")
		return b.String()
	}
	fmt.Fprintf(&b, "Temporales en el disco (%d):\n", len(inf.Temporales))
	for _, c := range inf.Temporales {
		acta := "sin acta"
		if c.Acta != "" {
			acta = "acta " + c.ActaEstado + ": " + c.Acta
		}
		fmt.Fprintf(&b, "  %s · %s · %s · %s\n", c.Ruta, proceso.Peso(c.Bytes), c.Estado, acta)
	}
	return b.String()
}
