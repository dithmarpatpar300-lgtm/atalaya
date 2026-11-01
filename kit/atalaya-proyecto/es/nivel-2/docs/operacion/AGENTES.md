# Agentes, modelos y effort · {{PROYECTO}}

La política para decidir **qué agente, qué modelo, qué effort y cuánto contexto** lleva cada tarea.

- **Principio:** se usa **la configuración mínima capaz de resolver la tarea con la confiabilidad que pide su riesgo**, y se escala **solo con una razón técnica escrita**.
- **El método:** primero se clasifica la tarea (§2) y después se asigna con la matriz (§3). Esa clasificación es el «scheduler».
- **Cómo encaja con lo demás:**
  - `PRIORIZACION.md` decide el **nivel** del encargo (riesgo × tamaño) y por eso cuántos roles entran en el ciclo;
  - este archivo decide **con qué** trabaja cada rol;
  - las reglas generales siguen en `AGENTS.md`.

## 1. Lo que hay de verdad (verifícalo al instalar)

| Modelo | Nombre en Claude Code | Peso relativo | Para qué sirve |
| --- | --- | --- | --- |
| Haiku | `haiku` | 1× | Búsqueda, lectura, cambios mecánicos, resúmenes de salidas |
| Sonnet | `sonnet` | ~20× | Implementación, depuración común, ensayos, maquetas |
| Opus | `opus` | ~40× | Arquitectura, auditoría crítica, depuración difícil, coordinación |
| Fable | `fable` | ~100× | El más capaz. Solo como último escalón (§4) |

> Los pesos son los medidos en el proyecto de origen (octubre de 2026). Los modelos y precios cambian: el instalador los confirma con {{DUENO}} y corrige esta tabla. Esta nota se borra al instalar.

- **Effort:** los cuatro admiten `low`, `medium`, `high`, `xhigh` y `max`. **Aquí el effort siempre se fija explícito.** No se confía en el valor por omisión.
- **Dónde se fija:**
  - el frontmatter de `.claude/agents/<rol>.md` (`model:` y `effort:`);
  - o cada llamada a `Agent` (`model` y `effort`), que manda sobre el archivo.

  Este documento es la instrucción expresa que autoriza al coordinador a fijar el effort por llamada.
- **Los roles de `.claude/agents/` se cargan al abrir la sesión.** Los creados durante una sesión no aparecen en ella («Agent type not found»). En ese caso se lanza `general-purpose` con el `model` y el `effort` de la matriz, y el encargo empieza con «Lee `.claude/agents/<rol>.md` y síguelo como tus instrucciones de rol». El frontmatter de un rol lleva solo `name`, `description`, `model` y `effort`; la `description` no puede tener «: » sin comillas, porque rompe el YAML.
- **Un agente recibe el `AGENTS.md` que la sesión cargó al abrirse, no el del disco.** Si las reglas cambiaron durante la sesión, el encargo dice «lee `AGENTS.md` del disco».
- **Un agente tiene un piso de decenas de miles de tokens** aun en una tarea trivial (medido: unos 78 000 con Haiku). Lo que cabe en pocos minutos lo hace el coordinador.
- **Qué significa el precio:** con suscripción se paga en límite de uso; con clave de API, en dinero. El peso relativo sirve como guía; si el límite se consume en otra proporción, se mide y se corrige (§12).

## 2. Clasificación de las tareas

Cada encargo se clasifica con una **clase**. La clase sale de la pregunta «¿qué falta decidir?», no del tamaño.

**Zona de riesgo** de este proyecto: {{ZONA_DE_RIESGO}} (lo que guarda o transforma datos, seguridad y permisos, migraciones, secretos, scripts de despliegue, integraciones con sistemas externos, producción).

