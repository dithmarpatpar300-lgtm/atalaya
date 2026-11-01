# Esfuerzo proporcional y priorización · {{PROYECTO}}

Regla del dueño: un agente que tarda una hora o más en algo simple es tiempo perdido que no se recupera. Antes de cada encargo y de cada decisión se preguntan dos cosas: **¿vale la pena hacer tanto por algo tan simple o tan corto?** y **¿dónde está el objetivo claro en que se concentra el esfuerzo?** La calidad de la salida no baja: baja la repetición.

Lo que **no** cambia con esta regla: «verificado» sigue queriendo decir «lo ejecuté y vi el resultado»; las reglas sin excepción de `AGENTS.md` §4 no se negocian; los secretos, los datos protegidos y lo que el usuario ya escribió nunca se aplazan por caro.

## 0. Regla estricta del tiempo

«Si tenemos que esperar más de {{TOPE_MIN}} minutos por agente nunca progresaremos.» Sin excepción:

1. **Por omisión lo hace el coordinador,** directamente, si cabe en unos 20 minutos de su trabajo. Un agente solo cuando la tarea es grande **y** se puede hacer en paralelo con otra cosa útil.
2. **Ningún agente pasa de {{TOPE_MIN}} minutos.** El encargo se corta para que quepa; si no cabe, se parte en piezas o lo hace el coordinador.
3. **Una sola ronda:** codificar y que el coordinador verifique. Auditoría aparte solo para riesgo alto real (datos, secretos, migraciones, la puerta de entrada de producción), y de 20 minutos.
4. **Nada de «vuelve a ejecutar todo lo que afirmó el anterior».** Lo barato lo repite el coordinador; lo caro, nadie.
5. Si algo se demora, el coordinador lo dice a {{DUENO}} **antes** de que se cumpla el tope, con lo que falta y una salida más corta.

## 1. El nivel del encargo: riesgo × tamaño

El coordinador pone el nivel en la primera línea de cada encargo y elige el ciclo por él. **Con qué** trabaja cada rol (clase, modelo, effort y contexto) lo decide `AGENTES.md`: el nivel dice cuántos roles entran y la clase dice cuánta capacidad lleva cada uno.

| | **Pequeño** (un tema, pocos archivos, se prueba con unas pocas afirmaciones) | **Grande** (lógica nueva, varios archivos o una parte entera de una sub-etapa) |
| --- | --- | --- |
| **Riesgo bajo** (aspecto, textos, comentarios, pruebas, documentos; nada que pueda perder, cambiar o exponer un dato) | **Directo:** un agente, o el coordinador si es trivial. Sin auditoría: verifica el coordinador. Presupuesto: {{TOPE_MIN}} min | **Ligero:** codificador y **una** auditoría enfocada. Corrección solo si hay altos o bloqueantes; la verifica el coordinador. Presupuesto: {{TOPE_MIN}} min por agente |
| **Riesgo alto** ({{ZONA_DE_RIESGO}}) | **Enfocado:** codificador y auditoría **solo de lo que puede fallar**, con mutaciones en esa parte. Presupuesto: {{TOPE_MIN}} min por agente | **Completo:** codificar → auditar → corregir → auditar de nuevo, {{TOPE_MIN}} min por agente |

Ningún presupuesto pasa del tope (punto 2): un nivel más alto pone más agentes, no agentes más largos.

Si un agente pasa de su presupuesto, no sigue en silencio: actualiza la bitácora con lo que le falta y por qué, y el coordinador decide si recorta o amplía. Un trabajo que crece de nivel a mitad de camino se dice en cuanto se ve.

## 2. Cada prueba cara, una sola vez

Lo caro: {{PRUEBAS_CARAS}}. Lo barato: {{PRUEBAS_BARATAS}}.

1. **Lo caro lo corre una vez quien entrega el código,** al final y con el árbol terminado. El rol siguiente no lo repite: lee la salida y lo vuelve a correr solo si algo no cuadra o si su encargo lo pide expresamente. El coordinador repite solo lo barato.
2. **Mutaciones** solo para las pruebas nuevas que vigilan algo de riesgo medio o más. Una pasada; no se repiten si nada cambió después.
3. **Rendimiento** solo si el cambio toca un camino caliente (código que corre por cada tecla, por cada petición, sobre toda una lista). Una tanda alternada, antes y después; se vuelve a medir solo si la diferencia pasa del 10 %.
4. **Capturas** solo si cambia lo que se ve, y solo las que {{DUENO}} o el auditor necesitan mirar.
5. **Informes cortos:** lo hecho, lo medido con su cifra, lo no verificado, lo que necesita una decisión. Sin repetir el encargo ni lo que dice otro informe: se cita.

## 3. Los hallazgos de una auditoría: gravedad × costo de corregir

Cada hallazgo lleva, además de su gravedad (bloqueante, alto, medio, bajo), su **costo de corregir**, de 1 a 4, contando el cambio **y** su prueba:

| Costo | Qué significa |
| --- | --- |
| 1 | Texto, comentario o constante; se prueba leyendo o compilando |
| 2 | Cambio local, hasta unas 20 líneas en un archivo, con una afirmación o un paso de prueba |
| 3 | Lógica nueva, varios archivos o una medición que hay que montar |
| 4 | Toca algo cerrado o aprobado, necesita algo que no tenemos o una decisión de {{DUENO}} |

| Gravedad \ Costo | **Barato (1–2)** | **Caro (3–4)** |
| --- | --- | --- |
| **Medio o más** | **Corregir ya** | **Planificar:** medir antes de cambiar; si depende de otra cosa, aplazar con dueño y fecha |
| **Bajo** | **Al pasar,** si no exige una prueba nueva grande | **Aceptar o aplazar por escrito,** con el motivo |

Tres reglas encima de la cuadrícula:
1. Una **regresión** o algo que contradice lo aprobado es como mínimo **medio**.
2. **Secretos, datos protegidos o lo que el usuario ya escribió:** no se aplazan por costo.
3. Si la gravedad o el costo están **sin verificar**, la casilla es **medir primero**.

Quién hace qué: el auditor pone gravedad, costo, evidencia (medido, leído, no verificado) y la casilla que propone, y termina su recuento con una línea por casilla. El coordinador confirma o cambia y escribe la decisión con su motivo en el encargo del corrector. El corrector devuelve el costo real para calibrar. {{DUENO}} solo ve lo que es suyo: lo de aspecto, lo que se aplaza con gravedad media o más, y lo que contradice la especificación.

## 4. Lo aplazado

Lo que se aplaza va a la tabla «Aplazado» del archivo de estado, con dueño y con cuándo se revisa, no a un párrafo suelto.
