# Centro de operaciones · {{PROYECTO}}

Aquí está **cómo se trabaja** en este proyecto. Las reglas están en **`AGENTS.md`**, en la raíz: es su única copia y se carga sola en cada sesión. Esta carpeta guarda el detalle que esas reglas enlazan.

- **Qué se construye:** {{ESPECIFICACION}}.
- **Dónde quedó el trabajo:** `docs/trazabilidad/ESTADO-para-retomar.md`.

## Los archivos

| Archivo | Para qué | Quién lo lee |
| --- | --- | --- |
| `HERRAMIENTAS.md` | Qué funciona y qué falla en este equipo, por herramienta, con la orden que sirve; el ciclo de los temporales (§3) | Todo agente al empezar; el coordinador |
| `HERRAMIENTAS-registro.md` | La historia: una línea por cada fallo de herramienta, por cada uso de una skill que enseñó algo y por cada agente lanzado | Quien anota; el coordinador al cerrar una sub-etapa |
| `PRIORIZACION.md` | Esfuerzo proporcional: nivel del encargo (riesgo × tamaño), pruebas caras una sola vez, hallazgos por gravedad × costo | El coordinador al encargar; el auditor |
| `AGENTES.md` | Política de agentes: clases de tareas, matriz clase → rol → modelo y effort, escalamiento y desescalamiento, auditoría, paralelismo, contexto por agente y medición | El coordinador antes de lanzar cualquier agente |
| `ENCARGO-PLANTILLA.md` | La parte variable de cada encargo (el bloque fijo está en `.claude/agents/`) | El coordinador |
| `SKILLS.md` | Skills, plugins y herramientas que existen de verdad: cuándo sí y cuándo no | El coordinador; un agente si su encargo lo pide |
| `pruebas.md` | Cómo se prueba: baterías, topes, mutaciones, capturas, simulados y trampas conocidas | Codificador, auditor y ensayador |
| `version-y-kit.md` | (opcional) El paso a paso de una versión: compilar, kit, ensayo, entrega y cierre | El coordinador y el ensayador |

**Los roles del enjambre** están en `.claude/agents/` (`ejecutor`, `codificador`, `depurador`, `arquitecto`, `auditor`, `ensayador`), cada uno con su modelo y su effort por omisión (`AGENTES.md` §3).

**Las plantillas** (el acta de temporales) están en `docs/trazabilidad/_plantillas/`.

## Cómo se mantiene

- **Una regla se cambia en su único lugar.** Si es general, en `AGENTS.md`; si es de una herramienta, en `HERRAMIENTAS.md`. Nunca se copia a otro archivo: se enlaza.
- **Cada fallo de herramienta** deja una línea en `HERRAMIENTAS-registro.md`. Al cerrar una sub-etapa, el coordinador convierte en regla lo que se repite.
- **Topes de tamaño,** para que nadie lea a medias: `HERRAMIENTAS.md` unos 20 KB; `SKILLS.md` unos 8 KB; `AGENTES.md` unos 28 KB (lo lee el coordinador, no cada agente); el estado para retomar unos 6 KB. Lo que crece sin parar va a un registro o a la historia.
- **Lo viejo no se borra:** va a `docs/archivo/`.
