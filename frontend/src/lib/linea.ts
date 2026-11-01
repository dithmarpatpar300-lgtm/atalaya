// La línea de tiempo global: la geometría pura, aparte del componente, para
// probarla sin navegador. Todo va en milisegundos y en porcentajes del tramo visible.

import { duracion, hora } from "./formato.ts";
import { idioma, t } from "./i18n.ts";

export interface ChoqueLinea {
  hora: string;
  otro: string;
  /** El id del otro agente, para abrirlo desde la marca del cruce. */
  otro_id?: string;
  ruta: string;
}

export interface FilaLinea {
  id: string;
  nombre: string;
  proveedor: "Claude" | "Codex";
  tipo: "sesion" | "subagente" | "codex";
  padre?: string;
  clase: string;
  estado: string;
  inicio: string;
  fin: string;
  vivo: boolean;
  diario?: boolean;
  excede: boolean;
  choques: ChoqueLinea[];
}

export interface PuntoCuota {
  t: number; // segundos Unix
  u: number; // % usado de la ventana de 5 h
}

export interface RespuestaLinea {
  proyecto: string;
  dia: string;
  dias: string[];
  desde: string;
  hasta: string;
  ahora: string;
  tope_min: number;
  fuente?: string;
  filas: FilaLinea[];
  cuota: Record<string, PuntoCuota[] | undefined>;
}

export interface FilaOrdenada extends FilaLinea {
  nivel: 0 | 1;
  a: number;
  b: number;
}

const MIN = 60_000;

/** Cada sesión seguida de lo que lanzó; lo que no tiene a su sesión en el día va al final, por hora. */
export function ordenar(filas: FilaLinea[], conSesiones: boolean): FilaOrdenada[] {
  const conMs = filas.map((f) => ({ ...f, a: Date.parse(f.inicio), b: Date.parse(f.fin), nivel: 0 as 0 | 1 }));
  const porHora = (x: { a: number }, y: { a: number }) => x.a - y.a;
  const sesiones = conMs.filter((f) => f.tipo === "sesion").sort(porHora);
  const ids = new Set(sesiones.map((s) => s.id));
  const out: FilaOrdenada[] = [];
  for (const s of sesiones) {
    if (conSesiones) out.push(s);
    for (const h of conMs.filter((f) => f.tipo !== "sesion" && f.padre === s.id).sort(porHora)) out.push({ ...h, nivel: conSesiones ? 1 : 0 });
  }
  for (const h of conMs.filter((f) => f.tipo !== "sesion" && !(f.padre && ids.has(f.padre))).sort(porHora)) out.push(h);
  return out;
}

/**
 * El tramo visible: de la primera hora a la última, redondeado a media hora y dentro del día. Lo marcan los agentes:
 * una sesión abierta desde ayer no estira el reloj hasta las 00:00; su barra se corta en el borde. Lo que sigue vivo
 * sí lo alarga hasta ahora.
 */
export function tramo(filas: { a: number; b: number; tipo?: string; vivo?: boolean }[], desde: number, hasta: number): [number, number] {
  const agentes = filas.filter((f) => f.tipo !== "sesion");
  const base = agentes.length ? agentes : filas;
  if (base.length === 0) return [desde + 8 * 60 * MIN, desde + 18 * 60 * MIN];
  const media = 30 * MIN;
  let ini = Math.min(...base.map((f) => f.a));
  let fin = Math.max(...base.map((f) => f.b), ...filas.filter((f) => f.vivo).map((f) => f.b));
  ini = Math.max(desde, Math.floor(ini / media) * media);
  fin = Math.min(hasta, Math.ceil(fin / media) * media);
  if (fin - ini < 2 * media) fin = Math.min(hasta, ini + 2 * media);
  return [ini, fin];
}

/** Las marcas del reloj: el paso más corto que deja a lo sumo diez. */
export function marcas(ini: number, fin: number): number[] {
  const paso = [15, 30, 60, 120, 180, 240].map((m) => m * MIN).find((p) => (fin - ini) / p <= 10) ?? 360 * MIN;
  const out: number[] = [];
  for (let t = Math.ceil(ini / paso) * paso; t <= fin; t += paso) out.push(t);
  return out;
}

export const pct = (t: number, ini: number, fin: number) => Math.min(100, Math.max(0, ((t - ini) / (fin - ini)) * 100));

/** Cuántos agentes (sin las sesiones) trabajaban a la vez, en tramos de `paso`. */
export function paralelo(filas: FilaOrdenada[], ini: number, fin: number, paso = 5 * MIN): { t: number; n: number }[] {
  const ags = filas.filter((f) => f.tipo !== "sesion");
  const out: { t: number; n: number }[] = [];
  for (let t = ini; t < fin; t += paso) out.push({ t, n: ags.filter((f) => f.a < t + paso && f.b > t).length });
  return out;
}

/** Los agentes (no las sesiones) que trabajaron a la vez que `sel`. */
export function enParaleloCon(sel: FilaOrdenada, filas: FilaOrdenada[]): string[] {
  return filas.filter((f) => f.id !== sel.id && f.tipo !== "sesion" && f.a < sel.b && f.b > sel.a).map((f) => f.nombre);
}

/** La curva de la cuota en un viewBox de 1000 × alto: escalones, porque cada lectura vale hasta la siguiente. */
export function curva(puntos: PuntoCuota[] | undefined, ini: number, fin: number, alto: number): string {
  const dentro = (puntos ?? []).filter((p) => p.t * 1000 >= ini && p.t * 1000 <= fin);
  if (dentro.length === 0) return "";
  const y = (u: number) => (alto - (Math.min(100, Math.max(0, u)) / 100) * alto).toFixed(1);
  const x = (t: number) => ((pct(t * 1000, ini, fin) / 100) * 1000).toFixed(1);
  const out: string[] = [];
  dentro.forEach((p, i) => {
    if (i > 0) out.push(`${x(p.t)},${y(dentro[i - 1].u)}`);
    out.push(`${x(p.t)},${y(p.u)}`);
  });
  return out.join(" ");
}

/** El uso de la ventana a una hora: la última lectura antes de ella. */
export function cuotaEn(puntos: PuntoCuota[] | undefined, t: number): number | null {
  let v: number | null = null;
  for (const p of puntos ?? []) {
    if (p.t * 1000 > t) break;
    v = p.u;
  }
  return v;
}

// La hora y la duración salen de formato.ts, como en el resto del panel; aquí van en milisegundos.
export const hhmm = (t: number): string => hora(new Date(t));
export const minutos = (ms: number): string => duracion(Math.round(ms / MIN) * 60);

const MESES = ["ene", "feb", "mar", "abr", "may", "jun", "jul", "ago", "sep", "oct", "nov", "dic"];
const MESES_EN = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];

/** «Hoy», «Ayer» o «29 oct», con `hoy` en AAAA-MM-DD. */
export function nombreDia(dia: string, hoy: string): string {
  if (dia === hoy) return t("Hoy");
  const [a, m, d] = dia.split("-").map(Number);
  const ayer = new Date(Date.parse(`${hoy}T12:00:00`) - 24 * 60 * MIN);
  if (ayer.getFullYear() === a && ayer.getMonth() + 1 === m && ayer.getDate() === d) return t("Ayer");
  return idioma() === "en" ? `${MESES_EN[m - 1]} ${d}` : `${d} ${MESES[m - 1]}`;
}
