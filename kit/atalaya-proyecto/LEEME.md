# Kit atalaya-proyecto · LEEME

**Versión 2.0.** El kit reúne el centro de operaciones, el contrato de trazabilidad, el avance por fase, el registro en `proyectos.json` y el MCP `atalaya`.

## Qué es

Una forma de trabajar con Claude Code que se lleva a cualquier proyecto, nuevo o ya empezado, en tres niveles. Cada
nivel incluye el anterior y un proyecto sube cuando lo necesita. Se instala con la skill `atalaya-proyecto` (el
`SKILL.md` de esta carpeta), que reconoce el proyecto, pregunta en un solo bloque, muestra un plan archivo por archivo,
espera el sí, instala, registra y comprueba.

| Nivel | Para qué | Lo que instala | Lo que cambia para el dueño |
| --- | --- | --- | --- |
| **1 · indexable** | Todo proyecto: que Atalaya lo entienda y el MCP responda «¿cómo vamos?» | `AGENTS.md` corto con la sección de Atalaya, `.mcp.json`, `ESTADO-para-retomar.md`, `F0/AVANCE.md` con la idea en paquetes, su línea en `proyectos.json` | Casi nada: la IA actualiza el avance al cerrar un paquete y se lo cuenta |
| **2 · con agentes** | Cuando la IA reparte el trabajo entre subagentes o Codex | El centro de operaciones v1: clases de tarea y matriz de modelo y effort (`AGENTES.md`), la plantilla del encargo, cinco roles en `.claude/agents/`, bitácoras de avance, esfuerzo proporcional (`PRIORIZACION.md`), lo temporal con acta, `HERRAMIENTAS.md` y su registro, el freno de heredoc | Aprueba los lanzamientos grandes; la Salud y las Estadísticas de Atalaya empiezan a tener qué decir |
| **3 · completo** | Productos con usuarios, datos o dinero en juego, o que entregan versiones | Las ranuras de producción y de diseño, `version-y-kit.md` y el rol `ensayador` | Más aprobaciones (commits, publicación, aceptación) a cambio de trazabilidad completa |

**Por qué el nivel 2 es grande.** Los roles y la plantilla necesitan los temporales y la priorización. La entrega de versiones y producción corresponden al nivel 3.

## Qué trae

```
atalaya-proyecto/
├─ SKILL.md                 la skill: el procedimiento de instalación
├─ LEEME.md                 este archivo
├─ comun/                   los tres niveles
│  ├─ AGENTS-atalaya.md      la sección de Atalaya, al final de AGENTS.md
│  ├─ .mcp.json
│  ├─ scripts/hito.ps1      anota un hito en AVANCE.md con la hora del reloj
│  └─ docs/trazabilidad/    ESTADO-para-retomar.md y F0/AVANCE.md
├─ nivel-1/AGENTS.plantilla.md
├─ nivel-2/                 el centro de operaciones neutro v1 (sin versiones ni ensayador)
│  ├─ AGENTS.plantilla.md
│  ├─ .claude/              settings.json, hooks/sin-heredoc.mjs, agents/ (cinco roles)
│  ├─ docs/operacion/       README, AGENTES, PRIORIZACION, ENCARGO-PLANTILLA, HERRAMIENTAS(-registro), SKILLS, pruebas
│  ├─ docs/trazabilidad/    ESTADO-historia.md, _plantillas/ACTA-PLANTILLA.md
│  └─ scripts/avance.ps1
└─ nivel-3/                 docs/operacion/version-y-kit.md y .claude/agents/ensayador.md
```

## Las frases que lee Atalaya

Atalaya no copia las reglas: las lee de `AGENTS.md` y las cita por su línea. Estas frases tienen que
quedar escritas así (los números se pueden cambiar). Una regla que el proyecto no escribe se muestra como «no escrita» y
no se vigila.

| Frase | Para qué la usa Atalaya | Nivel |
| --- | --- | --- |
| «Ningún agente pasa de N minutos» | La alerta de tiempo y el rayado de la línea de tiempo | todos |
| «nunca hay dos sobre el mismo archivo» | La alerta de frontera y su columna en la Salud | todos |
| «se mira la cuota» | La columna de cuota de la Salud (si está instalada la herramienta Cuotas) | opcional |
| «Su primera línea es la asignación» | La columna de asignación | 2 y 3 |
| «El effort se fija siempre explícito» | La columna de effort | 2 y 3 |
| «Cada agente lleva una bitácora» | La columna de bitácora | 2 y 3 |
| «El acta es obligatoria» y «Los temporales en `X:\…`» | Las columnas de acta y de temporales, y dónde buscar las carpetas | 2 y 3 |

## Los marcadores

Cada `{{NOMBRE}}` se reemplaza con una respuesta; los opcionales sin respuesta se quitan con su frase o su sección.