| Clase | Qué es | Ejemplos genéricos |
| --- | --- | --- |
| **M · Mecánica** | Todo está decidido: solo hay que ejecutar bien. **En zona de riesgo** solo es M un cambio **literal** (copiar una huella, un número de versión, un texto aprobado) y el coordinador revisa el diff línea por línea. Cualquier otra cosa ahí es I1 como mínimo | Subir la versión en los archivos que la llevan; renombrar; textos ya aprobados; formatear un informe; extraer datos de un registro |
| **I1 · Implementación guiada** | Diseño cerrado: maqueta aprobada, contrato o firma escritos y pruebas definidas | Un componente de una maqueta aprobada; un endpoint con contrato escrito; una validación con su regla; las pruebas de un cambio |
| **I2 · Implementación con decisiones** | Hay que decidir cómo, en zona de riesgo | Concurrencia o conflictos de guardado; cifrado o llaves; permisos y roles; una integración con un sistema externo |
| **D1 · Depuración evidente** | Mensaje claro y causa localizada | Error de compilación o de tipos; una prueba que falla con un diff claro |
| **D2 · Depuración distribuida** | Varios componentes y causa no evidente | Un simulado al que le falta una ruta; una semilla que choca con un paso viejo; un documento generado que sale mal |
| **D3 · Depuración difícil** | Intermitente, carga, concurrencia, datos inconsistentes, sistema externo o causa desconocida | Una batería que falla solo con la máquina cargada; algo que solo falla contra el sistema real |
| **A · Arquitectura y diseño** | Decisiones que se propagan | Cambios de esquema o migraciones nuevas, flujo de datos, contratos entre componentes, sincronización, roles y permisos |
| **E1 · Auditoría de rutina** | Revisar algo de riesgo bajo o medio | Documentación, textos, un componente visual, una refactorización pequeña |
| **E2 · Auditoría crítica** | Revisar algo que puede perder, cambiar o exponer datos, o tocar producción | Llaves y respaldos, migraciones, guardado y conflictos, permisos, guiones de despliegue |
| **V · Validación y ensayo** | Demostrar que funciona con un procedimiento conocido, aparte de las pruebas de quien entrega | El ensayo de un kit de versión (`version-y-kit.md` §4); repetir una medición con la máquina libre |
| **R · Búsqueda e investigación** | R1: ubicar algo dentro del repositorio. R2: investigar fuera, con fuentes. R3: investigar para decidir arquitectura | R1: «¿dónde se usa esta constante?». R2: una norma o una biblioteca, leídas, nunca inventadas. R3: el estudio de un módulo nuevo |
| **U · Maquetas y diseño visual** | Proponer una pantalla para el visto bueno | Una pantalla nueva; una maqueta de una extensión |
| **T · Documentación** | T1: actualizar con datos dados. T2: redactar guías o métodos para {{DUENO}} | T1: estado, bitácoras, notas de versión. T2: la guía de una versión, este documento |

**Quién clasifica y cómo se desempata:**
- **Clasifica el coordinador,** y lo escribe en la línea de asignación.
- **Si una tarea parece de dos clases,** manda la que pide más capacidad en la matriz §3: primero el modelo y después el effort.
- **Ante la duda entre E1 y E2, es E2.**
- **Si no se sabe qué falta decidir,** es A o R3 hasta que se sepa.
- **Irreversible** quiere decir que no se deshace con git ni con una vuelta atrás ensayada. Por ejemplo:
  - datos reales transformados o borrados;
  - cambios en llaves, cifrado o respaldos;
  - lo que se envía a un sistema externo y no se puede retirar.

## 3. Matriz: clase → rol → modelo y effort

