import assert from "node:assert/strict";
import { test } from "node:test";
import type { Consulta, FilaAhorro } from "./api.ts";
import { ancho, claveDia, cuentas, delPeriodo, diasSeguidos, PASA, topeRedondo } from "./consultas.ts";

const consulta = (o: Partial<Consulta>): Consulta => ({
  hora: "2026-11-01T10:00:00-05:00",
  cliente: "",
  herramienta: "estado",
  argumentos: "",
  bytes: 0,
  ms: 0,
  error: false,
  respuesta: "",
  paso: -1,
  ...o,
});

test("delPeriodo deja la misma ventana que el servidor: ahora menos los días", () => {
  const ahora = Date.parse("2026-11-01T12:00:00Z");
  const lista = [consulta({ hora: "2026-11-01T11:00:00Z" }), consulta({ hora: "2026-10-31T11:59:00Z" }), consulta({ hora: "nada" })];
  assert.equal(delPeriodo(lista, 1, ahora).length, 1);
  assert.equal(delPeriodo(lista, 7, ahora).length, 2);
});

test("los filtros cuentan lo mismo que filtran", () => {
  const lista = [
    consulta({ ahorro_tokens: 10 }),
    consulta({ ahorro_tokens: 0, sin_medir: true, error: true }),
    consulta({ releyo: true, repetida: true, ahorro_tokens: 3 }),
  ];
  const c = cuentas(lista);
  assert.deepEqual(c, { todas: 3, ahorro: 2, releyo: 1, repetidas: 1, fallos: 1 });
  assert.equal(lista.filter(PASA.ahorro).length, c.ahorro);
});

test("diasSeguidos rellena los días sin consultas y termina en hoy", () => {
  const ahora = new Date(2026, 10, 1, 12).getTime();
  const fila = { clave: "2026-10-31", n: 31 } as FilaAhorro;
  const dias = diasSeguidos([fila], 3, ahora);
  assert.deepEqual(
    dias.map((d) => [d.clave, d.n]),
    [
      ["2026-10-30", 0],
      ["2026-10-31", 31],
      ["2026-11-01", 0],
    ],
  );
  assert.equal(diasSeguidos(null, 1, ahora)[0].clave, claveDia(new Date(ahora)));
});

test("topeRedondo y ancho", () => {
  assert.equal(topeRedondo(34), 50);
  assert.equal(topeRedondo(15191), 20000);
  assert.equal(topeRedondo(1), 1);
  assert.equal(topeRedondo(0), 1);
  assert.equal(ancho(0, 10), 0);
  assert.equal(ancho(1, 100000), 1.5);
  assert.equal(ancho(50, 100), 50);
});