| Marcador | Qué es | Ejemplo | Nivel |
| --- | --- | --- | --- |
| `{{PROYECTO}}` | Nombre (el de su carpeta) | Mi-Proyecto | todos |
| `{{QUE_ES}}` | Una o dos frases: qué se construye y para quién | Un sistema de ventas para una tienda de barrio. | 1 |
| `{{DUENO}}` | Quién decide | Ana | todos |
| `{{IDIOMA}}` | Idioma de los documentos | español | todos |
| `{{CONSOLA}}` | La consola del dueño | PowerShell | todos |
| `{{REGLAS_DE_LA_CONSOLA}}` **(opcional)** | Lo que esa consola pide (comillas, rutas, codificación) | «las rutas con espacios van entre comillas» | 2 y 3 |
| `{{DATOS_PROTEGIDOS}}` | Qué datos nunca salen del equipo | clientes, ventas y costos | todos |
| `{{PRUEBAS_BARATAS}}` | Las órdenes rápidas | `go vet; go test` | todos |
| `{{TOPE_MIN}}` | Minutos máximos por agente | `30` | todos |
| `{{URL_DEL_CENTRO}}` | El MCP de Atalaya | `http://127.0.0.1:7420/mcp` | todos |
| `{{PROYECTO_EN_URL}}` | El nombre del proyecto para la dirección del MCP, con los espacios como `%20`: las herramientas `estado`, `retomar` y `salud` lo usan cuando no se les nombra otro | `Mi%20Proyecto` | todos |
| `{{FECHA_DE_INSTALACION}}`, `{{HORA_DE_INSTALACION}}`, `{{NIVEL}}` | Hoy, ahora y el nivel elegido | 2026-11-02, 10:15, 1 | todos |
| `{{DONDE_ESTAMOS}}`, `{{LO_QUE_FALTA}}` | El estado real que se vio al reconocer | | todos |
| `{{AHORA}}` | Lo que viene ahora en la fase 0 | falta empezar P1, el esquema de datos | todos |
| `{{PLAN}}` | Dónde está la idea | `README.md` | todos |
| `{{QUE_ES_LA_FASE}}` | Una o dos frases con lo que deja la fase 0 | La fase 0 deja la primera versión usable en casa. | todos |
| `{{PAQUETES}}` | Una fila por paquete | `\| P1 \| Esquema SQLite \| 3 \| sin empezar \| 0 \| \|` | todos |
| `{{RAIZ}}` | La carpeta del repositorio | `C:\Proyectos\Mi-Proyecto` | 2 y 3 |
| `{{TMP}}` | La de temporales, **fuera** del repositorio | `C:\Proyectos\Mi-Proyecto-tmp` | 2 y 3 |
| `{{RAMA_TRABAJO}}`, `{{RAMA_PRODUCCION}}` | Ramas | `dev` / `main` | 2 y 3 |
| `{{ESPECIFICACION}}` | Dónde está lo que se construye | `docs/Especificacion.md` | 2 y 3 |
| `{{OTRAS_PROHIBICIONES_DE_DATOS}}`, `{{ALCANCE}}`, `{{OTRAS_REGLAS_DE_CODIGO}}` **(opcional)** | Reglas propias del dueño | | 2 y 3 |
| `{{ZONA_DE_RIESGO}}` | Dónde un error cuesta caro | caja, permisos, `deploy/` | 2 y 3 |
| `{{PRUEBAS_CARAS}}`, `{{NOMBRE_DE_LA_BATERIA}}`, `{{ORDEN}}`, `{{DURACION}}` | Las baterías de `pruebas.md` | | 2 y 3 |
| `{{SISTEMA_Y_CONSOLA}}` | Para `HERRAMIENTAS.md` | Windows 11 y PowerShell | 2 y 3 |
| `{{PUERTOS_RESERVADOS}}` **(opcional)** | Puertos que ya usa el equipo (pregunta 10) | 8080 el servidor | 2 y 3 |
| `{{RUTA_DE_MEMORIA}}` **(opcional)** | Carpeta de la memoria de Claude Code: la que muestra la sesión al instalar; si no la ves, se quita la frase | `ruta\a\memory` | 2 y 3 |
| `{{PRODUCCION}}`, `{{DOC_DE_PRODUCCION}}` **(opcional)** | Qué es producción y sus reglas | | 3 |
| `{{DESTINO_DE_ENTREGA}}`, `{{COMO_SE_COMPILA}}` **(opcional)** | Para `version-y-kit.md` | | 3 |
| `{{GUIA_DE_DISENO}}` **(opcional)** | Guía visual aprobada | | 3 |
| `{{ORDEN_DE_LA_DATAPEDIA}}` **(opcional)** | Cómo se consulta un buscador de documentación | | 2 y 3 |

## Cómo se instala la skill

Copia esta carpeta a `%USERPROFILE%\.claude\skills\atalaya-proyecto\` (o a `.claude/skills/` de un proyecto). Desde la
sesión siguiente, Claude Code la ofrece cuando pides preparar o registrar un proyecto. Sin la skill, sirve igual: abre
Claude Code en el proyecto de destino, di «sigue `<ruta>\SKILL.md`» y la ruta del kit.

## Lo que no se pudo verificar

- Los niveles 1, 2 y 3 se ensayaron con proyectos de prueba (uno nuevo y uno que ya tenía sus propias reglas, servidor
  MCP, hooks y documentos de operación) y con una copia de Atalaya que los vio enteros; el nivel 1, además, en un
  proyecto real. La entrega de versiones para un proyecto que se despliega (`version-y-kit.md`, «Si el proyecto se
  despliega») está escrita pero no se ensayó con un despliegue de verdad.
- Los modelos y pesos de `AGENTES.md` §1 son los de octubre de 2026: se confirman al instalar.
- El freno de heredoc se probó en Windows; en Linux o macOS debería comportarse igual, pero no se probó.
