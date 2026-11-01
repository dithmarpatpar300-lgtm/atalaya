// La geometría de los hilos de la línea de tiempo: filas de alto fijo y un solo SVG
// que dibuja el riel, los desvíos, las vueltas, el mismo archivo, los paralelos y las delegaciones. Sin React ni DOM:
// se prueba con `node --test` (hilos.test.mts) y el componente solo pinta lo que devuelve.
import type { Paso } from "./api.ts";

export const ALTO = { turno: 40, etapa: 30, paso: 40 } as const;
/** Columna del primer nivel y separación entre niveles, en px dentro de la canaleta. */
export const X0 = 34;
export const NIVEL = 22;
export const ANCHO_CANALETA = 120;
export const x = (nivel: number) => X0 + nivel * NIVEL;

export interface Fila {
  tipo: "turno" | "etapa" | "paso";
  /** El paso de la fila (en una etapa, el primero de la etapa). */
  paso: Paso;
  etapa: string;
  /** Número de etapa dentro del turno (1, 2…). */
  nEtapa: number;
  y: number;
  alto: number;
  cy: number;
}

/** Arma las filas: cada pedido es la fila de su turno; antes del primer paso de cada etapa va su rótulo. */
export function armarFilas(pasos: Paso[]): Fila[] {
  const filas: Fila[] = [];
  let y = 0;
  let etapaPrevia = "";
  let turnoPrevio = -1;
  let nEtapa = 0;
  const agregar = (tipo: Fila["tipo"], paso: Paso, etapa = "") => {
    const alto = ALTO[tipo];
    filas.push({ tipo, paso, etapa, nEtapa, y, alto, cy: y + alto / 2 });
    y += alto;
  };
  for (const p of pasos) {
    if (p.turno !== turnoPrevio) {
      turnoPrevio = p.turno;
      etapaPrevia = "";
      nEtapa = 0;
    }
    if (p.tipo === "Pide") {
      agregar("turno", p);
      continue;
    }
    if (p.etapa && p.etapa !== etapaPrevia) {
      etapaPrevia = p.etapa;
      nEtapa++;
      agregar("etapa", p, p.etapa);
    }
    agregar("paso", p);
  }
  return filas;
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
  const conPaso = filas.filter((f) => f.tipo !== "etapa");
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

  // Cada etapa cruza el riel con una marca corta.
  for (const f of filas) if (f.tipo === "etapa") trazos.push({ d: `M ${x(0) - 5} ${f.cy} H ${x(0) + 5}`, clase: "etapa" });

  for (const f of conPaso) nodos.push({ n: f.paso.n, cx: x(nivel(f)), cy: f.cy });
  return { trazos, nodos, alto };
}

/** Coma decimal y un decimal fijo: «0,4 k», «181,4 k», «4,92 M» (la regla: lo que cambia en vivo no mueve el diseño; las cifras no bailan). */
export function tokensCortos(n: number): string {
  if (n < 1000) return `${n}`;
  if (n < 1_000_000) return `${(n / 1000).toFixed(1).replace(".", ",")} k`;
  return `${(n / 1_000_000).toFixed(2).replace(".", ",")} M`;
}

/** La parte de la salida de su mensaje que le toca a un paso: las llamadas de un lote la reparten. */
export function salidaDelPaso(p: Paso): number {
  return esHerramienta(p) && p.paralelo > 1 ? Math.round(p.tok_salida / p.paralelo) : p.tok_salida;
}

/** La ventana de contexto del modelo, inferida: si alguna petición pasó de 200 k, es de 1 M. */
export function ventana(pasos: Paso[]): number {
  return pasos.some((p) => p.tok_contexto > 200_000) ? 1_000_000 : 200_000;
}
