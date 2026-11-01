import assert from "node:assert/strict";
import { test } from "node:test";
import { agrupar, type Decision, eleccion, elegida, hace, tituloDecision } from "./decisiones.ts";

const decision = (d: Partial<Decision>): Decision => ({
  id: "s~1",
  agente: "s",
  proyecto: "Atalaya",
  sesion: "Atalaya",
  tipo: "botones",
  hora: "2026-10-31T15:00:00Z",
  paso: 1,
  preguntas: [{ texto: "¿Hago el commit?", opciones: [{ etiqueta: "Sí", recomendada: true }, { etiqueta: "No" }] }],
  estado: "pendiente",
  estado_sesion: "En espera",
  ...d,
});

test("agrupa lo que espera, lo viejo sin respuesta y lo respondido; lo reemplazado no se lista", () => {
  const g = agrupar([
    decision({ id: "a" }),
    decision({ id: "b", vieja: true }),
    decision({ id: "c", estado: "respondida" }),
    decision({ id: "d", estado: "reemplazada" }),
  ]);
  assert.deepEqual(
    [g.esperando.map((d) => d.id), g.sinRespuesta.map((d) => d.id), g.respondidas.map((d) => d.id)],
    [["a"], ["b"], ["c"]],
  );
});

test("el título de varias preguntas nombra sus temas", () => {
  const d = decision({
    tipo: "preguntas",
    preguntas: [
      { texto: "¿Mido?", encabezado: "Medición", opciones: [] },
      { texto: "¿Abro D2?", encabezado: "D2", opciones: [] },
      { texto: "¿Effort?", encabezado: "Effort", opciones: [] },
    ],
  });
  assert.equal(tituloDecision(d), "3 decisiones: medición, d2 y effort");
  assert.equal(tituloDecision(decision({})), "¿Hago el commit?");
});

test("lo elegido, también con varias opciones o con texto libre", () => {
  const q = { texto: "x", multiple: true, opciones: [{ etiqueta: "A" }, { etiqueta: "B" }], respuesta: "A, B" };
  assert.ok(elegida(q, { etiqueta: "B" }));
  assert.ok(!elegida({ ...q, multiple: false }, { etiqueta: "B" }));
  const libre = decision({ estado: "respondida", preguntas: [{ texto: "x", opciones: [{ etiqueta: "Sí" }], respuesta: "Sí, pero mañana" }] });
  assert.equal(eleccion(libre), "Sí, pero mañana");
  assert.ok(!elegida(libre.preguntas[0], { etiqueta: "Sí" }));
});

test("hace cuánto", () => {
  const t = Date.parse("2026-10-31T15:00:00Z");
  assert.equal(hace("2026-10-31T14:54:00Z", t), "hace 6 min");
  assert.equal(hace("2026-10-31T12:00:00Z", t), "hace 3 h");
  assert.equal(hace("2026-10-28T15:00:00Z", t), "hace 3 d");
});
