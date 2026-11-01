import assert from "node:assert/strict";
import { test } from "node:test";
import { type Celda, type FilaProceso, filaInicial, hallazgos, peso, puntoDe, resumen } from "./salud.ts";
import type { Regla } from "./api.ts";

const celda = (regla: string, marca: Celda["marca"]): Celda => ({ regla, marca, titulo: regla, texto: "" });

function fila(id: string, marcas: Celda["marca"][], activo = false): FilaProceso {
  const ids = ["asignacion", "effort", "cuota", "bitacora", "acta", "temporales"];
  const celdas = marcas.map((m, i) => celda(ids[i], m));
  return {
    id,
    nombre: id,
    proveedor: "Claude",
    clase: "",
    tarea: "",
    estado: activo ? "Trabajando" : "Entregado",
    activo,
    inicio: "",
    ultimo: "",
    celdas,
    faltas: marcas.filter((m) => m === "falta").length,
    medias: marcas.filter((m) => m === "medias").length,
  };
}

const reglas: Regla[] = [
  { id: "asignacion", nombre: "Asignación", umbral: "", nota: "", omision: false },
  { id: "effort", nombre: "Effort", umbral: "", nota: "", omision: false },
  { id: "cuota", nombre: "Cuota mirada", umbral: "", nota: "", omision: false },
  { id: "bitacora", nombre: "Bitácora", umbral: "", nota: "", omision: false },
  { id: "acta", nombre: "Acta", umbral: "", nota: "", omision: false },
  { id: "temporales", nombre: "Temporales", umbral: "", nota: "", omision: false },
];

test("resumen cuenta faltas, medias y temporales correctamente", () => {
  const filas = [
    fila("bien", ["cumple", "cumple", "cumple", "cumple", "cumple", "no_aplica"]),
    fila("medidor", ["cumple", "cumple", "falta", "cumple", "falta", "falta"]),
    fila("w6b", ["cumple", "falta", "cumple", "medias", "medias", "cumple"], true),
    fila("sin dato", ["cumple", "cumple", "sin_dato", "no_aplica", "no_aplica", "cumple"]),
  ];
  const temporales = [
    { ruta: "a", nombre: "a", lugar: "", bytes: 0, archivos: 0, parciales: 0, mod: "", acta_estado: "abierta" as const, estado: "agente entregado" as const },
    { ruta: "b", nombre: "b", lugar: "", bytes: 0, archivos: 0, parciales: 0, mod: "", estado: "sin agente" as const },
    { ruta: "c", nombre: "c", lugar: "", bytes: 0, archivos: 0, parciales: 0, mod: "", acta_estado: "cerrada" as const, estado: "en uso" as const },
  ];
  const r = resumen({ reglas, filas, temporales });
  assert.equal(r.agentes, 4);
  assert.equal(r.enRegla, 2, "lo «sin dato» no saca de regla");
  assert.equal(r.faltas, 4);
  assert.equal(r.conFaltas, 2);
  assert.deepEqual(r.reglasConFaltas, ["effort", "cuota mirada", "acta", "temporales"]);
  assert.equal(r.medias, 2);
  assert.equal(r.mediasActivos, 2);
  assert.equal(r.temporales, 3);
  assert.equal(r.sinActaCerrada, 2);
});

test("hallazgos van de lo más grave a lo que no se sabe", () => {
  const f = fila("x", ["sin_dato", "medias", "falta", "cumple", "no_aplica", "falta"]);
  assert.deepEqual(
    hallazgos(f).map((c) => `${c.regla}:${c.marca}`),
    ["cuota:falta", "temporales:falta", "effort:medias", "asignacion:sin_dato"],
  );
  assert.equal(puntoDe(f), "bg-error");
  assert.equal(puntoDe(fila("y", ["medias"])), "bg-aviso");
  assert.equal(puntoDe(fila("z", ["cumple", "sin_dato"])), "bg-ok");
});

test("peso con coma decimal y un decimal fijo", () => {
  assert.equal(peso(512), "512 B");
  assert.equal(peso(2148), "2,1 KB");
  assert.equal(peso(41 * 1024 * 1024), "41,0 MB");
});

test("la fila inicial respeta la elegida y si no busca la primera con faltas", () => {
  const filas = [fila("a", ["cumple"]), fila("b", ["falta"]), fila("c", ["falta"])];
  assert.equal(filaInicial(filas, "c"), "c");
  assert.equal(filaInicial(filas, "no-existe"), "b");
  assert.equal(filaInicial([fila("a", ["cumple"])], null), "a");
  assert.equal(filaInicial([], null), null);
});