| Clase | Rol (`.claude/agents/`) | Modelo · effort | Revisión | Autonomía |
| --- | --- | --- | --- | --- |
| M | `ejecutor`, o el coordinador si cabe en minutos | Haiku · `medium` (`low` si es copiar o pegar) | Comprobación automática: `grep`, compilación, una prueba. El coordinador mira el diff | Ninguna: si algo no está en el encargo, se detiene |
| I1 | `codificador` | Sonnet · `medium` | Pruebas de quien entrega + lo barato del coordinador. La auditoría, según el nivel (`PRIORIZACION.md` §1) | Baja: sigue la especificación y escala las dudas |
| I2 | `codificador` | Sonnet · `high` | Auditoría E2 del diff | Media: decide lo local y escala lo estructural |
| D1 | El coordinador, o `codificador` | Sonnet · `medium` | La prueba que fallaba, ahora en verde | Media |
| D2 | `depurador` | Sonnet · `high` | Reproducir antes y después | Media |
| D3 | `depurador` | Opus · `high`. Hasta dos hipótesis independientes se reproducen en paralelo, cada una con un `depurador` en Sonnet · `medium`, con su puerto y su carpeta. Si las dos necesitan el mismo recurso compartido, van en serie | Reproducción demostrada; si toca datos o producción, auditoría E2 | Media, con hipótesis escritas |
| A | `arquitecto` | Opus · `high`. Si es irreversible: Opus · `xhigh` | Cuando es irreversible, una crítica cruzada (§6.3) y el visto bueno de {{DUENO}} | Alta para proponer; **ninguna para implementar** |
| E1 | `auditor` con `model: sonnet` | Sonnet · `high` | El coordinador confirma los hallazgos | Alta para buscar; ninguna para corregir |
| E2 | `auditor` | Opus · `high`. **`xhigh` solo si el cambio es irreversible** | El coordinador decide la casilla de cada hallazgo (`PRIORIZACION.md` §3) | Igual |
| V | `ensayador` | Sonnet · `medium` para una versión sin cambios de lógica; `high` si trae migración o lógica nueva de despliegue, o un fallo que no se explica | El veredicto con evidencia | Baja: el procedimiento es fijo |
| R1 | `Explore` | Haiku · `low` (`medium` si la búsqueda es amplia) | Se comprueba lo que devuelve | — |
| R2 | `general-purpose` | Sonnet · `high` | Fuentes citadas; lo no verificado, marcado | Media |
| R3 | `arquitecto` | Opus · `high` | Igual que A | Alta para proponer |
| U | `general-purpose`, con skills de diseño | Sonnet · `high` | El visto bueno de {{DUENO}} sobre la maqueta | Media dentro de la identidad aprobada |
| T1 | `ejecutor` | Haiku · `medium` | El coordinador lee el resultado | Baja |
| T2 | El coordinador, o `general-purpose` | Sonnet · `medium`; Opus · `medium` si define reglas o métodos | El coordinador; {{DUENO}} si es para él | Media |

**Los seis perfiles y sus roles:** Explorer = `Explore`; Architect = `arquitecto`; Implementer = `codificador` (y `ejecutor` para lo mecánico); Debugger = `depurador`; Auditor = `auditor`; Validator = `ensayador` en las versiones, y quien entrega corre una sola vez las pruebas caras.

**El coordinador** (la sesión principal) corre en Opus.
- **Effort:** `medium` para coordinar y revisar entregas; `high` para diseñar, decidir o revisar algo crítico.
- **`max`:** nunca por omisión. **Fable:** solo por escalamiento (§4).
- **Orquestar muchos agentes a la vez** («workflow»): solo si {{DUENO}} lo pide.

## 4. Escalamiento: cuándo y cómo se sube

**Un agente no cambia de modelo a mitad de camino.** Escalar quiere decir **detenerse y avisar**: el agente deja escrito lo que encontró y el coordinador relanza la tarea en el escalón siguiente, pasándole ese hallazgo para que no empiece de cero.

