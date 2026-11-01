// Atalaya es un panel de agentes. Lee, en solo lectura, las cuotas de
// la herramienta opcional Cuotas, si está instalada, los transcritos de Claude Code y la trazabilidad de cada proyecto, y
// sirve el panel en http://127.0.0.1:7420.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"atalaya/internal/avisos"
	"atalaya/internal/claude"
	"atalaya/internal/codexvivo"
	"atalaya/internal/cuotas"
	"atalaya/internal/datapedia"
	"atalaya/internal/delegadas"
	"atalaya/internal/demo"
	registro "atalaya/internal/proyectos"
	"atalaya/internal/servidor"
	"atalaya/internal/web"
)

type listaProyectos []string

func (l *listaProyectos) String() string     { return strings.Join(*l, ", ") }
func (l *listaProyectos) Set(v string) error { *l = append(*l, v); return nil }

// carpetaLocal: donde Atalaya guarda lo suyo (el índice, el diario, las trazas de Codex) y donde viven las
// herramientas vecinas (Cuotas). En Windows, %LOCALAPPDATA%; en Linux, $XDG_DATA_HOME o ~/.local/share; en macOS,
// ~/Library/Application Support. Nunca vacía: una carpeta vacía volvía relativas las rutas, y fuera de Windows
// Atalaya guardaba su base en la carpeta desde donde se lo lanzaba.
func carpetaLocal(casa, sistema string) string {
	if l := os.Getenv("LOCALAPPDATA"); l != "" {
		return l
	}
	switch sistema {
	case "windows":
		return filepath.Join(casa, "AppData", "Local")
	case "darwin":
		return filepath.Join(casa, "Library", "Application Support")
	}
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		return x
	}
	return filepath.Join(casa, ".local", "share")
}

