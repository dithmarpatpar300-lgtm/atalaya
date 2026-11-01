# Installing Atalaya

**English** · [Español](INSTALAR.md) · [Back to the README](../README.md)

This guide gets Atalaya running, with your projects registered and connected to Claude Code and Codex. It is tested on
Windows 11; everything you type is PowerShell. The panel is in English and Spanish
(the ES/EN button in the header); the kit is in Spanish.

## 1. Requirements

- **Windows 10 or 11.** (On Linux the panel runs, tried on Ubuntu under WSL 2, without notifications and without
  `scripts/codex.ps1`, which needs Windows; macOS compiles, untested.)
- **Claude Code** (CLI or desktop app) and/or **Codex CLI**: Atalaya follows the agents they launch.
- To build: **Go 1.27 or newer** and **Node.js 20 or newer**, with `npm`. Git to clone.

## 2. Download or build

**Download.** The repository's *Releases* page has one zip per system (`atalaya-<version>-windows-amd64.zip`,
`-linux-amd64`, `-darwin-arm64`) with the executable, this guide, the registry example and the kit. Unzip it into a
folder of your own and continue at step 3; no Go or Node needed.

**Build.**

```powershell
git clone <this repository's URL> atalaya
cd atalaya
.\scripts\compilar.ps1
```

The script installs the panel's dependencies the first time (`npm ci`), builds it, embeds it and leaves **a single
`atalaya.exe`** at the root. With `-Verificar` it also runs formatting, static analysis and every test.

