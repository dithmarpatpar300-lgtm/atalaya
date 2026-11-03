// Los hilos de la actividad: los pasos plegados por lote (un mensaje de la IA con sus llamadas en paralelo), los enlaces
// entre lotes que tocan el mismo archivo, la pista de radio fijo donde corren y el mapa de toda la sesión. Sin React ni
// DOM: se prueba con `node --test` (lotes.test.mts) y el componente solo pinta lo que devuelve.
import type { Paso } from "./api.ts";
import { type Fila, type MarcaEtapa } from "./hilos.ts";
import { t } from "./i18n.ts";

export const ALTO_LOTE = 40;
export const ALTO_HIJO = 30;
/** Ancho de la canaleta: fijo, para que un enlace nuevo no corra el texto de las filas. */
export const CANALETA = 120;
export const CANALETA_ANGOSTA = 56;
/** Separación entre hebras de un enlace y radio de sus esquinas, siempre el mismo. */
const HEBRA = 2.6;
const RADIO = 6;
const MAX_HEBRAS = 8;
/** Lo que la pista deja a la derecha del riel, para la llegada de lo que viene de antes. */
const MARGEN_RIEL = 22;

export interface Lote {
  /** El número de su primer paso: su identidad, no cambia mientras el lote crece. */
  n: number;
  filas: Fila[];
  /** Un pedido de la persona: va solo, es la fila de su turno. */
  turno: boolean;
  /** Los archivos que toca, cada uno una vez, en orden de aparición. */
  archivos: string[];
  edita: boolean;
  errores: number;
  /** La primera marca de etapa de sus pasos. */
  marca: MarcaEtapa | null;
}

/** Pliega los pasos seguidos de un mismo mensaje. Un pedido siempre va solo: es la fila de su turno. */
export function plegar(filas: readonly Fila[]): Lote[] {
  const lotes: Lote[] = [];
  for (const f of filas) {
    const p = f.paso;
    let l = lotes[lotes.length - 1];
    if (!l || l.turno || p.tipo === "Pide" || l.filas[l.filas.length - 1].paso.lote !== p.lote) {
      l = { n: p.n, filas: [], turno: p.tipo === "Pide", archivos: [], edita: false, errores: 0, marca: null };
      lotes.push(l);
    }
    l.filas.push(f);
    if (p.hilo && !l.archivos.includes(p.hilo)) l.archivos.push(p.hilo);
    if (p.tipo === "Edita" || p.tipo === "Escribe") l.edita = true;
    if (p.error) l.errores++;
    if (!l.marca && f.marca) l.marca = f.marca;
  }
  return lotes;
}

/** «edita 8, ejecuta 2, responde 1»: lo que hizo el lote, lo más repetido primero. */
export function resumenLote(l: Lote): string {
  const cuenta = new Map<string, number>();
  for (const f of l.filas) cuenta.set(f.paso.tipo, (cuenta.get(f.paso.tipo) ?? 0) + 1);
  return [...cuenta.entries()]
    .sort((a, b) => b[1] - a[1])
    .map(([tipo, n]) => `${t(tipo).toLowerCase()} ${n}`)
    .join(", ");
}

export const nombreArchivo = (hilo: string) => hilo.split("/").pop() ?? hilo;

/** Dos lotes (por su posición) que tocan los mismos archivos, sin otro lote de por medio que los toque. */
export interface Enlace {
  a: number;
  b: number;
  archivos: string[];
}

/**
 * Une cada archivo con la última vez que apareció, agrupado por par de lotes. Un archivo que ya se había tocado antes de
 * esta página es una llegada: un trazo corto junto al nodo, nunca una línea larga por toda la lista. `previos` son los
 * pasos anteriores a la página (null si no se tienen: entonces solo avisa `continua`).
 */
export function enlazar(lotes: readonly Lote[], previos: readonly Paso[] | null): { enlaces: Enlace[]; llegadas: Map<number, string> } {
  const antes = new Set((previos ?? []).map((p) => p.hilo).filter(Boolean));
  const primero = lotes[0]?.n ?? 0;
  const ultimo = new Map<string, number>();
  const grupos = new Map<string, Enlace>();
  const llegadas = new Map<number, string>();
  lotes.forEach((l, i) => {
    for (const archivo of l.archivos) {
      const a = ultimo.get(archivo);
      if (a !== undefined) {
        const clave = `${a}>${i}`;
        const g = grupos.get(clave);
        if (g) g.archivos.push(archivo);
        else grupos.set(clave, { a, b: i, archivos: [archivo] });
      } else if (!llegadas.has(i)) {
        const retoma = l.filas.some((f) => f.paso.hilo === archivo && f.paso.continua >= 0 && f.paso.continua < primero);
        if (antes.has(archivo) || retoma) llegadas.set(i, archivo);
      }
      ultimo.set(archivo, i);
    }
  });
  return { enlaces: [...grupos.values()], llegadas };
}

