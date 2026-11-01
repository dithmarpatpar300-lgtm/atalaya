// Automatic brake for the project (neutral operations center; docs/operacion/HERRAMIENTAS.md §2).
// Rejects Bash commands that push a program or a piece of code through a heredoc (python - <<EOF,
// node <<EOF, cat > file <<EOF, tee file <<EOF): backslashes `\\` arrive as `\` and long heredocs
// fail, and the written rule was not enough (several agents in a row repeated it).
// It also rejects `sed -i` and `perl -i`: the written rule was broken by six agents or more.
// Reads the hook's JSON from standard input; exits with 2 (and the reason on standard error) to block.
import { readFileSync } from "node:fs";

let orden = "";
try {
  // Some consoles prepend a byte order mark: it is stripped before reading the JSON.
  const entrada = readFileSync(0, "utf8");
  const sinMarca = entrada.charCodeAt(0) === 0xfeff ? entrada.slice(1) : entrada;
  orden = String(JSON.parse(sinMarca)?.tool_input?.command ?? "");
} catch {
  process.exit(0); // unreadable input: nothing is blocked
}

// The command without text: the body of each heredoc is removed (its first line stays, with the `<<` and what follows)
// and so is whatever sits between quotes. That way a commit message or a printf that mentions `perl` or `sed -i` is not stopped.
const SIN_TEXTO = orden
  .replace(/(<<-?\s*(['"]?)([A-Za-z_]\w*)\2[^\n]*)\n[\s\S]*?\n\3[ \t]*(?=\n|$)/g, "$1")
  .replace(/'[^']*'|"(?:\\.|[^"\\])*"/g, "");

// A real heredoc: `<<` or `<<-` and a delimiter, not counting the here-string `<<<`.
const HEREDOC = /(?<!<)<<-?\s*(['"]?)[A-Za-z_][A-Za-z0-9_]*\1/;
// Who receives it: an interpreter, or a `cat`/`tee` that writes a file.
const INTERPRETE = /(^|[\s;&|(])(python3?|py|node|deno|bun|ruby|perl|php)(\.exe)?(\s|$)/;
const ESCRIBE_ARCHIVO = /(^|[\s;&|(])(cat|tee)\b[^\n|]*(>{1,2}|\btee\b)|(^|[\s;&|(])tee\s+\S|<<-?\s*['"]?[A-Za-z_]\w*['"]?\s*>{1,2}/;

if (HEREDOC.test(orden) && (INTERPRETE.test(SIN_TEXTO) || ESCRIBE_ARCHIVO.test(SIN_TEXTO))) {
  process.stderr.write(
    "Command rejected by the project brake: it pushes code through a heredoc (<<). " +
      "Backslashes arrive broken and long heredocs fail. " +
      "Write the program or the file with Write (in your scratch work folder, see AGENTS.md) and run it, " +
      "or use Edit on the file. The progress log can still go through printf. " +
      "Rule: docs/operacion/HERRAMIENTAS.md, section 2.\n",
  );
  process.exit(2);
}

// `sed -i` or `perl -i` (also `-Ei`, `-pi`, `--in-place`), looking only at the command without text.
const EDITA_EN_SITIO = /(^|[\s;&|(])(sed|perl)(\.exe)?[ \t]+(?:[^\s;&|]+[ \t]+)*?(-[A-Za-z]*i[A-Za-z.]*|--in-place)(\s|=|$)/;

if (EDITA_EN_SITIO.test(SIN_TEXTO)) {
  process.stderr.write(
    "Command rejected by the project brake: it edits a file in place with sed -i or perl -i. " +
      "Changes are made with Edit (after a Read), or with Write for a new file; " +
      "sed -n for reading is still allowed. Rule: docs/operacion/HERRAMIENTAS.md, section 2.\n",
  );
  process.exit(2);
}
process.exit(0);
