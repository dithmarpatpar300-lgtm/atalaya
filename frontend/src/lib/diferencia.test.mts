import assert from "node:assert/strict";
import { test } from "node:test";
import { diferencia } from "./diferencia.ts";

const lineas = (d: ReturnType<typeof diferencia>) => d.map((l) => `${l.signo}${l.texto}`);

// Agregar un proyecto al final del registro: la línea que gana una coma se ve quitada y agregada en su lugar, y la
// nueva al final; lo demás, sin cambio.
test("un proyecto nuevo al final", () => {
  const antes = '[\n  { "nombre": "A" },\n  { "nombre": "B" }\n]\n';
  const despues = '[\n  { "nombre": "A" },\n  { "nombre": "B" },\n  { "nombre": "C" }\n]\n';
  assert.deepEqual(lineas(diferencia(antes, despues)), [" [", '   { "nombre": "A" },', '−  { "nombre": "B" }', '+  { "nombre": "B" },', '+  { "nombre": "C" }', " ]"]);
});

test("un archivo nuevo y uno que se vacía", () => {
  assert.deepEqual(lineas(diferencia("", "{\n}\n")), ["+{", "+}"]);
  assert.deepEqual(lineas(diferencia("[a]\n", "")), ["−[a]"]);
  assert.deepEqual(diferencia("igual\n", "igual\n"), [{ signo: " ", texto: "igual" }]);
});
