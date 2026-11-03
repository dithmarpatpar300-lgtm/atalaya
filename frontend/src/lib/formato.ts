// Cifras, horas y duraciones en el idioma del panel (lib/i18n.ts): coma decimal en español, punto en inglés.

import { decimal, idioma, t } from "./i18n.ts";

const DIAS = ["domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"];
const DAYS = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"];

export const dos = (n: number) => String(n).padStart(2, "0");

export function hora(fecha: Date): string {
  return `${dos(fecha.getHours())}:${dos(fecha.getMinutes())}`;
}

/** «dd/mm» (en inglés, «mm/dd»). */
export const diaYMes = (f: Date) => (idioma() === "en" ? `${dos(f.getMonth() + 1)}/${dos(f.getDate())}` : `${dos(f.getDate())}/${dos(f.getMonth() + 1)}`);

/** «dd/mm» si no es de hoy; vacío si es de hoy (la hora basta). */
export const diaSiNoHoy = (f: Date) => (f.toDateString() === new Date().toDateString() ? "" : diaYMes(f));

/** «16:30» hoy, «30/10 16:30» otro día. */
export function cuando(f: Date): string {
  const d = diaSiNoHoy(f);
  return d ? `${d} ${hora(f)}` : hora(f);
}

/** «a las 02:00» si es hoy o mañana temprano (menos de 24 h), «el martes a las 06:00» si no. */
export function cuandoReinicia(unix: number, ahora: number): string {
  const f = new Date(unix * 1000);
  if (unix - ahora < 20 * 3600) return t("a las {hora}", { hora: hora(f) });
  return idioma() === "en" ? `on ${DAYS[f.getDay()]} at ${hora(f)}` : `el ${DIAS[f.getDay()]} a las ${hora(f)}`;
}

/** «3 h 31 min», «3 d 7 h», «42 s». */
export function duracion(segundos: number): string {
  const s = Math.max(0, Math.round(segundos));
  if (s < 60) return `${s} s`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m} min`;
  const h = Math.floor(m / 60);
  if (h < 24) return m % 60 ? `${h} h ${m % 60} min` : `${h} h`;
  const d = Math.floor(h / 24);
  return h % 24 ? `${d} d ${h % 24} h` : `${d} d`;
}

export function haceCuanto(unixOFecha: number | string, ahora: number): string {
  const segundos = typeof unixOFecha === "number" ? unixOFecha : Date.parse(unixOFecha) / 1000;
  const s = ahora - segundos;
  if (s < 5) return t("ahora");
  return t("hace {d}", { d: duracion(s) });
}

// Una cifra decimal, sin ceros de más («4,1 s», «181,4 k»; «181.4 k» en inglés). No se usa Intl con «es-PE»: ese
// formato pone punto decimal y el panel mezclaba «643.4 k» con «181,4 k».
const numero = (n: number) => {
  const r = Math.round(n * 10) / 10;
  return Number.isInteger(r) ? String(r) : decimal(r, 1);
};

export function porcentaje(n: number): string {
  return `${numero(n)} %`;
}

/** Tokens en k o M, con una cifra decimal: «181,4 k», «4,9 M». */
export function tokens(n: number): string {
  if (n < 1000) return String(n);
  if (n < 1_000_000) return `${numero(n / 1000)} k`;
  return `${numero(n / 1_000_000)} M`;
}

export function ritmo(pph: number): string {
  return `${pph >= 0 ? "+" : ""}${numero(pph)} %/h`;
}

/** Minutos de un presupuesto escrito «30 min»; null si no se entiende. */
export function minutosDe(presupuesto: string): number | null {
  const m = /(\d+)\s*min/.exec(presupuesto);
  return m ? Number(m[1]) : null;
}
