# atalaya-proyecto kit · README

**Version 2.0.** The kit brings together the operations center, the traceability contract, per-phase progress, the registration in `proyectos.json` and the `atalaya` MCP.

The Spanish version of this kit is in `es/` (same layout; start at `es/LEEME.md`). File and folder names are part of Atalaya's contract and are the same in both languages (`AVANCE.md`, `ESTADO-para-retomar.md`, `docs/trazabilidad/`, `docs/operacion/`, `comun`, `nivel-1` to `nivel-3`…).

## What it is

A way of working with Claude Code that can be taken to any project, new or already under way, in three levels. Each
level includes the previous one, and a project goes up when it needs to. It is installed with the `atalaya-proyecto`
skill (the `SKILL.md` in this folder), which recognizes the project, asks in a single block, shows a file-by-file plan,
waits for the yes, installs, registers and checks.

| Level | What for | What it installs | What changes for the owner |
| --- | --- | --- | --- |
| **1 · indexable** | Every project: so Atalaya understands it and the MCP answers "how are we doing?" | A short `AGENTS.md` with the Atalaya section, `.mcp.json`, `ESTADO-para-retomar.md`, `F0/AVANCE.md` with the idea split into packages, its line in `proyectos.json` | Almost nothing: the AI updates the progress when it closes a package and tells the owner |
| **2 · with agents** | When the AI hands work out to subagents or Codex | Operations center v1: task classes and the model and effort matrix (`AGENTES.md`), the brief template, five roles in `.claude/agents/`, progress logs, proportional effort (`PRIORIZACION.md`), scratch files with a record, `HERRAMIENTAS.md` and its log, the heredoc stop | The owner approves the big launches; Atalaya's Health and Statistics start to have something to say |
| **3 · complete** | Products with users, data or money at stake, or that deliver versions | The production and design slots, `version-y-kit.md` and the `ensayador` (tester) role | More approvals (commits, publishing, acceptance) in exchange for full traceability |

**Why level 2 is big.** The roles and the template need the scratch files and the prioritization. Delivering versions and production belong to level 3.

## What it contains

```
atalaya-proyecto/
├─ SKILL.md                 the skill: the installation procedure
├─ README.md                this file
├─ es/                      the same kit in Spanish (es/LEEME.md is its README)
├─ comun/                   all three levels
│  ├─ AGENTS-atalaya.md      the Atalaya section, at the end of AGENTS.md
│  ├─ .mcp.json
│  ├─ scripts/hito.ps1      writes a milestone into AVANCE.md with the clock time
│  └─ docs/trazabilidad/    ESTADO-para-retomar.md and F0/AVANCE.md
├─ nivel-1/AGENTS.plantilla.md
├─ nivel-2/                 the neutral operations center v1 (no versions, no ensayador)
│  ├─ AGENTS.plantilla.md
│  ├─ .claude/              settings.json, hooks/sin-heredoc.mjs, agents/ (five roles)
│  ├─ docs/operacion/       README, AGENTES, PRIORIZACION, ENCARGO-PLANTILLA, HERRAMIENTAS(-registro), SKILLS, pruebas
│  ├─ docs/trazabilidad/    ESTADO-historia.md, _plantillas/ACTA-PLANTILLA.md
│  ├─ delegar/              the split between the two tools, a section of AGENTS.md depending on who coordinates
│  └─ scripts/              avance.ps1; codex.ps1 (Claude Code coordinates) or claude.ps1 (Codex coordinates)
└─ nivel-3/                 docs/operacion/version-y-kit.md and .claude/agents/ensayador.md
```

## The phrases Atalaya reads

Atalaya does not copy the rules: it reads them from `AGENTS.md` and quotes them by line. These phrases have to be
written like this (the numbers can change). A rule the project does not write is shown as "not written" and is not
watched.

| Phrase | What Atalaya uses it for | Level |
| --- | --- | --- |
| "No agent runs longer than N minutes" | The time alert and the striping of the timeline | all |
| "never two on the same file" | The file-boundary alert and its column in Health | all |
| "the quota is checked" | The quota column in Health (if the Quotas tool is installed) | optional |
| "Its first line is the assignment" | The assignment column | 2 and 3 |
| "The effort is always set explicitly" | The effort column | 2 and 3 |
| "Every agent keeps a log" | The log column | 2 and 3 |
| "The record is mandatory" and "Scratch files in `X:\…`" | The record and scratch-files columns, and where to look for the folders | 2 and 3 |

## The placeholders

Each `{{NAME}}` is replaced with an answer; optional ones with no answer are removed together with their sentence or section.

