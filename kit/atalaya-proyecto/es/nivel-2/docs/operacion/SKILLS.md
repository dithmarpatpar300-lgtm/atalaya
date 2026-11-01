# Skills, plugins y apoyos · {{PROYECTO}}

Qué apoyos existen **de verdad** en este equipo, cuándo valen la pena y cuándo no.

- **Los nombres exactos** son los de la lista de skills de la sesión: no se adivinan. Si un nombre de aquí no aparece en la lista, se dice y no se usa.
- **El registro de usos** (qué sirvió y qué no) está en `HERRAMIENTAS-registro.md`, parte 2.
- **Este archivo nace casi vacío a propósito:** cada equipo tiene plugins distintos. El instalador lo llena con lo que la sesión de {{DUENO}} realmente ofrece, y no antes.

## 1. Cómo decidir

1. **¿La tarea cae en alguna fila de abajo?** Si no, no se usa ninguna.
2. **¿El beneficio compensa el costo?** Una skill carga sus instrucciones en el contexto (de cientos a miles de tokens). Para una comprobación de una línea no compensa.
3. **Se anota la decisión** en el informe («se usó X porque…» o «ninguna, porque…»). Si enseñó algo, va una línea en el registro.

**Quién invoca una skill:** el coordinador, siempre; un agente, si tiene la herramienta `Skill` y la skill aparece en su lista. Si no, lo dice en su informe y el coordinador la ejecuta sobre su entrega.

**Buscar skills nuevas** (solo cuando una tarea no cae en ninguna fila): con las palabras de la tarea y de la pila real, no con búsquedas genéricas. Filtro de calidad: más de 1 000 instalaciones, una fuente conocida y un repositorio con más de 100 estrellas; lo demás, con desconfianza. **Que no choque:** si hace lo mismo que una que ya hay, no se propone; no puede contradecir `AGENTS.md` §4. **Nadie instala nada sin el sí de {{DUENO}}**, y un agente nunca instala. Cada búsqueda deja una línea en el registro, también cuando no encontró nada útil.

## 2. Críticas (se usan siempre)

| Apoyo | Para qué | Cuidado |
| --- | --- | --- |
| Read, Edit, Write, Grep, Glob | Leer, cambiar y buscar | Reglas en `HERRAMIENTAS.md` §2 |
| Bash y la consola del equipo | Órdenes, guiones, git | `HERRAMIENTAS.md` §4 y §5 |
| Agentes (`.claude/agents/`, `Explore`, `general-purpose`) | El enjambre por rol y las búsquedas amplias | Solo para lo grande y paralelizable (`PRIORIZACION.md` §0) |
| Monitor | Esperar un archivo de fin sin `sleep` largo | `HERRAMIENTAS.md` §6 |
| El hook `sin-heredoc.mjs` | Frena el heredoc y el `sed -i` en Bash | Solo en Claude Code y con la sesión abierta en el repositorio |
| Memoria del proyecto y `AGENTS.md` | Continuidad entre sesiones y reglas | `AGENTS.md` §5 |

## 3. Muy recomendadas (si existen en la sesión)

| Apoyo | Úsalo cuando | No lo uses para |
| --- | --- | --- |
| `code-review`, `security-review` | Al cerrar una versión; cambios en sesión, permisos, cifrado o datos protegidos | Cambios solo de aspecto o de documentos |
| `superpowers:verification-before-completion` | Antes de dar algo por hecho | — |
| Skills de revisión de migraciones o de esquema | Un cambio de esquema nuevo | Migraciones ya aplicadas, que no se editan |
| `gh` (en la consola) | Consultar el remoto y el estado de las ramas | Publicar en `{{RAMA_PRODUCCION}}` sin el sí de {{DUENO}} |

## 4. Útiles en casos específicos

| Apoyo | Úsalo cuando | Cuidado |
| --- | --- | --- |
| Skills de diseño y accesibilidad | Pantallas nuevas, maquetas, textos de avisos | La identidad aprobada manda |
| Skills de depuración y estrategia de pruebas | Un fallo que no se reproduce a la primera; decidir qué se prueba | Errores con mensaje claro |
| Skills de arquitectura | Una pieza nueva con decisiones de datos o de límites | Lo ya decidido en la especificación |
| `simplify` | Tras una ronda de correcciones | Buscar defectos: no los busca |
| `update-config` | Cambiar hooks o permisos de `.claude/settings.json` | Solo con el sí de {{DUENO}} |
| WebSearch y WebFetch | Fuentes externas | Sitios que responden 403/418: `curl` (`HERRAMIENTAS.md` §10) |

## 5. No aplican a este proyecto

[Lista de plugins y conectores que el equipo tiene pero este proyecto no usa. El instalador la escribe al ver la lista real.]

## 6. Lo que ya no existe

[Nombres que algún documento viejo menciona y que ya no están. Vacío al instalar.]
