import assert from "node:assert/strict";
import { test } from "node:test";
import { borradorVacio, cuerpo, elegir, elegirOtra, motivoImagen, type PreguntaDelCentro, respondida, textoRespuesta, visibles } from "./preguntas.ts";

const qs: PreguntaDelCentro[] = [
  { id: "medir", texto: "¿Mido ahora?", opciones: [{ etiqueta: "Ahora" }, { etiqueta: "Después" }] },
  { id: "d2", texto: "¿Abro D2?", opciones: [{ etiqueta: "Sí, en paralelo", abre: ["effort"] }, { etiqueta: "No" }] },
  { id: "effort", texto: "¿Effort?", opciones: [{ etiqueta: "Medio" }, { etiqueta: "Alto" }], depende: { pregunta: "d2", opcion: "Sí, en paralelo" } },
  { id: "pruebas", texto: "¿Qué corro?", multiple: true, opciones: [{ etiqueta: "C3" }, { etiqueta: "W6" }] },
];

test("una pregunta que depende de una opción solo aparece si se eligió", () => {
  let b = borradorVacio();
  assert.deepEqual(
    visibles(qs, b.elecciones).map((q) => q.id),
    ["medir", "d2", "pruebas"],
  );
  b = elegir(b, qs[1], "Sí, en paralelo");
  assert.deepEqual(
    visibles(qs, b.elecciones).map((q) => q.id),
    ["medir", "d2", "effort", "pruebas"],
  );
  b = elegir(b, qs[2], "Alto");
  b = elegir(b, qs[1], "No"); // volver y cambiar: la que dependía se cierra
  assert.deepEqual(
    visibles(qs, b.elecciones).map((q) => q.id),
    ["medir", "d2", "pruebas"],
  );
  // y lo que se había elegido en ella no se envía
  assert.equal(cuerpo(qs, b, "", []).elecciones.effort, undefined);
});

test("opción única reemplaza; múltiple suma y quita; «otra» necesita texto", () => {
  let b = elegir(borradorVacio(), qs[0], "Ahora");
  b = elegir(b, qs[0], "Después");
  assert.deepEqual(b.elecciones.medir, ["Después"]);
  b = elegir(b, qs[3], "C3");
  b = elegir(b, qs[3], "W6");
  b = elegir(b, qs[3], "C3");
  assert.deepEqual(b.elecciones.pruebas, ["W6"]);

  b = elegirOtra(b, qs[0]);
  assert.deepEqual(b.elecciones.medir, []);
  assert.ok(!respondida(qs[0], b), "«otra» sin texto no responde");
  b = { ...b, textos: { ...b.textos, medir: "  mañana temprano " } };
  assert.ok(respondida(qs[0], b));
  assert.equal(textoRespuesta(qs[0], b), "«mañana temprano»");
  assert.deepEqual(cuerpo(qs, b, " ok ", []).textos, { medir: "mañana temprano" });
  assert.equal(cuerpo(qs, b, " ok ", []).nota, "ok");
});

test("las imágenes: tipo, tamaño y cuántas", () => {
  assert.equal(motivoImagen("image/png", 1000, 0), null);
  assert.match(motivoImagen("application/pdf", 1000, 0) ?? "", /PNG/);
  assert.match(motivoImagen("image/png", 6 * 1024 * 1024, 0) ?? "", /5 MB/);
  assert.match(motivoImagen("image/png", 10, 4) ?? "", /hasta 4/);
});
