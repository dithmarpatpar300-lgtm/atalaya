// La salud del proceso: tipos de /api/proceso y las cuentas de la sección, puras para
// poder probarlas. El backend decide cada marca (backend/internal/proceso); aquí solo se cuenta y se rotula.

import type { Regla } from "./api.ts";

export type Marca = "cumple" | "falta" | "medias" | "no_aplica" | "sin_dato";
export type Periodo = "hoy" | "fase" | "todo";

export interface Celda {
  regla: string;
  marca: Marca;
  titulo: string;
  texto: string;
  ruta?: string;
  /** El paso del transcrito que lo prueba y de quién es (-1 si no hay); la ruta ya no lo lleva pegado. */
  paso?: number;
  agente?: string;
}

export interface FilaProceso {
  id: string;
  nombre: string;
  proveedor: "Claude" | "Codex";
  clase: string;
  tarea: string;
  estado: string;
  activo: boolean;
  inicio: string;
  ultimo: string;
  fase?: string;
  celdas: Celda[];
  faltas: number;
  medias: number;
}

export interface CarpetaTemporal {
  ruta: string;
  nombre: string;
  lugar: string;
  bytes: number;
  archivos: number;
  parciales: number;
  mod: string;
  acta?: string;
  acta_estado?: "abierta" | "cerrada";
  agente?: string;
  agente_nombre?: string;
  estado: "en uso" | "agente entregado" | "sin agente";
}

export interface InformeProceso {
  reglas: Regla[];
  lugares: string[];
  filas: FilaProceso[];
  temporales: CarpetaTemporal[];
  periodo: Periodo;
  fase?: string;
  hora: number;
}

/** La marca en signo y en palabra: el color nunca va solo. */
export const MARCA: Record<Marca, { signo: string; palabra: string }> = {
  cumple: { signo: "✓", palabra: "cumple" },
  falta: { signo: "✕", palabra: "falta" },
  medias: { signo: "!", palabra: "a medias" },
  no_aplica: { signo: "—", palabra: "no aplica" },
  sin_dato: { signo: "?", palabra: "sin dato" },
};

/** Clases de color de cada marca: el chip de la celda y la tarjeta del hallazgo. */
export const COLOR_MARCA: Record<Marca, { texto: string; fondo: string }> = {
  cumple: { texto: "text-ok", fondo: "bg-ok-suave" },
  falta: { texto: "text-error", fondo: "bg-error-suave" },
  medias: { texto: "text-aviso", fondo: "bg-aviso-suave" },
  no_aplica: { texto: "text-tenue", fondo: "bg-transparent" },
  sin_dato: { texto: "text-suave", fondo: "bg-elevado" },
};

/** En regla: ninguna falta ni nada a medias en lo que le aplica (lo «sin dato» no cuenta en contra). */
export const enRegla = (f: FilaProceso) => f.faltas === 0 && f.medias === 0;

/** El punto de la fila: rojo con alguna falta, ámbar con algo a medias, verde en regla. */
export function puntoDe(f: FilaProceso): string {
  if (f.faltas > 0) return "bg-error";
  if (f.medias > 0) return "bg-aviso";
  return "bg-ok";
}

/** Los hallazgos de un agente, del más grave al más leve: faltas, a medias y lo que no se pudo saber. */
export function hallazgos(f: FilaProceso): Celda[] {
  const orden: Partial<Record<Marca, number>> = { falta: 0, medias: 1, sin_dato: 2 };
  return f.celdas.filter((c) => c.marca in orden).sort((a, b) => orden[a.marca]! - orden[b.marca]!);
}

export interface Resumen {
  agentes: number;
  enRegla: number;
  faltas: number;
  conFaltas: number;
  /** Las reglas con alguna falta, por su nombre, sin repetir y en el orden de las columnas. */
  reglasConFaltas: string[];
  medias: number;
  mediasActivos: number;
  temporales: number;
  sinActaCerrada: number;
}

export function resumen(inf: Pick<InformeProceso, "reglas" | "filas" | "temporales">): Resumen {
  const nombres = new Map(inf.reglas.map((r) => [r.id, r.nombre]));
  const conFalta = new Set<string>();
  let faltas = 0;
  let medias = 0;
  let mediasActivos = 0;
  for (const f of inf.filas) {
    faltas += f.faltas;
    medias += f.medias;
    if (f.activo) mediasActivos += f.medias;
    for (const c of f.celdas) if (c.marca === "falta") conFalta.add(c.regla);
  }
  return {
    agentes: inf.filas.length,
    enRegla: inf.filas.filter(enRegla).length,
    faltas,
    conFaltas: inf.filas.filter((f) => f.faltas > 0).length,
    reglasConFaltas: inf.reglas.filter((r) => conFalta.has(r.id)).map((r) => (nombres.get(r.id) ?? r.id).toLowerCase()),
    medias,
    mediasActivos,
    temporales: inf.temporales.length,
    sinActaCerrada: inf.temporales.filter((t) => t.acta_estado !== "cerrada").length,
  };
}

/** Bytes con coma decimal y siempre un decimal: «41,0 MB». */
export function peso(b: number): string {
  const u = ["B", "KB", "MB", "GB", "TB"];
  let n = b;
  let i = 0;
  while (n >= 1024 && i < u.length - 1) {
    n /= 1024;
    i++;
  }
  return i === 0 ? `${b} B` : `${n.toFixed(1).replace(".", ",")} ${u[i]}`;
}

/** La fila que se muestra al llegar: la que sigue elegida, si no la primera con faltas, si no la primera. */
export function filaInicial(filas: FilaProceso[], elegida: string | null): string | null {
  if (elegida && filas.some((f) => f.id === elegida)) return elegida;
  return (filas.find((f) => f.faltas > 0) ?? filas[0])?.id ?? null;
}
