// Takes the README screenshots of Atalaya from a running demo, driving Microsoft Edge (or Chrome) without a window
// through its DevTools protocol. Node.js 22+ (built-in fetch and WebSocket), nothing to install.
//
// Start the demo first, with a neutral folder (it shows up in the panel's paths) and on a free port:
//     .\atalaya.exe -demo -demo-carpeta "$env:PUBLIC\atalaya-demo" -escucha 127.0.0.1:7421
// Then:
//     node scripts/capturas.mjs [base URL, default http://127.0.0.1:7421]
// One PNG per section goes to docs/img/ (light theme; the agents view also in dark). The panel keeps a live
// connection open, so a page never goes "idle": each shot waits a fixed time after loading instead.

import { spawn } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";

const base = process.argv[2] ?? "http://127.0.0.1:7421";
const ancho = 1440;
const alto = 900;
const espera = 6000; // ms after navigating: the panel fetches its data, then renders it
const puerto = 9333;
const raiz = join(dirname(fileURLToPath(import.meta.url)), "..");
const destino = join(raiz, "docs", "img");

const tomas = [
  { seccion: "", archivo: "proyectos.png" },
  { seccion: "agentes", archivo: "agentes.png" },
  { seccion: "agentes", archivo: "agentes-oscuro.png", oscuro: true },
  { seccion: "decisiones", archivo: "decisiones.png" },
  { seccion: "salud", archivo: "salud.png" },
  { seccion: "linea", archivo: "linea.png" },
  { seccion: "estadisticas", archivo: "estadisticas.png" },
];

const candidatos = [
  process.env.NAVEGADOR,
  "C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe",
  "C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe",
  "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
  "/usr/bin/microsoft-edge",
  "/usr/bin/google-chrome",
  "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
].filter(Boolean);
const navegador = candidatos.find((c) => existsSync(c));
if (!navegador) throw new Error("No Edge or Chrome found: set NAVEGADOR to its executable.");

const dormir = (ms) => new Promise((r) => setTimeout(r, ms));

try {
  await fetch(`${base}/api/salud`);
} catch {
  throw new Error(`No Atalaya at ${base}: start it with -demo first.`);
}

mkdirSync(destino, { recursive: true });
// A profile of its own, thrown away at the end: the panel's preferences (localStorage) never come from a real browser.
const perfil = mkdtempSync(join(tmpdir(), "atalaya-capturas-"));
const proceso = spawn(navegador, ["--headless=new", "--disable-gpu", "--hide-scrollbars", `--remote-debugging-port=${puerto}`, `--user-data-dir=${perfil}`, "about:blank"], { stdio: "ignore" });

try {
  // The DevTools endpoint takes a moment to come up.
  let pestana;
  for (let i = 0; i < 40 && !pestana; i++) {
    try {
      pestana = await (await fetch(`http://127.0.0.1:${puerto}/json/new?about:blank`, { method: "PUT" })).json();
    } catch {
      await dormir(250);
    }
  }
  if (!pestana) throw new Error("The browser's DevTools endpoint did not answer.");

  const ws = new WebSocket(pestana.webSocketDebuggerUrl);
  await new Promise((ok, mal) => {
    ws.onopen = ok;
    ws.onerror = mal;
  });
  let id = 0;
  const pendientes = new Map();
  ws.onmessage = (m) => {
    const r = JSON.parse(m.data);
    if (r.id && pendientes.has(r.id)) {
      pendientes.get(r.id)(r);
      pendientes.delete(r.id);
    }
  };
  const enviar = (method, params = {}) =>
    new Promise((ok, mal) => {
      const n = ++id;
      pendientes.set(n, (r) => (r.error ? mal(new Error(`${method}: ${r.error.message}`)) : ok(r.result)));
      ws.send(JSON.stringify({ id: n, method, params }));
    });

  await enviar("Emulation.setDeviceMetricsOverride", { width: ancho, height: alto, deviceScaleFactor: 1, mobile: false });
  // The alerts are marked as seen (the panel keeps that in localStorage): otherwise their band covers the top of every
  // shot. They still show on each agent's row and in the health view.
  const ids = [];
  for (const p of await (await fetch(`${base}/api/portafolio`)).json()) {
    const r = await (await fetch(`${base}/api/alertas?proyecto=${encodeURIComponent(p.nombre)}`)).json();
    ids.push(...r.alertas.map((a) => a.id));
  }
  await enviar("Page.navigate", { url: `${base}/` });
  await dormir(1500);
  await enviar("Runtime.evaluate", { expression: `localStorage.setItem("atalaya:alertas-vistas", ${JSON.stringify(JSON.stringify(ids))})` });
  for (const t of tomas) {
    await enviar("Emulation.setEmulatedMedia", { features: [{ name: "prefers-color-scheme", value: t.oscuro ? "dark" : "light" }] });
    // about:blank first: a change of #anchor alone would not reload the page.
    await enviar("Page.navigate", { url: "about:blank" });
    await enviar("Page.navigate", { url: `${base}/${t.seccion ? "#" + t.seccion : ""}` });
    await dormir(espera);
    const { data } = await enviar("Page.captureScreenshot", { format: "png" });
    writeFileSync(join(destino, t.archivo), Buffer.from(data, "base64"));
    console.log(`== ${t.archivo} (${Math.round((data.length * 3) / 4 / 1024)} KB)`);
  }
  ws.close();
} finally {
  proceso.kill();
  await dormir(500);
  rmSync(perfil, { recursive: true, force: true, maxRetries: 5, retryDelay: 300 });
}
