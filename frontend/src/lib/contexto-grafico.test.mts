import assert from "node:assert/strict";
import { test } from "node:test";
import {
  colocarEnCarril,
  compactacionVisible,
  desdeDeRango,
  escalaY,
  rangoPorOmision,
  rangosPosibles,
  tramos,
  ULTIMAS,
} from "./contexto-grafico.ts";
import { repartirRotulos } from "./rotulos.ts";

test("el rango por omisión empieza en la última compactación con curva detrás", () => {
  // Una sesión larga: 1500 peticiones; dos compactaciones antes de la serie (índice negativo) y dos dentro.
  const comps = [{ peticion: -300 }, { peticion: -2 }, { peticion: 400 }, { peticion: 1200 }];
  assert.equal(rangoPorOmision(1500, comps), "compactacion");
  assert.equal(desdeDeRango("compactacion", 1500, comps), 1192);
  assert.deepEqual(rangosPosibles(1500, comps), ["compactacion", "ultimas", "todas"]);
  // Las de antes de la serie y la que cae fuera del rango no se dibujan: ni pegadas al borde.
  const desde = desdeDeRango("compactacion", 1500, comps);
  assert.deepEqual(
    comps.map((k) => compactacionVisible(k.peticion, desde, 1500)),
    [false, false, false, true],
  );
  assert.deepEqual(
    comps.map((k) => compactacionVisible(k.peticion, 0, 1500)),
    [false, false, true, true],
  );
});

test("sin compactación útil: las últimas en una serie larga, todas en una corta", () => {
  assert.equal(rangoPorOmision(1500, [{ peticion: 1495 }]), "ultimas");
  assert.equal(desdeDeRango("ultimas", 1500, []), 1500 - ULTIMAS);
  assert.equal(rangoPorOmision(120, []), "todas");
  assert.deepEqual(rangosPosibles(120, [{ peticion: -5 }]), ["todas"]);
  // Pedir «compactacion» sin ninguna dentro vuelve al principio, sin romperse.
  assert.equal(desdeDeRango("compactacion", 120, []), 0);
});

test("la escala tiene un tope redondo por encima del máximo", () => {
  assert.deepEqual(escalaY(967_100), { tope: 1_250_000, ticks: [250_000, 500_000, 750_000, 1_000_000, 1_250_000] });
  assert.deepEqual(escalaY(100_000).ticks, [25_000, 50_000, 75_000, 100_000, 125_000]);
  const e = escalaY(150_000);
  assert.ok(e.tope >= 150_000 * 1.08);
  assert.equal(e.ticks[e.ticks.length - 1], e.tope);
  assert.ok(escalaY(0).tope > 0);
});

test("los tramos cambian con la etapa o con el turno", () => {
  const ps = [
    { turno: 1, etapa: "" },
    { turno: 1, etapa: "" },
    { turno: 1, etapa: "Leer" },
    { turno: 2, etapa: "Leer" },
    { turno: 2, etapa: "Leer" },
  ];
  assert.deepEqual(
    tramos(ps).map((t) => [t.desde, t.hasta]),
    [
      [0, 2],
      [2, 3],
      [3, 5],
    ],
  );
});

/** Dos cajas [left, left+ancho) separadas al menos por `hueco`. */
function sinChoques(cajas: { left: number; ancho: number }[], hueco: number) {
  const o = [...cajas].sort((a, b) => a.left - b.left);
  for (let j = 1; j < o.length; j++) assert.ok(o[j - 1].left + o[j - 1].ancho + hueco <= o[j].left + 1e-9, `chocan ${j - 1} y ${j}`);
}

test("choques: muchas compactaciones y muchos turnos juntos nunca se montan", () => {
  for (const carril of [280, 640, 1100]) {
    // 60 compactaciones: rachas apiñadas, dos en el mismo punto y algunas en los bordes.
    const xs = [0, 0, 1, 2, 3, ...Array.from({ length: 40 }, (_, i) => 100 + i * 4), carril / 2, carril / 2, carril - 1, carril, carril];
    const items = xs.map((x) => ({ x, ancho: 92 }));
    const g = colocarEnCarril(items, carril, () => 34);
    sinChoques(g, 6);
    for (const x of g) assert.ok(x.left >= 0 && x.left + x.ancho <= carril, "se sale del carril");
    // Nadie se pierde: cada compactación está en un solo grupo.
    assert.deepEqual(
      g.flatMap((x) => x.miembros).sort((a, b) => a - b),
      items.map((_, i) => i),
    );

    // Y 300 cambios de turno apiñados en la fila de arriba.
    const cortes = Array.from({ length: 300 }, (_, i) => (i < 200 ? i * 0.5 : 100 + (i - 200) * ((carril - 100) / 100)));
    const r = repartirRotulos(cortes, carril, 84, 8, 64);
    sinChoques(
      r.map((x) => ({ left: x.left, ancho: x.max })),
      8,
    );
    for (const x of r) assert.ok(x.left + x.max <= carril);
  }
});

test("dos compactaciones separadas quedan cada una en su sitio", () => {
  const g = colocarEnCarril(
    [
      { x: 100, ancho: 80 },
      { x: 400, ancho: 80 },
    ],
    640,
    () => 30,
  );
  assert.deepEqual(
    g.map((x) => [x.miembros, x.left]),
    [
      [[0], 60],
      [[1], 360],
    ],
  );
  // Junto al borde, se corre hacia adentro en vez de cortarse.
  assert.equal(colocarEnCarril([{ x: 2, ancho: 80 }], 640, () => 30)[0].left, 0);
  assert.equal(colocarEnCarril([{ x: 639, ancho: 80 }], 640, () => 30)[0].left, 560);
  assert.deepEqual(colocarEnCarril([], 640, () => 30), []);
});
