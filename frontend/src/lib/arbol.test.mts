import assert from "node:assert/strict";
import { test } from "node:test";
import type { Agente } from "./api.ts";
import { nombreAgente, ordenarArbol, titulos, trazosArbol } from "./arbol.ts";

const ag = (id: string, ultimo: string, padre?: string) => ({ id, padre, ultimo, proveedor: "Claude", tipo: padre ? "subagente" : "sesion" }) as Agente;
const codex = (id: string, ultimo: string, padre?: string) => ({ id, padre, ultimo, proveedor: "Codex", tipo: "codex" }) as Agente;

test("cada subagente va bajo su sesión, del más reciente al más viejo", () => {
  const lista = [ag("s~b", "2026-10-31T10:05:00Z", "s"), ag("t", "2026-10-31T10:04:00Z"), ag("s", "2026-10-31T10:03:00Z"), ag("s~a", "2026-10-31T10:09:00Z", "s")];
  const filas = ordenarArbol(lista);
  assert.deepEqual(
    filas.map((f) => [f.agente.id, f.nivel, f.padre]),
    [
      ["t", 0, -1],
      ["s", 0, -1],
      ["s~a", 1, 1],
      ["s~b", 1, 1],
    ],
  );
  assert.equal(trazosArbol(filas).length, 2);
});

test("un subagente sin su sesión en la lista va suelto", () => {
  const filas = ordenarArbol([ag("s~a", "2026-10-31T10:00:00Z", "s")]);
  assert.deepEqual([filas[0].nivel, filas[0].huerfano, filas[0].sinSesion], [0, true, false]);
  assert.equal(trazosArbol(filas).length, 0);
});

test("las llamadas a Codex cuelgan de su sesión, junto a los subagentes", () => {
  const lista = [codex("c1", "2026-10-31T10:08:00Z", "s"), ag("s", "2026-10-31T10:03:00Z"), ag("s~a", "2026-10-31T10:05:00Z", "s"), codex("c2", "2026-10-31T09:00:00Z")];
  const filas = ordenarArbol(lista);
  assert.deepEqual(
    filas.map((f) => [f.agente.id, f.nivel, f.padre, f.sinSesion]),
    [
      ["s", 0, -1, false],
      ["c1", 1, 0, false],
      ["s~a", 1, 0, false],
      ["c2", 0, -1, true],
    ],
  );
  const trazos = trazosArbol(filas);
  assert.deepEqual(
    trazos.map((t) => t.codex),
    [true, false],
    "el hilo de Codex se dibuja celeste y discontinuo",
  );
});

test("una llamada a Codex cuya sesión quedó fuera del filtro va suelta, pero no «sin sesión»", () => {
  const [f] = ordenarArbol([codex("c1", "2026-10-31T10:08:00Z", "s")]);
  assert.deepEqual([f.nivel, f.huerfano, f.sinSesion], [0, true, false]);
});

test("títulos: la sesión se nombra por su proyecto, con su nombre propio o su primer pedido debajo", () => {
  const ses = { ...ag("s", "2026-10-31T10:00:00Z"), rol: "Sesión", proyecto: "Tienda Demo", tarea: "Retoma la fase 5", clase: "" } as Agente;
  assert.deepEqual(titulos(ses), { principal: "Tienda Demo", clase: "", alterno: "Retoma la fase 5", completo: "Retoma la fase 5" });
  assert.deepEqual(titulos({ ...ses, titulo: "Retomar con Atalaya MCP" }), {
    principal: "Tienda Demo",
    clase: "",
    alterno: "Retomar con Atalaya MCP",
    completo: "Retomar con Atalaya MCP — Retoma la fase 5",
  });
  assert.equal(titulos({ ...ses, titulo: "tienda-demo" }).alterno, "Retoma la fase 5", "un nombre igual al proyecto no aporta");
  assert.equal(titulos({ ...ses, tarea: "" }).alterno, "Sin encargo");
});

test("títulos: subagentes y Codex siguen con su rol y su clase, y su tarea debajo", () => {
  const sub = { ...ag("s~a", "2026-10-31T10:00:00Z", "s"), rol: "Codificador", clase: "I2", proyecto: "Tienda Demo", tarea: "T1: medir" } as Agente;
  assert.deepEqual([titulos(sub).principal, titulos(sub).alterno, nombreAgente(sub)], ["Codificador", "T1: medir", "Codificador I2"]);
  const cx = { ...codex("c", "2026-10-31T10:00:00Z", "s"), rol: "Segunda opinión", clase: "E2", proyecto: "Tienda Demo", tarea: "e2-t1b-sol" } as Agente;
  assert.deepEqual([nombreAgente(cx), titulos(cx).alterno], ["Segunda opinión E2", "e2-t1b-sol"]);
});

test("una sesión interactiva de Codex va como raíz sin decir que le falta sesión", () => {
  const filas = ordenarArbol([{ ...codex("tui", "2026-10-31T10:00:00Z"), interactiva: true } as Agente, codex("c", "2026-10-31T09:00:00Z")]);
  assert.deepEqual(
    filas.map((f) => [f.agente.id, f.nivel, f.sinSesion]),
    [
      ["tui", 0, false],
      ["c", 0, true],
    ],
  );
});
