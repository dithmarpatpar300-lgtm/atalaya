import { decimal, t } from "./i18n.ts";
// La geometría de los hilos de la línea de tiempo: filas de alto fijo y un solo SVG
// que dibuja el riel, los desvíos, las vueltas, el mismo archivo, los paralelos y las delegaciones. Sin React ni DOM:
// se prueba con `node --test` (hilos.test.mts) y el componente solo pinta lo que devuelve.
import type { Paso } from "./api.ts";

/** Toda fila mide 40 px: una página de N pasos mide N × 40, sin importar sus etapas (la etapa es una marca dentro de
 * la fila, no una fila aparte de 30 px que hacía variar el alto de una página a otra). */
export const ALTO_FILA = 40;
/** Columna del primer nivel y separación entre niveles, en px dentro de la canaleta. */
export const X0 = 34;
export const NIVEL = 22;
export const ANCHO_CANALETA = 120;
export const x = (nivel: number) => X0 + nivel * NIVEL;

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
  y: number;
  alto: number;
  cy: number;
}

/**
 * Arma las filas, todas de 40 px: cada pedido es la fila de su turno y cada paso la suya; la etapa va como marca en su
 * primera fila. `previos` son los pasos anteriores al primero de `pasos` (para numerar las etapas y saber si la de la
 * primera fila ya venía de antes); null si no se tienen.
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
    const y = filas.length * ALTO_FILA;
    filas.push({ tipo: p.tipo === "Pide" ? "turno" : "paso", paso: p, marca, y, alto: ALTO_FILA, cy: y + ALTO_FILA / 2 });
  }
  return filas;
}

/** El nombre de la marca, el mismo a la vista, en su `title` y en la etiqueta de la fila: «Etapa 2 · sigue · Corregir». */
export function nombreEtapa(m: MarcaEtapa): string {
  return [m.n ? t("Etapa {n}", { n: m.n }) : t("Etapa"), m.sigue ? t("sigue") : "", m.etapa].filter(Boolean).join(" · ");
}

export interface Trazo {
  d: string;
  /** riel y desvío: gris neutro; hilo: color del archivo, punteado; paralelo: acento; delega: celeste discontinuo. */
  clase: "riel" | "desvio" | "vuelta" | "hilo" | "paralelo" | "delega" | "etapa";
  color?: string;
}

export interface Nodo {
  n: number;
  cx: number;
  cy: number;
}

const esHerramienta = (p: Paso) => p.tipo !== "Pide" && p.herramienta !== "Respuesta";

/** El color de cada hilo (archivo), por orden de aparición: los cinco tonos de la identidad, en ciclo. */
export function coloresDeHilo(pasos: Paso[]): Map<string, string> {
  const m = new Map<string, string>();
  for (const p of pasos) if (p.hilo && !m.has(p.hilo)) m.set(p.hilo, `var(--hilo-${(m.size % 5) + 1})`);
  return m;
}

/** Todo lo que dibuja el SVG. Un enlace a un paso que no está entre las filas (anterior al corte) no se dibuja. */
export function geometria(filas: Fila[], colores: Map<string, string>): { trazos: Trazo[]; nodos: Nodo[]; alto: number } {
  const trazos: Trazo[] = [];
  const nodos: Nodo[] = [];
  const conPaso = filas;
  const alto = filas.length ? filas[filas.length - 1].y + filas[filas.length - 1].alto : 0;
  if (conPaso.length === 0) return { trazos, nodos, alto };
  const porN = new Map(conPaso.map((f) => [f.paso.n, f]));

  // El riel principal, del primer al último nodo.
  trazos.push({ d: `M ${x(0)} ${conPaso[0].cy} V ${conPaso[conPaso.length - 1].cy}`, clase: "riel" });

  // Desvíos: por cada nivel, cada tramo seguido de pasos a ese nivel o más. Un codo baja del nivel anterior, el riel
  // del desvío corre a su lado y otro codo vuelve con flecha al paso que retoma.
  const nivel = (f: Fila) => (f.tipo === "turno" ? 0 : f.paso.profundidad);
  const maxNivel = Math.max(0, ...conPaso.map(nivel));
  for (let k = 1; k <= maxNivel; k++) {
    let i = 0;
    while (i < conPaso.length) {
      if (nivel(conPaso[i]) < k) {
        i++;
        continue;
      }
      const s = i;
      while (i < conPaso.length && nivel(conPaso[i]) >= k) i++;
      const e = i - 1;
      const antes = conPaso[s - 1];
      const despues = conPaso[e + 1];
      const xa = x(k - 1);
      const xb = x(k);
      const ys = conPaso[s].cy;
      const ye = conPaso[e].cy;
      if (antes) trazos.push({ d: `M ${xa} ${antes.cy + 7} V ${ys - 11} Q ${xa} ${ys} ${xa + 11} ${ys} H ${xb - 6}`, clase: "desvio" });
      if (e > s) trazos.push({ d: `M ${xb} ${ys + 6} V ${ye - 6}`, clase: "desvio" });
      if (despues) trazos.push({ d: `M ${xb} ${ye + 6} V ${despues.cy - 11} Q ${xb} ${despues.cy} ${xb - 11} ${despues.cy} H ${xa + 9}`, clase: "vuelta" });
    }
  }

  // Mismo archivo: un arco punteado a la izquierda une el paso que vuelve con el que dejó ese archivo.
  for (const f of conPaso) {
    const q = f.paso.continua >= 0 ? porN.get(f.paso.continua) : undefined;
    if (!q) continue;
    const xq = x(nivel(q)) - 6;
    const xp = x(nivel(f)) - 6;
    const curva = Math.max(6, Math.min(x(0) - 4, 10 + (f.cy - q.cy) / 40));
    const xc = x(0) - curva - 14;
    trazos.push({ d: `M ${xq} ${q.cy} C ${xc} ${q.cy} ${xc} ${f.cy} ${xp} ${f.cy}`, clase: "hilo", color: colores.get(f.paso.hilo) });
  }

  // Paralelo: arcos cortos entre los nodos de las llamadas de un mismo mensaje.
  for (let i = 1; i < conPaso.length; i++) {
    const a = conPaso[i - 1];
    const b = conPaso[i];
    if (a.paso.lote === b.paso.lote && b.paso.paralelo > 1 && esHerramienta(a.paso) && esHerramienta(b.paso)) {
      const xb = x(nivel(b));
      trazos.push({ d: `M ${x(nivel(a)) + 6} ${a.cy} C ${xb + 20} ${a.cy} ${xb + 20} ${b.cy} ${xb + 6} ${b.cy}`, clase: "paralelo" });
    }
  }

  // Delegación: un trazo discontinuo hacia el borde de la canaleta.
  for (const f of conPaso) if (f.paso.subagente) trazos.push({ d: `M ${x(nivel(f)) + 7} ${f.cy} H ${ANCHO_CANALETA}`, clase: "delega" });

  // Donde empieza una etapa, una marca corta cruza el riel en el borde de arriba de su fila (no tapa el nodo).
  for (const f of filas) if (f.marca && !f.marca.sigue) trazos.push({ d: `M ${x(0) - 5} ${f.y + 1} H ${x(0) + 5}`, clase: "etapa" });

  for (const f of conPaso) nodos.push({ n: f.paso.n, cx: x(nivel(f)), cy: f.cy });
  return { trazos, nodos, alto };
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
