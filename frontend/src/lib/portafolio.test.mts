import assert from "node:assert/strict";
import { test } from "node:test";
import {
  type ArchivoContrato,
  type AvancePortafolio,
  chipNivel,
  cifraAvance,
  cuentas,
  filtrar,
  fraseFalta,
  hiloEnTexto,
  horas,
  leFaltaAlgo,
  lineaFase,
  type ProyectoPortafolio,
  proyectoElegido,
  saludHoy,
  tramosHilo,
} from "./portafolio.ts";

const IDS = ["agents", "mcp", "estado", "avance", "operacion", "subagentes", "registro"];

const contrato = (faltan: string[] = []): ArchivoContrato[] => IDS.map((id) => ({ id, nombre: id, ruta: id, esta: !faltan.includes(id) }));

const avance = (a: Partial<AvancePortafolio> = {}): AvancePortafolio => ({
  fase: "F5",
  ruta: "docs/trazabilidad/F5/AVANCE.md",
  ahora: "T1b: pruebas de inventario",
  calculado: 82.8,
  horas: 53,
  horas_hechas: 40,
  horas_en_curso: 3.9,
  horas_avanzadas: 43.9,
  ...a,
});

const proyecto = (p: Partial<ProyectoPortafolio> = {}): ProyectoPortafolio => ({
  nombre: "Tienda Demo",
  sigla: "TD",
  ruta: "C:\\Proyectos\\Tienda Demo",
  nivel: 3,
  existe: true,
  fase: "F5",
  contrato: contrato(["avance"]),
  avance: null,
  salud: { en_regla: 2, total: 3 },
  vivos: 1,
  consultas: 4,
  decisiones: 0,
  ...p,
});

test("la cifra y la nota del avance, con coma", () => {
  assert.deepEqual(cifraAvance(avance()), { cifra: "82,8 %", nota: "43,9 de 53 h" });
  assert.deepEqual(cifraAvance(avance({ calculado: 50, horas: 10, horas_avanzadas: 5 })), { cifra: "50,0 %", nota: "5 de 10 h" });
  assert.deepEqual(cifraAvance(null), { cifra: "—", nota: "sin AVANCE.md" });
  assert.equal(cifraAvance(avance({ horas: 0 })).cifra, "—");
  assert.equal(horas(43.94), "43,9");
  assert.equal(horas(53), "53");
});

test("el hilo: hecho, en curso y lo que falta, por horas", () => {
  assert.deepEqual(tramosHilo(avance()), [
    { tipo: "hecho", peso: 40 },
    { tipo: "curso", peso: 3.9 },
    { tipo: "falta", peso: 53 - 40 - 3.9 },
  ]);
  // Sin nada en curso, el tramo ámbar no se dibuja; todo hecho, solo verde.
  assert.deepEqual(
    tramosHilo(avance({ horas_en_curso: 0 })).map((t) => t.tipo),
    ["hecho", "falta"],
  );
  assert.deepEqual(tramosHilo(avance({ horas_hechas: 53, horas_en_curso: 0 })), [{ tipo: "hecho", peso: 53 }]);
  // Lo que se pasa del total se recorta.
  assert.deepEqual(tramosHilo(avance({ horas: 10, horas_hechas: 8, horas_en_curso: 5 })), [
    { tipo: "hecho", peso: 8 },
    { tipo: "curso", peso: 2 },
  ]);
  assert.deepEqual(tramosHilo(null), [{ tipo: "vacio", peso: 1 }]);
  assert.equal(hiloEnTexto(avance({ horas: 10, horas_hechas: 3, horas_en_curso: 2 })), "3 h hechas, 2 h avanzadas en curso y 5 h por hacer, de 10 h");
  assert.equal(hiloEnTexto(null), "Sin avance que dibujar");
});

test("los filtros y sus cuentas", () => {
  const ps = [
    proyecto(),
    proyecto({ nombre: "Recetario-Demo", vivos: 0, contrato: contrato(["mcp", "avance"]) }),
    proyecto({ nombre: "Completo", vivos: 0, contrato: contrato() }),
    proyecto({ nombre: "Fantasma", vivos: 0, existe: false, contrato: contrato() }),
  ];
  assert.deepEqual(cuentas(ps), { todos: 4, vivos: 1, falta: 3 });
  assert.deepEqual(
    filtrar(ps, "falta").map((p) => p.nombre),
    ["Tienda Demo", "Recetario-Demo", "Fantasma"],
  );
  assert.equal(leFaltaAlgo(ps[2]), false);
  assert.equal(filtrar(ps, "todos"), ps);
});

test("la frase de lo que falta, dicha de los archivos", () => {
  assert.equal(fraseFalta(proyecto()), "Le falta AVANCE.md.");
  assert.equal(fraseFalta(proyecto({ contrato: contrato(["mcp", "avance"]) })), "Le faltan .mcp.json y AVANCE.md.");
  assert.equal(fraseFalta(proyecto({ contrato: contrato(["mcp", "estado", "avance"]) })), "Le faltan .mcp.json, el estado para retomar y AVANCE.md.");
  assert.equal(fraseFalta(proyecto({ contrato: contrato(IDS) })), "No tiene contrato: le faltan los 7 archivos.");
  assert.equal(fraseFalta(proyecto({ contrato: contrato() })), "Tiene todo el contrato de trazabilidad.");
  assert.match(fraseFalta(proyecto({ existe: false })), /no está en el disco/);
});

test("rótulos de la tarjeta", () => {
  assert.equal(chipNivel(3), "nivel 3");
  assert.equal(chipNivel(0), "sin contrato");
  assert.equal(lineaFase(proyecto({ avance: avance() })), "F5 · T1b: pruebas de inventario");
  assert.equal(lineaFase(proyecto()), "F5 · sin AVANCE.md");
  assert.equal(lineaFase(proyecto({ fase: undefined })), "Sin fases en docs/trazabilidad");
  assert.equal(lineaFase(proyecto({ existe: false })), "La carpeta no está en el disco");
  assert.deepEqual(saludHoy({ en_regla: 2, total: 3 }).cifra, "2 de 3");
  assert.deepEqual(saludHoy({ en_regla: 0, total: 0 }).cifra, "—");
});

test("el panel muestra el elegido si sigue a la vista", () => {
  const ps = [proyecto({ nombre: "A" }), proyecto({ nombre: "B" })];
  assert.equal(proyectoElegido(ps, "B")?.nombre, "B");
  assert.equal(proyectoElegido(ps, "Z")?.nombre, "A");
  assert.equal(proyectoElegido(ps, null)?.nombre, "A");
  assert.equal(proyectoElegido([], "A"), null);
});
