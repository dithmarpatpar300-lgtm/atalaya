package servidor

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"atalaya/internal/alertas"
	"atalaya/internal/cuotas"
)

// La línea de tiempo global: todos los agentes de un proyecto en un día, sobre
// el reloj. Cada fila dice de cuándo a cuándo trabajó, quién lo lanzó, si pasó del tope de AGENTS.md y a qué hora
// editó un archivo que otro agente vivo también editaba; arriba va la cuota de 5 h de Claude y Codex según Cuotas.

type choqueLinea struct {
	Hora   time.Time `json:"hora"`
	Otro   string    `json:"otro"`
	OtroID string    `json:"otro_id"`
	Ruta   string    `json:"ruta"`
}

type filaLinea struct {
	ID        string        `json:"id"`
	Nombre    string        `json:"nombre"`
	Proveedor string        `json:"proveedor"`
	Tipo      string        `json:"tipo"` // sesion | subagente | codex
	Padre     string        `json:"padre,omitempty"`
	Clase     string        `json:"clase"`
	Estado    string        `json:"estado"`
	Inicio    time.Time     `json:"inicio"`
	Fin       time.Time     `json:"fin"` // el último paso, o ahora si sigue vivo
	Vivo      bool          `json:"vivo"`
	Excede    bool          `json:"excede"` // un agente (no la sesión) que pasó del tope
	Choques   []choqueLinea `json:"choques"`
	// Diario: la fila sale del diario porque su transcrito o su traza ya se borraron (sin cruces ni pasos).
	Diario bool `json:"diario,omitempty"`
	// presupuesto: el del encargo («30 min»); manda sobre el tope del proyecto para Excede (alertas.TopeDe).
	presupuesto string
}

type respuestaLinea struct {
	Proyecto string                    `json:"proyecto"`
	Dia      string                    `json:"dia"`
	Dias     []string                  `json:"dias"` // los días con agentes, del más nuevo al más viejo (maxDiasLinea a lo sumo)
	Desde    time.Time                 `json:"desde"`
	Hasta    time.Time                 `json:"hasta"`
	Ahora    time.Time                 `json:"ahora"`
	TopeMin  int                       `json:"tope_min"`
	Fuente   string                    `json:"fuente,omitempty"` // dónde escribe AGENTS.md el tope
	Filas    []filaLinea               `json:"filas"`
	Cuota    map[string][]cuotas.Punto `json:"cuota"`
}

// maxDiasLinea: los días que se ofrecen; los viejos salen del diario (soloDelDiario).
const maxDiasLinea = maxDiasDiario

func (s *Servidor) verLinea(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	escribir(w, s.lineaDeTiempo(s.proyectoOMayor(q.Get("proyecto")), q.Get("dia")))
}

func (s *Servidor) lineaDeTiempo(proyecto, dia string) respuestaLinea {
	ahora := s.ahora()
	reglas := s.reglas.Reglas(s.raizDe(proyecto))
	cruces := s.crucesPorAgente(proyecto, ahora)
	equipo := s.equipoTrabajando()

	var todas []filaLinea
	for _, a := range s.Claude.Agentes() {
		if a.Proyecto != proyecto {
			continue
		}
		v := s.vistaConEquipo(a, false, equipo)
		f := filaLinea{ID: v.ID, Nombre: v.Nombre, Proveedor: "Claude", Tipo: v.Tipo, Padre: v.Padre, Clase: claseDeVista(v), Estado: v.Estado,
			Inicio: v.Inicio, Fin: v.Ultimo, Vivo: activo(v.Estado), Choques: []choqueLinea{}, presupuesto: v.Presupuesto}
		for _, c := range cruces[a.ID] {
			f.Choques = append(f.Choques, choqueLinea{Hora: c.Hora, Otro: c.Otro, OtroID: c.OtroID, Ruta: c.Ruta})
		}
		todas = append(todas, f)
	}
	for _, v := range s.todos() {
		if v.Proveedor != "Codex" || v.Proyecto != proyecto {
			continue
		}
		todas = append(todas, filaLinea{ID: v.ID, Nombre: v.Nombre, Proveedor: "Codex", Tipo: "codex", Padre: v.Padre,
			Clase: claseDeVista(v), Estado: v.Estado, Inicio: v.Inicio, Fin: v.Ultimo, Vivo: activo(v.Estado), Choques: []choqueLinea{},
			presupuesto: v.Presupuesto})
	}
	// Los días que la memoria ya no tiene salen del diario.
	en := map[string]bool{}
	for _, f := range todas {
		en[f.ID] = true
	}
	for _, a := range s.soloDelDiario("", en) {
		if a.Proyecto != proyecto {
			continue
		}
		f := filaLinea{ID: a.ID, Nombre: a.Nombre, Proveedor: a.Proveedor, Tipo: a.Tipo, Padre: a.Padre, Clase: claseCorta(a.Clase, a.Corto),
			Estado: Entregado, Inicio: a.Inicio, Fin: a.Fin, Choques: []choqueLinea{}, Diario: true}
		if a.Presupuesto > 0 {
			f.presupuesto = strconv.FormatFloat(a.Presupuesto, 'f', -1, 64) + " min"
		}
		todas = append(todas, f)
	}
	out := armarLinea(proyecto, dia, ahora, reglas.Tope, todas, s.historialCuota)
	out.Fuente = reglas.Fuente("tiempo")
	return out
}

