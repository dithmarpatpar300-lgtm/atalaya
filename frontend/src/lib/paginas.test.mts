import assert from "node:assert/strict";
import { test } from "node:test";
import { anchoCasilla, CASILLAS, cuantasPaginas, paginaDe, paginasVisibles, relleno, sinRepetir, ventanaAnclada, ventanaViva } from "./paginas.ts";

const dibujo = (actual: number, total: number) =>
  paginasVisibles(actual, total)
    .map((c) => (c.tipo === "pagina" ? String(c.numero) : "…"))
    .join(" ");

test("la ventana de números: primera, última y una vecina, «…» solo si esconde dos o más", () => {
  const casos: [number, number, number, string][] = [
    [0, 1, 1, ""], [1, 1, 1, "1"], [2, 1, 2, "1 2"],
    [5, 1, 5, "1 2 3 4 5"], [7, 1, 7, "1 2 3 4 5 6 7"],
    [8, 1, 4, "1 2 3 4 5 … 8"], [8, 5, 8, "1 … 4 5 6 7 8"],
    [9, 1, 4, "1 2 3 4 5 … 9"], [9, 5, 5, "1 … 4 5 6 … 9"], [9, 6, 9, "1 … 5 6 7 8 9"],
    [12, 1, 4, "1 2 3 4 5 … 12"], [12, 5, 5, "1 … 4 5 6 … 12"],
    [12, 6, 6, "1 … 5 6 7 … 12"], [12, 7, 7, "1 … 6 7 8 … 12"],
    [12, 8, 8, "1 … 7 8 9 … 12"], [12, 9, 12, "1 … 8 9 10 11 12"],
    [52, 1, 4, "1 2 3 4 5 … 52"], [52, 5, 5, "1 … 4 5 6 … 52"],
    [52, 26, 26, "1 … 25 26 27 … 52"], [52, 48, 48, "1 … 47 48 49 … 52"],
    [52, 49, 52, "1 … 48 49 50 51 52"],
    [100, 1, 4, "1 2 3 4 5 … 100"], [100, 5, 5, "1 … 4 5 6 … 100"],
    [100, 50, 50, "1 … 49 50 51 … 100"], [100, 96, 96, "1 … 95 96 97 … 100"],
    [100, 97, 100, "1 … 96 97 98 99 100"],
    [3, 9, 9, "1 2 3"], [12, 0, 0, "1 2 3 4 5 … 12"],
    [12, -3, -3, "1 2 3 4 5 … 12"], [100, 200, 200, "1 … 96 97 98 99 100"],
  ];
  for (const [total, desde, hasta, esperado] of casos) {
    for (let p = desde; p <= hasta; p++) assert.equal(dibujo(p, total), esperado, `${p}/${total}`);
  }
});

test("siempre las mismas casillas", () => {
  for (let total = 1; total <= 130; total++) {
    for (let p = 1; p <= total; p++) assert.equal(paginasVisibles(p, total).length, Math.min(total, CASILLAS), `${p}/${total}`);
  }
});

test("ventana bien formada", () => {
  for (let total = 1; total <= 130; total++) {
    for (let p = 1; p <= total; p++) {
      const casillas = paginasVisibles(p, total);
      assert.deepEqual(casillas[0], { tipo: "pagina", numero: 1 });
      assert.deepEqual(casillas[casillas.length - 1], { tipo: "pagina", numero: total });
      assert.ok(casillas.some((c) => c.tipo === "pagina" && c.numero === p));
      const lados = new Set<string>();
      let anterior = 0;
      let hueco = false;
      for (const c of casillas) {
        if (c.tipo === "hueco") {
          assert.ok(!hueco && !lados.has(c.lado));
          lados.add(c.lado);
          hueco = true;
        } else {
          if (anterior) assert.ok(hueco ? c.numero - anterior >= 3 : c.numero - anterior === 1, `${p}/${total}`);
          anterior = c.numero;
          hueco = false;
        }
      }
    }
  }
});

test("ancho de casilla según las cifras", () => {
  for (const [total, cifras] of [[0, 1], [1, 1], [9, 1], [10, 2], [99, 2], [100, 3], [104, 3], [1000, 4]]) {
    assert.equal(anchoCasilla(total), `max(1.75rem, calc(${cifras}ch + 0.5rem))`);
  }
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
