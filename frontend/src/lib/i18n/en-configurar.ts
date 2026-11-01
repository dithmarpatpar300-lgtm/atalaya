// Inglés de la configuración: el asistente de la primera vez y la sección Configurar (components/panel/configurar.tsx),
// y su entrada en la cabecera y la paleta. Clave: la frase en español tal como la pasa t(); valor: la misma en inglés.
export const EN_CONFIGURAR: Record<string, string> = {
  // La sección y sus pasos
  "Configurar Atalaya": "Set up Atalaya",
  "Atalaya mira primero qué hay en este equipo y solo te pregunta lo que no puede saber. Cada cambio se muestra antes de hacerse y se puede deshacer.":
    "Atalaya first looks at what is on this computer and only asks what it cannot find out. Every change is shown before it is made and can be undone.",
  "Saltar por ahora": "Skip for now",
  Idioma: "Language",
  "Tus herramientas": "Your tools",
  "Quién coordina": "Who coordinates",
  "Conectar por MCP": "Connect over MCP",
  Anterior: "Back",
  "Siguiente: {paso}": "Next: {paso}",

  // Idioma
  "El idioma del panel. Se guarda en este navegador y se puede cambiar cuando quieras con el botón ES/EN de la cabecera.":
    "The panel's language. It is saved in this browser and you can change it any time with the ES/EN button in the header.",

  // Herramientas
  "Atalaya {version} en {sistema}. Esto es lo que encontró de Claude Code y de Codex; con una de las dos alcanza.":
    "Atalaya {version} on {sistema}. This is what it found of Claude Code and Codex; either one is enough.",
  Instalada: "Installed",
  "No está en el PATH": "Not on the PATH",
  "Tiene datos en {ruta}": "Has data in {ruta}",
  "Sin datos todavía en {ruta}": "No data yet in {ruta}",
  "Conectada a Atalaya (configuración global)": "Connected to Atalaya (global settings)",
  "Sin conexión global (puede ir por proyecto, en .mcp.json)": "No global connection (it can go per project, in .mcp.json)",
  "Sin conexión a Atalaya": "Not connected to Atalaya",
  "Cómo instalar {nombre}": "How to install {nombre}",
  "Comprobar de nuevo": "Check again",
  "Comprobando…": "Checking…",

  // Quién coordina
  "Quién reparte el trabajo y quién ayuda. Atalaya pone en la raíz del árbol a la sesión que coordina y, en cada proyecto con agentes, el kit lleva las reglas para ese reparto.":
    "Who splits up the work and who helps. Atalaya puts the coordinating session at the root of the tree and, in every project with agents, the kit carries the rules for that split.",
  "Claude Code coordina, Codex ayuda": "Claude Code coordinates, Codex helps",
  "Codex coordina, Claude Code ayuda": "Codex coordinates, Claude Code helps",
  "Solo Claude Code": "Claude Code only",
  "Solo Codex": "Codex only",
  "La sesión de Claude Code reparte el trabajo y le pasa a Codex lo que tiene la especificación cerrada.":
    "The Claude Code session splits up the work and hands Codex whatever has a closed spec.",
  "La sesión de Codex reparte el trabajo y le pasa a Claude Code el diseño y la depuración difícil.":
    "The Codex session splits up the work and hands Claude Code the design and the hard debugging.",
  "Sin Codex: Atalaya sigue a tus sesiones y subagentes de Claude Code.": "Without Codex: Atalaya follows your Claude Code sessions and subagents.",
  "Sin Claude Code: Atalaya sigue a tus sesiones de Codex, de la terminal y de la app.":
    "Without Claude Code: Atalaya follows your Codex sessions, from the terminal and from the app.",
  "Con Codex de principal, Codex le pasa a Claude Code el diseño y la depuración con scripts/claude.ps1, que corre sin pantalla, solo lee salvo que se le pida escribir y nunca aplica su parche solo. Atalaya lo muestra colgado de la sesión de Codex que lo lanzó.":
    "With Codex as primary, Codex hands Claude Code the design and debugging through scripts/claude.ps1, which runs headless, only reads unless asked to write, and never applies its own patch. Atalaya shows it under the Codex session that launched it.",
  guardado: "saved",
  Guardar: "Save",
  "Guardando…": "Saving…",

  // Proyectos
  "Atalaya agrupa a los agentes por la carpeta del proyecto. Estas son las carpetas donde trabajaron tus agentes en los últimos 30 días; marca las que quieres seguir.":
    "Atalaya groups agents by the project's folder. These are the folders your agents worked in over the last 30 days; tick the ones you want to follow.",
  "No hay carpetas nuevas con agentes recientes.": "No new folders with recent agents.",
  "Registrar {nombre}": "Register {nombre}",
  "Claude: {n} sesiones": "Claude: {n} sessions",
  "Claude: 1 sesión": "Claude: 1 session",
  "Codex: 1 sesión": "Codex: 1 session",
  "Codex: {n} sesiones": "Codex: {n} sessions",
  "última {cuando}": "last {cuando}",
  ".mcp.json con Atalaya": ".mcp.json with Atalaya",
  "Nivel de {nombre}": "Level of {nombre}",
  "1 · Mirar": "1 · Watch",
  "2 · Con agentes": "2 · With agents",
  "3 · Completo": "3 · Complete",
  "Solo lo registra: Atalaya sigue a sus agentes.": "Only registers it: Atalaya follows its agents.",
  "Además, el kit de operación: roles, reglas y avance por fase.": "Plus the operating kit: roles, rules and progress per phase.",
  "Además, entregas por versión y un rol de ensayo.": "Plus versioned releases and a rehearsal role.",
  "Otra carpeta": "Another folder",
  "Otra carpeta (ruta completa, por ejemplo {ejemplo})": "Another folder (full path, for example {ejemplo})",
  Agregar: "Add",
  "Ya registrados": "Already registered",
  "la carpeta ya no existe": "the folder no longer exists",
  Quitar: "Remove",
  "No quitar": "Keep",
  "Ver el cambio": "Review the change",

  // La vista previa y el historial
  "Cambia {ruta}": "Changes {ruta}",
  "Crea {ruta}": "Creates {ruta}",
  "Antes de escribir, Atalaya guarda una copia; desde el historial de abajo se puede deshacer.":
    "Before writing, Atalaya keeps a copy; it can be undone from the history below.",
  "Atalaya corre esta orden sin intérprete de órdenes; desde el historial de abajo se puede deshacer (corre la orden inversa).":
    "Atalaya runs this command without a shell; it can be undone from the history below (by running the inverse command).",
  Aplicar: "Apply",
  "Aplicando…": "Applying…",
  Cancelar: "Cancel",
  "Historial de cambios": "Change history",
  Deshacer: "Undo",
  deshecho: "undone",
  falló: "failed",

  // Conectar por MCP
  "Por MCP tus agentes le preguntan a Atalaya («¿cómo vamos?», «¿qué hizo el codificador?») en vez de releer documentos. Cada herramienta se conecta una vez; Atalaya lo ve en cuanto llega la conexión.":
    "Over MCP your agents ask Atalaya (\"how are we doing?\", \"what did the coder do?\") instead of rereading documents. Each tool connects once; Atalaya sees it as soon as the connection arrives.",
  "conectada para todos los proyectos": "connected for every project",
  "conectada en algunos proyectos (.mcp.json)": "connected in some projects (.mcp.json)",
  "sin conectar": "not connected",
  conectada: "connected",
  "Para todos tus proyectos, una vez (en un proyecto con el kit, su .mcp.json ya lo hace y además le dice a Atalaya de qué proyecto se trata):":
    "For all your projects, once (in a project with the kit, its .mcp.json already does it and also tells Atalaya which project it is):",
  "Una vez, para todos tus proyectos:": "Once, for all your projects:",
  "Conectar por mí": "Connect it for me",
  Desconectar: "Disconnect",
  "Prefiero hacerlo yo": "I would rather do it myself",
  "No está instalada en este equipo.": "It is not installed on this computer.",
  Copiada: "Copied",

  "Para comprobarlo, abre una sesión nueva y pídele: «pregúntale a Atalaya cómo vamos». El paso se pone en verde cuando llega esa consulta.":
    "To check it, open a new session and ask it: \"ask Atalaya how we are doing\". The step turns green when that query arrives.",
  "consultó a Atalaya {cuando}": "queried Atalaya {cuando}",
  "se conectó {cuando}; falta su primera consulta": "connected {cuando}; its first query is still missing",
  "{estado}; falta su primera consulta": "{estado}; its first query is still missing",

  // El kit en cada proyecto
  "El kit en cada proyecto": "The kit in each project",
  "Atalaya no toca tus proyectos: el kit (reglas, roles, avance por fase) lo instala tu agente con la skill atalaya-proyecto, que funde los textos con lo que el proyecto ya tiene. Aquí Atalaya copia la skill a tu herramienta y te da el pedido para cada proyecto.":
    "Atalaya does not touch your projects: your agent installs the kit (rules, roles, progress per phase) with the atalaya-proyecto skill, which merges the texts with what the project already has. Here Atalaya copies the skill into your tool and gives you the request for each project.",
  "Elige primero quién coordina: la skill va a esa herramienta.": "Choose who coordinates first: the skill goes to that tool.",
  "La skill en {nombre}": "The skill in {nombre}",
  "al día": "up to date",
  "otra versión": "another version",
  "no está": "missing",
  "sin el kit junto al ejecutable": "no kit next to the executable",
  "Copiar la skill": "Copy the skill",
  "Preparando la vista previa…": "Preparing the preview…",
  "Poner la skill al día": "Update the skill",
  "Todavía no hay proyectos registrados.": "No projects registered yet.",
  "Usa la skill atalaya-proyecto para preparar el proyecto {ruta} en el nivel {nivel}. Atalaya está en {origen}.":
    "Use the atalaya-proyecto skill to prepare the project {ruta} at level {nivel}. Atalaya is at {origen}.",
  "kit instalado": "kit installed",
  "sin AGENTS.md": "no AGENTS.md",
  "falta el kit": "kit missing",
  "El pedido para {agente}": "The request for {agente}",
  "tu agente": "your agent",
  "Ábrelo en una sesión nueva en esa carpeta. Atalaya lo pone en verde cuando ve AGENTS.md y el avance de la fase 0.":
    "Open it in a new session in that folder. Atalaya turns it green when it sees AGENTS.md and the phase 0 progress.",

  // Avisos y arranque
  "Avisos y arranque": "Notifications and startup",
  Avisos: "Notifications",
  "Aviso de prueba enviado: debería verse en la esquina de la pantalla.": "Test notification sent: it should show in the corner of the screen.",
  "Los avisos del sistema están apagados en este equipo (por ahora solo hay en Windows); lo que te espera se ve igual en la campana del panel.":
    "System notifications are off on this computer (for now they only exist on Windows); what is waiting for you still shows in the panel's bell.",
  "Cuando un agente te espera con una decisión": "When an agent is waiting on a decision from you",
  "Cuando salta una alerta de error": "When an error alert fires",
  "Enviar un aviso de prueba": "Send a test notification",
  "Arrancar con el sistema": "Start with the system",
  "arranca al iniciar sesión": "starts when you sign in",
  "no arranca solo": "does not start on its own",
  "En este sistema Atalaya no sabe arrancar solo; la guía de instalación dice cómo.":
    "On this system Atalaya cannot start on its own; the install guide explains how.",
  "Un acceso directo a {programa} en tu carpeta de inicio de Windows.": "A shortcut to {programa} in your Windows Startup folder.",
  "Una unidad de systemd de tu usuario que corre {programa} al iniciar sesión.": "A systemd user unit that runs {programa} when you sign in.",
  "Que no arranque solo": "Stop starting on its own",

  // Privacidad y datos
  "Privacidad y datos": "Privacy and data",
  "Todo queda en este equipo: Atalaya no manda nada afuera. Aquí eliges qué proyectos se indexan para buscar y cuánto recuerda el diario.":
    "Everything stays on this computer: Atalaya sends nothing out. Here you choose which projects are indexed for search and how much the diary remembers.",
  "La datapedia": "The datapedia",
  "Un proyecto fuera de la datapedia se sigue igual en el panel, pero sus documentos, su memoria y las trazas de sus agentes no se indexan ni salen en las búsquedas.":
    "A project left out of the datapedia is still followed in the panel, but its documents, its memory and its agents' traces are not indexed and do not show up in searches.",
  "en la datapedia": "in the datapedia",
  "fuera de la datapedia": "left out of the datapedia",
  Sacar: "Leave out",
  "Volver a indexar": "Index again",
  "El diario": "The diary",
  "Un archivo por día con lo que hicieron tus agentes, para las estadísticas y la línea de lo que los transcritos ya no tienen.":
    "One file per day with what your agents did, for the statistics and the timeline of what the transcripts no longer have.",
  "Elige primero quién coordina: esta preferencia se guarda junto a esa.": "Choose who coordinates first: this preference is saved together with that one.",
  "Días que guarda el diario": "Days the diary keeps",
  "{n} días": "{n} days",
  Siempre: "Forever",
  "Preparando…": "Preparing…",
  "Elige un cambio arriba: su vista previa aparece aquí, antes de aplicarlo.": "Choose a change above: its preview shows here before you apply it.",

  // Listo
  "Ni Claude Code ni Codex están instalados todavía": "Neither Claude Code nor Codex is installed yet",
  "Falta elegir quién coordina": "Who coordinates is not chosen yet",
  "1 proyecto registrado": "1 project registered",
  "{n} proyectos registrados": "{n} projects registered",
  "Ningún proyecto registrado todavía": "No project registered yet",
  "Conectadas a Atalaya: {lista}": "Connected to Atalaya: {lista}",
  "Ninguna herramienta conectada por MCP": "No tool connected over MCP",
  "Elige quién coordina para terminar; lo demás se puede completar después.": "Choose who coordinates to finish; the rest can be completed later.",
  "Todo esto queda en Configuración (el engranaje de la cabecera) para cambiarlo cuando quieras.":
    "All of this stays in Setup (the gear in the header) so you can change it any time.",
  "Ir a la portada": "Go to the home page",
};
