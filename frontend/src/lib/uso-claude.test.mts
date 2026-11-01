import assert from "node:assert/strict";
import { test } from "node:test";
import { fijarIdioma } from "./i18n.ts";
import {
  camino,
  edad,
  escala,
  etiquetaCorta,
  etiquetaVentana,
  filasTabla,
  frescura,
  marcasTiempo,
  masCercano,
  proximaConsulta,
  separarEtiquetas,
  seriesDe,
  trazos,
  type UsoClaude,
  usaUsoClaude,
  vistaVentana,
} from "./uso-claude.ts";

// El ejemplo del contrato (docs/trazabilidad/F6/temporales/2026-11-01-uso-claude-contrato.md).
const ejemplo = (): UsoClaude => ({
  estado: "ok",
  detalle: "",
  tomada_en: 1793545000,
  edad_s: 120,
  proxima_en: 1793545600,
  limitado_hasta: null,
  ventanas: [
    { clave: "session", etiqueta: "5 horas", porcentaje: 13, reinicia_en: 1793560200, duracion_min: 300, severidad: "normal", activa: false, vencida: false },
    { clave: "weekly_all", etiqueta: "Semana · todos los modelos", porcentaje: 49, reinicia_en: 1793703600, duracion_min: 10080, severidad: "normal", activa: true, vencida: false },
    { clave: "weekly_scoped:Fable", etiqueta: "Semana · Fable", porcentaje: 0, reinicia_en: 1793703600, duracion_min: 10080, severidad: "normal", activa: false, vencida: false },
  ],
  extra_activo: false,
  historial: [{ t: 1793545000, v: { session: 13, weekly_all: 49, "weekly_scoped:Fable": 0 } }],
});

test("la fuente de /usage vale salvo sin Claude Code", () => {
  assert.equal(usaUsoClaude(null), false);
  assert.equal(usaUsoClaude({ ...ejemplo(), estado: "sin-claude" }), false);
  assert.equal(usaUsoClaude({ ...ejemplo(), estado: "viejo" }), true);
  assert.equal(usaUsoClaude({ ...ejemplo(), estado: "error" }), true);
});

test("los nombres de las ventanas, en los dos idiomas", () => {
  fijarIdioma("es");
  assert.equal(etiquetaVentana("weekly_scoped:Fable"), "Semana · Fable");
  assert.equal(etiquetaCorta("weekly_scoped:Fable"), "Fable");
  assert.equal(etiquetaCorta("session"), "5 h");
  fijarIdioma("en");
  assert.equal(etiquetaVentana("weekly_all"), "Week · all models");
  assert.equal(etiquetaVentana("session"), "5 hours");
  assert.equal(etiquetaVentana("otra", "Algo nuevo"), "Algo nuevo");
  fijarIdioma("es");
});

test("la edad corre con el reloj desde tomada_en; sin reloj, la del servidor", () => {
  const u = ejemplo();
  assert.equal(edad(u, 0), 120);
  assert.equal(edad(u, u.tomada_en + 300), 300);
  assert.equal(edad({ ...u, tomada_en: 0 }, 1793545500), 120);
});

test("la frescura sale del estado del servidor, no de una cuenta propia", () => {
  fijarIdioma("es");
  const u = ejemplo();
  assert.deepEqual(
    [frescura(u, u.tomada_en + 120).texto, frescura(u, u.tomada_en + 120).tono],
    ["hace 2 min · /usage", "normal"],
  );
  // Con 34 min de edad pero estado «ok», sigue siendo «ok»: el servidor decide cuándo es vieja.
  assert.equal(frescura(u, u.tomada_en + 34 * 60).tono, "normal");
  const vieja = frescura({ ...u, estado: "viejo" }, u.tomada_en + 34 * 60);
  assert.deepEqual([vieja.texto, vieja.tono], ["vieja: hace 34 min", "aviso"]);
  const hasta = new Date(2026, 10, 1, 15, 10).getTime() / 1000;
  assert.equal(frescura({ ...u, estado: "limitado", limitado_hasta: hasta }, u.tomada_en).texto, "limitada hasta las 15:10");
  assert.equal(frescura({ ...u, estado: "limitado" }, u.tomada_en).texto, "limitada por /usage");
  assert.equal(frescura({ ...u, estado: "error", detalle: "salió con código 1" }, u.tomada_en).tono, "error");
  assert.match(frescura({ ...u, estado: "error", detalle: "salió con código 1" }, u.tomada_en).titulo, /salió con código 1/);
  assert.equal(frescura({ ...u, estado: "sin-suscripcion" }, u.tomada_en).texto, "sin suscripción: /usage no da la cuota");
});