| Placeholder | What it is | Example | Level |
| --- | --- | --- | --- |
| `{{PROYECTO}}` | Name (that of its folder) | Mi-Proyecto | all |
| `{{QUE_ES}}` | One or two sentences: what is being built and for whom | A sales system for a neighborhood shop. | 1 |
| `{{DUENO}}` | Who decides (the owner) | Ana | all |
| `{{IDIOMA}}` | Language of the documents | English | all |
| `{{CONSOLA}}` | The owner's shell | PowerShell | all |
| `{{REGLAS_DE_LA_CONSOLA}}` **(optional)** | What that shell requires (quotes, paths, encoding) | "paths with spaces go in quotes" | 2 and 3 |
| `{{DATOS_PROTEGIDOS}}` | Which data never leaves the machine | customers, sales and costs | all |
| `{{PRUEBAS_BARATAS}}` | The quick commands | `go vet; go test` | all |
| `{{TOPE_MIN}}` | Maximum minutes per agent | `30` | all |
| `{{URL_DEL_CENTRO}}` | Atalaya's MCP | `http://127.0.0.1:7420/mcp` | all |
| `{{PROYECTO_EN_URL}}` | The project name for the MCP address, with spaces as `%20`: the `estado`, `retomar` and `salud` tools use it when no other is named | `Mi%20Proyecto` | all |
| `{{FECHA_DE_INSTALACION}}`, `{{HORA_DE_INSTALACION}}`, `{{NIVEL}}` | Today, now and the chosen level | 2026-11-02, 10:15, 1 | all |
| `{{DONDE_ESTAMOS}}`, `{{LO_QUE_FALTA}}` | The real status seen when recognizing the project | | all |
| `{{AHORA}}` | What comes now in phase 0 | start P1, the data schema | all |
| `{{PLAN}}` | Where the idea is | `README.md` | all |
| `{{QUE_ES_LA_FASE}}` | One or two sentences on what phase 0 leaves behind | Phase 0 leaves the first version usable at home. | all |
| `{{PAQUETES}}` | One row per package | `\| P1 \| SQLite schema \| 3 \| not started \| 0 \| \|` | all |
| `{{RAIZ}}` | The repository folder | `C:\Proyectos\Mi-Proyecto` | 2 and 3 |
| `{{TMP}}` | The scratch-files folder, **outside** the repository | `C:\Proyectos\Mi-Proyecto-tmp` | 2 and 3 |
| `{{RAMA_TRABAJO}}`, `{{RAMA_PRODUCCION}}` | Branches | `dev` / `main` | 2 and 3 |
| `{{ESPECIFICACION}}` | Where what is being built is described | `docs/Especificacion.md` | 2 and 3 |
| `{{OTRAS_PROHIBICIONES_DE_DATOS}}`, `{{ALCANCE}}`, `{{OTRAS_REGLAS_DE_CODIGO}}` **(optional)** | The owner's own rules | | 2 and 3 |
| `{{ZONA_DE_RIESGO}}` | Where a mistake is costly | cash register, permissions, `deploy/` | 2 and 3 |
| `{{PRUEBAS_CARAS}}`, `{{NOMBRE_DE_LA_BATERIA}}`, `{{ORDEN}}`, `{{DURACION}}` | The batteries in `pruebas.md` | | 2 and 3 |
| `{{SISTEMA_Y_CONSOLA}}` | For `HERRAMIENTAS.md` | Windows 11 and PowerShell | 2 and 3 |
| `{{PUERTOS_RESERVADOS}}` **(optional)** | Ports the machine already uses (question 10) | 8080 the server | 2 and 3 |
| `{{RUTA_DE_MEMORIA}}` **(optional)** | Claude Code's memory folder: the one the session shows at install; if you cannot see it, the sentence is removed | `path\to\memory` | 2 and 3 |
| `{{PRODUCCION}}`, `{{DOC_DE_PRODUCCION}}` **(optional)** | What production is and its rules | | 3 |
| `{{DESTINO_DE_ENTREGA}}`, `{{COMO_SE_COMPILA}}` **(optional)** | For `version-y-kit.md` | | 3 |
| `{{GUIA_DE_DISENO}}` **(optional)** | Approved visual guide | | 3 |
| `{{ORDEN_DE_LA_DATAPEDIA}}` **(optional)** | How a documentation search tool is queried | | 2 and 3 |

## How to install the skill

Copy this folder to `%USERPROFILE%\.claude\skills\atalaya-proyecto\` (or to a project's `.claude/skills/`). From the next
session, Claude Code offers it when you ask to set up or register a project. Without the skill it works just as well:
open Claude Code in the target project, say "follow `<path>\SKILL.md`" and give the path of the kit.

## What could not be verified

- Levels 1, 2 and 3 were rehearsed with test projects (a new one and one that already had its own rules, MCP server,
  hooks and operations documents) and with a copy of Atalaya that saw them whole; level 1, in addition, on a real
  project. Version delivery for a project that is deployed (`version-y-kit.md`, "If the project is deployed") is
  written but was not rehearsed with a real deployment.
- The models and weights of `AGENTES.md` §1 are those of October 2026: they are confirmed at install.
- The heredoc stop was tested on Windows; on Linux or macOS it should behave the same, but it was not tested.
