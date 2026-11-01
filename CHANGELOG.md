# Changelog · Novedades

## 1.3.0 — 2026-11-01

What Atalaya saves you, the plan's quota in real time, and the kit in English. · Lo que Atalaya te ahorra, la cuota del
plan al momento y el kit en inglés.

**English.**

- **Queries shows what Atalaya saved.** Each MCP query now says which documents its answer replaced, how many tokens
  that kept out of the agent's context, whether the agent read the document anyway, and whether it repeated the same
  question. The Queries view leads with that figure and its method, then breaks it down by tool, project and role, day
  by day (charts with a table view), and lists the status documents agents read when a query would have done. The
  query log has filters and a detail with what each query replaced. Queries are saved, so they survive a restart with
  their client, latency and size, and a worktree counts in its repository. New `GET /api/consultas/resumen`.
- **The plan's quota, read and pushed.** Atalaya reads Claude's plan usage with `claude -p /usage` (5-hour window,
  weekly, weekly per model) every 10 minutes or when you ask, with an interactive history chart. A Claude Code mod can
  also push each change the moment a window moves a point (`POST /api/uso-claude/empujar`, local only).
- **The kit in English.** The `atalaya-proyecto` kit is now in English, and the Spanish kit lives under `es/`; the skill
  installs in the project's or the conversation's language. Atalaya reads progress files, brief assignment lines and
  the `AGENTS.md` rules in both languages, and file names stay the same in both.
- **A faster server.** About 2.3 times more panel requests than 1.2.1 (8,330 instead of 3,564 in 30 s, identical
  responses): per-step marks are computed once, agent logs are parsed once per version, and the queries summary is
  kept until the next change.
- **Clearer alerts and charts.** Alert cards, the alert band and panel messages share one grid with the icon on the
  title's line and severity shown by icon, label and edge. The context chart puts stage and compaction labels in lanes
  that never overlap, with a range selector and a table view. An agent's progress follows its latest milestone when the
  log's first line falls behind, and says so.
- **Steady page numbers.** Every paginator shows the same number of page boxes, all the same width, so the arrows
  never move when you change pages, and on a phone the numbers stay inside their card.
- Smaller: lists by pages keep a constant height with no repeated rows, the privacy step keeps its layout, `preguntar`
  rejects repeated option labels, and the agent detail keeps its width on phones.

**Español.**

- **Consultas dice lo que Atalaya ahorró.** Cada consulta MCP dice ahora qué documentos reemplazó su respuesta, cuántos
  tokens no entraron por eso al contexto del agente, si el agente leyó igual el documento y si repitió la misma
  pregunta. La vista Consultas abre con esa cifra y su método, y la desglosa por herramienta, proyecto y rol, día a día
  (gráficos con vista de tabla), y lista los documentos de estado que los agentes leyeron cuando una consulta habría
  bastado. El registro tiene filtros y un detalle con lo que reemplazó cada consulta. Las consultas se guardan: sobreviven
  a un reinicio con su cliente, su tiempo y su tamaño, y un worktree cuenta en su repositorio. Nuevo
  `GET /api/consultas/resumen`.
- **La cuota del plan, leída y empujada.** Atalaya lee el uso del plan de Claude con `claude -p /usage` (ventana de 5
  horas, semanal y semanal por modelo) cada 10 minutos o cuando lo pides, con un gráfico interactivo de su historia. Un
  mod de Claude Code también puede empujar cada cambio en el momento en que una ventana se mueve un punto
  (`POST /api/uso-claude/empujar`, solo local).
- **El kit en inglés.** El kit `atalaya-proyecto` está ahora en inglés, y el kit en español vive en `es/`; la skill
  instala en la lengua del proyecto o de la conversación. Atalaya lee los archivos de avance, la línea de asignación de
  los encargos y las reglas de `AGENTS.md` en las dos lenguas, y los nombres de archivo son los mismos en ambas.
- **Un servidor más rápido.** Unas 2,3 veces más pedidos del panel que la 1.2.1 (8330 en vez de 3564 en 30 s, con las
  mismas respuestas): las marcas de cada paso se calculan una vez, las bitácoras se leen una vez por versión y el resumen
  de consultas se guarda hasta el próximo cambio.
- **Alertas y gráficos más claros.** Las tarjetas de alerta, la banda y los mensajes del panel comparten una rejilla con
  el icono en la línea del título y la gravedad por icono, etiqueta y borde. El gráfico de contexto pone las etiquetas de
  etapas y compactaciones en carriles que no se pisan, con selector de rango y vista de tabla. El avance de un agente
  sigue a su último hito cuando la primera línea de la bitácora queda atrás, y lo dice.
