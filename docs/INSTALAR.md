# Instalar Atalaya

[English](INSTALL.md) · **Español** · [Volver al README](../README.es.md)

Esta guía deja Atalaya corriendo, con tus proyectos registrados y conectado a Claude Code y a Codex. Está probada en
Windows 11; todo lo que se teclea es PowerShell.

## 1. Lo que hace falta

- **Windows 10 u 11.** (En Linux y macOS compila y debería andar, sin notificaciones, pero no se probó.)
- **Claude Code** (la CLI o la app de escritorio) y/o **Codex CLI**: Atalaya sigue a los agentes que ellos lanzan.
- Para compilar: **Go 1.27 o más nuevo** y **Node.js 20 o más nuevo**, con `npm`. Git para clonar.

## 2. Descargar o compilar

**Descargar.** En la página de versiones (*Releases*) del repositorio está un zip por sistema
(`atalaya-<versión>-windows-amd64.zip`, `-linux-amd64`, `-darwin-arm64`), con el ejecutable, esta guía, el ejemplo del
registro y el kit. Descomprímelo en una carpeta tuya y sigue en el paso 3; no hace falta Go ni Node.

**Compilar.**

```powershell
git clone <la dirección de este repositorio> atalaya
cd atalaya
.\scripts\compilar.ps1
```

El script instala las dependencias del panel la primera vez (`npm ci`), lo compila, lo embebe y deja **un solo
`atalaya.exe`** en la raíz. Con `-Verificar` corre además el formato, el análisis y todas las pruebas.

Los zip de una versión los arma `.\scripts\empaquetar.ps1` después de compilar, en `dist/`; para un solo sistema, `go
build ./cmd/atalaya` dentro de `backend/`, con `GOOS` y `GOARCH` (por ejemplo `linux` y `amd64`).

## 3. Encender

```powershell
.\atalaya.exe
```

y abre **http://127.0.0.1:7420**. Sin proyectos registrados, la portada dice cómo registrar el primero; aun así ya
verás los agentes que encuentre en los transcritos de Claude Code.

Opciones útiles:

| Opción | Qué hace | Por omisión |
| --- | --- | --- |
| `-escucha 127.0.0.1:7421` | Otro puerto (solo acepta `127.0.0.1` o `localhost`) | `127.0.0.1:7420` |
| `-proyectos <archivo>` | Otro registro de proyectos | `proyectos.json` junto al ejecutable |
| `-proyecto "Nombre=C:\ruta"` | Suma un proyecto sin tocar el registro (se repite) | — |
| `-claude <carpeta>` | Dónde están los transcritos de Claude Code | `%USERPROFILE%\.claude\projects` |
| `-codex <carpeta>` | Dónde escribe `codex.ps1` las trazas de Codex | `%LOCALAPPDATA%\Atalaya\codex` |
| `-avisos=false` | Sin notificaciones de Windows | encendidas |
| `-diario <carpeta>` | Dónde va el diario (vacía: sin diario) | `%LOCALAPPDATA%\Atalaya\diario` |
| `-db <archivo>` | El índice de la datapedia (se rehace si se borra) | `%LOCALAPPDATA%\Atalaya\atalaya.db` |
| `-demo` | Un mundo de demo: proyectos, agentes, alertas y decisiones inventados, para probarlo sin agentes | apagado |
| `-demo-carpeta <carpeta>` | Dónde se escribe ese mundo (se borra y se rehace al arrancar) | la carpeta temporal |

Para que arranque con Windows, un acceso directo en la carpeta de inicio:

```powershell
$s = (New-Object -ComObject WScript.Shell).CreateShortcut("$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\Atalaya.lnk")
$s.TargetPath = (Resolve-Path .\atalaya.exe).Path; $s.WorkingDirectory = (Get-Location).Path; $s.WindowStyle = 7; $s.Save()
```

## 4. Registrar tus proyectos

Atalaya junta los agentes por la **carpeta** del proyecto, así que el nombre registrado es el de su carpeta.

**Con la skill (recomendado).** Copia la skill a Claude Code una vez:

```powershell
Copy-Item -Recurse .\kit\atalaya-proyecto "$env:USERPROFILE\.claude\skills\atalaya-proyecto"
```

Después, abre Claude Code **dentro de tu proyecto** y pide «prepara este proyecto para Atalaya». La skill lee el
proyecto, te hace un cuestionario en un solo bloque, te muestra el plan y, con tu sí:

- instala el kit del nivel que elijas: **1 · indexable** (para todo proyecto), **2 · con agentes** (roles en
  `.claude/agents/`, un freno de heredoc, documentos de operación y `scripts/codex.ps1`) o **3 · completo** (entrega de
  versiones y un rol de ensayo);
