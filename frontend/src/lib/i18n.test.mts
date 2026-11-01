import assert from "node:assert/strict";
import { test } from "node:test";
import { haceCuanto, porcentaje, tokens } from "./formato.ts";
import { EN_AGENTES } from "./i18n/en-agentes.ts";
import { EN_CABECERA } from "./i18n/en-cabecera.ts";
import { EN_VISTAS } from "./i18n/en-vistas.ts";
import { decimal, fijarIdioma, t } from "./i18n.ts";

test("t: en español, la clave tal cual; en inglés, su traducción con los huecos llenos", () => {
  fijarIdioma("es");
  assert.equal(t("hace {d}", { d: "6 min" }), "hace 6 min");
  fijarIdioma("en");
  assert.equal(t("hace {d}", { d: "6 min" }), "6 min ago");
  assert.equal(t("Una frase sin traducir"), "Una frase sin traducir");
  fijarIdioma("es");
});

test("decimal y formato: coma en español, punto en inglés", () => {
  fijarIdioma("es");
  assert.equal(decimal(4.06), "4,1");
  assert.equal(tokens(181_400), "181,4 k");
  assert.equal(porcentaje(23.33), "23,3 %");
  assert.equal(haceCuanto(100, 102), "ahora");
  fijarIdioma("en");
  assert.equal(decimal(4.06), "4.1");
  assert.equal(tokens(181_400), "181.4 k");
  assert.equal(haceCuanto(0, 360), "6 min ago");
  fijarIdioma("es");
});

// Cada diccionario es de un área; una clave en dos daría la traducción del último que se mezcla.
test("los diccionarios no repiten claves y respetan los huecos", () => {
  const vistas = new Map<string, string>();
  for (const [area, dic] of Object.entries({ EN_CABECERA, EN_AGENTES, EN_VISTAS })) {
    for (const [es, en] of Object.entries(dic)) {
      assert.ok(!vistas.has(es), `«${es}» está en ${vistas.get(es)} y en ${area}`);
      vistas.set(es, area);
      const huecos = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort();
      assert.deepEqual(huecos(en), huecos(es), `«${es}» → «${en}»: los huecos no coinciden`);
    }
  }
});
