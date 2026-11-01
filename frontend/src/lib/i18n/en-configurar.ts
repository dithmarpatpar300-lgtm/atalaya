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
  "Con Codex de principal, Atalaya ya sigue sus sesiones; que Codex le pase trabajo a Claude Code con su traza (claude.ps1) llega en la próxima entrega.":
    "With Codex as primary, Atalaya already follows its sessions; Codex handing work to Claude Code with its own trace (claude.ps1) comes in the next release.",
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