func main() {
	casa, _ := os.UserHomeDir()
	local := carpetaLocal(casa, runtime.GOOS)
	escucha := flag.String("escucha", "127.0.0.1:7420", "dirección; solo se acepta 127.0.0.1 o localhost")
	transcritos := flag.String("claude", filepath.Join(casa, ".claude", "projects"), "carpeta de los transcritos de Claude Code")
	dirCuotas := flag.String("cuotas", filepath.Join(local, "Cuotas"), "carpeta de Cuotas (cuotas.exe y sus archivos)")
	cada := flag.Duration("cada", 2*time.Second, "cada cuánto se miran los archivos")
	dirCodex := flag.String("codex", filepath.Join(local, "Atalaya", "codex"), "carpeta de la traza en vivo de Codex (la escribe scripts/codex.ps1)")
	dirDelegadas := flag.String("claude-delegado", filepath.Join(local, "Atalaya", "claude"), "carpeta de las fichas de las sesiones de Claude Code lanzadas por scripts/claude.ps1")
	estadoCodex := os.Getenv("CODEX_HOME")
	if estadoCodex == "" {
		estadoCodex = filepath.Join(casa, ".codex")
	}
	dirEstadoCodex := flag.String("codex-estado", estadoCodex, "carpeta de Codex (CODEX_HOME) con las sesiones interactivas en sessions/ (vacía: no se siguen)")
	rutaDB := flag.String("db", filepath.Join(local, "Atalaya", "atalaya.db"), "índice de la datapedia (SQLite; se rehace si se borra)")
	var proyectos listaProyectos
	flag.Var(&proyectos, "proyecto", "Nombre=Ruta de un proyecto con docs/trazabilidad (se repite; se suma al registro)")
	// Las notificaciones son de Windows: en los demás sistemas, apagadas de fábrica.
	conAvisos := flag.Bool("avisos", runtime.GOOS == "windows", "notificaciones de Windows de lo que te espera (decisiones y alertas de error)")
	dirDiario := flag.String("diario", filepath.Join(local, "Atalaya", "diario"), "carpeta del diario, un JSON por día (vacía: sin diario)")
	rutaRegistro := flag.String("proyectos", "", "registro de proyectos (por omisión, "+registro.Archivo+" junto al ejecutable)")
	modoDemo := flag.Bool("demo", false, "usar proyectos y agentes inventados")
	// La carpeta del mundo de demo aparece en las rutas que muestra el panel: para capturas, una neutra
	// (por ejemplo C:\Users\Public\atalaya-demo) en vez de la temporal del usuario.
	carpetaDemo := flag.String("demo-carpeta", filepath.Join(os.TempDir(), "atalaya-demo"), "dónde se escribe el mundo de -demo (se borra y se rehace)")
	flag.Parse()
	if *modoDemo {
		m, err := demo.Preparar(*carpetaDemo, time.Now())
		if err != nil {
			log.Fatalf("preparar demo: %v", err)
		}
		*transcritos, *dirCodex, *rutaRegistro = m.Claude, m.Codex, m.Registro
		*dirDiario, *rutaDB, *dirCuotas = m.Diario, m.DB, m.Cuotas
		*conAvisos, *dirEstadoCodex, *dirDelegadas = false, "", ""
		proyectos = nil
		log.Printf("mundo de demo: %s", m.Raiz)
	}

	host, _, err := net.SplitHostPort(*escucha)
	if err != nil || (host != "127.0.0.1" && host != "localhost") {
		log.Fatalf("-escucha debe ser 127.0.0.1:<puerto> o localhost:<puerto>; vino %q", *escucha)
	}

	s := &servidor.Servidor{
		Cuotas: cuotas.NuevoLector(filepath.Join(*dirCuotas, "cuotas.exe"), *dirCuotas),
		Claude: claude.NuevoIndice(*transcritos),
		Web:    web.FS(),
		// La traza en vivo de Codex que escribe scripts/codex.ps1.
		CodexVivo: codexvivo.NuevoLector(*dirCodex),
		Base:      "http://" + *escucha,
		Diario:    *dirDiario,
	}
	if *conAvisos {
		// 15 s entre dos avisos: varias cosas seguidas llegan en uno solo.
		s.Avisos, s.Notificador = avisos.Nuevo(15*time.Second), avisos.Windows{}
		s.RutaAvisos = filepath.Join(local, "Atalaya", "avisos.json")
	}
	ruta, explicito := rutaDelRegistro(*rutaRegistro)
	if s.Proyectos, err = leerProyectos(ruta, explicito, proyectos); err != nil {
		log.Fatalf("registro de proyectos: %v", err)
	}
	s.RutaRegistro = ruta
	// Para la configuración desde el panel: dónde buscar Claude Code y Codex, y dónde guardar las copias y el historial
	// de lo que cambia (junto al índice de la datapedia: la carpeta de Atalaya, o la del mundo de demo).
	s.Casa, s.CodexHome, s.Datos = casa, estadoCodex, filepath.Dir(*rutaDB)
	// El kit viaja junto al ejecutable (el zip, y el repositorio en desarrollo): de ahí se copia la skill.
	if exe, err := os.Executable(); err == nil {
		if k := filepath.Join(filepath.Dir(exe), "kit", "atalaya-proyecto"); existeCarpeta(k) {
			s.Kit = k
		}
	}
	// Las sesiones interactivas de Codex (la TUI): se cuelgan del proyecto que contiene su carpeta.
	s.CodexVivo.Sesiones, s.CodexVivo.RaizDe = *dirEstadoCodex, s.RaizDeCarpeta
	// Las sesiones de Claude Code que lanzó claude.ps1 (casi siempre desde Codex): su ficha dice de quién cuelgan.
	if *dirDelegadas != "" {
		s.Delegadas = delegadas.NuevoLector(*dirDelegadas)
		s.Claude.Delegadas = s.Delegadas.Delegacion
	}
	s.Registro = func() ([]servidor.Proyecto, error) { return leerProyectos(ruta, explicito, proyectos) }

	var dpProyectos []datapedia.Proyecto
	for _, p := range s.Proyectos {
		if !p.FueraDatapedia {
			dpProyectos = append(dpProyectos, datapedia.Proyecto{Nombre: p.Nombre, Raiz: p.Raiz})
		}
	}
	if dp, err := datapedia.Abrir(*rutaDB, *transcritos, dpProyectos); err != nil {
		log.Printf("datapedia desactivada: %v", err)
	} else {
		dp.Virtuales = servidor.Trazas(s.Claude)
		dp.Fuera = s.FueraDeDatapedia
		s.Datapedia = dp
		defer dp.Cerrar()
	}

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt)
	defer parar()

	inicio := time.Now()
	go s.Vigilar(ctx, *cada)

	srv := &http.Server{Addr: *escucha, Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		cierre, c := context.WithTimeout(context.Background(), 3*time.Second)
		defer c()
		_ = srv.Shutdown(cierre)
	}()
	fmt.Printf("Atalaya %s en http://%s (arranque %s)\n", servidor.Version, *escucha, time.Since(inicio).Round(time.Millisecond))
	if s.Web == nil {
		fmt.Println("Sin frontend embebido: en frontend/ corre `npm run dev` y abre http://localhost:3000.")
	}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