**Disparadores válidos.** Hay que escribir cuál fue y su evidencia:
1. **Hay más de una interpretación razonable** de la especificación y elegir cambia el resultado.
2. **El alcance se sale del encargo:** un implementador necesita cambiar un archivo que el encargo no nombra, o un auditor/depurador/arquitecto encuentra la causa o el riesgo en otra parte del sistema.
3. **Una contradicción** entre la especificación, el código, las pruebas o lo aprobado.
4. **Riesgo de pérdida o corrupción de datos,** o de exponer un secreto o datos protegidos.
5. **La solución cambia la arquitectura:** un contrato, el esquema, el flujo de datos, la frontera con otro sistema.
6. **Concurrencia o carreras:** dos escritores, candados, temporizadores, eventos fuera de orden.
7. **Toca producción** o un guion de despliegue.
8. **Una hipótesis que no se puede verificar** con confianza, o dos intentos fallidos con la misma idea.
9. **El presupuesto de tiempo del nivel** se va a pasar sin resultado verificable. Lo normal es avisar y recortar el alcance; se relanza en otro peldaño solo si la causa de la demora es uno de los disparadores 1 a 8.

**«Es difícil», «quiero estar seguro» o «por si acaso» no son disparadores.**

**La escalera.** Se sube **un peldaño** cada vez:

`Haiku medium → Sonnet medium → Sonnet high → Opus high → Opus xhigh → Fable high`

- **Cómo se elige qué subir:** dentro del mismo modelo, primero el effort, cuando el problema es de **profundidad** (más pasos, más casos); de modelo, cuando es de **juicio** (disparadores 1, 3, 5 y 8).
- **Fable** entra solo si se cumplen las tres: (1) Opus `xhigh` dejó una contradicción sin resolver; (2) es una decisión A o E2 **irreversible**; (3) {{DUENO}} dio **su sí**. Lleva un presupuesto de 30 minutos.
- **Tope:** dos escalamientos por tarea. Si con dos no se resuelve, el coordinador toma la tarea o le pregunta a {{DUENO}}.
- **El formato del aviso,** en la bitácora y en el mensaje final, una línea:

  `ESCALAR · disparador N · evidencia (archivo:línea o salida) · lo que ya está hecho · lo que falta`

## 5. Desescalamiento: cuándo se baja

Cuando un rol caro ya cerró la decisión, la ejecución pasa a uno más barato. **Una decisión está cerrada** si hay un documento en `docs/trazabilidad/<fase>/` que trae: (1) los archivos que se tocan; (2) los contratos o firmas; (3) las pruebas de aceptación; (4) ninguna pregunta abierta.

**El patrón:** `arquitecto` (Opus · `high`) decide y escribe el diseño → `codificador` (Sonnet · `medium`) implementa por módulos, y lo repetitivo va a `ejecutor` (Haiku) → `auditor` revisa **solo el diff** contra el diseño.

**Lo que nunca se baja por historial:** la clase de una auditoría la da el riesgo del cambio. Una E2 sigue siendo E2 aunque las anteriores hayan salido limpias: cero hallazgos no prueban que el código esté bien.

**Lo que sí se ajusta con historial:** el effort dentro de la misma clase y modelo (por ejemplo, un E2 irreversible con tres rondas seguidas sin hallazgos medios puede pasar de `xhigh` a `high`, si el coordinador lo escribe en la asignación), y se anota todo escalamiento innecesario para calibrar.

## 6. Auditoría

1. **La pregunta del auditor** no es «¿parece funcionar?», sino **«¿de qué maneras podría estar equivocado aunque parezca funcionar?»**: supuestos falsos, casos límite, efectos secundarios, contradicciones con lo aprobado, seguridad y datos, mantenibilidad, lo que la prueba no cubre.
2. **La independencia:**
   - el auditor no escribió nada de lo que revisa y no lo corrige;
   - recibe la especificación, el diseño y el **commit exacto** que audita;
   - lee el informe del implementador **después** de formar su propia lista, y lo trata como afirmaciones que hay que verificar;
   - cada afirmación importante la demuestra con una prueba o una mutación en una copia.
3. **La crítica cruzada** se hace solo en A o E2 irreversibles. Un segundo auditor revisa con **otro foco** (uno mira seguridad y secretos, otro datos y vuelta atrás). No se lanzan dos auditores iguales «para tener otra opinión».
4. **Cada hallazgo** lleva gravedad × costo y su casilla (`PRIORIZACION.md` §3). El informe dice lo que **no** pudo verificar.
5. **Cuándo hay auditoría:** lo dice el nivel (`PRIORIZACION.md` §1). Su clase (E1 o E2) la da el riesgo del cambio, no su tamaño. Ante la duda, E2.
6. **Si a un auditor lanzado como E1 le toca revisar algo que es E2,** se detiene y escribe `ESCALAR · reclasificar a E2 · evidencia`.

