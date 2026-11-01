import assert from "node:assert/strict";
import { test } from "node:test";
import { cuotaEn, curva, enParaleloCon, type FilaLinea, marcas, minutos, nombreDia, ordenar, paralelo, pct, tramo } from "./linea.ts";

const fila = (f: Partial<FilaLinea>): FilaLinea => ({
  id: "x",
  nombre: "x",
  proveedor: "Claude",
  tipo: "subagente",
  clase: "",
  estado: "Entregado",
  inicio: "2026-10-31T10:00:00Z",
  fin: "2026-10-31T10:20:00Z",
  vivo: false,
  excede: false,
  choques: [],
  ...f,
});

const filas = [
  fila({ id: "b", tipo: "subagente", padre: "s1", inicio: "2026-10-31T10:30:00Z", fin: "2026-10-31T10:50:00Z" }),
  fila({ id: "s1", tipo: "sesion", inicio: "2026-10-31T09:10:00Z", fin: "2026-10-31T12:00:00Z" }),
  fila({ id: "a", tipo: "subagente", padre: "s1", inicio: "2026-10-31T10:00:00Z", fin: "2026-10-31T10:40:00Z" }),
  fila({ id: "c", tipo: "codex", proveedor: "Codex", padre: "otra", inicio: "2026-10-31T11:00:00Z", fin: "2026-10-31T11:05:00Z" }),
];

test("cada sesión va seguida de sus agentes; los huérfanos al final", () => {
  assert.deepEqual(
    ordenar(filas, true).map((f) => `${f.id}${f.nivel}`),
    ["s10", "a1", "b1", "c0"],
  );
  assert.deepEqual(
    ordenar(filas, false).map((f) => `${f.id}${f.nivel}`),
    ["a0", "b0", "c0"],
  );
});

test("el tramo va de media hora en media hora y las marcas no pasan de diez", () => {
  const o = ordenar(filas, true);
  const dia = Date.parse("2026-10-31T00:00:00Z");
  const [ini, fin] = tramo(o, dia, dia + 86_400_000);
  // La sesión de 09:10 a 12:00 no marca el tramo: lo marcan sus agentes (10:00 a 11:05).
  assert.equal(new Date(ini).toISOString(), "2026-10-31T10:00:00.000Z");
  assert.equal(new Date(fin).toISOString(), "2026-10-31T11:30:00.000Z");
  // Una sesión viva alarga el final hasta su último paso (ahora).
  const viva = o.map((f) => (f.tipo === "sesion" ? { ...f, vivo: true } : f));
  assert.equal(new Date(tramo(viva, dia, dia + 86_400_000)[1]).toISOString(), "2026-10-31T12:00:00.000Z");
  // Un día con solo la sesión: la sesión lo marca.
  const [i2, f2] = tramo(
    o.filter((f) => f.tipo === "sesion"),
    dia,
    dia + 86_400_000,
  );
  assert.equal(new Date(i2).toISOString(), "2026-10-31T09:00:00.000Z");
  assert.equal(new Date(f2).toISOString(), "2026-10-31T12:00:00.000Z");
  const ms = marcas(ini, fin);
  assert.ok(ms.length <= 10 && ms.length >= 3);
  assert.equal(pct(ini, ini, fin), 0);
  assert.equal(pct(fin + 1, ini, fin), 100);
});

test("a la vez y en paralelo no cuentan la sesión", () => {
  const o = ordenar(filas, true);
  const [ini, fin] = [Date.parse("2026-10-31T10:30:00Z"), Date.parse("2026-10-31T10:40:00Z")];
  assert.deepEqual(
    paralelo(o, ini, fin).map((p) => p.n),
    [2, 2],
  );
  const a = o.find((f) => f.id === "a")!;
  assert.deepEqual(enParaleloCon(a, o), ["x"]);
});

test("la cuota va en escalones y se lee la última antes de la hora", () => {
  const ini = Date.parse("2026-10-31T10:00:00Z");
  const fin = ini + 100 * 60_000;
  const pts = [
    { t: ini / 1000, u: 10 },
    { t: (ini + 50 * 60_000) / 1000, u: 60 },
  ];
  assert.equal(curva(pts, ini, fin, 100), "0.0,90.0 500.0,90.0 500.0,40.0");
  assert.equal(curva([], ini, fin, 100), "");
  assert.equal(cuotaEn(pts, ini + 49 * 60_000), 10);
  assert.equal(cuotaEn(pts, ini + 51 * 60_000), 60);
  assert.equal(cuotaEn(pts, ini - 1), null);
});

test("textos", () => {
  assert.equal(minutos(29 * 60_000), "29 min");
  assert.equal(minutos(75 * 60_000), "1 h 15 min");
  assert.equal(nombreDia("2026-10-31", "2026-10-31"), "Hoy");
  assert.equal(nombreDia("2026-10-30", "2026-10-31"), "Ayer");
  assert.equal(nombreDia("2026-10-29", "2026-10-31"), "29 oct");
});