- **Números de página quietos.** Todo paginador muestra la misma cantidad de casillas, todas del mismo ancho, así que
  las flechas no se mueven al cambiar de página, y en el teléfono los números quedan dentro de su tarjeta.
- Menores: las listas por páginas mantienen el alto y no repiten filas, el paso de privacidad no salta, `preguntar`
  rechaza opciones repetidas y el detalle de un agente conserva su ancho en el teléfono.

## 1.2.1 — 2026-11-01

Minor fixes and polish. · Arreglos menores y pulido.

**English.**

- **Faster server.** It handles about ten times more panel requests than 1.2.0 (measured with real data: 1,904 instead
  of 182 in 30 s) and idles at a lower CPU. Each step's command is normalized once instead of on every request, the
  last quota check before each Codex launch is found in a single pass, secret masking skips patterns whose hints are
  absent and caches files that did not change, and each project's progress is read once per update cycle.
- **An open panel reloads itself** when `atalaya.exe` is updated, instead of running the old panel against the new
  server.
- **Safer write gate.** Apply and undo run one at a time with unique temporary files; a failure to write the change
  history is shown instead of reported as success; retrying a failed undo continues where it stopped instead of
  repeating commands that already ran.
- **Panel fixes.** Lists by pages keep a focusable row for the keyboard, Health no longer shows the previous project or
  period while loading, page numbers do not come back after switching lists, the setup assistant ignores out-of-order
  replies and pauses while the tab is hidden, and the last page of an agent's activity is capped, with a "new" button.
- Smaller: deleted `claude.ps1` records are dropped, recent folders are grouped the same way every time, and the
  skill's state is cached until its files change.

**Español.**

- **Un servidor más rápido.** Atiende unas diez veces más pedidos del panel que la 1.2.0 (medido con datos reales: 1904
  en vez de 182 en 30 s) y en reposo usa menos CPU. La orden de cada paso se normaliza una vez y no en cada pedido, la
  última mirada a la cuota antes de cada lanzamiento de Codex se encuentra en una sola pasada, tapar secretos se salta
  los patrones cuyas pistas no están y guarda los archivos que no cambiaron, y el avance de cada proyecto se lee una vez
  por vuelta.
- **Un panel abierto se recarga solo** cuando se actualiza `atalaya.exe`, en vez de seguir con el panel viejo contra el
  servidor nuevo.
- **Una puerta de escritura más segura.** Aplicar y deshacer van de a uno, con temporales únicos; si no se puede
  escribir el historial de cambios, se muestra en vez de darlo por bueno; reintentar un deshacer que falló sigue donde
  paró en vez de repetir órdenes que ya corrieron.
- **Arreglos del panel.** Las listas por páginas mantienen una fila alcanzable con el teclado, Salud ya no muestra el
  proyecto o el periodo anterior mientras carga, el número de página no vuelve al cambiar de lista, el asistente ignora
  respuestas fuera de orden y se pausa con la pestaña oculta, y la última página de la actividad de un agente tiene
  tope, con un botón de «nuevos».
- Menores: las fichas de `claude.ps1` borradas se sueltan, las carpetas recientes se agrupan siempre igual y el estado
  de la skill se guarda hasta que sus archivos cambian.

## 1.2.0 — 2026-11-01

Set everything up from the panel, Codex as coordinator, Linux and long lists by pages. · Todo se configura desde el
panel, Codex puede coordinar, Linux y las listas largas por páginas.

**English.**

- **Setup assistant.** The first time, the panel opens a step-by-step assistant (afterwards it lives behind the gear in
  the header). It looks at what is on the computer first and only asks what it cannot find out: language, your tools
  (Claude Code and Codex, versions, data folders), who coordinates, your projects (folders your agents worked in over
  the last 30 days, with a level each), the MCP connection, the kit's skill and a request for your agent per project,
  notifications and starting with the system, and privacy. Each step turns green when Atalaya actually sees the result:
  for MCP, when the first query arrives from each tool.
- **A careful write gate.** Atalaya only writes a closed list of things: its project registry and settings, the skill
  folder of each tool, the startup shortcut (Windows) or systemd user unit (Linux), and the official `claude mcp` and
  `codex mcp` commands, run without a shell. Every change is previewed first, backed up, logged and can be undone; undo
  never overwrites later edits, only reverts the steps that actually ran, and can be retried. Only the panel can write,
  never the MCP, and never a file inside your projects.
