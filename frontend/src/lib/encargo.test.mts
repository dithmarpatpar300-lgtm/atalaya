import assert from "node:assert/strict";
import { test } from "node:test";
import type { Agente, Paso } from "./api.ts";
import { archivosTocados, linaje, posicionHito } from "./encargo.ts";

function agente(p: Partial<Agente>): Agente {
  return {
    id: "x", proyecto: "P", proveedor: "Claude", tipo: "subagente", rol: "Codificador", tarea: "", modelo: "", effort: "", clase: "", presupuesto: "",
    base: "", estado: "Entregado", inicio: "2026-10-31T10:00:00Z", ultimo: "2026-10-31T10:30:00Z", tokens: { entrada: 0, cache_escrita: 0, cache_leida: 0, salida: 0 },
    avance: null, bitacora: "", informe: "", pasos: 0, ...p,
  };
}

test("linaje: padre, yo, hermanos en paralelo primero e hijos", () => {
  const ses = agente({ id: "s", tipo: "sesion", rol: "Sesión", tarea: "Fase 5" });
  const yo = agente({ id: "s~a", padre: "s", clase: "I2" });
  const par = agente({ id: "s~b", padre: "s", rol: "Auditor", inicio: "2026-10-31T10:10:00Z", ultimo: "2026-10-31T10:20:00Z" });
  const antes = agente({ id: "s~c", padre: "s", rol: "Depurador", inicio: "2026-10-31T09:00:00Z", ultimo: "2026-10-31T09:20:00Z" });
  const nodos = linaje(yo, [ses, yo, par, antes]);
  assert.deepEqual(
    nodos.map((n) => [n.nombre, n.nivel, n.nota]),
    [
      ["P", 0, "Fase 5"],
      ["Codificador I2", 1, "este"],
      ["Auditor", 1, "en paralelo"],
      ["Depurador", 1, "antes"],
    ],
  );
  assert.equal(nodos[1].yo, true);
});

test("linaje: Codex es hijo de su sesión, y la sesión muestra a sus hijos de Codex", () => {
  const ses = agente({ id: "s", tipo: "sesion", rol: "Sesión", tarea: "Fase 5", titulo: "Auditar W6" });
  const sub = agente({ id: "s~a", padre: "s", clase: "I2" });
  const cx = agente({ id: "codex~1", padre: "s", proveedor: "Codex", tipo: "codex", rol: "Segunda opinión", clase: "E2", tarea: "e2-w6b-sol", inicio: "2026-10-31T10:05:00Z", ultimo: "2026-10-31T10:09:00Z" });
  const todos = [ses, sub, cx];
  assert.deepEqual(
    linaje(cx, todos).map((n) => [n.nombre, n.nivel, n.padre, n.nota, n.proveedor]),
    [
      ["P", 0, -1, "Auditar W6", "Claude"],
      ["Segunda opinión E2", 1, 0, "este", "Codex"],
      ["Codificador I2", 1, 0, "en paralelo", "Claude"],
    ],
  );
  assert.deepEqual(
    linaje(ses, todos).map((n) => [n.nombre, n.nivel, n.nota]),
    [
      ["P", 0, "este"],
      ["Codificador I2", 1, "subagente"],
      ["Segunda opinión E2", 1, "e2-w6b-sol"],
    ],
  );
});

test("linaje: la sesión fuera de la lista se nombra por su id, y Codex sin sesión lo dice", () => {
  const cx = agente({ id: "codex~1", padre: "abcdef1234", proveedor: "Codex", tipo: "codex", rol: "Auditoría", tarea: "e1-x" });
  assert.deepEqual(
    linaje(cx, [cx]).map((n) => [n.id, n.nombre, n.nota]),
    [
      ["abcdef1234", "Sesión abcdef12", "fuera de la lista"],
      ["codex~1", "Auditoría", "este"],
    ],
  );
  const suelta = { ...cx, padre: undefined };
  assert.deepEqual(
    linaje(suelta, [suelta]).map((n) => [n.nombre, n.nota]),
    [["Auditoría", "sin sesión enlazada"]],
  );
});

test("archivos: conteos, orden y frontera", () => {
  const p = (tipo: string, hilo: string) => ({ tipo, hilo }) as Paso;
  const filas = archivosTocados([p("Lee", "src/a.rs"), p("Lee", "src/b.rs"), p("Edita", "src/b.rs"), p("Busca", "src/a.rs"), p("Ejecuta", "")], ["b.rs"]);
  assert.deepEqual(
    filas.map((f) => [f.nombre, f.lee, f.busca, f.edita, f.nombrado]),
    [
      ["b.rs", 1, 0, 1, true],
      ["a.rs", 1, 1, 0, false],
    ],
  );
  assert.equal(archivosTocados([p("Lee", "x.rs")])[0].nombrado, null, "sin nombrados no hay frontera");
});

test("hitos sobre el tiempo", () => {
  const inicio = new Date(2026, 9, 31, 21, 32).toISOString();
  assert.equal(posicionHito("21:32", inicio, 30), 0);
  assert.equal(posicionHito("21:47", inicio, 30), 0.5);
  assert.equal(posicionHito("23:00", inicio, 30), 1, "pasado del presupuesto: al final");
  assert.equal(posicionHito("nada", inicio, 30), null);
});
