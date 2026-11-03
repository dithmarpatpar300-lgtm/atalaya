import assert from "node:assert/strict";
import { test } from "node:test";
import type { Paso } from "./api.ts";
import { armarFilas, nombreEtapa, salidaDelPaso, tokensCortos, ventana } from "./hilos.ts";

let n = 0;
function paso(p: Partial<Paso>): Paso {
  return {
    n: n++, hora: "2026-10-31T10:00:00Z", tipo: "Lee", herramienta: "Read", resumen: "", meta: "", error: false, hecho: true,
    turno: 1, etapa: "", lote: n, paralelo: 1, hilo: "", profundidad: 0, desvio: false, continua: -1,
    tok_entra: 0, tok_salida: 0, tok_contexto: 0, tok_tras: 0, ...p,
  };
}

test("filas: el pedido es el turno y la etapa es una marca en su primera fila", () => {
  n = 0;
  const pasos = [paso({ tipo: "Pide", herramienta: "Mensaje" }), paso({ etapa: "Leer" }), paso({ etapa: "Leer" }), paso({ etapa: "Corregir" })];
  const filas = armarFilas(pasos);
  assert.deepEqual(filas.map((f) => f.tipo), ["turno", "paso", "paso", "paso"]);
  assert.deepEqual(filas.map((f) => f.marca && [f.marca.n, f.marca.etapa, f.marca.sigue]), [null, [1, "Leer", false], null, [2, "Corregir", false]]);
  assert.equal(nombreEtapa(filas[3].marca!), "Etapa 2 · Corregir");
});

test("la primera fila de una página dice su etapa aunque haya empezado antes", () => {
  n = 0;
  const antes = [paso({ tipo: "Pide", turno: 3 }), paso({ turno: 3, etapa: "Leer" }), paso({ turno: 3, etapa: "Corregir" })];
  const pagina = [paso({ turno: 3, etapa: "Corregir" }), paso({ turno: 3, etapa: "Corregir" }), paso({ turno: 3, etapa: "Probar" })];
  const filas = armarFilas(pagina, antes);
  assert.deepEqual(filas.map((f) => f.marca && [f.marca.n, f.marca.etapa, f.marca.sigue]), [[2, "Corregir", true], null, [3, "Probar", false]]);
  assert.equal(nombreEtapa(filas[0].marca!), "Etapa 2 · sigue · Corregir");
  // Una etapa que empieza justo en la primera fila no es «sigue».
  assert.equal(armarFilas([paso({ turno: 3, etapa: "Probar" })], antes)[0].marca?.sigue, false);
  // Sin los pasos de antes (una página vieja) no se sabe el número hasta el próximo turno.
  const vieja = armarFilas([...pagina, paso({ tipo: "Pide", turno: 4 }), paso({ turno: 4, etapa: "Leer" })], null);
  assert.deepEqual(vieja.map((f) => f.marca?.n ?? null), [null, null, null, null, 1]);
  assert.equal(nombreEtapa(vieja[0].marca!), "Etapa · Corregir");
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