`.\scripts\empaquetar.ps1` builds a release's zips into `dist/` after building; for a single system, `go build
./cmd/atalaya` inside `backend/`, with `GOOS` and `GOARCH` (for example `linux` and `amd64`).

## 3. Run

```powershell
.\atalaya.exe
```

and open **http://127.0.0.1:7420**. With no projects registered, the home page explains how to register the first
one; you will already see the agents it finds in Claude Code's transcripts.

Useful options:

| Option | What it does | Default |
| --- | --- | --- |
| `-escucha 127.0.0.1:7421` | Another port (only `127.0.0.1` or `localhost` are accepted) | `127.0.0.1:7420` |
| `-proyectos <file>` | Another project registry | `proyectos.json` next to the executable |
| `-proyecto "Name=C:\path"` | Adds a project without touching the registry (repeatable) | — |
| `-claude <folder>` | Where Claude Code keeps its transcripts | `%USERPROFILE%\.claude\projects` |
| `-codex <folder>` | Where `codex.ps1` writes Codex traces | `%LOCALAPPDATA%\Atalaya\codex` |
| `-codex-estado <folder>` | Codex's own folder, with the interactive sessions in `sessions\` (empty: not followed) | `%CODEX_HOME%`, or `%USERPROFILE%\.codex` |
| `-avisos=false` | No Windows notifications | on |
| `-diario <folder>` | Where the diary goes (empty: no diary) | `%LOCALAPPDATA%\Atalaya\diario` |
| `-db <file>` | The datapedia index (rebuilt if deleted) | `%LOCALAPPDATA%\Atalaya\atalaya.db` |
| `-demo` | A demo world: made-up projects, agents, alerts and decisions, to try it without any agent | off |
| `-demo-carpeta <folder>` | Where that world is written (wiped and rebuilt at startup) | the temp folder |

To start it with Windows, add a shortcut to the Startup folder:

```powershell
$s = (New-Object -ComObject WScript.Shell).CreateShortcut("$env:APPDATA\Microsoft\Windows\Start Menu\Programs\Startup\Atalaya.lnk")
$s.TargetPath = (Resolve-Path .\atalaya.exe).Path; $s.WorkingDirectory = (Get-Location).Path; $s.WindowStyle = 7; $s.Save()
```

## 4. Register your projects

Atalaya groups agents by the project's **folder**, so the registered name is the folder's name.

**With the skill (recommended).** Copy the skill into Claude Code once:

```powershell
Copy-Item -Recurse .\kit\atalaya-proyecto "$env:USERPROFILE\.claude\skills\atalaya-proyecto"
```

Then open Claude Code **inside your project** and ask «prepara este proyecto para Atalaya» ("prepare this project for
Atalaya"). The skill reads the project, asks you one block of questions, shows you the plan and, once you say yes:

- installs the kit at the level you choose: **1 · indexable** (for any project), **2 · with agents** (roles in
  `.claude/agents/`, a heredoc guard, operating documents and `scripts/codex.ps1`) or **3 · complete** (release
  delivery and a rehearsal role);
- opens phase 0 in `docs/trazabilidad/F0/AVANCE.md` with its work packages and hours;
- **merges** with what you already have (`AGENTS.md`, `CLAUDE.md`, `.mcp.json`, `.claude/settings.json`) without
  overwriting anything;
- registers the project in Atalaya, which shows it within seconds without restarting.

With Codex, or without the skill, ask the agent to follow `kit/atalaya-proyecto/SKILL.md` from this repository.

**By hand.** Copy `proyectos.ejemplo.json` to `proyectos.json` next to `atalaya.exe` and keep one line per project:

```json
[
  { "nombre": "My-Project", "ruta": "C:\\Projects\\My-Project", "sigla": "MP", "nivel": 1 }
]
```

Atalaya rereads the file by itself. What each level expects to find in the project is in
`kit/atalaya-proyecto/LEEME.md`.

## 5. Connect the agents over MCP

This lets agents ask Atalaya ("how are we doing?", "what did the coder do?", "who touched this file?") instead of
rereading documents. Atalaya must be running.

**Claude Code**: a `.mcp.json` at the project root (the skill creates it). `proyecto` is the folder name, with spaces as
`%20`; with it, `estado`, `retomar` and `salud` answer for that project without the agent naming it.

```json
{
  "mcpServers": {
    "atalaya": { "type": "http", "url": "http://127.0.0.1:7420/mcp?proyecto=My-Project" }
  }
}
```

Claude Code asks you to approve the server the first time you open the project.

**Codex CLI**: once, for every project:

```powershell
codex mcp add atalaya --url http://127.0.0.1:7420/mcp
```

The panel's **Consultas** section shows every call, with its answer and how many tokens it cost.

## 6. Following Codex runs

Atalaya sees Codex in three ways:

- **Live**, when it is launched with `scripts\codex.ps1` (the level 2 kit installs it in the project). The script runs
  `codex exec` in a separate copy of the repository, writes the trace to `%LOCALAPPDATA%\Atalaya\codex`, records the
  Claude Code session that launched it and leaves the report in `docs/trazabilidad/<phase>/codex/`. With `-Escribe`,
  Codex works in that copy and whatever it changes comes back as a patch to review before applying.

  ```powershell
  .\scripts\codex.ps1 -Encargo .\task.md -Tema review-login -Clase E1 -Modelo <Codex model>
  ```

- **From the registry**, through the lines of `docs/trazabilidad/<phase>/codex/REGISTRO.md`, even without a trace.

- **Interactive sessions** (the TUI, plain `codex`, and those opened from the Codex app), live: Codex keeps each one in
  `%USERPROFILE%\.codex\sessions\YYYY\MM\DD\rollout-*.jsonl` and Atalaya reads it (read-only) as a Codex agent of the
  project whose folder contains the session's working folder, labelled «Codex · interactiva», with its steps, model,
  effort and tokens. It is *working* while a turn is open, *waiting* for half an hour after the turn ends (Codex leaves
  no mark when the TUI closes) and then *ended*. Sessions from the last 30 days are read; compressed ones (`.jsonl.zst`)
  and `auth.json` are never read.

## 7. Notifications, decisions and quota

- **Notifications:** on by default. The bell in the header turns each type (decisions and error alerts) on or off and
  sends a test. If none show up, check Windows' Do Not Disturb.
- **Decisions:** the section shows what your sessions ask you. Those that arrive through the `preguntar` tool are
  answered in the panel; the rest, in the agent's app.
- **Quota:** the strip at the top reads a separate tool, `cuotas.exe` (`-cuotas <folder>`), which does not ship with
  Atalaya. Without it, the strip says there is no data and everything else works the same.

## 8. Data and uninstalling

Atalaya keeps the datapedia index, the diary, the notification settings and the Codex traces in
`%LOCALAPPDATA%\Atalaya` (on Linux, `~/.local/share/Atalaya` or `$XDG_DATA_HOME/Atalaya`; on macOS,
`~/Library/Application Support/Atalaya`). To uninstall, stop `atalaya.exe` and delete that folder, the repository folder and, if you
copied it, the skill in `%USERPROFILE%\.claude\skills\atalaya-proyecto`. What the kit installed in your projects
(`AGENTS.md`, `docs/trazabilidad/`…) stays: it is yours.

## Troubleshooting

| Symptom | What to check |
| --- | --- |
| "port already in use" | Another `atalaya.exe` is running, or another program uses 7420: `-escucha 127.0.0.1:7421`. |
| No agents appear | That Claude Code keeps its transcripts in `%USERPROFILE%\.claude\projects` (otherwise, `-claude`). |
| A project does not group its agents | That its name in `proyectos.json` is its folder's name. |
| `estado` answers for another project | The `?proyecto=` in the project's `.mcp.json`. |
| Health says "por omisión" (by default) | The project's `AGENTS.md` lacks the phrase Atalaya looks for: the kit installs it verbatim. |
