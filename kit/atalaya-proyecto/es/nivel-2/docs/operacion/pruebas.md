# Guía de pruebas · {{PROYECTO}}

Qué correr y qué trampas ya se pisaron. Este archivo nace con **los principios** que valieron en el proyecto de origen; las órdenes concretas de {{PROYECTO}} las llena el instalador (§2) y el equipo las completa con el uso. Cada regla nueva trae entre paréntesis la entrada del registro de donde salió (`HERRAMIENTAS-registro.md`).

## 1. Barato y caro

- **Barato:** {{PRUEBAS_BARATAS}}.
- **Caro:** {{PRUEBAS_CARAS}}.
- **Lo caro lo corre una sola vez quien entrega el código** (`PRIORIZACION.md` §2). Al integrar, el coordinador corre la suite completa **de lo barato**, no la cara: esa ya la corrió quien entregó; el codificador corre solo sus paquetes.
- **Si cambia una regla aprobada,** su prueba se pone al día en el mismo commit, no al preparar una versión.

## 2. Las baterías de este proyecto

| Batería | Orden | Dura | Notas |
| --- | --- | --- | --- |
| {{NOMBRE_DE_LA_BATERIA}} | {{ORDEN}} | {{DURACION}} | |

**Cada batería corre sola:** ningún agente trabajando, ninguna otra carga, y dos baterías nunca juntas. Una prueba que falla en la batería y pasa sola se repite primero con la máquina libre y con varias repeticiones, antes de abrir un diagnóstico.

## 3. Mutaciones

Se rompe el código a propósito y alguna prueba debe fallar; si «sobrevive», la prueba no sirve. Solo para pruebas nuevas de riesgo medio o más, una pasada.

1. **Nunca en el repositorio:** en una copia en la carpeta de trabajo (o con la facilidad de superposición del lenguaje, si la hay). Siempre una **mutación de control que debe morir**.
2. **Comprobar que el build mutado se compiló.** Si la mutación rompe los tipos, la compilación falla, queda la salida buena de antes y la prueba «pasa» con el código sin mutar. Tras mutar, la compilación debe terminar sin error antes de correr la prueba.
3. **Tras un corte a medias, recompilar:** la salida de compilación puede seguir compilada con la mutación.
4. **El programa que corre las pruebas y cuenta mutantes** decodifica la salida en UTF-8, corre antes la **línea base sin mutar**, y cuenta también las pruebas **canceladas** (un mutante que cuelga está muerto, pero se dice que murió por tiempo).
5. **Una mutación con una sola de dos defensas sobrevive con razón:** se mutan las dos juntas y se anota que cada una por separado es equivalente.

## 4. Capturas

- Las capturas se hacen solo si cambia lo que se ve, y solo las que el dueño o el auditor necesitan mirar.
- Una variable que pide «una carpeta de capturas» es una carpeta, no un interruptor.
- Con la batería completa, las capturas de todas las fases se reescriben: solo con el filtro de los pasos propios, y antes de entregar `git status` sobre `capturas/` no muestra archivos ajenos.

## 5. Servidores simulados y semillas

- Los simulados son todo ficticio y escuchan solo en `127.0.0.1`, en puertos altos libres.
- **Antes de sembrar un dato** en un simulado, se busca (Grep) si un paso viejo lo usa: si no, un paso viejo que lo teclea como «nuevo» empieza a decir «ya existe».
- **Ruta nueva que consulta la interfaz:** se suma al simulado, o responde 404 y la batería anota «errores de consola que el paso no esperaba».
- Las limpiezas de pruebas borran **por valor exacto, nunca por patrón**, y se cuenta después.

## 6. Trampas del navegador (si el proyecto tiene interfaz web)

- **La primera tecla se pierde** si llega antes de hidratar: las esperas reintentan la acción, no solo la comprobación.
- **Los avisos tapan clics:** un aviso entra después de montarse y empuja lo de abajo. Antes de un clic en una tabla, cerrar los avisos.
- **Un clic que reintenta esconde un clic perdido.** Para un clic que debe caer en lo que se ve ahora: medir y hacer clic sin reintento, con una pausa entre pulsar y soltar.
- **El diálogo abierto es `dialog[open]`,** no el primer `<dialog>`.
- Lo que se lee del DOM no vuelve como se escribió (`outline: 3px solid …` → `rgb(…) solid 3px`): se compara por partes. Dentro de un contenedor escalado, las medidas salen escaladas.

## 7. Trampas de la base de datos (si la hay)

- **Una base de pruebas vieja guarda lo que el código retiró.** Si una prueba de catálogo falla en una PC y no en otra, se busca el nombre en el historial de git.
- **Tabla nueva con clave foránea:** se suma a los `TRUNCATE` de las pruebas de carga.
- **Candados:** para ver quién espera, se mira la vista de bloqueos, no la de actividad.
- **El clúster temporal** se crea con mensajes en un idioma fijo (`C`), porque toma el de Windows y una prueba que busca el texto de un error falla.

## 8. «Una prueba que no corrió no pasó»

**Cero pruebas corridas es un fallo del filtro, no un «todo bien».** Tras cada corrida con filtro, se mira cuántas corrieron.

1. **El filtro se copia con sus tildes.**
2. **Un `.ps1` escrito con la herramienta de archivos es UTF-8 sin BOM** y PowerShell 5.1 lo lee como ANSI: un filtro con caracteres no ASCII corre 0 pasos sin avisar. Dentro del `.ps1`, lo no ASCII como `[char]0x00B7`; y se revisa la sintaxis antes de correrlo.
3. **Antes de fiarse de un 0,** se comprueba que el detector encuentra algo que se sabe que existe. No se tira a `/dev/null` la salida de lo que se leerá después.
4. **Sin la variable que activa las pruebas con base,** «ok» puede significar «saltadas». Se cuentan las saltadas.
5. **Caracteres invisibles o ambiguos** en una prueba (U+0301, U+00A0, U+2028): van como escape.
6. **Un resultado declarado** (un `verificacion.json`, una captura de otro rol) **no es una prueba que acabas de correr.**