## 7. Implementación

El implementador (`codificador`, `ejecutor`): **sigue la especificación;** **cambia lo mínimo** y respeta los contratos; **no introduce arquitectura nueva** (ni dependencias, ni abstracciones que nadie pidió, ni «ya que estaba»); **corre las pruebas que le tocan** y reporta las desviaciones; y **si la especificación no alcanza o se contradice, no improvisa en silencio:** se detiene y escala (§4, disparadores 1 y 3).

## 8. Paralelización

**Sí se paraleliza:** módulos independientes con frontera de carpetas escrita; búsquedas `Explore` sobre preguntas distintas; hipótesis independientes de un D3; auditorías de áreas distintas del mismo cambio; redacción de documentos que no se tocan entre sí. Después, **el coordinador consolida**.

**No se paraleliza:** lo que tiene dependencia (el orden es `descubrimiento → decisión → implementación → validación`; nada se implementa con el diseño abierto); dos escritores sobre el mismo archivo o carpeta; dos agentes sobre el mismo recurso compartido (base de pruebas, puerto); dos baterías largas a la vez, ni una batería con un agente trabajando.

**Si dos escritores necesitan carpetas vecinas,** cada uno va en su propio worktree (`isolation: "worktree"`), y el coordinador integra.

## 9. Contexto por agente: lo mínimo necesario

| Rol | Recibe | No recibe |
| --- | --- | --- |
| `Explore` | La pregunta y las carpetas donde buscar | Reglas, historia, encargo largo |
| `ejecutor` | El encargo con los pasos exactos, los archivos y la comprobación final | Especificación completa, historia, estado |
| `codificador` | El encargo, las secciones de la especificación o el diseño, los archivos, los contratos, las restricciones, `HERRAMIENTAS.md` y `pruebas.md` | La historia del proyecto, el registro de herramientas, informes que no tocan |
| `depurador` | El síntoma, cómo se reproduce, lo ya descartado y los archivos sospechosos | El diseño completo, salvo que la causa esté ahí |
| `auditor` | La especificación y el diseño, el commit exacto, las decisiones previas que aplican y las reglas que no se negocian | El razonamiento del implementador antes de su propia revisión |
| `arquitecto` | El problema, el estado, la especificación y los límites del código | Detalles de implementación que todavía no tocan |
| `ensayador` | El kit, su informe de versión, el último ensayo y `version-y-kit.md` | El código fuente, salvo para comprobar lo servido |

**Reglas comunes:**
- `AGENTS.md` se carga solo (§1). No hace falta copiarlo en el encargo.
- Los roles mandan leer `HERRAMIENTAS.md` y `PRIORIZACION.md` enteros; mientras los dos juntos quepan en unos 8 000 tokens. Si crecen, se recortan los preámbulos de los roles.
- Se nombran archivos y líneas en vez de pegar su contenido.
- Ningún agente lee entera la historia del estado ni el registro de herramientas.

## 10. Coordinación: cómo se evitan los errores entre agentes

- **Base fija.** Cada encargo dice sobre qué commit trabaja (`{{RAMA_TRABAJO}}` en `<hash>`). El auditor audita ese commit o ese diff, no «lo que haya».
- **Un escritor por archivo.** La frontera va escrita en el encargo, y el coordinador la comprueba con `git status` antes de integrar.
- **Las decisiones viven en un solo documento.** Si dos agentes podrían decidir lo mismo, decide el `arquitecto` o el coordinador antes de lanzarlos.
- **Sin trabajo duplicado.** Antes de lanzar, el coordinador revisa las bitácoras activas (`scripts/avance.ps1`).
- **Sin contexto perdido.** Lo que un agente encontró y otro necesita va en su informe, y el encargo siguiente lo nombra.
- **Información desactualizada.** Si el árbol cambió desde el encargo, el agente lo dice antes de seguir: `git log` desde el hash del encargo.

