# Plantilla de encargo de agente · {{PROYECTO}}

El coordinador redacta cada encargo sobre esta plantilla. El encargo es lo único que el agente sabe de la tarea, así que debe bastarse solo.

**Los bloques fijos no van aquí.** «Antes de empezar», «Límites», «Bitácora» y «Entrega» viven en el archivo de cada rol, y Claude Code los carga solo al lanzar el agente: `.claude/agents/ejecutor.md`, `codificador.md`, `depurador.md`, `arquitecto.md`, `auditor.md` y `ensayador.md`.

Cada archivo trae su modelo y su effort por omisión. El coordinador los cambia en la llamada cuando la matriz de `AGENTES.md` §3 lo pide (por ejemplo, `codificador` con `effort: high` para I2).

**Para otro rol** (investigador, maquetador, redactor…) se usa `general-purpose` y el encargo empieza con estas dos líneas: «Lee `AGENTS.md` y `docs/operacion/HERRAMIENTAS.md` antes de empezar» y «Sigue los límites y la bitácora de `.claude/agents/codificador.md`». Después van las reglas de esta tarea.

---

**Asignación: nivel [directo | ligero | enfocado | completo] · clase [M | I1 | I2 | D1-D3 | A | E1 | E2 | V | R1-R3 | U | T1 | T2] · rol [ejecutor | codificador | depurador | arquitecto | auditor | ensayador | Explore | general-purpose] · [modelo]/[effort] · presupuesto [{{TOPE_MIN}} min] · base `{{RAMA_TRABAJO}}` en [hash]**

Esa línea viene de `PRIORIZACION.md` §1 (el nivel) y de `AGENTES.md` §2 y §3 (la clase, el rol y la configuración).

**Presupuesto** (`AGENTES.md` §11), una línea por punto:
- el problema concreto;
- el contexto que necesita y el que no;
- si es en paralelo o qué espera;
- cómo se verificará.

[Si el rol no carga en esta sesión, o si las reglas cambiaron desde que se abrió (`AGENTES.md` §1), el encargo empieza con: «Lee `.claude/agents/<rol>.md` y `AGENTS.md` del disco, y síguelos».]
- **Si te sales de la configuración** del rol, di el porqué en una frase.
- **Si vas a pasarte del presupuesto,** dilo en la bitácora con lo que falta, en cuanto lo veas.
- **Si aparece un disparador** de `AGENTES.md` §4, escala en lugar de seguir.

Eres el **[rol] de [sub-etapa y parte]** del proyecto {{PROYECTO}} ([la pila que le toca]). [Una frase sobre qué está en juego: datos, una pieza irreversible, lo que verá el usuario…]

## Qué leer primero

[Lista corta y ordenada:
- las secciones de la especificación;
- los informes y las líneas que devolvió la datapedia (si la hay);
- los archivos de código que debe conocer antes de tocar nada.

Si la lista está completa, el agente no necesita consultar la datapedia.]

## Qué hacer

[El trabajo, por partes, con las decisiones ya tomadas y las que no le tocan.
- **Para un auditor:** qué mirar con más cuidado.
- **Para un corrector:** la lista de hallazgos, la decisión sobre cada uno y que devuelva el costo real.]

**Pruebas caras, una sola vez** (`PRIORIZACION.md` §2 y `docs/operacion/pruebas.md`): [quién corre la batería completa; si hay mutaciones y de qué; si se mide rendimiento y por qué; qué capturas hacen falta. Lo que ya corrió otro rol y no cambió, no se repite.]

## Reglas de esta tarea

[Solo lo propio de esta tarea; las reglas generales ya están en `AGENTS.md` y en el rol:
- qué carpetas puede tocar y cuáles no (la frontera, si hay otro agente trabajando en paralelo);
- qué puede ejecutar: bases, puertos;
- si la tarea toca producción: {{DOC_DE_PRODUCCION}}.]

## Bitácora y entrega

- **Bitácora:** `docs/trazabilidad/[fase]/avance/[rol].md`.
- **Acta de temporales:** `docs/trazabilidad/[fase]/temporales/[aaaa-mm-dd]-[rol]-[tema].md`, abierta al crear la carpeta de trabajo y cerrada **antes** de borrarla (`HERRAMIENTAS.md` §3). El informe la nombra.
- **Informe:** `docs/trazabilidad/[fase]/[NN]-[tipo]-[tema].md`, con [lo propio de la tarea].
- **Lo demás** (qué lleva el informe, cómo se cierra) viene en el archivo del rol.
