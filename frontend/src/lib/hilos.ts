import { decimal, t } from "./i18n.ts";
// Las filas de la línea de tiempo con sus etapas, los colores de los hilos y las cifras de cada paso. Sin React ni DOM:
// se prueba con `node --test` (hilos.test.mts). Los lotes y la pista de los hilos están en lotes.ts.
import type { Paso } from "./api.ts";

/** La etapa que marca una fila: la primera de cada etapa, y la primera de la página si su etapa empezó antes. */
export interface MarcaEtapa {
  etapa: string;
  /** Número de etapa dentro del turno (1, 2…); null si no se sabe (una página vieja, sin los pasos de antes). */
  n: number | null;
  /** La etapa empezó en una página anterior. */
  sigue: boolean;
}

export interface Fila {
  tipo: "turno" | "paso";
  paso: Paso;
  marca: MarcaEtapa | null;
}

/**
 * Arma las filas: cada pedido es la fila de su turno y cada paso la suya; la etapa va como marca en su primera fila.
 * `previos` son los pasos anteriores al primero de `pasos` (para numerar las etapas y saber si la de la primera fila ya
 * venía de antes); null si no se tienen.
 */
export function armarFilas(pasos: readonly Paso[], previos: readonly Paso[] | null = []): Fila[] {
  const filas: Fila[] = [];
  let etapaPrevia = "";
  let turnoPrevio = -1;
  let nEtapa = 0;
  // Sin los pasos de antes, el número de etapa se sabe recién desde el próximo turno.
  let conocido = previos !== null;
  const avanzar = (p: Paso): MarcaEtapa | null => {
    if (p.turno !== turnoPrevio) {
      if (turnoPrevio !== -1 || p.tipo === "Pide") conocido = true;
      turnoPrevio = p.turno;
      etapaPrevia = "";
      nEtapa = 0;
    }
    if (p.tipo === "Pide" || !p.etapa || p.etapa === etapaPrevia) return null;
    etapaPrevia = p.etapa;
    nEtapa++;
    return { etapa: p.etapa, n: conocido ? nEtapa : null, sigue: false };
  };
  for (const p of previos ?? []) avanzar(p);
  for (const p of pasos) {
    let marca = avanzar(p);
    // La primera fila de la página dice su etapa aunque haya empezado antes (mismo turno y misma etapa que el último
    // de `previos`). Sin `previos` no se sabe si empezó antes: se marca como nueva, sin número.
    if (filas.length === 0 && !marca && p.tipo !== "Pide" && p.etapa) marca = { etapa: p.etapa, n: conocido ? nEtapa : null, sigue: true };
    filas.push({ tipo: p.tipo === "Pide" ? "turno" : "paso", paso: p, marca });
  }
  return filas;
}

/** El nombre de la marca, el mismo a la vista, en su `title` y en la etiqueta de la fila: «Etapa 2 · sigue · Corregir». */
export function nombreEtapa(m: MarcaEtapa): string {
  return [m.n ? t("Etapa {n}", { n: m.n }) : t("Etapa"), m.sigue ? t("sigue") : "", m.etapa].filter(Boolean).join(" · ");
}

const esHerramienta = (p: Paso) => p.tipo !== "Pide" && p.herramienta !== "Respuesta";

/** El color de cada hilo (archivo), por orden de aparición: los cinco tonos de la identidad, en ciclo. */
export function coloresDeHilo(pasos: Paso[]): Map<string, string> {
  const m = new Map<string, string>();
  for (const p of pasos) if (p.hilo && !m.has(p.hilo)) m.set(p.hilo, `var(--hilo-${(m.size % 5) + 1})`);
  return m;
}

/** Coma decimal y un decimal fijo: «0,4 k», «181,4 k», «4,92 M» (la regla: lo que cambia en vivo no mueve el diseño; las cifras no bailan). */
export function tokensCortos(n: number): string {
  if (n < 1000) return `${n}`;
  if (n < 1_000_000) return `${decimal(n / 1000)} k`;
  return `${decimal(n / 1_000_000, 2)} M`;
}

/** El resumen corto del resultado de un paso que arma el servidor («12 líneas», «error», «Exit code 1»), en el idioma del panel. */
export function metaPaso(meta: string): string {
  if (meta === "error") return t("error");
  if (meta === "1 línea") return t("1 línea");
  let m = /^(\d+) líneas$/.exec(meta);
  if (m) return t("{n} líneas", { n: m[1] });
  m = /^(\d+) caracteres$/.exec(meta);
  if (m) return t("{n} caracteres", { n: m[1] });
  m = /^salida 0 · (\d+) líneas$/.exec(meta);
  if (m) return t("salida 0 · {n} líneas", { n: m[1] });
  return meta;
}

/** La parte de la salida de su mensaje que le toca a un paso: las llamadas de un lote la reparten. */
export function salidaDelPaso(p: Paso): number {
  return esHerramienta(p) && p.paralelo > 1 ? Math.round(p.tok_salida / p.paralelo) : p.tok_salida;
}

/** La ventana de contexto del modelo, inferida: si alguna petición pasó de 200 k, es de 1 M. */
export function ventana(pasos: Paso[]): number {
  return pasos.some((p) => p.tok_contexto > 200_000) ? 1_000_000 : 200_000;
}
