import assert from "node:assert/strict";
import { test } from "node:test";
import { detalleBarra, fija, marcasDe, porAgente, porcentajeDe, recortarVistas } from "./alertas.ts";
import type { Alerta, Regla } from "./api.ts";

const al = (id: string, nivel: Alerta["nivel"], agente: string, otro?: string): Alerta => ({
  id,
  nivel,
  regla: id.split(":")[0],
  proyecto: "RG",
  agente,
  nombre: agente,
  titulo: "",
  texto: "",
  hora: "",
  valor: 0,
  max: 0,
  chip: id,
  otro,
});

test("el chip de cada agente es su alerta más grave, y la frontera va en los dos", () => {
  const m = porAgente([al("contexto:a", "aviso", "a"), al("quieto:a", "info", "a"), al("frontera:x:a+b", "error", "a", "b"), al("quieto:c", "info", "c")]);
  assert.equal(m.get("a")?.id, "frontera:x:a+b");
  assert.equal(m.get("b")?.id, "frontera:x:a+b");
  assert.equal(m.get("c")?.id, "quieto:c");
  assert.equal(m.size, 3);
});

test("un cruce ya cerrado (sin chip) no marca la fila", () => {
  const cerrado = { ...al("frontera:x:a+b:cerrado", "info", "a", "b"), chip: "" };
  const m = porAgente([cerrado, al("contexto:b", "aviso", "b")]);
  assert.equal(m.has("a"), false);
  assert.equal(m.get("b")?.id, "contexto:b");
});

test("se guardan solo las últimas vistas", () => {
  assert.deepEqual([...recortarVistas(new Set(["a", "b", "c", "d"]), 2)], ["c", "d"]);
  assert.deepEqual([...recortarVistas(new Set(["a"]), 2)], ["a"]);
});

const reglas: Regla[] = [
  { id: "tiempo", nombre: "Tiempo del agente", umbral: "30 min", nota: "Aviso a los 25 min y alerta al cumplirse.", omision: true },
  { id: "contexto", nombre: "Contexto", umbral: "80 %", nota: "", omision: true },
  { id: "frontera", nombre: "Frontera de archivos", umbral: "siempre", nota: "", omision: true },
];

test("las cifras de la barra llevan siempre un decimal y coma", () => {
  assert.equal(fija(88), "88,0");
  assert.equal(fija(Number.NaN), "0,0");
  assert.equal(porcentajeDe({ valor: 176000, max: 200000 }), "88,0 %");
  assert.equal(porcentajeDe({ valor: 34, max: 30 }), "113,3 %");
  assert.equal(porcentajeDe({ valor: 1, max: 0 }), "0,0 %");
  assert.equal(detalleBarra({ regla: "contexto", valor: 180019, max: 200000 }), "180,0 k de 200,0 k");
  assert.equal(detalleBarra({ regla: "contexto", valor: 944873, max: 1000000 }), "944,9 k de 1,0 M");
  assert.equal(detalleBarra({ regla: "tiempo", valor: 26.7, max: 30 }), "26 de 30 min");
});

test("las marcas del umbral salen de las reglas", () => {
  assert.deepEqual(marcasDe({ regla: "contexto", max: 200000 }, reglas), [{ pos: 0.8, nombre: "umbral 80 %" }]);
  // Con un presupuesto propio de 60 min, el aviso cae 5 min antes del tope.
  assert.deepEqual(marcasDe({ regla: "tiempo", max: 60 }, reglas), [{ pos: 55 / 60, nombre: "aviso 55 min" }]);
  assert.deepEqual(marcasDe({ regla: "frontera", max: 0 }, reglas), []);
  assert.deepEqual(marcasDe({ regla: "contexto", max: 200000 }, []), []);
  assert.deepEqual(marcasDe({ regla: "tiempo", max: 4 }, reglas), []);
});
