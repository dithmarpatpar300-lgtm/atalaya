import assert from "node:assert/strict";
import { test } from "node:test";
import type { Paso } from "./api.ts";
import { armarFilas } from "./hilos.ts";
import {
  ALTO_HIJO,
  ALTO_LOTE,
  cambiosDeDia,
  CANALETA,
  CANALETA_ANGOSTA,
  coloresDeArchivos,
  colocar,
  columnas,
  enlazar,
  franja,
  marcoMapa,
  opacidadCelda,
  plegar,
  renglones,
  resumenLote,
  trazar,
} from "./lotes.ts";

let n = 0;
function paso(p: Partial<Paso>): Paso {
  const i = n++;
  return {
    n: i, hora: "2026-10-31T10:00:00Z", tipo: "Lee", herramienta: "Read", resumen: "", meta: "", error: false, hecho: true,
    turno: 1, etapa: "", lote: i + 1, paralelo: 1, hilo: "", profundidad: 0, desvio: false, continua: -1,
    tok_entra: 0, tok_salida: 0, tok_contexto: 0, tok_tras: 0, ...p,
  };
}
const lotesDe = (pasos: Paso[]) => plegar(armarFilas(pasos));

test("plegar: un mensaje es un lote, y un pedido siempre va solo", () => {
  n = 0;
  const lotes = lotesDe([
    paso({ tipo: "Pide", lote: 5 }),
    paso({ lote: 5, hilo: "a.rs" }),
    paso({ lote: 5, hilo: "a.rs", tipo: "Edita", error: true }),
    paso({ lote: 5, hilo: "b.rs", etapa: "Leer" }),
    paso({ lote: 6, tipo: "Responde" }),
    paso({ lote: 7, tipo: "Pide" }),
    paso({ lote: 7, tipo: "Pide" }),
  ]);
  assert.deepEqual(lotes.map((l) => [l.n, l.filas.length, l.turno]), [[0, 1, true], [1, 3, false], [4, 1, false], [5, 1, true], [6, 1, true]]);
  assert.deepEqual(lotes[1].archivos, ["a.rs", "b.rs"]);
  assert.equal(lotes[1].edita, true);
  assert.equal(lotes[1].errores, 1);
  assert.equal(lotes[1].marca?.etapa, "Leer");
  assert.equal(lotes[0].edita, false);
});

test("resumen: lo más repetido primero", () => {
  n = 0;
  const [l] = lotesDe([paso({ lote: 1, tipo: "Ejecuta" }), paso({ lote: 1, tipo: "Edita" }), paso({ lote: 1, tipo: "Edita" }), paso({ lote: 1, tipo: "Responde" })]);
  assert.equal(resumenLote(l), "edita 2, ejecuta 1, responde 1");
});

test("enlaces: cada archivo con su última aparición, agrupados por par de lotes", () => {
  n = 0;
  const lotes = lotesDe([
    paso({ hilo: "a.rs", lote: 1 }), paso({ hilo: "b.rs", lote: 1 }), // 0
    paso({ hilo: "c.rs", lote: 2 }), // 1
    paso({ hilo: "a.rs", lote: 3 }), paso({ hilo: "b.rs", lote: 3 }), // 2
    paso({ hilo: "c.rs", lote: 4 }), paso({ hilo: "a.rs", lote: 4 }), // 3
  ]);
  const { enlaces, llegadas } = enlazar(lotes, []);
  assert.deepEqual(enlaces, [
    { a: 0, b: 2, archivos: ["a.rs", "b.rs"] },
    { a: 1, b: 3, archivos: ["c.rs"] },
    { a: 2, b: 3, archivos: ["a.rs"] },
  ]);
  assert.equal(llegadas.size, 0);
});

test("lo que viene de antes de la página es una llegada, una por nodo", () => {
  n = 10;
  const previos = [paso({ hilo: "viejo.rs" })];
  n = 20;
  const lotes = lotesDe([
    paso({ hilo: "viejo.rs", lote: 1 }), paso({ hilo: "viejo2.rs", lote: 1 }), // 0
    paso({ hilo: "nuevo.rs", lote: 2 }), // 1
    paso({ hilo: "sin-previos.rs", lote: 3, continua: 3 }), // 2: `continua` lo sabe aunque no estén los pasos de antes
  ]);
  const { llegadas } = enlazar(lotes, [...previos, { ...previos[0], hilo: "viejo2.rs" }]);
  assert.deepEqual([...llegadas], [[0, "viejo.rs"], [2, "sin-previos.rs"]]);
  // Una página vieja, sin los pasos de antes: solo avisa `continua`.
  assert.deepEqual([...enlazar(lotes, null).llegadas], [[2, "sin-previos.rs"]]);
});

