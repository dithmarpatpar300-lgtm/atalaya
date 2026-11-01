import assert from "node:assert/strict";
import { test } from "node:test";
import { cuantasPaginas, paginaDe, paginasVisibles, ventanaAnclada, ventanaViva } from "./paginas.ts";

const dibujo = (actual: number, total: number) =>
  paginasVisibles(actual, total)
    .map((c) => (c.tipo === "pagina" ? String(c.numero) : "…"))
    .join(" ");

test("la ventana de números: primera, última y dos vecinas, «…» solo si esconde dos o más", () => {
  assert.equal(dibujo(6, 99), "1 … 4 5 6 7 8 … 99");
  assert.equal(dibujo(3, 99), "1 2 3 4 5 … 99");
  assert.equal(dibujo(1, 21), "1 2 3 … 21");
  assert.equal(dibujo(4, 21), "1 2 3 4 5 6 … 21");
  assert.equal(dibujo(21, 21), "1 … 19 20 21");
  assert.equal(dibujo(1, 1), "1");
  assert.equal(dibujo(9, 3), "1 2 3");
  assert.equal(dibujo(1, 0), "");
});

test("cuántas páginas: al menos una", () => {
  assert.equal(cuantasPaginas(0, 25), 1);
  assert.equal(cuantasPaginas(25, 25), 1);
  assert.equal(cuantasPaginas(26, 25), 2);
  assert.equal(cuantasPaginas(509, 25), 21);
});

const k = (e: { id: string }) => e.id;
const lista = (...ids: string[]) => ids.map((id) => ({ id, v: id.toUpperCase() }));

test("la página 1 es la de ahora: lo que llega se ve", () => {
  const v = ventanaViva(lista("e", "d", "c", "b", "a"), k, 1, 2, null);
  assert.deepEqual(v.visibles.map(k), ["e", "d"]);
  assert.deepEqual([v.desde, v.hasta, v.total, v.paginas, v.nuevos], [1, 2, 5, 3, 0]);
});

test("en otra página el orden se congela: lo que llega no corre las filas y se cuenta aparte", () => {
  const foto = ["d", "c", "b", "a"];
  const ahora = [{ id: "f", v: "F" }, { id: "e", v: "E" }, { id: "d", v: "D" }, { id: "c", v: "c al día" }, { id: "b", v: "B" }, { id: "a", v: "A" }];
  const v = ventanaViva(ahora, k, 2, 2, foto);
  assert.deepEqual(v.visibles.map((e) => e.v), ["B", "A"]);
  assert.deepEqual([v.desde, v.hasta, v.total, v.nuevos], [3, 4, 4, 2]);
  // El dato de cada fila es el de ahora, aunque el orden sea el de la foto.
  assert.equal(ventanaViva(ahora, k, 1, 2, foto).visibles[0].id, "f");
  assert.equal(ventanaViva(ahora, k, 2, 1, foto).visibles[0].v, "c al día");
});

test("una página que ya no existe se acota a la última", () => {
  const v = ventanaViva(lista("a", "b", "c"), k, 9, 2, null);
  assert.deepEqual([v.pagina, v.desde, v.hasta], [2, 3, 3]);
  assert.deepEqual(ventanaViva([], k, 1, 25, null), { visibles: [], desde: 0, hasta: 0, total: 0, paginas: 1, pagina: 1, nuevos: 0 });
});

test("anclada: la última página llena y crece por abajo; la primera es la corta", () => {
  assert.deepEqual(ventanaAnclada(509, 509, 50, 11), { inicio: 459, fin: 509, paginas: 11, pagina: 11 });
  assert.deepEqual(ventanaAnclada(509, 509, 50, 1), { inicio: 0, fin: 9, paginas: 11, pagina: 1 });
  assert.deepEqual(ventanaAnclada(509, 509, 50, 2), { inicio: 9, fin: 59, paginas: 11, pagina: 2 });
  // Llegan 7 pasos: la última página los suma y las demás no cambian.
  assert.deepEqual(ventanaAnclada(516, 509, 50, 11), { inicio: 459, fin: 516, paginas: 11, pagina: 11 });
  assert.deepEqual(ventanaAnclada(516, 509, 50, 2), { inicio: 9, fin: 59, paginas: 11, pagina: 2 });
  // Pocos pasos: una sola página con todo.
  assert.deepEqual(ventanaAnclada(30, 30, 50, 1), { inicio: 0, fin: 30, paginas: 1, pagina: 1 });
  assert.deepEqual(ventanaAnclada(0, 0, 50, 1), { inicio: 0, fin: 0, paginas: 1, pagina: 1 });
});

test("anclada: la página de cada posición", () => {
  for (const [total, ancla] of [[509, 509], [516, 509], [120, 100], [50, 50]] as const) {
    for (let i = 0; i < total; i++) {
      const p = paginaDe(i, total, ancla, 50);
      const v = ventanaAnclada(total, ancla, 50, p);
      assert.ok(v.inicio <= i && i < v.fin, `${i} en la página ${p} (${v.inicio}–${v.fin}) con ${total}/${ancla}`);
    }
  }
});
