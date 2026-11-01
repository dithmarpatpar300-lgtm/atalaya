# Changelog · Novedades

## 1.1.0 — 2026-10-31

The panel in English, a demo mode and CI. · El panel en inglés, un modo demo y CI.

**English.**

- **English panel.** An ES/EN button in the header switches the whole panel; it is remembered per browser and, until
  you choose, follows the browser's language. Numbers use a decimal point in English and a comma in Spanish, and the
  messages the server builds (alerts, health checks, progress warnings) are translated too. Your projects' own text
  (names, briefs, plans, decisions) stays as written.
- **Progress reads the plan.** While a new phase's `AVANCE.md` table has no packages yet, Atalaya takes them from
  the package table of the document on its `Plan:` line, all not started, and says so. A row with only a name is a
  placeholder, not a package, and long package names no longer break the layout.
- **Demo mode.** `atalaya.exe -demo` fills a throwaway folder with sample projects and agents, so you can try the panel
  without your own; the README screenshots come from it.
- The Cuotas strip is hidden when the Cuotas tool is not installed, and a level 1 project is asked for its 5 files.
- CI on GitHub Actions builds and checks every push and pull request on a clean Windows machine; every clone gets LF
  line endings.

**Español.**

- **El panel en inglés.** Un botón ES/EN en la cabecera cambia todo el panel; se recuerda en cada navegador y, hasta
  que eliges, sigue el idioma del navegador. Las cifras van con punto decimal en inglés y con coma en español, y los
  mensajes que arma el servidor (alertas, chequeos de salud, avisos del avance) también se traducen. El texto de tus
  proyectos (nombres, encargos, planes, decisiones) queda como se escribió.
- **El avance lee el plan.** Mientras la tabla de `AVANCE.md` de una fase nueva no tiene paquetes, Atalaya los
  toma de la tabla de paquetes del documento de su línea `Plan:`, todos sin empezar, y lo avisa. Una fila con solo el nombre es
  un marcador, no un paquete, y los nombres largos ya no rompen la vista.
- **Modo demo.** `atalaya.exe -demo` llena una carpeta desechable con proyectos y agentes de muestra, para probar el
  panel sin los tuyos; de ahí salen las capturas del README.
- La franja de Cuotas se oculta si la herramienta Cuotas no está instalada, y a un proyecto de nivel 1 se le piden sus
  5 archivos.
- CI en GitHub Actions compila y revisa cada push y pull request en un Windows limpio; todo clon recibe finales de línea
  LF.

## 1.0.0 — 2026-10-31

First stable release. · Primera versión estable.

**English.** Atalaya follows Claude Code sessions and subagents and Codex runs live, step by step; checks each agent
against the rules in the project's `AGENTS.md`; sends Windows notifications for pending decisions and error alerts;
answers agents over MCP (`que_hizo`, `buscar`, `archivo`, `salud`, `estado`, `retomar`, `preguntar`); computes phase
progress from `AVANCE.md`; keeps a daily diary that survives transcript cleanup; shows a timeline, statistics per class
and model and a weekly summary; searches project documents and agent turns; and ships the `atalaya-proyecto` skill,
which installs a tiered operating kit (levels 1 to 3) in any project and registers it. Tested on Windows 11 with
Claude Code and Codex CLI.

**Español.** Atalaya sigue en vivo, paso a paso, las sesiones y subagentes de Claude Code y las llamadas a Codex;
revisa cada agente contra las reglas del `AGENTS.md` del proyecto; avisa por Windows de las decisiones que esperan y de
las alertas de error; responde a los agentes por MCP (`que_hizo`, `buscar`, `archivo`, `salud`, `estado`, `retomar`,
`preguntar`); calcula el avance de la fase desde `AVANCE.md`; guarda un diario por día que sobrevive al borrado de los
transcritos; muestra la línea de tiempo, las estadísticas por clase y modelo y un resumen por semana; busca en los
documentos y en los turnos de los agentes; y trae la skill `atalaya-proyecto`, que instala un kit de operación por
niveles (1 a 3) en cualquier proyecto y lo registra. Probado en Windows 11 con Claude Code y Codex CLI.