- abre la fase 0 en `docs/trazabilidad/F0/AVANCE.md` con sus paquetes y horas;
- **fusiona** con lo que ya tengas (`AGENTS.md`, `CLAUDE.md`, `.mcp.json`, `.claude/settings.json`) sin pisar nada;
- registra el proyecto en Atalaya, que lo muestra en unos segundos sin reiniciarse.

Con Codex, o sin la skill, pídele al agente que siga `kit/atalaya-proyecto/SKILL.md` de este repositorio.

**A mano.** Copia `proyectos.ejemplo.json` como `proyectos.json` junto a `atalaya.exe` y deja una línea por proyecto:

```json
[
  { "nombre": "Mi-Proyecto", "ruta": "C:\\Proyectos\\Mi-Proyecto", "sigla": "MP", "nivel": 1 }
]
```

Atalaya relee el archivo solo. Lo que cada nivel espera encontrar en el proyecto está en
`kit/atalaya-proyecto/LEEME.md`.

## 5. Conectar los agentes por MCP

Así los agentes le preguntan a Atalaya («¿cómo vamos?», «¿qué hizo el codificador?», «¿quién tocó este archivo?») en
vez de releer documentos. Atalaya tiene que estar encendido.

**Claude Code**: un `.mcp.json` en la raíz del proyecto (la skill lo deja hecho). `proyecto` es el nombre de la
carpeta, con los espacios como `%20`; con él, `estado`, `retomar` y `salud` responden de ese proyecto sin que el agente
lo nombre.

```json
{
  "mcpServers": {
    "atalaya": { "type": "http", "url": "http://127.0.0.1:7420/mcp?proyecto=Mi-Proyecto" }
  }
}
```

Claude Code pide aprobar el servidor la primera vez que abres el proyecto.

**Codex CLI**: una vez, para todos los proyectos:

```powershell
codex mcp add atalaya --url http://127.0.0.1:7420/mcp
```

La sección **Consultas** del panel muestra cada llamada, con su respuesta y cuántos tokens costó.

## 6. Seguir las llamadas a Codex

Atalaya ve a Codex de dos maneras:

- **En vivo**, cuando se lanza con `scripts\codex.ps1` (el kit de nivel 2 lo instala en el proyecto). El script corre
  `codex exec` en una copia aparte del repositorio, escribe la traza en `%LOCALAPPDATA%\Atalaya\codex`, anota la sesión
  de Claude Code que lo lanzó y deja el informe en `docs/trazabilidad/<fase>/codex/`. Con `-Escribe`, Codex trabaja en
  esa copia y lo que cambie queda como un parche que se revisa antes de aplicarlo.

  ```powershell
  .\scripts\codex.ps1 -Encargo .\encargo.md -Tema revisar-login -Clase E1 -Modelo <modelo de Codex>
  ```

- **Del registro**, por las líneas de `docs/trazabilidad/<fase>/codex/REGISTRO.md`, aunque no haya traza.

Las sesiones interactivas de Codex (el TUI) no se siguen todavía.

## 7. Avisos, decisiones y cuota

- **Avisos:** encendidos por omisión. En la campana de la cabecera se encienden y apagan por tipo (decisiones y
  alertas de error) y se prueban. Si no aparecen, revisa el modo «No molestar» de Windows.
- **Decisiones:** la sección muestra lo que tus sesiones te preguntan. Las que llegan por la herramienta `preguntar` se
  responden en el panel; las demás, en la app del agente.
- **Cuota:** la franja de arriba lee una herramienta aparte, `cuotas.exe` (`-cuotas <carpeta>`), que no viene con
  Atalaya. Sin ella, la franja dice que no hay datos y todo lo demás funciona igual.

## 8. Datos y desinstalar

Atalaya guarda en `%LOCALAPPDATA%\Atalaya` el índice de la datapedia, el diario, la configuración de los avisos y las
trazas de Codex. Para desinstalar, apaga `atalaya.exe`, borra esa carpeta, la del repositorio y, si la copiaste, la
skill de `%USERPROFILE%\.claude\skills\atalaya-proyecto`. En tus proyectos queda lo que instaló el kit (`AGENTS.md`,
`docs/trazabilidad/`…), que es tuyo.

## Si algo falla

| Síntoma | Qué mirar |
| --- | --- |
| «el puerto está ocupado» | Otro `atalaya.exe` ya corre, u otro programa usa el 7420: `-escucha 127.0.0.1:7421`. |
| No aparece ningún agente | Que Claude Code guarde sus transcritos en `%USERPROFILE%\.claude\projects` (si no, `-claude`). |
| Un proyecto no junta sus agentes | Que su nombre en `proyectos.json` sea el de su carpeta. |
| `estado` responde de otro proyecto | El `?proyecto=` del `.mcp.json` del proyecto. |
| La salud dice «por omisión» | El `AGENTS.md` del proyecto no tiene la frase que Atalaya busca: el kit la instala tal cual. |