/** El color de cada archivo, por primera aparición en la página: lo que llega después no repinta a los de antes. */
export function coloresDeArchivos(lotes: readonly Lote[]): Map<string, string> {
  const m = new Map<string, string>();
  for (const l of lotes) for (const f of l.archivos) if (!m.has(f)) m.set(f, `var(--serie-${(m.size % 5) + 1})`);
  return m;
}

export interface Colocado extends Enlace {
  /** Distancia al riel de su primera hebra. */
  x: number;
  carril: number;
}

export interface Pista {
  rail: number;
  colocados: Colocado[];
  /** Los que no caben en la canaleta: su lote de llegada lo dice con el trazo corto. */
  sobran: Enlace[];
}

/**
 * Pista de radio fijo: cada enlace toma el carril libre más cercano al riel, los cortos adentro. Dos enlaces que solo se
 * tocan en un nodo comparten carril: se leen como una línea que sigue. Con menos sitio (el teléfono), `soloDe` deja los
 * del lote elegido.
 */
export function colocar(enlaces: readonly Enlace[], ancho: number, soloDe?: number): Pista {
  const rail = ancho - MARGEN_RIEL;
  const visibles = soloDe === undefined ? [...enlaces] : enlaces.filter((e) => e.a === soloDe || e.b === soloDe);
  const carriles: Enlace[][] = [];
  const carrilDe = new Map<Enlace, number>();
  visibles
    .sort((p, q) => p.b - p.a - (q.b - q.a) || p.a - q.a)
    .forEach((e) => {
      let k = 0;
      while ((carriles[k] ?? []).some((o) => o.a < e.b && e.a < o.b)) k++;
      (carriles[k] ??= []).push(e);
      carrilDe.set(e, k);
    });
  const xCarril: number[] = [];
  let acum = 12;
  for (const lista of carriles) {
    const anchoCarril = Math.max(...lista.map((e) => (Math.min(e.archivos.length, MAX_HEBRAS) - 1) * HEBRA));
    // Si su hebra más lejana no cabe, ni este carril ni los de afuera se dibujan.
    if (acum + anchoCarril > rail - 4) break;
    xCarril.push(acum);
    acum += anchoCarril + 6;
  }
  const colocados: Colocado[] = [];
  const sobran: Enlace[] = [];
  for (const e of visibles) {
    const carril = carrilDe.get(e) ?? 0;
    if (carril < xCarril.length) colocados.push({ ...e, carril, x: xCarril[carril] });
    else sobran.push(e);
  }
  return { rail, colocados, sobran };
}

export interface Columnas {
  angosto: boolean;
  plantilla: string;
  hora: number;
  canaleta: number;
  /** La columna de archivos: el nombre con sus puntos, solo los puntos, o nada. */
  archivos: "nombre" | "puntos" | null;
  tokens: boolean;
}

/** Las columnas de una fila según lo que mide la lista (0: aún no se midió). Lo primero que se va es lo que repite el inspector. */
export function columnas(lista: number, conTokens: boolean): Columnas {
  if (lista > 0 && lista < 420) {
    return { angosto: true, plantilla: `40px ${CANALETA_ANGOSTA}px 30px minmax(0,1fr) 14px`, hora: 40, canaleta: CANALETA_ANGOSTA, archivos: null, tokens: false };
  }
  const amplia = lista === 0 || lista >= 760;
  const anchoArchivos = amplia ? 230 : lista >= 620 ? 150 : 66;
  const tokens = conTokens && (amplia || lista >= 560);
  const plantilla = `46px ${CANALETA}px 42px minmax(0,1fr) ${anchoArchivos}px ${tokens ? `${amplia ? 84 : 72}px ` : ""}14px`;
  return { angosto: false, plantilla, hora: 46, canaleta: CANALETA, archivos: anchoArchivos === 66 ? "puntos" : "nombre", tokens };
}

export interface Renglon {
  clave: string;
  y: number;
  alto: number;
  /** La posición de su lote. */
  i: number;
  lote: Lote;
  /** El paso, en una fila de lote abierto; null en la fila del lote. */
  hijo: Fila | null;
}

/** Las filas que se ven: cada lote y, si está abierto, sus pasos. También el centro de cada lote y el alto total. */
export function renglones(lotes: readonly Lote[], abierto: (l: Lote) => boolean): { filas: Renglon[]; cy: number[]; alto: number } {
  const filas: Renglon[] = [];
  const cy: number[] = [];
  let y = 0;
  lotes.forEach((lote, i) => {
    filas.push({ clave: `l${lote.n}`, y, alto: ALTO_LOTE, i, lote, hijo: null });
    cy.push(y + ALTO_LOTE / 2);
    y += ALTO_LOTE;
    if (lote.filas.length > 1 && abierto(lote)) {
      for (const hijo of lote.filas) {
        filas.push({ clave: `p${hijo.paso.n}`, y, alto: ALTO_HIJO, i, lote, hijo });
        y += ALTO_HIJO;
      }
    }
  });
  return { filas, cy, alto: y };
}

