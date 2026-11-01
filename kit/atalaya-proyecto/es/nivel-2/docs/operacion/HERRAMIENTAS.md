# Herramientas · qué funciona en este equipo · {{PROYECTO}}

Lo que aquí se dice que falla **no se intenta**: no se reintenta «por si acaso». Cada regla viene de un fallo medido.

- **La historia de cada regla,** con su fecha, está en `HERRAMIENTAS-registro.md`. Las reglas generales están en `AGENTS.md`.
- **Herramienta principal:** Claude Code. Equipo: {{SISTEMA_Y_CONSOLA}}.
- **Pruebas:** `pruebas.md`. **Producción:** {{DOC_DE_PRODUCCION}}.
- **Este archivo nace con las lecciones del proyecto de origen** (Windows 11, Git Bash y PowerShell 5.1). El instalador borra lo que no aplica a tu equipo y agrega lo propio; lo nuevo se anota primero en `HERRAMIENTAS-registro.md` y sube aquí cuando se repite.

## 1. Datapedia (opcional)

Si el proyecto tiene un índice o buscador de su documentación, se consulta cuando el encargo no trae la lista de lecturas y antes de responder sobre algo ya decidido. Lo que devuelve se comprueba contra el archivo. {{ORDEN_DE_LA_DATAPEDIA}}

## 2. Leer y escribir archivos

- **Read para leer,** con `offset` y `limit` si el archivo es largo (el tope es de unos 25 000 tokens). Las lecturas independientes van en un mismo mensaje.
- **Antes de un Edit:**
  - un **Read del archivo por la misma ruta** y en esta conversación. Después de compactar la conversación, de cambiar de ruta (incluida una mudanza del proyecto) o de un `git pull`, se vuelve a leer el tramo;
  - **Grep, `cat` y `sed -n` no cuentan como lectura;**
  - el ancla, **de una sola línea**, copiada tal cual (tabuladores y espacios incluidos; sin la columna del número de línea);
  - **no se mezclan** `sed` y Edit sobre un mismo archivo.
- **`replace_all` sobre una clase o palabra corta** toca también las que la contienen (`p-6` está dentro de `gap-6`): se reemplaza con su contexto, o línea por línea.
- **Prohibido escribir archivos con heredoc, `sed -i`, `perl -i`, `Set-Content` de código, `python -c` o `node -e`.**
  - En Bash lo frena un hook del proyecto: `.claude/hooks/sin-heredoc.mjs`. Si salta, no es un fallo: el programa se escribe con Write en la carpeta de trabajo y se ejecuta, o se usa Edit.
  - Siguen pasando: `printf … > archivo` para la bitácora, la cadena-aquí `<<<` y `git commit -F - <<'EOF'`.
- **Escapes:** los caracteres invisibles o ambiguos (U+00A0, U+2028, acento combinante) van siempre como escape en el código, y se verifican con `od -c` después de escribirlos.
- **Python en Windows** escribe CRLF en modo texto, y `-I` ignora `PYTHONIOENCODING`. Para tildes en la salida: `PYTHONIOENCODING=utf-8` sin `-I`.

## 3. Lo temporal: se crea, se usa, se deja su acta y se destruye

Todo lo temporal va en `{{TMP}}`; no va al repositorio ni a la carpeta temporal del sistema, que suele compartirse entre agentes.

**Regla del dueño.** Los temporales dejan huecos: carpetas olvidadas, procesos vivos, pruebas sin rastro. Por eso **nada temporal queda sin acta ni sin destruir**, y el acta se guarda **antes** de borrar.