test("colores: por primera aparición; lo que llega después no repinta a los de antes", () => {
  n = 0;
  const antes = lotesDe([paso({ hilo: "a", lote: 1 }), paso({ hilo: "b", lote: 2 })]);
  const despues = [...antes, ...lotesDe([paso({ hilo: "c", lote: 3 })])];
  const m = coloresDeArchivos(despues);
  assert.deepEqual([...coloresDeArchivos(antes)], [["a", "var(--serie-1)"], ["b", "var(--serie-2)"]]);
  assert.equal(m.get("a"), "var(--serie-1)");
  assert.equal(m.get("c"), "var(--serie-3)");
});

test("pista: los cortos adentro, y los que solo se tocan en un nodo comparten carril", () => {
  const p = colocar([{ a: 0, b: 1, archivos: ["x"] }, { a: 1, b: 2, archivos: ["y"] }, { a: 0, b: 2, archivos: ["z", "w"] }], CANALETA);
  assert.equal(p.rail, CANALETA - 22);
  const por = new Map(p.colocados.map((e) => [e.archivos[0], e]));
  assert.equal(por.get("x")?.carril, 0);
  assert.equal(por.get("y")?.carril, 0, "x e y se tocan en el lote 1: una línea que sigue");
  assert.equal(por.get("z")?.carril, 1);
  assert.equal(por.get("x")?.x, 12);
  assert.equal(por.get("z")?.x, 18);
});

test("pista: lo que no cabe en la canaleta sobra y no pasa de su ancho", () => {
  const enlaces = Array.from({ length: 40 }, (_, i) => ({ a: 0, b: i + 1, archivos: ["a", "b", "c", "d", "e", "f", "g", "h", "i"] }));
  const p = colocar(enlaces, CANALETA);
  assert.ok(p.sobran.length > 0);
  assert.equal(p.colocados.length + p.sobran.length, 40);
  for (const e of p.colocados) assert.ok(p.rail - e.x - 7 * 2.6 > 0, "la hebra más lejana queda dentro");
  // Sobran los de afuera, los largos: los cortos siguen a la vista.
  assert.equal(p.colocados.at(0)?.b, 1);
});

test("pista angosta: solo los del lote elegido", () => {
  const enlaces = [{ a: 0, b: 1, archivos: ["x"] }, { a: 1, b: 2, archivos: ["y"] }, { a: 2, b: 3, archivos: ["z"] }];
  const p = colocar(enlaces, CANALETA_ANGOSTA, 1);
  assert.equal(p.rail, CANALETA_ANGOSTA - 22);
  assert.deepEqual(p.colocados.map((e) => e.archivos[0]), ["x", "y"]);
});

test("renglones: 40 por lote y 30 por paso abierto, sin abrir uno de un paso", () => {
  n = 0;
  const lotes = lotesDe([paso({ lote: 1 }), paso({ lote: 1 }), paso({ lote: 1 }), paso({ lote: 2 })]);
  const { filas, cy, alto } = renglones(lotes, () => true);
  assert.deepEqual(filas.map((f) => f.clave), ["l0", "p0", "p1", "p2", "l3"]);
  assert.deepEqual(filas.map((f) => f.y), [0, 40, 70, 100, 130]);
  assert.deepEqual(cy, [20, 150]);
  assert.equal(alto, 2 * ALTO_LOTE + 3 * ALTO_HIJO);
  assert.equal(renglones(lotes, () => false).alto, 80);
});

