// Las estadísticas por clase: tipos de /api/estadisticas y lo que la sección calcula para
// dibujar, puro para poder probarlo. Las medianas, los p90 y la matriz los decide el backend (internal/estadisticas).

export type PeriodoEstadisticas = "semana" | "mes" | "fase";
export type MedidaEstadisticas = "minutos" | "salida" | "nuevos";

export interface Medida {
  mediana: number;
  p90: number;
}

export interface ClaseEstadistica {
  clase: string;
  nombre: string;
  proveedor: "Claude" | "Codex";
  n: number;
  minutos: Medida;
  salida: Medida;
  nuevos: Medida;
  presupuesto: number;
  presupuesto_fuente: string;
  matriz?: string;
  matriz_fuente?: string;
  en_matriz: number;
  /** Los agentes que forman la mediana, del más nuevo al más viejo (40 a lo sumo). */
  agentes?: AgenteClase[];
}

export interface AgenteClase {
  id: string;
  nombre: string;
  proveedor: "Claude" | "Codex";
  modelo: string;
  minutos: number;
  presupuesto: number;
  fin: string;
}

export interface CeldaEstadistica {
  clase: string;
  modelo: string;
  n: number;
  minutos: number;
  en_matriz: boolean;
  matriz?: string;
}

export interface SemanaEstadisticas {
  desde: string;
  agentes: number;
  claude: number;
  codex: number;
  opus: number;
  minutos: number;
  mediana: number;
  nuevos: number;
}

export interface InformeEstadisticas {
  totales: {
    agentes: number;
    claude: number;
    codex: number;
    minutos: number;
    salida: number;
    nuevos: number;
    cache_pct: number;
    opus: number;
    opus_pct: number;
    meta_opus: number;
    fuente_opus?: string;
  };
  clases: ClaseEstadistica[];
  modelos: string[];
  celdas: CeldaEstadistica[];
  semanas: SemanaEstadisticas[];
  dias_diario: number;
  matriz?: string;
  periodo: PeriodoEstadisticas;
  fase?: string;
  tope: string;
}

/** Menos de esto, §12 de AGENTES.md no saca conclusiones de una clase. */
export const MINIMO_SENAL = 5;

const coma = (n: number, dec = 1) => (Number.isFinite(n) ? n : 0).toFixed(dec).replace(".", ",");

/** Tokens con un decimal fijo: «41,1 k», «1,19 M». */
export function miles(n: number): string {
  if (n < 1000) return String(Math.round(n));
  if (n < 1_000_000) return `${coma(n / 1000)} k`;
  return `${coma(n / 1_000_000, 2)} M`;
}

/** Una cifra de la medida elegida, siempre con los mismos decimales (lo que cambia en vivo no mueve el diseño). */
export function cifra(medida: MedidaEstadisticas, v: number): string {
  return medida === "minutos" ? `${coma(v)} min` : miles(v);
}

/** El valor de la medida en una clase. */
export const medidaDe = (c: ClaseEstadistica, m: MedidaEstadisticas): Medida => c[m];

/**
 * La escala de las barras: el mayor p90 y, en minutos, el mayor presupuesto, con un 8 % de aire para que la marca
 * del presupuesto no quede pegada al borde.
 */
export function escala(clases: ClaseEstadistica[], m: MedidaEstadisticas): number {
  let max = 0;
  for (const c of clases) {
    max = Math.max(max, medidaDe(c, m).p90, m === "minutos" ? c.presupuesto : 0);
  }
  return max > 0 ? max * 1.08 : 1;
}

/** Porcentaje del ancho con un decimal, para el estilo de la barra. */
export const ancho = (v: number, max: number) => `${Math.min(100, Math.max(0, (v / max) * 100)).toFixed(1)}%`;

export interface Lectura {
  texto: string;
  tono: "ok" | "aviso" | "suave";
}

/** Lo que dice una clase: si hay datos para concluir, si su p90 pasa del presupuesto y si sigue la matriz. */
export function lectura(c: ClaseEstadistica, m: MedidaEstadisticas): Lectura {
  const partes: string[] = [];
  let tono: Lectura["tono"] = "suave";
  if (m === "minutos") {
    if (c.minutos.p90 > c.presupuesto) {
      partes.push(`El p90 pasa del presupuesto de ${coma(c.presupuesto, 0)} min (${c.presupuesto_fuente}): 1 de cada 10 se pasa.`);
      tono = "aviso";
    } else {
      partes.push(`Dentro del presupuesto de ${coma(c.presupuesto, 0)} min (${c.presupuesto_fuente}), incluso en el p90.`);
      tono = "ok";
    }
  } else if (m === "salida") {
    partes.push("Tokens que escribió el modelo, por agente.");
  } else {
    partes.push("Entrada que no vino de la caché, por agente: muchos indican demasiado contexto (§9).");
  }
  if (c.matriz) {
    const fuera = c.n - c.en_matriz;
    partes.push(fuera === 0 ? `Todos con la matriz (${c.matriz}).` : `${fuera} de ${c.n} fuera de la matriz (${c.matriz}).`);
    if (fuera > 0 && tono === "ok") tono = "aviso";
  } else if (c.clase !== "—") {
    partes.push("La matriz no fija esta clase para este proveedor.");
  }
  if (c.n < MINIMO_SENAL) {
    partes.push(`Con ${c.n} ${c.n === 1 ? "tarea" : "tareas"}, §12 todavía no saca conclusiones.`);
    if (tono === "aviso") tono = "suave";
  }
  return { texto: partes.join(" "), tono };
}

/** La celda de la tabla clase × modelo: «21,9 (5)», o «—» si no hay agentes. */
export function celdaTexto(c: CeldaEstadistica | undefined): string {
  return c ? `${coma(c.minutos)} (${c.n})` : "—";
}

/** Las clases de la tabla clase × modelo, en el orden de las barras y sin repetir. */
export function filasTabla(clases: ClaseEstadistica[]): string[] {
  const out: string[] = [];
  for (const c of clases) if (!out.includes(c.clase)) out.push(c.clase);
  return out;
}
