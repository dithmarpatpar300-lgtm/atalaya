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
	"strings"
	"time"

	"atalaya/internal/avisos"
	"atalaya/internal/claude"
	"atalaya/internal/codexvivo"
	"atalaya/internal/cuotas"
	"atalaya/internal/datapedia"
	registro "atalaya/internal/proyectos"
	"atalaya/internal/servidor"
	"atalaya/internal/web"
)

type listaProyectos []string

func (l *listaProyectos) String() string     { return strings.Join(*l, ", ") }
func (l *listaProyectos) Set(v string) error { *l = append(*l, v); return nil }

func main() {
	casa, _ := os.UserHomeDir()
	local := os.Getenv("LOCALAPPDATA")
	escucha := flag.String("escucha", "127.0.0.1:7420", "dirección; solo se acepta 127.0.0.1 o localhost")
	transcritos := flag.String("claude", filepath.Join(casa, ".claude", "projects"), "carpeta de los transcritos de Claude Code")
	dirCuotas := flag.String("cuotas", filepath.Join(local, "Cuotas"), "carpeta de Cuotas (cuotas.exe y sus archivos)")
	cada := flag.Duration("cada", 2*time.Second, "cada cuánto se miran los archivos")
	dirCodex := flag.String("codex", filepath.Join(local, "Atalaya", "codex"), "carpeta de la traza en vivo de Codex (la escribe scripts/ops/codex.ps1)")
	rutaDB := flag.String("db", filepath.Join(local, "Atalaya", "atalaya.db"), "índice de la datapedia (SQLite; se rehace si se borra)")
	var proyectos listaProyectos
	flag.Var(&proyectos, "proyecto", "Nombre=Ruta de un proyecto con docs/trazabilidad (se repite; se suma al registro)")
	conAvisos := flag.Bool("avisos", true, "notificaciones de Windows de lo que te espera (decisiones y alertas de error)")
	dirDiario := flag.String("diario", filepath.Join(local, "Atalaya", "diario"), "carpeta del diario, un JSON por día (vacía: sin diario)")
	rutaRegistro := flag.String("proyectos", "", "registro de proyectos (por omisión, "+registro.Archivo+" junto al ejecutable)")
	flag.Parse()

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
	s.Registro = func() ([]servidor.Proyecto, error) { return leerProyectos(ruta, explicito, proyectos) }

	var dpProyectos []datapedia.Proyecto
	for _, p := range s.Proyectos {
		dpProyectos = append(dpProyectos, datapedia.Proyecto{Nombre: p.Nombre, Raiz: p.Raiz})
	}
	if dp, err := datapedia.Abrir(*rutaDB, *transcritos, dpProyectos); err != nil {
		log.Printf("datapedia desactivada: %v", err)
	} else {
		dp.Virtuales = servidor.Trazas(s.Claude)
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
				out = append(out, servidor.Proyecto{Nombre: p.Nombre, Raiz: p.Ruta, Sigla: p.Sigla, Nivel: p.Nivel, Registrado: true})
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
