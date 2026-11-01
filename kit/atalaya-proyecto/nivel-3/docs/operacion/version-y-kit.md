# Versión y entrega · {{PROYECTO}}

El paso a paso para sacar una versión y entregarla. Las secciones 1 a 4 son para un proyecto que se entrega **como
paquete** (un ejecutable, un instalador, un zip). Si el proyecto **se despliega** (una web, un servicio), vale la
sección «Si el proyecto se despliega» y las demás se borran al instalar.

- **Las reglas que lo rodean están en `AGENTS.md`:** entrega en {{DESTINO_DE_ENTREGA}}, guiones con `-Simular`, nunca `{{RAMA_PRODUCCION}}` sin el sí de {{DUENO}}.

## Si el proyecto se despliega

1. **Antes:** el código en `{{RAMA_TRABAJO}}`, con commit; las pruebas baratas pasan ({{PRUEBAS_BARATAS}}) y la batería
   cara la corrió una sola vez quien entregó (`pruebas.md`). Las migraciones nuevas, con su vuelta atrás escrita.
2. **La versión:** el número sube donde lo lleve el proyecto y queda una etiqueta del commit (`v<versión>`) y sus notas.
3. **El ensayo:** el despliegue se corre primero simulado (`-Simular` o el equivalente de la herramienta) y se lee qué
   haría. Si el proyecto tiene un entorno de prueba, se despliega ahí y se comprueba lo nuevo.
4. **Producción, con el sí de {{DUENO}}:** se pasa a `{{RAMA_PRODUCCION}}` y se despliega con la orden real, nunca en
   el horario que {{DUENO}} haya prohibido.
5. **La comprobación:** que responde, que trae lo nuevo (un texto o un número propio de la versión) y que los registros
   no muestran errores nuevos. Si algo falla, la vuelta atrás escrita en el paso 1, sin improvisar.
6. **El cierre:** las notas de la versión en el estado del proyecto y, si el ensayador intervino, su informe.

## 1. Antes de empezar

- **Lo visual tiene su maqueta aprobada,** y el código está en `{{RAMA_TRABAJO}}`, con commit.
- **Las pruebas baratas pasan:** {{PRUEBAS_BARATAS}}.
- **Lo caro lo corre una sola vez** quien entrega (`pruebas.md`).
- **Al subir la versión,** antes de la batería larga se busca la versión vieja escrita en las pruebas (`Grep` del número viejo): suelen llevarla escrita varias.
- **Las migraciones nuevas** se dicen en las notas de la versión, con su vuelta atrás.

## 2. Compilar desde un árbol limpio

1. **Un worktree limpio** del commit de la versión, en `{{TMP}}\release-<versión>\`, sin cambios sin guardar (con su acta: `HERRAMIENTAS.md` §3).
2. **El producto:** {{COMO_SE_COMPILA}}. Se comprueba:
   - que el binario sale de un árbol sin modificar;
   - que arranca (`-help` sale con código 0);
   - que trae lo nuevo, buscando un texto propio de la versión.
3. **La huella:** SHA-256 del producto, guardada junto a él.
4. **Al terminar,** se cierra el acta del worktree y recién entonces se borra, con la ruta literal.

## 3. Armar el kit

**La copia de trabajo** va en {{TMP}} o en la carpeta de kits del proyecto. **Lleva:** el producto y su huella; los guiones de instalación y actualización, con la huella nueva (solo cambian esa huella y su texto); la guía con su parte nueva; un `LEEME-<versión>.txt`.

**Qué dice el LEEME:** qué trae la versión; qué hace el dueño esa noche, en orden; cómo volver atrás y qué cuidar al hacerlo; las migraciones, o «sin migraciones».

**El control del kit, antes de entregarlo:**
- **Nada prohibido:** 0 archivos de secretos (`.env*`, llaves, certificados), volcados o respaldos de bases, datos reales, ni registros (`.log`).
- **Ningún secreto:** ni `PRIVATE KEY` ni cadenas base64 largas.
- **La sintaxis de cada `.ps1`:** `Parser::ParseFile`, con `$t = $null; $e = $null` antes.
- **Las huellas escritas** en el LEEME, la guía y los guiones coinciden con las calculadas.

## 4. El ensayo

- **Lo hace un agente independiente,** que no armó el kit y no lo corrige: el rol `ensayador`.
- **Su veredicto:** «listo para instalar» o «no listo», en su informe (el siguiente número después del de la versión).
- **Lo que se ensaya, siempre:**
  1. pasar de la versión instalada a la nueva: código 0, la configuración idéntica y las migraciones al día;
  2. repetirlo: no cambia nada;
  3. volver atrás a mano y regresar;
  4. comprobar que lo servido trae lo nuevo;
  5. las comprobaciones baratas del §3, sobre la carpeta de entrega.
- **Lo que no se ensaya** (dicho siempre en el informe): lo que solo existe en producción real, las cuentas reales, las tareas programadas reales y los sistemas externos reales.
- **Los hallazgos** van con gravedad × costo (`PRIORIZACION.md` §3). Si es medio o más, se corrige y se ensaya de nuevo lo que cambió.

## 5. Entregar

- **Destino:** {{DESTINO_DE_ENTREGA}}, en dos formas: la carpeta `Kit {{PROYECTO}} <versión>` y su `.zip` con el contenido **en la raíz** (sin carpeta superior) y las entradas con `/`.
- **Se comprueba** que el zip se abre y que la carpeta, el zip y la copia de trabajo tienen las mismas huellas (0 diferencias).

## 6. Cerrar

1. **El informe de la versión** (`docs/trazabilidad/<fase>/NN-release-<versión>.md`): qué cambió, pruebas con cifras, el kit con sus huellas, el ensayo y su resultado, lo no verificado y lo aprendido sobre herramientas.
2. **El estado para retomar,** al día: solo la sección vigente.
3. **Commit en `{{RAMA_TRABAJO}}`** con la versión, las huellas y el resultado del ensayo; después el push de esa rama.
4. **Publicar en `{{RAMA_PRODUCCION}}`,** con etiqueta y release: **solo cuando {{DUENO}} lo pide**.

## 7. Lo que hace el dueño (no el coordinador)

- Instalar con el `LEEME` en el entorno real; probar con datos reales; decir si se publica.
