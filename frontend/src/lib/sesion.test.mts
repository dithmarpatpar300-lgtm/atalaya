import assert from "node:assert/strict";
import { test } from "node:test";
import { efecto, loQueSiguio, type PedidoSesion, pedidoInicial } from "./sesion.ts";

const p = (c: Partial<PedidoSesion>): PedidoSesion => ({
  n: 0,
  hora: "",
  tipo: "pedido",
  texto: "",
  caracteres: "",
  lanzados: [],
  consultas: 0,
  cuota: 0,
  commits: 0,
  pasos: 0,
  etapas: [],
  ...c,
});

test("lo que siguió a un pedido, en una línea", () => {
  assert.equal(loQueSiguio(p({ lanzados: ["a"], consultas: 6, cuota: 2 })), "lanzó 1 agente, 6 consultas a Atalaya, leyó la cuota");
  assert.equal(loQueSiguio(p({ commits: 1 })), "comiteó");
  assert.equal(loQueSiguio(p({ pasos: 3 })), "3 pasos");
  assert.equal(efecto(p({ lanzados: ["a", "b"] })), "2 agentes lanzados");
  assert.equal(efecto(p({ commits: 1, consultas: 2 })), "commit");
  assert.equal(efecto(p({ pasos: 1 })), "1 paso");
});

test("el pedido inicial es el elegido o el último", () => {
  const ps = [p({ n: 1 }), p({ n: 9 })];
  assert.equal(pedidoInicial(ps, 1)?.n, 1);
  assert.equal(pedidoInicial(ps, null)?.n, 9);
  assert.equal(pedidoInicial([], null), null);
});