**El ciclo, siempre en este orden:**
1. **Crear.** Una subcarpeta por tarea: `{{TMP}}\<aaaa-mm-dd>-<rol>-<tema>\`. El coordinador también. Nadie toca la de otro. Se crea justo antes de usarla, no «por si acaso».
2. **Abrir el acta,** en el mismo momento: `docs/trazabilidad/<fase>/temporales/<aaaa-mm-dd>-<rol>-<tema>.md` (modelo en `docs/trazabilidad/_plantillas/ACTA-PLANTILLA.md`), con el motivo. Se completa **por partes, desde temprano**: a un agente cortado a medias se le pierde todo lo que no había escrito.
3. **Usar.** Ahí van scripts auxiliares, copias para mutaciones, servidores simulados, perfiles de navegador, clústeres y worktrees temporales, salidas largas y borradores.
4. **Cerrar el acta antes de borrar.** Lleva, en este orden:
   - el motivo y la base (`{{RAMA_TRABAJO}}` en `<hash>`);
   - lo que se creó: carpetas, procesos, puertos, bases, worktrees;
   - lo que se hizo y se corrió, cada cosa con su orden y su resultado **en cifras** (conteos, tiempos, pasó o falló), no con la salida entera;
   - lo que sobrevive y adónde pasó: informe, `deploy/`, `scripts/` o memoria;
   - lo que falló o enseñó (esa línea va también a `HERRAMIENTAS-registro.md`);
   - lo que no se verificó.
5. **Destruir** con la ruta literal (con variables, el borrado se bloquea: `rm -rf "$VAR/…"` en Bash, `rd` o `Remove-Item` con `$variable` en PowerShell). Antes se cierran los procesos y puertos propios, **buscándolos por la ruta de la carpeta en su línea de órdenes** (un proceso en segundo plano sigue vivo aunque ya no haga falta). Una carpeta con trabajo a medias no se borra: se completa el acta o se le pregunta al dueño.
6. **Confirmar.** `Test-Path` (o `ls`) da que no existe, no queda ningún proceso propio ni puerto escuchando, y se añade al final del acta `Borrado confirmado: <fecha y hora> · carpeta ausente · 0 procesos · 0 puertos`. La hora sale del reloj (`Get-Date`, `date`), no de la memoria.

**Alrededor del ciclo:**
- **Nunca va ahí,** ni un momento, y el acta tampoco lo copia: secretos, llaves, respaldos ni datos protegidos.
- **Alcance:** vale para `{{TMP}}`, los worktrees de versión (`version-y-kit.md` §2), la carpeta temporal del sistema (por ejemplo los perfiles de navegador de las pruebas) y la carpeta de trabajo (scratchpad) de la sesión de Claude Code.
- **El informe de la tarea** nombra el acta con su ruta; ya no basta decir «borré mi carpeta».
- **Si hubo un corte** (límite de uso, agente detenido, caída), lo primero al retomar es el hueco: se lee lo que quedó en la carpeta, se escribe el acta marcándola «reconstruida» y recién entonces se borra.
- **Quién vigila:** el coordinador mira `{{TMP}}` al abrir cada sesión y al cerrar cada sub-etapa (`AGENTS.md` §2 y §7). Toda carpeta sin acta cerrada se la cuenta al dueño.
- **Lo que se reutiliza no vive aquí.** Lo que deba sobrevivir va al repositorio (informe, `deploy/`, `scripts/`) o, si es de una sesión a otra, a la memoria.

## 4. Bash

- **Las órdenes miden menos de unos 6 KB.** Lo más largo se escribe con Write en la carpeta de trabajo y se ejecuta.
- **`\\` llega como `\`:** nada de barras dobles dentro de una orden.
- **Una comprobación que acaba en `grep` sin coincidencias** da «Exit code 1»: se añade `; true`.
- **`find` solo en las carpetas que importan.** `-not -path` no poda: recorre `node_modules` entero. Lo mismo `du` sobre carpetas enormes: en Windows se mide con PowerShell y `Measure-Object`.
- **La salida larga** se termina con `| head -60` o `| tail`.
- **Rutas para Node en Windows:** `/d/…` llega mal a `node`; se usa `D:/…`.

## 5. PowerShell 5.1 (si el equipo es Windows)

- **Sin prosa en español dentro de la orden:** la palabra «del» se toma por la orden de borrar y se bloquea. Tampoco hay `&&`: se usa `; if ($?) { … }`.
- **Un `Remove-Item` dentro de una orden más larga** hace que el freno bloquee la orden entera (incluso con `/L` de un `robocopy` por delante): el borrado va aparte, con ruta literal.
- **Las órdenes de consulta se cierran con `exit 0`:** el código raro suele ser el del último ejecutable.
- **Nada de `Get-ChildItem -Recurse`** sobre el repositorio entero (`node_modules`, worktrees). Se resume por carpeta, o se usa `Test-Path`.
- **Codificación:**
  - sin BOM, PowerShell lee como ANSI: para leer, `Get-Content -Encoding UTF8`;
  - un `.ps1` va en ASCII o, si lleva tildes, en UTF-8 **con** BOM. Se comprueba con `[regex]::Matches($t,'[^\x00-\x7F]').Count`;
  - `*>` y `Tee-Object` pueden escribir UTF-16 o UTF-8 con BOM: se mira la codificación antes de leer.
- **Sintaxis:**
  - `"${variable}:"` dentro de comillas dobles (`"$variable:"` se lee como unidad con unidad de disco);
  - todo guion propio se revisa antes de correrlo: `$t = $null; $e = $null; [System.Management.Automation.Language.Parser]::ParseFile($ruta, [ref]$t, [ref]$e)`;
  - una función propia no se llama `H` ni de una o dos letras (los alias ganan);
  - `.Substring(0,[Math]::Min(N,$s.Length))`; se comprueba `$null` antes de llamar a un método;
  - las variables no distinguen mayúsculas (`$t` pisa `$T`);
  - `$ErrorActionPreference = 'Stop'` corta un guion en el aviso normal que `git` escribe por stderr: en los guiones que llaman a programas, `Continue` y se mira `$LASTEXITCODE`.
- **Ejecutables:**
  - no se usa `2>&1` sobre ellos: PowerShell convierte su salida de errores en `NativeCommandError`;
  - una función que llama a un ejecutable manda su salida a `| Out-Host` y devuelve solo `$LASTEXITCODE`;
  - `Start-Process -ArgumentList` parte un argumento con espacios: se pasa entre comillas dentro de la cadena;
  - `Start-Process … -Wait` sobre un servicio que no termina se cuelga: se arranca sin `-Wait` y se espera con la comprobación de que escucha.
- **`[IO.File]::ReadAllText`** usa el directorio de inicio de la consola, no el de `Set-Location`: siempre con ruta absoluta.
- **Un PATH recién ampliado** no llega a una consola abierta antes: `$env:Path = [Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [Environment]::GetEnvironmentVariable('Path','User')`.

## 6. Esperas y procesos en segundo plano

- **`sleep N; orden` con N de 25 o más está bloqueado** en Claude Code (medido: 24 pasa, 30 no). Para esperar, `run_in_background` o la herramienta **Monitor** con un bucle `until` que avisa cuando aparece el archivo de fin.
- **`run_in_background` con un `&` dentro** termina «completada» al instante aunque el proceso siga vivo: la orden entera va sin `&`.
- **Lo que tarda más de 2 minutos** se lanza en segundo plano, y el guion escribe un `FIN-*.txt` al terminar.
- **Antes de relanzar una batería, se mata la anterior entera:** el proceso padre y sus hijos, buscándolos por la carpeta en la línea de órdenes.
- **Al terminar, ningún proceso propio queda vivo.** Se comprueba (§3, paso 6).

## 7. Navegador (si el proyecto tiene interfaz web)

- **El integrado:** se empieza con `navigate` a una dirección real (`http://127.0.0.1:PUERTO/…`; `about:blank` y `data:` no valen). Para `next dev` se usa `localhost`.
  - Si el panel está oculto, las capturas agotan el tiempo y no hay teclas reales: se lee con `get_page_text`, `find` o `javascript_tool`, y el informe dice que no hubo teclas reales.
  - `computer type` no dispara `keydown`: para lo que reacciona a cada tecla se usa `computer key`.
- **Para teclas reales y capturas fiables:** un navegador sin ventana dirigido por el protocolo de depuración. **No se cierra ningún navegador que uno no haya abierto.**
- **Un PDF no se puede mirar con Read** si no hay `pdftoppm`: se lee su texto, o se imprime y se abre en el mismo navegador sin ventana. No se instala nada sin decisión del dueño.
- **Puertos reservados** (la única lista; los agentes no usan ninguno salvo que su encargo lo diga, y sus servidores simulados van en puertos altos libres):

  | Puerto | De quién |
  | --- | --- |
  | {{PUERTOS_RESERVADOS}} | |

## 8. Git y `gh`

- **Commits:** los hace el coordinador, en `{{RAMA_TRABAJO}}`.
  - Con `git add` de los archivos propios, nunca `-A` a ciegas: puede haber archivos ajenos sin seguimiento.
  - Antes del commit se mira lo preparado (`git diff --cached --stat`) y que no vaya nada sensible.
- **El mensaje del commit:** en Bash, `git commit -F - <<'EOF' … EOF`; en PowerShell, el mensaje va a un archivo temporal y se usa `git commit -F <archivo>`.
- **«Dubious ownership»** (el repositorio lo creó otra cuenta o se copió desde otra PC): se resuelve por orden, `git -c "safe.directory=<ruta>" …`, sin tocar la configuración global; o copiando el proyecto con la cuenta actual.
- **Copiar o mudar carpetas grandes:** `robocopy origen destino /E /COPY:DAT /DCOPY:DAT /R:1 /W:1 /NFL /NDL /NP /MT:16 /XJ` (el código 1 significa «copió»). Se verifica con `robocopy … /L` (0 diferencias) y la huella SHA-256 de algunos archivos clave; el original no se borra hasta que el dueño dé el sí.

## 9. Bases de datos

- **La clave** se lee de `.env` hacia una variable del proceso, sin imprimirla. Nunca `cat` ni `type` del `.env`.
- **Antes de suponer el nombre de una tabla o columna,** se consulta el catálogo.
- **Las limpiezas borran por valor exacto, nunca por patrón** (un `LIKE` borró datos reales de una base de pruebas), y se cuenta después.
- **En la base de desarrollo no se crean datos de prueba;** las pruebas usan su propia base, cuyo nombre lo verifica el código.

## 10. Web y documentos

- **WebFetch da 418 o 403 en algunos sitios oficiales y PDF.** Se lee con `curl` y un User-Agent de navegador.
- **Una norma o cifra externa** se cita de su fuente oficial vigente, con la fecha en que se leyó.

## 11. Cada fallo de herramienta va a revisión

Vale para el coordinador y para cada agente.

- **Qué cuenta como fallo:** una llamada que devuelve error, se bloquea, agota el tiempo o entrega una salida que no sirve (cortada, demasiado larga, vacía cuando no debía).
- **Qué hace quien lo sufre:** no repite la misma llamada. Busca la causa, encuentra la orden que sí funciona y la anota: un agente, en su informe, en «Lo aprendido sobre herramientas»; el coordinador, al final de la parte 1 de `HERRAMIENTAS-registro.md`.
- **Qué dice la línea:** la herramienta, qué se intentaba, el mensaje, la causa (**verificada** o **supuesta**) y la orden de reemplazo, escrita para que otro pueda copiarla. Si ya tenía regla y volvió a pasar, lleva la palabra **repetido**.
- **Al cerrar cada sub-etapa,** el coordinador compara el registro y **sube aquí** lo que se repite. Si una regla escrita no basta (el mismo fallo en varios agentes seguidos), propone al dueño un freno automático, como el hook de §2.