test("un 429 dice cuándo se puede volver a leer, sin ser error, y se olvida al pasar la hora", () => {
  fijarIdioma("es");
  const u = ejemplo();
  const espera = new Date(2026, 10, 1, 10, 32).getTime() / 1000;
  const f = frescura(u, espera - 60, espera);
  assert.deepEqual([f.texto, f.tono], ["se puede volver a leer a las 10:32", "tenue"]);
  assert.equal(frescura(u, espera + 1, espera).texto.startsWith("hace"), true);
  assert.equal(frescura(u, espera - 60, espera, true).texto, "sin respuesta de Atalaya");
});

test("una ventana vencida nunca muestra el % viejo", () => {
  fijarIdioma("es");
  const [ses] = ejemplo().ventanas;
  const antes = vistaVentana(ses, ses.reinicia_en - 2 * 3600 - 600);
  assert.deepEqual([antes.pct, antes.cuenta], [13, "reinicia en 2 h 10 min"]);
  assert.deepEqual([vistaVentana({ ...ses, vencida: true }, ses.reinicia_en - 60).pct, vistaVentana({ ...ses, vencida: true }, 0).cuenta], [null, "se reinició · falta leerla"]);
  // Mientras llega la lectura que lo marca, el reinicio que ya pasó en el reloj también cuenta.
  assert.equal(vistaVentana(ses, ses.reinicia_en + 1).pct, null);
  assert.equal(vistaVentana({ ...ses, porcentaje: 140 }, ses.reinicia_en - 10).pct, 100);
});

test("las series tienen orden y color fijos; desde el segundo modelo, discontinuas", () => {
  const u = ejemplo();
  u.historial.push({ t: 1793545600, v: { "weekly_scoped:Opus": 3 } });
  const s = seriesDe(u);
  assert.deepEqual(
    s.map((x) => [x.clave, x.color, x.discontinua]),
    [
      ["session", 0, false],
      ["weekly_all", 1, false],
      ["weekly_scoped:Fable", 2, false],
      ["weekly_scoped:Opus", 2, true],
    ],
  );
  // Sin la ventana de 5 horas, la semana conserva su color.
  const sinSesion = seriesDe({ ...ejemplo(), ventanas: ejemplo().ventanas.slice(1), historial: [] });
  assert.equal(sinSesion[0].color, 1);
});

test("las dos fuentes no hacen saltar la línea; el reinicio baja a 0 sin hueco (la serie real del 01-11)", () => {
  // La semanal alterna 57/58 entre /usage y el mod; la de 5 h se reinicia a las 14:10 (con su hora del servidor).
  const h = [
    { t: 100, v: { session: 80, weekly_all: 57 }, r: { session: 600 } },
    { t: 200, v: { session: 81, weekly_all: 58 }, r: { session: 600 } },
    { t: 300, v: { session: 81, weekly_all: 57 }, r: { session: 600 } },
    { t: 400, v: { session: 81, weekly_all: 58 }, r: { session: 601 } }, // el mod da otra hora: no es reinicio
    { t: 700, v: { session: 0, weekly_all: 58 } },
    { t: 800, v: { session: 1, weekly_all: 59 } },
  ];
  const { tramos, reinicios } = trazos(h, ["session", "weekly_all"], 0, 10_000);
  assert.deepEqual(tramos.get("weekly_all"), [[[100, 57], [200, 58], [300, 58], [400, 58], [700, 58], [800, 59]]]);
  assert.deepEqual(tramos.get("session"), [[[100, 80], [200, 81], [300, 81], [400, 81], [601, 81], [601, 0], [700, 0], [800, 1]]]);
  assert.deepEqual(reinicios.map((r) => [r.t, r.claves]), [[600, ["session"]]]);
});

