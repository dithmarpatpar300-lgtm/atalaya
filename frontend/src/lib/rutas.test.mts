import assert from "node:assert/strict";
import { test } from "node:test";
import { resolverRuta } from "./rutas.ts";

test("enlaces relativos de un documento", () => {
  assert.equal(resolverRuta("docs/operacion/README.md", "AGENTES.md"), "docs/operacion/AGENTES.md");
  assert.equal(resolverRuta("docs/operacion/README.md", "../adr/0013-x.md#limites"), "docs/adr/0013-x.md");
  assert.equal(resolverRuta("docs/trazabilidad/F5/08.md", "./avance/c.md:12"), "docs/trazabilidad/F5/avance/c.md");
  assert.equal(resolverRuta("AGENTS.md", "docs/operacion/README.md"), "docs/operacion/README.md");
  assert.equal(resolverRuta("docs/a/b.md", "/docs/c.md"), "docs/c.md");
});