test("trazos: el riel, cada hebra con esquinas de radio fijo, y el elegido encima", () => {
  const colores = new Map([["a", "var(--serie-1)"], ["b", "var(--serie-2)"], ["c", "var(--serie-3)"]]);
  const p = colocar([{ a: 0, b: 2, archivos: ["a", "b"] }, { a: 0, b: 1, archivos: ["c"] }], CANALETA);
  const cy = [20, 60, 100];
  const trazos = trazar(p, cy, 120, 1, colores, new Map());
  assert.equal(trazos[0].d, "M 98 0 V 120");
  // El carril de afuera lleva dos cortes, los de sus tramos horizontales; luego sus hebras; el elegido va al final.
  assert.equal(trazos.filter((x) => x.color === "var(--superficie)").length, 2);
  const ultimo = trazos[trazos.length - 1];
  assert.equal(ultimo.color, "var(--serie-3)");
  assert.equal(ultimo.opacidad, 1);
  assert.match(ultimo.d, /^M 91 20 H 92 A 6 6 0 0 0 86 26 V 54 A 6 6 0 0 0 92 60 H 91$/);
  assert.equal(trazos.find((x) => x.color === "var(--serie-1)")?.opacidad, 0.42);
});

test("trazos: un salto corto no se pasa con un radio mayor que media altura", () => {
  const p = colocar([{ a: 0, b: 1, archivos: ["a"] }], CANALETA);
  const [, hebra] = trazar(p, [20, 28], 40, -1, new Map(), new Map());
  assert.match(hebra.d, / A 4 4 0 0 0 /);
});

test("trazos: una sola llegada por nodo, también para lo que no cupo", () => {
  const p = { rail: 98, colocados: [], sobran: [{ a: 0, b: 1, archivos: ["x"] }, { a: 0, b: 2, archivos: ["y"] }] };
  const trazos = trazar(p, [20, 60, 100], 120, -1, new Map([["y", "var(--serie-2)"]]), new Map([[1, "viejo"]]));
  const cortos = trazos.filter((x) => x.raya);
  assert.equal(cortos.length, 2);
  assert.equal(cortos[0].d, "M 105 60 H 108 A 4 4 0 0 0 112 56 V 45");
  assert.equal(cortos[1].color, "var(--serie-2)");
});

test("franja: lo que retoma y lo que deja, en palabras", () => {
  n = 0;
  const lotes = lotesDe([paso({ hilo: "src/a.rs", lote: 1 }), paso({ hilo: "b.rs", lote: 1 }), paso({ lote: 2 })]);
  const enlaces = [{ a: 5, b: 0, archivos: ["x", "y"] }, { a: 0, b: 7, archivos: ["z"] }];
  assert.equal(franja(lotes[0], 0, enlaces, "23:33"), "23:33, 2 pasos: retoma 2 archivos y sigue 1 archivo después. a.rs, b.rs");
  assert.equal(franja(lotes[1], 1, [], "23:40"), "23:40, 1 paso");
});

test("columnas: lo que no cabe se va, y el teléfono es otra plantilla", () => {
  assert.equal(columnas(900, true).plantilla, "46px 120px 42px minmax(0,1fr) 230px 84px 14px");
  assert.equal(columnas(0, false).plantilla, "46px 120px 42px minmax(0,1fr) 230px 14px", "sin medir todavía, la ancha");
  assert.equal(columnas(650, true).plantilla, "46px 120px 42px minmax(0,1fr) 150px 72px 14px");
  assert.deepEqual([columnas(570, true).archivos, columnas(570, true).tokens], ["puntos", true]);
  assert.deepEqual([columnas(500, true).archivos, columnas(500, true).tokens], ["puntos", false]);
  const tel = columnas(340, true);
  assert.deepEqual([tel.angosto, tel.archivos, tel.tokens, tel.plantilla], [true, null, false, "40px 56px 30px minmax(0,1fr) 14px"]);
});

test("mapa: la intensidad, el marco y los cambios de día", () => {
  assert.equal(opacidadCelda(0), 0.16);
  assert.equal(opacidadCelda(1), 1);
  assert.deepEqual(marcoMapa(0, 100, 200), { top: 0, alto: 50 });
  assert.deepEqual(marcoMapa(100, 200, 200), { top: 50, alto: 50 });
  // Un tramo minúsculo se ve igual, y el marco no se sale del mapa.
  assert.deepEqual(marcoMapa(9999, 10000, 10000), { top: 98.5, alto: 1.5 });
  assert.deepEqual(marcoMapa(0, 0, 0), { top: 0, alto: 100 });
  assert.deepEqual(cambiosDeDia([{ hora: "2026-10-31T10:00:00" }, { hora: "2026-10-31T12:00:00" }, { hora: "2026-11-01T01:00:00" }]), [0, 2]);
});