test("los tramos se cortan en los huecos y en los reinicios, que se marcan", () => {
  const h = [
    { t: 0, v: { session: 10, weekly_all: 40 } },
    { t: 600, v: { session: 20, weekly_all: 41 } },
    { t: 1200, v: { session: 30, weekly_all: 42 }, r: { session: 1500 } },
    { t: 1800, v: { session: 2, weekly_all: 43 } }, // reinicio de 5 h, con su hora del servidor
    { t: 1800 + 3600, v: { session: 5, weekly_all: 44 } }, // hueco de 1 h
    { t: 1800 + 4200, v: { session: 7, weekly_all: 1 } }, // la semana baja sin hora: reinicio inferido
  ];
  const { tramos, reinicios } = trazos(h, ["session", "weekly_all"], 0, 10_000);
  // La de 5 h: sube, baja a 0 a las 1500 y sigue en el mismo tramo; el hueco de 1 h corta. La semanal: el hueco corta,
  // y la caída de 44 a 1 es un reinicio que baja a 0 sin hueco.
  assert.deepEqual(tramos.get("session")!.map((tr) => tr.length), [6, 2]);
  assert.deepEqual(tramos.get("weekly_all")!.map((tr) => tr.length), [4, 4]);
  assert.deepEqual(
    reinicios.map((r) => [r.t, r.claves, r.exacto]),
    [
      [1500, ["session"], true],
      [6000, ["weekly_all"], false],
    ],
  );
  assert.equal(reinicios[1].desde, 5400);
  // Fuera del periodo no cuenta.
  assert.equal(trazos(h, ["session"], 5000, 10_000).tramos.get("session")!.flat().length, 2);
});

test("la escala va de 0 a 100 % y de ini a fin", () => {
  const e = escala(100, 200, 500, 100);
  assert.equal(e.x(150), 250);
  assert.equal(e.y(0), 100);
  assert.equal(e.y(100), 0);
  assert.equal(e.y(130), 0);
  assert.equal(e.t(250), 150);
  assert.equal(e.t(-20), 100);
  assert.equal(camino([[[100, 0], [200, 50]], [[150, 100]]], e.x, e.y), "M0.0,100.0L500.0,50.0M250.0,0.0");
});

test("las marcas del tiempo: cada 3 h en un día, cada día en una semana", () => {
  fijarIdioma("es");
  const fin = new Date(2026, 10, 1, 10, 14).getTime() / 1000;
  const dia = marcasTiempo(fin - 24 * 3600, fin, 24);
  assert.equal(dia.length, 8);
  assert.ok(dia.every((m) => /^\d\d:00$/.test(m.rotulo) && Number(m.rotulo.slice(0, 2)) % 3 === 0));
  const semana = marcasTiempo(fin - 168 * 3600, fin, 168);
  assert.deepEqual(semana.map((m) => m.rotulo), ["lun 26", "mar 27", "mié 28", "jue 29", "vie 30", "sáb 31", "dom 1"]);
  assert.equal(marcasTiempo(fin - 720 * 3600, fin, 720).length, 6);
});

test("el punto más cercano al cursor", () => {
  const pts = [{ t: 0 }, { t: 10 }, { t: 30 }];
  assert.deepEqual([masCercano(pts, -5), masCercano(pts, 4), masCercano(pts, 6), masCercano(pts, 21), masCercano(pts, 99), masCercano([], 1)], [0, 0, 1, 2, 2, -1]);
});

test("las etiquetas del final no se pisan y no salen del área", () => {
  assert.deepEqual(separarEtiquetas([50, 50, 52], 13, 100), [50, 63, 76]);
  assert.deepEqual(separarEtiquetas([100, 100], 13, 100), [87, 100]);
  assert.deepEqual(separarEtiquetas([10], 13, 100), [10]);
});

test("la tabla: la última lectura de cada hora, de la más nueva a la más vieja", () => {
  const h = [0, 600, 3000, 3700, 7300].map((tt) => ({ t: tt, v: { session: tt / 100 } }));
  assert.deepEqual(filasTabla(h, 0, 10_000, 24).map((p) => p.t), [7300, 3700, 3000]);
  assert.deepEqual(filasTabla(h, 3600, 10_000, 24).map((p) => p.t), [7300, 3700]);
});

test("se vuelve a pedir al minuto o justo después del próximo reinicio", () => {
  const u = ejemplo();
  assert.equal(proximaConsulta(u, u.tomada_en), 60);
  assert.equal(proximaConsulta(u, u.ventanas[0].reinicia_en - 10), 12);
  assert.equal(proximaConsulta({ ...u, limitado_hasta: u.tomada_en + 5 }, u.tomada_en), 7);
});
