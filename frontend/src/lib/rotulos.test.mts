import assert from "node:assert/strict";
import { test } from "node:test";
import { repartirRotulos } from "./rotulos.ts";

test("los rótulos nunca se montan", () => {
  // 30 etapas apiñadas al final, como una sesión larga con muchos cambios de turno.
  const xs = [5, 60, 200, 210, 220, ...Array.from({ length: 30 }, (_, i) => 400 + i * 7)];
  const r = repartirRotulos(xs, 640);
  for (let j = 1; j < r.length; j++) {
    assert.ok(r[j - 1].left + r[j - 1].max + 8 <= r[j].left, `se monta ${j - 1} con ${j}`);
  }
  for (const x of r) assert.ok(x.left + x.max <= 640 - 8 && x.max >= 40);
  assert.deepEqual(
    r.map((x) => x.k),
    [0, 2, 5, 17, 29],
  );
});

test("sin ancho medido no hay rótulos, y el último no se sale del borde", () => {
  assert.deepEqual(repartirRotulos([10, 300], 0), []);
  assert.deepEqual(
    repartirRotulos([10, 610], 640).map((x) => x.k),
    [0],
  );
  assert.equal(repartirRotulos([10], 640)[0].max, 200);
});