// rutaDelRegistro: la de -proyectos o, sin ella, proyectos.json junto al ejecutable.
func rutaDelRegistro(ruta string) (string, bool) {
	if ruta != "" {
		return ruta, true
	}
	if exe, err := os.Executable(); err == nil {
		return filepath.Join(filepath.Dir(exe), registro.Archivo), false
	}
	return "", false
}

// leerProyectos: los del registro (proyectos.json junto al ejecutable, o el de -proyectos) y, sumados, los de
// -proyecto; uno de -proyecto con el nombre de uno registrado le cambia la ruta. Sin registro ni opciones, ninguno.
// Lo usa el arranque y, cada vez que el registro cambia, la relectura en vivo (servidor/registro.go).
func leerProyectos(ruta string, explicito bool, opciones []string) ([]servidor.Proyecto, error) {
	var out []servidor.Proyecto
	if ruta != "" {
		ps, err := registro.Leer(ruta)
		switch {
		case err == nil:
			for _, p := range ps {
				if registro.NombreDistinto(p) {
					log.Printf("%s: %q no se llama como su carpeta; sus agentes se nombran %q y no se juntarán con él", ruta, p.Nombre, filepath.Base(p.Ruta))
				}
				out = append(out, servidor.Proyecto{Nombre: p.Nombre, Raiz: p.Ruta, Sigla: p.Sigla, Nivel: p.Nivel, Registrado: true,
					FueraDatapedia: !p.EnDatapedia()})
			}
		case errors.Is(err, os.ErrNotExist) && !explicito:
			// Sin registro junto al ejecutable (go run, por ejemplo): solo las opciones.
		default:
			return nil, err
		}
	}
	for _, o := range opciones {
		pr, ok := servidor.NombreProyecto(o)
		if !ok {
			return nil, fmt.Errorf("-proyecto debe ser Nombre=Ruta; vino %q", o)
		}
		pr.Sigla = registro.Sigla(pr.Nombre)
		reemplazo := false
		for i := range out {
			if strings.EqualFold(out[i].Nombre, pr.Nombre) {
				out[i].Raiz, reemplazo = pr.Raiz, true
			}
		}
		if !reemplazo {
			out = append(out, pr)
		}
	}
	// Sin registro ni opciones, ningún proyecto: el panel igual muestra los agentes de los transcritos, y «Tus proyectos»
	// explica cómo registrar uno (la skill atalaya-proyecto).
	return out, nil
}

// existeCarpeta: la ruta es una carpeta.
func existeCarpeta(r string) bool {
	fi, err := os.Stat(r)
	return err == nil && fi.IsDir()
}
