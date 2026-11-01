import assert from "node:assert/strict";
import { test } from "node:test";
import { cuantasPaginas, paginaDe, paginasVisibles, relleno, sinRepetir, ventanaAnclada, ventanaViva } from "./paginas.ts";

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

// El recorrido que medió el Validador en «Documentos de estado que editó» (23, de a 8): 1 → 2 → 3 → 2 → 1 → 3. La
// foto se toma al salir de la 1, como en usePaginaViva.
function recorrido<T>(elementos: T[], clave: (e: T) => string, porPagina: number, paginas: number[]) {
  let foto: string[] | null = null;
  return paginas.map((p) => {
    foto = p === 1 ? null : (foto ?? elementos.map(clave));
    return ventanaViva(elementos, clave, p, porPagina, foto).visibles.map(clave);
  });
}

test("documentos: 23 con rutas cortas repetidas pero id propio dan 8, 8, 7, 8, 8, 7 filas sin claves repetidas", () => {
  const cortas = ["docs/trazabilidad/F5/AVANCE.md", "docs/trazabilidad/ESTADO-para-retomar.md", "docs/trazabilidad/F0/AVANCE.md"];
  const docs = Array.from({ length: 23 }, (_, i) => ({ id: `d:/copia-${i}/doc-${i}.md`, ruta: cortas[i % 3] }));
  const vistas = recorrido(docs, (d) => d.id, 8, [1, 2, 3, 2, 1, 3]);
  assert.deepEqual(vistas.map((v) => v.length), [8, 8, 7, 8, 8, 7]);
  for (const v of vistas) assert.equal(new Set(v).size, v.length);
  assert.equal(new Set(vistas.slice(0, 3).flat()).size, 23, "las tres páginas cubren los 23, sin repetir");
});

test("defensa: claves repetidas no multiplican filas en ninguna página ni en la foto", () => {
  // 23 elementos con 3 claves repetidas (como cuando la clave era la ruta corta): 20 claves distintas.
  const ids = Array.from({ length: 20 }, (_, i) => `k${i}`);
  const datos = [...ids.slice(0, 5), "k1", ...ids.slice(5, 12), "k7", "k1", ...ids.slice(12)].map((id, i) => ({ id, v: i }));
  assert.equal(datos.length, 23);
  const vistas = recorrido(datos, k, 8, [1, 2, 3, 2, 1, 3]);
  assert.deepEqual(vistas.map((v) => v.length), [8, 8, 4, 8, 8, 4]);
  for (const v of vistas) assert.equal(new Set(v).size, v.length, `sin claves repetidas: ${v.join(" ")}`);
  // La primera aparición manda: k1 conserva su dato y su lugar.
  assert.equal(ventanaViva(datos, k, 1, 8, null).visibles[1].v, 1);
  const v2 = ventanaViva(datos, k, 2, 8, datos.map(k));
  assert.deepEqual([v2.total, v2.paginas, v2.nuevos], [20, 3, 0]);
});

test("sinRepetir y relleno", () => {
  const l = lista("a", "b");
  assert.equal(sinRepetir(l, k), l, "sin repetidas es la misma lista");
  assert.deepEqual(sinRepetir(lista("a", "b", "a"), k).map(k), ["a", "b"]);
  assert.equal(relleno(7, 8, 3), 1);
  assert.equal(relleno(8, 8, 3), 0);
  assert.equal(relleno(0, 50, 2), 50, "mientras carga, la página entera");
  assert.equal(relleno(3, 8, 1), 0, "con una sola página no se reserva nada");
});