export interface Trazo {
  d: string;
  color: string;
  ancho: number;
  opacidad: number;
  raya?: string;
}

/** Todo lo que corre por la canaleta: el riel, las hebras de cada enlace y lo que viene de antes. */
export function trazar(
  pista: Pista,
  cy: readonly number[],
  alto: number,
  eleccion: number,
  colores: ReadonlyMap<string, string>,
  llegadas: ReadonlyMap<number, string>,
): Trazo[] {
  const { rail } = pista;
  const color = (archivo: string) => colores.get(archivo) ?? "var(--riel)";
  const trazos: Trazo[] = [{ d: `M ${rail} 0 V ${alto}`, color: "var(--riel)", ancho: 1.75, opacidad: 1 }];
  const toca = (e: Enlace) => e.a === eleccion || e.b === eleccion;
  const orden = [...pista.colocados].sort((p, q) => Number(toca(p)) - Number(toca(q)) || p.carril - q.carril);
  for (const e of orden) {
    const ya = cy[e.a];
    const yb = cy[e.b];
    const on = toca(e);
    const r = Math.min(RADIO, (yb - ya) / 2);
    const xMin = rail - e.x;
    // Un corte del color del fondo donde su tramo horizontal cruza los carriles de adentro.
    if (e.carril > 0) {
      for (const y of [ya, yb]) trazos.push({ d: `M ${rail - 7} ${y} H ${xMin + r + 1}`, color: "var(--superficie)", ancho: 4.5, opacidad: 1 });
    }
    e.archivos.slice(0, MAX_HEBRAS).forEach((archivo, k) => {
      const xl = rail - e.x - k * HEBRA;
      trazos.push({
        d: `M ${rail - 7} ${ya} H ${xl + r} A ${r} ${r} 0 0 0 ${xl} ${ya + r} V ${yb - r} A ${r} ${r} 0 0 0 ${xl + r} ${yb} H ${rail - 7}`,
        color: color(archivo),
        ancho: on ? 1.75 : 1.25,
        opacidad: on ? 1 : 0.42,
      });
    });
  }
  // Un solo trazo corto por nodo, a la derecha, aunque sean varios archivos: lo que viene de antes y lo que no cupo.
  const cortos = new Map(llegadas);
  for (const e of pista.sobran) if (!cortos.has(e.b)) cortos.set(e.b, e.archivos[0]);
  for (const [b, archivo] of cortos) {
    const y = cy[b];
    trazos.push({
      d: `M ${rail + 7} ${y} H ${rail + 10} A 4 4 0 0 0 ${rail + 14} ${y - 4} V ${y - 15}`,
      color: color(archivo),
      ancho: 1.5,
      opacidad: b === eleccion ? 1 : 0.7,
      raya: "2 3",
    });
  }
  return trazos;
}

/** La franja de la elección: «23:33, 7 pasos: retoma 2 archivos y sigue 1 después. a.rs, b.rs». */
export function franja(l: Lote, i: number, enlaces: readonly Enlace[], cuando: string): string {
  const cuantos = (n: number) => (n === 1 ? t("1 archivo") : t("{n} archivos", { n }));
  const retoma = enlaces.reduce((s, e) => s + (e.b === i ? e.archivos.length : 0), 0);
  const deja = enlaces.reduce((s, e) => s + (e.a === i ? e.archivos.length : 0), 0);
  const pasos = l.filas.length === 1 ? t("1 paso") : t("{n} pasos", { n: l.filas.length });
  const base = retoma || deja ? t("{cuando}, {pasos}: retoma {a} y sigue {b} después", { cuando, pasos, a: cuantos(retoma), b: cuantos(deja) }) : `${cuando}, ${pasos}`;
  return l.archivos.length ? `${base}. ${l.archivos.map(nombreArchivo).join(", ")}` : base;
}

/** Más intensa, más ediciones; nunca invisible. */
export const opacidadCelda = (edita: number) => Math.min(1, 0.16 + edita * 1.35);

/** El marco del mapa sobre los pasos a la vista, en % del alto; nunca tan chico que no se vea. */
export function marcoMapa(inicio: number, fin: number, total: number): { top: number; alto: number } {
  if (total <= 0) return { top: 0, alto: 100 };
  const alto = Math.max(1.5, ((fin - inicio) / total) * 100);
  return { top: Math.min(100 - alto, (inicio / total) * 100), alto };
}

/** Las celdas donde cambia el día, con su posición: la primera siempre. */
export function cambiosDeDia(celdas: readonly { hora: string }[]): number[] {
  const out: number[] = [];
  let dia = "";
  celdas.forEach((c, i) => {
    const d = new Date(c.hora).toDateString();
    if (d !== dia) out.push(i);
    dia = d;
  });
  return out;
}
