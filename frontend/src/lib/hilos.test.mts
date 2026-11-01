import assert from "node:assert/strict";
import { test } from "node:test";
import type { Paso } from "./api.ts";
import { armarFilas, coloresDeHilo, geometria, nombreEtapa, salidaDelPaso, tokensCortos, ventana } from "./hilos.ts";

let n = 0;
function paso(p: Partial<Paso>): Paso {
  return {
    n: n++, hora: "2026-10-31T10:00:00Z", tipo: "Lee", herramienta: "Read", resumen: "", meta: "", error: false, hecho: true,
    turno: 1, etapa: "", lote: n, paralelo: 1, hilo: "", profundidad: 0, desvio: false, continua: -1,
    tok_entra: 0, tok_salida: 0, tok_contexto: 0, tok_tras: 0, ...p,
  };
}

test("filas: el pedido es el turno, todas de 40 px, y la etapa es una marca en su primera fila", () => {
  n = 0;
  const pasos = [paso({ tipo: "Pide", herramienta: "Mensaje" }), paso({ etapa: "Leer" }), paso({ etapa: "Leer" }), paso({ etapa: "Corregir" })];
  const filas = armarFilas(pasos);
  assert.deepEqual(filas.map((f) => f.tipo), ["turno", "paso", "paso", "paso"]);
  assert.deepEqual(filas.map((f) => f.marca && [f.marca.n, f.marca.etapa, f.marca.sigue]), [null, [1, "Leer", false], null, [2, "Corregir", false]]);
  assert.deepEqual(filas.map((f) => f.y), [0, 40, 80, 120]);
  assert.equal(geometria(filas, new Map()).alto, 160);
  assert.equal(nombreEtapa(filas[3].marca!), "Etapa 2 · Corregir");
});

test("el alto de una página es N × 40 tenga las etapas que tenga", () => {
  n = 0;
  const muchas = Array.from({ length: 50 }, (_, i) => paso({ etapa: `E${i % 6}` }));
  const una = Array.from({ length: 50 }, () => paso({ etapa: "Sola" }));
  assert.equal(geometria(armarFilas(muchas), new Map()).alto, 2000);
  assert.equal(geometria(armarFilas(una), new Map()).alto, 2000);
  assert.equal(armarFilas(muchas).length, 50);
});

test("la primera fila de una página dice su etapa aunque haya empezado antes", () => {
  n = 0;
  const antes = [paso({ tipo: "Pide", turno: 3 }), paso({ turno: 3, etapa: "Leer" }), paso({ turno: 3, etapa: "Corregir" })];
  const pagina = [paso({ turno: 3, etapa: "Corregir" }), paso({ turno: 3, etapa: "Corregir" }), paso({ turno: 3, etapa: "Probar" })];
  const filas = armarFilas(pagina, antes);
  assert.deepEqual(filas.map((f) => f.marca && [f.marca.n, f.marca.etapa, f.marca.sigue]), [[2, "Corregir", true], null, [3, "Probar", false]]);
  assert.equal(nombreEtapa(filas[0].marca!), "Etapa 2 · sigue · Corregir");
  // La que sigue no cruza el riel (no empieza aquí); la nueva sí, en el borde de arriba de su fila.
  const marcas = geometria(filas, new Map()).trazos.filter((t) => t.clase === "etapa");
  assert.deepEqual(marcas.map((t) => t.d), ["M 29 81 H 39"]);
  // Una etapa que empieza justo en la primera fila no es «sigue».
  assert.equal(armarFilas([paso({ turno: 3, etapa: "Probar" })], antes)[0].marca?.sigue, false);
  // Sin los pasos de antes (una página vieja) no se sabe el número hasta el próximo turno.
  const vieja = armarFilas([...pagina, paso({ tipo: "Pide", turno: 4 }), paso({ turno: 4, etapa: "Leer" })], null);
  assert.deepEqual(vieja.map((f) => f.marca?.n ?? null), [null, null, null, null, 1]);
  assert.equal(nombreEtapa(vieja[0].marca!), "Etapa · Corregir");
});

test("geometría: desvío, vuelta, mismo archivo, paralelo y delegación", () => {
  n = 0;
  const pasos = [
    paso({ hilo: "a.rs" }), // 0
    paso({ hilo: "b.rs", profundidad: 1, lote: 50, paralelo: 2 }), // 1
    paso({ hilo: "c.rs", profundidad: 1, lote: 50, paralelo: 2 }), // 2
    paso({ hilo: "a.rs", tipo: "Edita", continua: 0 }), // 3
    paso({ tipo: "Delega", herramienta: "Agent", subagente: "s~x" }), // 4
  ];
  const filas = armarFilas(pasos);
  const { trazos, nodos } = geometria(filas, coloresDeHilo(pasos));
  const de = (c: string) => trazos.filter((t) => t.clase === c);
  assert.equal(de("riel").length, 1);
  assert.equal(de("desvio").length, 2, "el codo de entrada y el riel del desvío");
  assert.equal(de("vuelta").length, 1);
  assert.equal(de("hilo").length, 1);
  assert.equal(de("hilo")[0].color, "var(--hilo-1)", "a.rs fue el primer hilo");
  assert.equal(de("paralelo").length, 1);
  assert.equal(de("delega").length, 1);
  assert.deepEqual(nodos.map((d) => d.cx), [34, 56, 56, 34, 34]);
});

test("un enlace a un paso fuera del corte no se dibuja", () => {
  n = 10;
  const filas = armarFilas([paso({ hilo: "a.rs", continua: 3 })]);
  assert.equal(geometria(filas, new Map()).trazos.filter((t) => t.clase === "hilo").length, 0);
});

test("cifras y reparto", () => {
  assert.equal(tokensCortos(412), "412");
  assert.equal(tokensCortos(108000), "108,0 k");
  assert.equal(tokensCortos(4920000), "4,92 M");
  assert.equal(salidaDelPaso(paso({ tok_salida: 900, paralelo: 3 })), 300);
  assert.equal(salidaDelPaso(paso({ tipo: "Responde", herramienta: "Respuesta", tok_salida: 900, paralelo: 3 })), 900);
  assert.equal(ventana([paso({ tok_contexto: 250_000 })]), 1_000_000);
  assert.equal(ventana([paso({ tok_contexto: 90_000 })]), 200_000);
});
