import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import { EN_AGENTES } from "./i18n/en-agentes.ts";
import { EN_CABECERA } from "./i18n/en-cabecera.ts";
import { EN_CONFIGURAR } from "./i18n/en-configurar.ts";
import { EN_VISTAS } from "./i18n/en-vistas.ts";

const src = path.join(path.dirname(fileURLToPath(import.meta.url)), "..");
const EN: Record<string, string> = { ...EN_CABECERA, ...EN_AGENTES, ...EN_VISTAS, ...EN_CONFIGURAR };

/** Cada t("…") con la frase escrita en el código, en todo src/ (sin las pruebas). */
function clavesDelCodigo(): { clave: string; donde: string }[] {
  const fuera: { clave: string; donde: string }[] = [];
  const llamada = /\bt\(\s*(["'`])((?:\\.|(?!\1).)*)\1/g;
  function recorrer(d: string) {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name);
      if (e.isDirectory()) recorrer(p);
      else if (/\.tsx?$/.test(e.name)) {
        // Sin comentarios: un ejemplo como t("Tus proyectos") en un comentario no es una llamada.
        const texto = fs
          .readFileSync(p, "utf8")
          .replace(/\/\*[\s\S]*?\*\//g, (c) => c.replace(/[^\n]/g, " "))
          .replace(/(^|[^:])\/\/.*$/gm, "$1");
        for (const m of texto.matchAll(llamada)) {
          // Una plantilla con ${…} no es una clave fija: no se puede comprobar aquí.
          if (m[1] === "`" && m[2].includes("${")) continue;
          const clave = m[1] === "`" ? m[2] : JSON.parse(`"${m[2].replace(/\\'/g, "'").replace(/"/g, '\\"')}"`);
          const linea = texto.slice(0, m.index).split("\n").length;
          fuera.push({ clave, donde: `${path.relative(src, p)}:${linea}` });
        }
      }
    }
  }
  recorrer(src);
  return fuera;
}

test("cada frase que el código pasa a t() tiene su inglés", () => {
  const faltan = clavesDelCodigo().filter(({ clave }) => !(clave in EN));
  assert.deepEqual(
    faltan.map(({ clave, donde }) => `${donde} «${clave}»`),
    [],
  );
});
