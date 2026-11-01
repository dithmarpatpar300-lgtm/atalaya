import assert from "node:assert/strict";
import { test } from "node:test";
import { COLOR_HILO, type FaseAvance, horas, horasQueFaltan, paqueteInicial, tonoEstado, tramos, uno } from "./avance.ts";

const fase = (paquetes: FaseAvance["paquetes"]): Pick<FaseAvance, "paquetes"> => ({ paquetes });

test("cifras con coma y horas sin decimales de más", () => {
  assert.equal(uno(82.8), "82,8");
  assert.equal(uno(100), "100,0");
  assert.equal(horas(4), "4");
  assert.equal(horas(4.5), "4,5");
});

test("los tramos pesan por horas y solo lo en curso lleva relleno", () => {
  const ts = tramos(
    fase([
      { id: "W1", que: "a", horas: 3, estado: "hecho", pct: 100 },
      { id: "W6", que: "b", horas: 4, estado: "en curso", pct: 85 },
      { id: "W12", que: "c", horas: 0, estado: "sin empezar", pct: 0 },
      { id: "W9", que: "d", horas: 2, estado: "bloqueado", pct: 10 },
    ]),
  );
  assert.deepEqual(
    ts.map((t) => [t.paquete.id, t.peso, t.fondo, t.relleno]),
    [
      ["W1", 3, COLOR_HILO.hecho, 0],
      ["W6", 4, COLOR_HILO.falta, 85],
      ["W12", 1, COLOR_HILO.falta, 0],
      ["W9", 2, COLOR_HILO.bloqueado, 0],
    ],
  );
  assert.match(ts[1].aria, /W6, b, 4 horas, en curso, 85 %/);
});

test("lo que falta y el paquete inicial", () => {
  const f = fase([
    { id: "W5", que: "", horas: 3, estado: "en curso", pct: 50 },
    { id: "W6", que: "", horas: 4, estado: "en curso", pct: 85 },
    { id: "W12", que: "", horas: 3, estado: "sin empezar", pct: 0 },
  ]);
  assert.equal(Math.round(horasQueFaltan(f) * 10) / 10, 5.1);
  assert.equal(paqueteInicial(f, null)?.id, "W5");
  assert.equal(paqueteInicial(f, "W12")?.id, "W12");
  assert.equal(paqueteInicial(f, "no-existe")?.id, "W5");
  assert.equal(paqueteInicial(fase([]), null), null);
  assert.equal(tonoEstado("en curso").clase, "text-aviso");
  assert.equal(tonoEstado("").palabra, "sin empezar");
});