- **Codex as coordinator.** `scripts/claude.ps1`, the mirror of `codex.ps1`, lets a Codex session hand Claude Code a
  task that runs headless on a throwaway copy: read-only by default, edits only in the copy with `-Escribe` (leaving a
  patch that is never applied on its own), no shell or web except the commands named with `-Ordenes`, never skipping
  permissions and without your Claude Code hooks, plugins or other MCP servers. Atalaya shows each task under the Codex
  session that launched it. The kit installs it, and the reverse split of work, when Codex coordinates.
- **Interactive Codex sessions.** Sessions from the Codex TUI and the desktop app are followed live as agents of their
  project.
- **Privacy.** Leave a project out of the datapedia (its documents, memory and agent traces are no longer indexed) and
  choose how many days the diary keeps.
- **Linux.** Tested on Ubuntu under WSL 2: the panel and the assistant work end to end, data goes to
  `~/.local/share/Atalaya`, CI runs on Ubuntu as well as Windows, and the Windows tools WSL can see are not mistaken for
  Linux ones. Notifications remain Windows-only.
- **Long lists by pages.** The team, commits, documents and questions of a session, the activity of an agent (older
  pages come from the server, so nothing is cut at 3000 steps), Health and answered decisions page like a table, with
  page sizes; data arriving live never moves the page you are reading. The timeline keeps its rows at a fixed height.
- Smaller: progress warns when the latest milestone is later than the file was saved; the kit's `scripts/hito.ps1`
  writes milestones with the system clock.

**Español.**

- **El asistente de configuración.** La primera vez, el panel abre un asistente por pasos (después queda en el
  engranaje de la cabecera). Primero mira qué hay en el equipo y solo pregunta lo que no puede saber: el idioma, tus
  herramientas (Claude Code y Codex, versiones, carpetas de datos), quién coordina, tus proyectos (las carpetas donde
  trabajaron tus agentes en los últimos 30 días, con su nivel), la conexión por MCP, la skill del kit y el pedido para
  tu agente en cada proyecto, los avisos y el arranque con el sistema, y la privacidad. Cada paso se pone en verde
  cuando Atalaya ve el resultado; en el MCP, cuando llega la primera consulta de cada herramienta.
- **Una puerta de escritura con cuidado.** Atalaya solo escribe una lista cerrada: su registro de proyectos y su
  configuración, la carpeta de la skill de cada herramienta, el acceso directo de inicio (Windows) o la unidad de
  systemd del usuario (Linux), y las órdenes oficiales `claude mcp` y `codex mcp`, sin intérprete de órdenes. Cada cambio
  se muestra antes, se respalda, queda en el historial y se puede deshacer; deshacer nunca pisa lo cambiado después,
  revierte solo los pasos que llegaron a hacerse y se puede reintentar. Solo escribe el panel, nunca el MCP, y nunca un
  archivo dentro de tus proyectos.
- **Codex de coordinador.** `scripts/claude.ps1`, el espejo de `codex.ps1`, deja que una sesión de Codex le pase a
  Claude Code un encargo que corre sin pantalla en una copia desechable: por omisión solo lee; con `-Escribe` edita solo
  en la copia y deja un parche que nunca se aplica solo; sin shell ni web salvo las órdenes que se nombren con
  `-Ordenes`; nunca se salta los permisos y no usa tus hooks, plugins ni otros MCP de Claude Code. Atalaya muestra cada
  encargo colgado de la sesión de Codex que lo lanzó. El kit lo instala, con el reparto al revés, cuando coordina Codex.
- **Las sesiones interactivas de Codex.** Las de la TUI y las de la app de escritorio se siguen en vivo como agentes de
  su proyecto.
- **Privacidad.** Un proyecto puede quedar fuera de la datapedia (sus documentos, su memoria y las trazas de sus agentes
  dejan de indexarse), y se elige cuántos días guarda el diario.
- **Linux.** Probado en Ubuntu con WSL 2: el panel y el asistente funcionan de punta a punta, los datos van a
  `~/.local/share/Atalaya`, el CI corre en Ubuntu además de Windows, y las herramientas de Windows que WSL ve no se
  confunden con las de Linux. Los avisos siguen siendo solo de Windows.
- **Las listas largas por páginas.** El equipo, los commits, los documentos y las preguntas de una sesión, la actividad
  de un agente (las páginas viejas llegan del servidor, así que nada se corta en 3000 pasos), Salud y las decisiones
  respondidas se paginan como una tabla, con tamaño de página; lo que llega en vivo nunca mueve la página que lees. La
  línea de tiempo mantiene sus filas en un alto fijo.
- Menores: el avance avisa cuando el último hito es posterior al guardado del archivo; `scripts/hito.ps1` del kit anota
  los hitos con la hora del reloj.

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
