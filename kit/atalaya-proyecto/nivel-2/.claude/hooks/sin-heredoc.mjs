// Freno automático del proyecto (Centro de operaciones neutro; docs/operacion/HERRAMIENTAS.md §2).
// Rechaza las órdenes de Bash que meten un programa o un trozo de código por heredoc (python - <<EOF,
// node <<EOF, cat > archivo <<EOF, tee archivo <<EOF): las barras `\\` llegan como `\` y los heredocs
// largos fallan, y la regla escrita no bastó (varios agentes seguidos la repitieron).
// Rechaza también `sed -i` y `perl -i`: la regla escrita se incumplió en seis agentes o más.
// Lee de la entrada estándar el JSON del hook; sale con 2 (y el porqué por la salida de errores) para bloquear.
import { readFileSync } from "node:fs";

let orden = "";
try {
  // Algunas consolas anteponen la marca de orden de bytes: se quita antes de leer el JSON.
  const entrada = readFileSync(0, "utf8");
  const sinMarca = entrada.charCodeAt(0) === 0xfeff ? entrada.slice(1) : entrada;
  orden = String(JSON.parse(sinMarca)?.tool_input?.command ?? "");
} catch {
  process.exit(0); // entrada ilegible: no se bloquea nada
}

// La orden sin texto: se quita el cuerpo de cada heredoc (queda su primera línea, con el `<<` y lo que sigue)
// y lo que va entre comillas. Así un mensaje de commit o un printf que nombre `perl` o `sed -i` no se frena.
const SIN_TEXTO = orden
  .replace(/(<<-?\s*(['"]?)([A-Za-z_]\w*)\2[^\n]*)\n[\s\S]*?\n\3[ \t]*(?=\n|$)/g, "$1")
  .replace(/'[^']*'|"(?:\\.|[^"\\])*"/g, "");

// Un heredoc de verdad: `<<` o `<<-` y un delimitador, sin contar la cadena-aquí `<<<`.
const HEREDOC = /(?<!<)<<-?\s*(['"]?)[A-Za-z_][A-Za-z0-9_]*\1/;
// Quién lo recibe: un intérprete, o un `cat`/`tee` que escribe un archivo.
const INTERPRETE = /(^|[\s;&|(])(python3?|py|node|deno|bun|ruby|perl|php)(\.exe)?(\s|$)/;
const ESCRIBE_ARCHIVO = /(^|[\s;&|(])(cat|tee)\b[^\n|]*(>{1,2}|\btee\b)|(^|[\s;&|(])tee\s+\S|<<-?\s*['"]?[A-Za-z_]\w*['"]?\s*>{1,2}/;

if (HEREDOC.test(orden) && (INTERPRETE.test(SIN_TEXTO) || ESCRIBE_ARCHIVO.test(SIN_TEXTO))) {
  process.stderr.write(
    "Orden rechazada por el freno del proyecto: mete código por heredoc (<<). " +
      "Las barras invertidas llegan rotas y los heredocs largos fallan. " +
      "Escribe el programa o el archivo con Write (en tu carpeta de trabajo de temporales, ver AGENTS.md) y ejecútalo, " +
      "o usa Edit sobre el archivo. La bitácora de avance sí puede ir con printf. " +
      "Regla: docs/operacion/HERRAMIENTAS.md, apartado 2.\n",
  );
  process.exit(2);
}

// `sed -i` o `perl -i` (también `-Ei`, `-pi`, `--in-place`), mirando solo la orden sin texto.
const EDITA_EN_SITIO = /(^|[\s;&|(])(sed|perl)(\.exe)?[ \t]+(?:[^\s;&|]+[ \t]+)*?(-[A-Za-z]*i[A-Za-z.]*|--in-place)(\s|=|$)/;

if (EDITA_EN_SITIO.test(SIN_TEXTO)) {
  process.stderr.write(
    "Orden rechazada por el freno del proyecto: edita un archivo en el sitio con sed -i o perl -i. " +
      "Los cambios se hacen con Edit (con un Read antes), o con Write para un archivo nuevo; " +
      "sed -n para leer sigue permitido. Regla: docs/operacion/HERRAMIENTAS.md, apartado 2.\n",
  );
  process.exit(2);
}
process.exit(0);
