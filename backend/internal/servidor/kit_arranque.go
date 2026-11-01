package servidor

import (
	"bytes"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"atalaya/internal/configurar"
)

// Dos pasos más del asistente que escriben fuera de la carpeta de Atalaya, por la misma puerta (vista previa, copia,
// historial y deshacer): copiar la skill atalaya-proyecto a Claude Code o a Codex (S5), y arrancar Atalaya con el
// sistema (S7). Nunca dentro de un proyecto.

// NombreSkill: la carpeta de la skill, en el kit y en cada herramienta.
const NombreSkill = "atalaya-proyecto"

// destinoSkill: dónde busca sus skills cada herramienta; vacío si no se sabe.
func (s *Servidor) destinoSkill(herramienta string) string {
	switch {
	case herramienta == "claude" && s.Casa != "":
		return filepath.Join(s.Casa, ".claude", "skills", NombreSkill)
	case herramienta == "codex" && s.CodexHome != "":
		return filepath.Join(s.CodexHome, "skills", NombreSkill)
	}
	return ""
}

// archivosKit: los archivos del kit, por ruta relativa con «/», ordenados.
func archivosKit(dir string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(dir, func(r string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			rel, _ := filepath.Rel(dir, r)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out, err
}

// EstadoSkill: «al día» si cada archivo del kit está igual en la herramienta, «distinta» si la skill está pero no
// igual (otra versión), «falta» si no está; vacío si Atalaya no tiene el kit o no sabe dónde va.
func (s *Servidor) estadoSkill(herramienta string) string {
	dest := s.destinoSkill(herramienta)
	if s.Kit == "" || dest == "" {
		return ""
	}
	if _, err := os.Stat(filepath.Join(dest, "SKILL.md")); err != nil {
		return "falta"
	}
	rels, err := archivosKit(s.Kit)
	if err != nil {
		return ""
	}
	for _, rel := range rels {
		a, err1 := os.ReadFile(filepath.Join(s.Kit, filepath.FromSlash(rel)))
		b, err2 := os.ReadFile(filepath.Join(dest, filepath.FromSlash(rel)))
		if err1 != nil || err2 != nil || !bytes.Equal(a, b) {
			return "distinta"
		}
	}
	return "al día"
}

// dentroDe: r está dentro de dir (o es dir).
func dentroDe(r, dir string) bool {
	rel, err := filepath.Rel(dir, r)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

// permitidoKitArranque: la parte de la lista cerrada que agrega este archivo (puerta).
func (s *Servidor) permitidoKitArranque(r string) bool {
	for _, h := range []string{"claude", "codex"} {
		if d := s.destinoSkill(h); d != "" && dentroDe(filepath.Clean(r), d) {
			return true
		}
	}
	a := s.arranque()
	return a.Archivo != "" && configurar.MismaRuta(r, a.Archivo)
}

// programaDelSistema: los programas del sistema de la lista cerrada (para el arranque).
func programaDelSistema(nombre string) (string, bool) {
	switch {
	case nombre == "powershell" && runtime.GOOS == "windows":
		raiz := os.Getenv("SystemRoot")
		if raiz == "" {
			raiz = `C:\Windows`
		}
		return filepath.Join(raiz, "System32", "WindowsPowerShell", "v1.0", "powershell.exe"), true
	case nombre == "systemctl" && runtime.GOOS == "linux":
		if r, err := exec.LookPath("systemctl"); err == nil {
			return r, true
		}
	}
	return "", false
}

type pedidoSkill struct {
	Herramienta string `json:"herramienta"`
}

// proponerSkill: copiar (o poner al día) la skill del kit en la carpeta de skills de una herramienta. Un archivo que ya
// está igual no se toca; uno de más que la persona agregó allí, tampoco.
func (s *Servidor) proponerSkill(w http.ResponseWriter, r *http.Request) {
	if !delPanel(r) {
		fallo(w, http.StatusForbidden, "solo el panel puede cambiar la configuración")
		return
	}
	var p pedidoSkill
	if !leerCuerpo(w, r, &p) {
		return
	}
	dest := s.destinoSkill(p.Herramienta)
	if dest == "" {
		fallo(w, http.StatusBadRequest, "la herramienta es claude o codex")
		return
	}
	if s.Kit == "" {
		fallo(w, http.StatusConflict, "Atalaya no encuentra el kit (la carpeta kit\\atalaya-proyecto junto a atalaya.exe)")
		return
	}
	rels, err := archivosKit(s.Kit)
	if err != nil || len(rels) == 0 {
		fallo(w, http.StatusConflict, "no se pudo leer el kit")
		return
	}
	pu, err := s.puerta()
	if err != nil {
		fallo(w, http.StatusConflict, err.Error())
		return
	}
	var cambios []configurar.Cambio
	for _, rel := range rels {
		b, err := os.ReadFile(filepath.Join(s.Kit, filepath.FromSlash(rel)))
		if err != nil {
			fallo(w, http.StatusConflict, "no se pudo leer el kit: "+rel)
			return
		}
		ruta := filepath.Join(dest, filepath.FromSlash(rel))
		if actual, err := os.ReadFile(ruta); err == nil && bytes.Equal(actual, b) {
			continue
		}
		cambios = append(cambios, configurar.Cambio{Tipo: configurar.Archivo, Ruta: ruta, Despues: string(b)})
	}
	if len(cambios) == 0 {
		fallo(w, http.StatusConflict, "la skill ya está al día")
		return
	}
	nombre := map[string]string{"claude": "Claude Code", "codex": "Codex"}[p.Herramienta]
	prop, err := pu.Proponer("skill", "Skill "+NombreSkill+" para "+nombre+" ("+dest+")", cambios)
	if err != nil {
		fallo(w, http.StatusBadRequest, err.Error())
		return
	}
	escribir(w, prop)
}

// Arranque: si Atalaya arranca con el sistema y cómo.
type Arranque struct {
	Disponible bool   `json:"disponible"` // este sistema sabe (Windows: acceso directo de inicio; Linux: systemd del usuario)
	Activo     bool   `json:"activo"`
	Archivo    string `json:"archivo,omitempty"`
	Programa   string `json:"programa,omitempty"` // el atalaya que arrancaría
}

func (s *Servidor) arranque() Arranque {
	exe, err := os.Executable()
	if err != nil {
		return Arranque{}
	}
	a := Arranque{Programa: exe}
	switch runtime.GOOS {
	case "windows":
		if ad := os.Getenv("APPDATA"); ad != "" {
			a.Archivo = filepath.Join(ad, "Microsoft", "Windows", "Start Menu", "Programs", "Startup", "Atalaya.lnk")
		}
	case "linux":
		if s.Casa != "" {
			a.Archivo = filepath.Join(s.Casa, ".config", "systemd", "user", "atalaya.service")
		}
	}
	a.Disponible = a.Archivo != ""
	if a.Disponible {
		_, err := os.Stat(a.Archivo)
		a.Activo = err == nil
	}
	return a
}

// comillaPS: un texto entre comillas simples de PowerShell (la comilla se dobla).
func comillaPS(t string) string { return "'" + strings.ReplaceAll(t, "'", "''") + "'" }

// unidadSystemd: la unidad de usuario que arranca este atalaya al iniciar sesión.
func unidadSystemd(exe string) string {
	return "[Unit]\nDescription=Atalaya: el panel de tus agentes\n\n[Service]\nExecStart=" + strconvQuote(exe) +
		"\nRestart=on-failure\n\n[Install]\nWantedBy=default.target\n"
}

// strconvQuote: una ruta con espacios entre comillas dobles, como las quiere systemd.
func strconvQuote(r string) string {
	if strings.ContainsAny(r, " \t") {
		return `"` + strings.ReplaceAll(r, `"`, `\"`) + `"`
	}
	return r
}

type pedidoArranque struct {
	Activar bool `json:"activar"`
}

// proponerArranque: en Windows, el acceso directo de la carpeta de inicio (como dice la guía), creado y borrado con
// PowerShell; en Linux, la unidad de systemd del usuario y `systemctl --user enable` (sin --now: este Atalaya ya corre).
func (s *Servidor) proponerArranque(w http.ResponseWriter, r *http.Request) {
	if !delPanel(r) {
		fallo(w, http.StatusForbidden, "solo el panel puede cambiar la configuración")
		return
	}
	var p pedidoArranque
	if !leerCuerpo(w, r, &p) {
		return
	}
	a := s.arranque()
	if !a.Disponible {
		fallo(w, http.StatusConflict, "en este sistema Atalaya no sabe arrancar solo; mira la guía de instalación")
		return
	}
	if p.Activar == a.Activo {
		fallo(w, http.StatusConflict, map[bool]string{true: "Atalaya ya arranca con el sistema", false: "Atalaya no arranca con el sistema"}[a.Activo])
		return
	}
	pu, err := s.puerta()
	if err != nil {
		fallo(w, http.StatusConflict, err.Error())
		return
	}
	var cambios []configurar.Cambio
	resumen := "Arrancar Atalaya con el sistema"
	if !p.Activar {
		resumen = "No arrancar Atalaya con el sistema"
	}
	switch runtime.GOOS {
	case "windows":
		crear := "$s = (New-Object -ComObject WScript.Shell).CreateShortcut(" + comillaPS(a.Archivo) + "); $s.TargetPath = " +
			comillaPS(a.Programa) + "; $s.WorkingDirectory = " + comillaPS(filepath.Dir(a.Programa)) + "; $s.WindowStyle = 7; $s.Save()"
		borrar := "Remove-Item -LiteralPath " + comillaPS(a.Archivo)
		args := func(orden string) []string { return []string{"-NoProfile", "-NonInteractive", "-Command", orden} }
		c := configurar.Cambio{Tipo: configurar.Orden, Programa: "powershell", Argumentos: args(crear), Deshacer: args(borrar)}
		if !p.Activar {
			c.Argumentos, c.Deshacer = c.Deshacer, c.Argumentos
		}
		cambios = append(cambios, c)
	default:
		if p.Activar {
			cambios = append(cambios,
				configurar.Cambio{Tipo: configurar.Archivo, Ruta: a.Archivo, Despues: unidadSystemd(a.Programa)},
				configurar.Cambio{Tipo: configurar.Orden, Programa: "systemctl", Argumentos: []string{"--user", "enable", "atalaya.service"},
					Deshacer: []string{"--user", "disable", "atalaya.service"}})
		} else {
			cambios = append(cambios, configurar.Cambio{Tipo: configurar.Orden, Programa: "systemctl",
				Argumentos: []string{"--user", "disable", "atalaya.service"}, Deshacer: []string{"--user", "enable", "atalaya.service"}})
		}
	}
	prop, err := pu.Proponer("arranque", resumen, cambios)
	if err != nil {
		fallo(w, http.StatusBadRequest, err.Error())
		return
	}
	escribir(w, prop)
}
