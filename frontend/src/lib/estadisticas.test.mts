import assert from "node:assert/strict";
import { test } from "node:test";
import { ancho, type ClaseEstadistica, celdaTexto, cifra, escala, filasTabla, lectura, miles } from "./estadisticas.ts";

const clase = (c: Partial<ClaseEstadistica>): ClaseEstadistica => ({
  clase: "I2",
  nombre: "Implementación con decisiones",
  proveedor: "Claude",
  n: 5,
  minutos: { mediana: 21.9, p90: 38.5 },
  salida: { mediana: 41_100, p90: 66_300 },
  nuevos: { mediana: 120_000, p90: 300_000 },
  presupuesto: 30,
  presupuesto_fuente: "las asignaciones de la clase",
  matriz: "sonnet/high",
  matriz_fuente: "AGENTES.md:73",
  en_matriz: 5,
  ...c,
});

test("cifras con coma y decimales fijos", () => {
  assert.equal(cifra("minutos", 21.94), "21,9 min");
  assert.equal(cifra("minutos", 4), "4,0 min");
  assert.equal(miles(41_100), "41,1 k");
  assert.equal(miles(1_187_389), "1,19 M");
  assert.equal(miles(512), "512");
});

test("la escala deja aire y cuenta el presupuesto solo en minutos", () => {
  const cs = [clase({}), clase({ clase: "E1", minutos: { mediana: 1, p90: 2 }, presupuesto: 45 })];
  assert.equal(escala(cs, "minutos"), 45 * 1.08);
  assert.equal(escala(cs, "salida"), 66_300 * 1.08);
  assert.equal(escala([], "minutos"), 1);
  assert.equal(ancho(15, 30), "50.0%");
  assert.equal(ancho(60, 30), "100.0%");
});

test("la lectura avisa cuando el p90 pasa del presupuesto o sale de la matriz", () => {
  const pasa = lectura(clase({}), "minutos");
  assert.equal(pasa.tono, "aviso");
  assert.match(pasa.texto, /El p90 pasa del presupuesto de 30 min/);
  assert.match(pasa.texto, /Todos con la matriz \(sonnet\/high\)/);

  const bien = lectura(clase({ minutos: { mediana: 10, p90: 20 } }), "minutos");
  assert.equal(bien.tono, "ok");

  const fuera = lectura(clase({ minutos: { mediana: 10, p90: 20 }, en_matriz: 3 }), "minutos");
  assert.equal(fuera.tono, "aviso");
  assert.match(fuera.texto, /2 de 5 fuera de la matriz/);

  // Con menos de 5, §12 no concluye: no se pinta como aviso.
  const poca = lectura(clase({ n: 2, en_matriz: 2 }), "minutos");
  assert.equal(poca.tono, "suave");
  assert.match(poca.texto, /Con 2 tareas, §12 todavía no saca conclusiones/);

  const sinMatriz = lectura(clase({ matriz: undefined, en_matriz: 0 }), "salida");
  assert.match(sinMatriz.texto, /La matriz no fija esta clase/);
});

test("la tabla clase × modelo", () => {
  assert.equal(celdaTexto({ clase: "I2", modelo: "sonnet", n: 5, minutos: 21.94, en_matriz: true }), "21,9 (5)");
  assert.equal(celdaTexto(undefined), "—");
  assert.deepEqual(filasTabla([clase({}), clase({ clase: "E2" }), clase({ clase: "E2", proveedor: "Codex" })]), ["I2", "E2"]);
});