func (s *Servidor) historialCuota(desde, hasta time.Time) map[string][]cuotas.Punto {
	if s.Cuotas == nil {
		return map[string][]cuotas.Punto{}
	}
	return s.Cuotas.Historial(desde, hasta, 5*time.Minute)
}

// armarLinea: las filas del día pedido (hoy si no se pide o no se entiende), recortadas a él.
func armarLinea(proyecto, dia string, ahora time.Time, tope time.Duration, todas []filaLinea, cuota func(desde, hasta time.Time) map[string][]cuotas.Punto) respuestaLinea {
	loc := ahora.Location()
	desde := time.Date(ahora.Year(), ahora.Month(), ahora.Day(), 0, 0, 0, 0, loc)
	if d, err := time.ParseInLocation("2006-01-02", dia, loc); err == nil {
		desde = d
	}
	hasta := desde.AddDate(0, 0, 1)
	out := respuestaLinea{Proyecto: proyecto, Dia: desde.Format("2006-01-02"), Desde: desde, Hasta: hasta, Ahora: ahora,
		TopeMin: int(tope.Minutes()), Filas: []filaLinea{}, Dias: []string{}}

	visto := map[string]bool{}
	for _, f := range todas {
		if f.Inicio.IsZero() {
			continue
		}
		if f.Vivo && f.Fin.Before(ahora) {
			f.Fin = ahora
		}
		if f.Fin.Before(f.Inicio) {
			f.Fin = f.Inicio
		}
		for d := f.Inicio.In(loc); !d.After(f.Fin); d = d.AddDate(0, 0, 1) {
			visto[d.Format("2006-01-02")] = true
		}
		visto[f.Fin.In(loc).Format("2006-01-02")] = true
		if !f.Inicio.Before(hasta) || f.Fin.Before(desde) {
			continue
		}
		t := alertas.TopeDe(alertas.Reglas{Tope: tope}, f.presupuesto)
		f.Excede = f.Tipo != "sesion" && t > 0 && f.Fin.Sub(f.Inicio) > t
		out.Filas = append(out.Filas, f)
	}
	for d := range visto {
		out.Dias = append(out.Dias, d)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(out.Dias)))
	if len(out.Dias) > maxDiasLinea {
		out.Dias = out.Dias[:maxDiasLinea]
	}
	sort.SliceStable(out.Filas, func(i, j int) bool { return out.Filas[i].Inicio.Before(out.Filas[j].Inicio) })
	out.Cuota = cuota(desde, hasta)
	return out
}

// claseDeVista: «I2 · sonnet-5-5/high», o lo que haya.
func claseDeVista(v AgenteVista) string {
	m := v.ModeloCorto
	if m == "" {
		m = modeloCorto(v.Modelo, v.Effort)
	}
	return claseCorta(v.Clase, m)
}
