// El avance por fase: tipos de /api/avance y lo que se calcula para dibujar el hilo de la
// fase. El % lo calcula el backend (internal/avance); aquí solo se rotula y se reparte el ancho.

import { decimal, t } from "./i18n.ts";

export type EstadoPaquete = "hecho" | "en curso" | "sin empezar" | "bloqueado";

export interface Paquete {
  id: string;
  que: string;
  horas: number;
  estado: EstadoPaquete | string;
  pct: number;
  evidencia?: string;
}

export interface FaseAvance {
  fase: string;
  ruta: string;
  plan?: string;
  ahora: string;
  declarado: number;
  calculado: number;
  horas: number;
  horas_hechas: number;
  paquetes: Paquete[];
  hitos: { hora: string; texto: string }[];
  avisos: string[];
}

export interface RespuestaAvance {
  proyecto: string;
  fases: FaseAvance[];
  sin?: string;
}

/** Coma decimal y siempre un decimal: «82,8». */
export const uno = (n: number) => decimal(n);

/** Horas sin decimales si son enteras: «4», «4,5». */
export const horas = (n: number) => (Number.isInteger(n) ? String(n) : uno(n));

/** Los colores del hilo, de la identidad: verde lo hecho, ámbar lo avanzado en curso, gris lo que falta. */
// Lo que falta usa el gris del tema: #3f3f45 en oscuro y más claro en claro (en claro, #3f3f45 se ve casi negro).
export const COLOR_HILO = { hecho: "#3f9f6a", curso: "#c9a227", falta: "var(--borde-fuerte)", bloqueado: "#b4483f" } as const;

/** El estado en palabra y su clase de texto: el color nunca va solo. */
export function tonoEstado(e: string): { palabra: string; clase: string } {
  switch (e) {
    case "hecho":
      return { palabra: "hecho", clase: "text-ok" };
    case "en curso":
      return { palabra: "en curso", clase: "text-aviso" };
    case "bloqueado":
      return { palabra: "bloqueado", clase: "text-error" };
    default:
      return { palabra: e || "sin empezar", clase: "text-suave" };
  }
}

export interface Tramo {
  paquete: Paquete;
  /** Peso del tramo en el ancho (sus horas; 1 si no las dice, para que se vea). */
  peso: number;
  fondo: string;
  /** Lo avanzado dentro del tramo (0–100), pintado encima del fondo; solo en curso. */
  relleno: number;
  aria: string;
}

export function tramos(f: Pick<FaseAvance, "paquetes">): Tramo[] {
  return f.paquetes.map((p) => {
    const hecho = p.estado === "hecho";
    const curso = p.estado === "en curso";
    const fondo = hecho ? COLOR_HILO.hecho : p.estado === "bloqueado" ? COLOR_HILO.bloqueado : COLOR_HILO.falta;
    return {
      paquete: p,
      peso: p.horas > 0 ? p.horas : 1,
      fondo,
      relleno: curso ? Math.max(0, Math.min(100, p.pct)) : 0,
      aria: t("{id}, {que}, {horas} horas, {estado}, {pct} %", { id: p.id, que: p.que || t("sin descripción"), horas: horas(p.horas), estado: t(tonoEstado(p.estado).palabra), pct: p.pct }),
    };
  });
}

/** Las horas que faltan: lo que no está hecho de cada paquete. */
export function horasQueFaltan(f: Pick<FaseAvance, "paquetes">): number {
  return f.paquetes.reduce((s, p) => s + (p.horas * (100 - p.pct)) / 100, 0);
}

/** El paquete que se muestra al llegar: el elegido si existe, si no el primero en curso, si no el primero. */
export function paqueteInicial(f: Pick<FaseAvance, "paquetes">, elegido: string | null): Paquete | null {
  return f.paquetes.find((p) => p.id === elegido) ?? f.paquetes.find((p) => p.estado === "en curso") ?? f.paquetes[0] ?? null;
}