## 11. Antes de lanzar agentes: el presupuesto

El coordinador responde por escrito en el encargo, debajo de la línea de asignación:
1. ¿Qué problema concreto resuelve este agente?
2. ¿Qué clase es y qué configuración le toca?
3. ¿Qué contexto necesita, y cuál no?
4. ¿Se puede hacer en paralelo, o depende de algo?
5. ¿Cómo se verificará el resultado?

**Si no hay una función clara, no se lanza.**

## 12. Medición, sin burocracia

- **Cada informe de agente** termina con una línea: `Config: rol · clase · modelo/effort · minutos · escaló (sí: motivo / no) · hallazgos o retrabajo`. El implementador no conoce el retrabajo al entregar: deja «—».
- **El coordinador** la pasa a `HERRAMIENTAS-registro.md`, parte 3, y completa los hallazgos y el retrabajo cuando se conocen.
- **Al cerrar una sub-etapa** mira estas señales. Solo cuentan las clases con **5 tareas o más**:

| Señal | Qué indica | Qué se hace |
| --- | --- | --- |
| Más del 30 % de las tareas de una clase escalaron | Esa configuración es insuficiente | Subir un peldaño la configuración de esa clase |
| Una clase nunca escala y sus auditorías dan 0 hallazgos | Puede estar sobrada | Probar un peldaño menos en la siguiente |
| Una corrección o reversión después de una entrega «verificada» | Defecto introducido | Revisar la clase y el auditor |
| Más de dos rondas de corrección | Retrabajo: diseño o especificación insuficiente | Mirar la clase A y el encargo, no solo el implementador |
| Muchos tokens **nuevos** (no de caché) por agente | Demasiado contexto | Recortar lo que se le pasa (§9) |
| Porcentaje de agentes en Opus | Mezcla de costo | Meta inicial sugerida: menos del 40 % sin que suban los defectos |

## 13. Ejemplos genéricos

1. **Subir de versión** (número, huellas, notas): M. `ejecutor` en Haiku · `medium`, con la lista exacta de `version-y-kit.md`; el coordinador revisa el diff literal.
2. **Una pantalla nueva:** maqueta U (Sonnet · `high`) y visto bueno de {{DUENO}}; si toca roles o permisos, A (`arquitecto`); la ruta del servidor es I2; la pantalla es I1; al final, E2 sobre el diff.
3. **Cambiar una constante que decide quién hace qué:** I1 con riesgo alto, `codificador` en Sonnet · `medium`, más un E2 de 20 minutos solo sobre lo que cambia.
4. **Un cambio de esquema o migración:** A (`arquitecto`, Opus · `high`), I2 (`codificador`, Sonnet · `high`), E2 (`auditor`, Opus · `high`; `xhigh` y crítica cruzada si transforma o borra datos reales) y V con la vuelta atrás ensayada.
5. **Un error de compilación tras un cambio:** D1; lo arregla el coordinador.
6. **Una prueba que falla solo con la batería completa:** D2; `depurador` en Sonnet · `high`, reproduciendo con carga artificial.
7. **Algo que falla solo contra el sistema real:** D3; `depurador` en Opus · `high`, dos hipótesis en paralelo contra un simulado. La prueba real la corre {{DUENO}}.
8. **Revisión de llaves y respaldos antes de una versión mayor:** E2 irreversible; `auditor` en Opus · `xhigh` con crítica cruzada.
9. **Estudiar un módulo nuevo:** R3/A con `arquitecto` en Opus · `high`; las búsquedas en el repositorio las hace `Explore` en Haiku · `low`.
10. **«¿Dónde se usa esta constante?»:** R1; `Explore` en Haiku · `low`, o Grep del coordinador.
