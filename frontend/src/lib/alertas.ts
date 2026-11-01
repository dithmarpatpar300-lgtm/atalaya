import type { Alerta, NivelAlerta, Regla } from "./api.ts";
import { decimal, t, tServidor } from "./i18n.ts";

/** El orden de gravedad: menor es más grave. */
export const GRAVEDAD: Record<NivelAlerta, number> = { error: 0, aviso: 1, info: 2 };

/** El nivel en palabra: el color y el icono nunca van solos. */
export const NOMBRE_NIVEL: Record<NivelAlerta, string> = { error: "Error", aviso: "Aviso", info: "Info" };

/**
 * Clases de color por nivel. Nunca van solas: cada alerta dice su regla y su nivel en texto.
 * `texto`, `fondo` y `barra` sirven al chip de la fila y a la barra; `carril`, `linea` y `borde`, a la tarjeta de la
 * alerta (la columna de nivel a la izquierda, su filo y el contorno de la tarjeta).
 */
export const COLOR: Record<NivelAlerta, { texto: string; fondo: string; barra: string; carril: string; linea: string; borde: string }> = {
  error: { texto: "text-error", fondo: "bg-error-suave", barra: "bg-error", carril: "bg-error-suave", linea: "border-error/20", borde: "border-error/35" },
  aviso: { texto: "text-aviso", fondo: "bg-aviso-suave", barra: "bg-aviso", carril: "bg-aviso-suave", linea: "border-aviso/20", borde: "border-aviso/35" },
  info: { texto: "text-suave", fondo: "bg-elevado", barra: "bg-borde-fuerte", carril: "bg-elevado", linea: "border-borde", borde: "border-borde" },
};

/** Una cifra con siempre `dec` decimales y coma decimal (regla: lo que cambia en vivo no mueve el diseño; «88,0», no «88»). */
export function fija(n: number, dec = 1): string {
  return decimal(Number.isFinite(n) ? n : 0, dec);
}

/** Cuánto lleva de su tope, en porcentaje con un decimal: «88,0 %». Puede pasar de 100 (un presupuesto vencido). */
export function porcentajeDe(a: Pick<Alerta, "valor" | "max">): string {
  return `${fija(a.max > 0 ? (a.valor / a.max) * 100 : 0)} %`;
}

const fijaTokens = (n: number) => (n < 1_000_000 ? `${fija(n / 1000)} k` : `${fija(n / 1_000_000)} M`);

/** Lo que mide la barra, en sus unidades: «180,0 k de 200,0 k» (tokens), «26 de 30 min». */
export function detalleBarra(a: Pick<Alerta, "regla" | "valor" | "max">): string {
  if (a.regla === "contexto") return t("{valor} de {max}", { valor: fijaTokens(a.valor), max: fijaTokens(a.max) });
  if (a.regla === "tiempo") return t("{valor} de {max} min", { valor: Math.floor(a.valor), max: Math.round(a.max) });
  return t("{valor} de {max}", { valor: fija(a.valor), max: fija(a.max) });
}

/** Una marca sobre la barra: dónde cae (0 a 1) y qué es. */
export interface Marca {
  pos: number;
  nombre: string;
}

/**
 * Las marcas del umbral sobre la barra de una alerta, si las reglas lo dicen: el umbral de contexto («80 %») o el
 * aviso del presupuesto de tiempo (su nota dice «Aviso a los 25 min» de un tope de 30; con un presupuesto propio el
 * margen es el mismo). Sin la regla, o si la marca cae fuera de la barra, no hay marca.
 */
export function marcasDe(a: Pick<Alerta, "regla" | "max">, reglas: readonly Regla[]): Marca[] {
  if (a.max <= 0) return [];
  const r = reglas.find((x) => x.id === a.regla);
  if (!r) return [];
  const dentro = (m: Marca) => (m.pos > 0 && m.pos < 1 ? [m] : []);
  if (a.regla === "contexto") {
    const pct = /(\d+(?:[.,]\d+)?)\s*%/.exec(r.umbral);
    if (!pct) return [];
    const v = Number(pct[1].replace(",", "."));
    return dentro({ pos: v / 100, nombre: t("umbral {valor}", { valor: tServidor(r.umbral.trim()) }) });
  }
  if (a.regla === "tiempo") {
    const tope = /(\d+)\s*min/.exec(r.umbral);
    const aviso = /aviso a los (\d+) min/i.exec(r.nota);
    if (!tope || !aviso) return [];
    const margen = Number(tope[1]) - Number(aviso[1]);
    if (margen <= 0) return [];
    const en = a.max - margen;
    return dentro({ pos: en / a.max, nombre: t("aviso {n} min", { n: Math.round(en) }) });
  }
  return [];
}

/**
 * La alerta más grave de cada agente, para el chip de su fila. La frontera toca a dos agentes: va en los dos.
 */
export function porAgente(alertas: Alerta[]): Map<string, Alerta> {
  const m = new Map<string, Alerta>();
  const poner = (id: string, a: Alerta) => {
    const ya = m.get(id);
    if (!ya || GRAVEDAD[a.nivel] < GRAVEDAD[ya.nivel]) m.set(id, a);
  };
  for (const a of alertas) {
    // Sin chip (un cruce de archivos ya cerrado): solo vive en el centro de alertas, la fila queda limpia.
    if (!a.chip) continue;
    poner(a.agente, a);
    if (a.otro) poner(a.otro, a);
  }
  return m;
}

/** Las vistas se guardan para todos los proyectos; se conservan las últimas `max`, las más viejas ya no vuelven. */
export function recortarVistas(vistas: ReadonlySet<string>, max = 300): Set<string> {
  const v = [...vistas];
  return new Set(v.slice(Math.max(0, v.length - max)));
}

const CLAVE_VISTAS = "atalaya:alertas-vistas";

export function leerVistas(): Set<string> {
  try {
    const v = JSON.parse(localStorage.getItem(CLAVE_VISTAS) ?? "[]") as unknown;
    return new Set(Array.isArray(v) ? v.filter((x): x is string => typeof x === "string") : []);
  } catch {
    return new Set();
  }
}

export function guardarVistas(v: ReadonlySet<string>) {
  try {
    localStorage.setItem(CLAVE_VISTAS, JSON.stringify([...recortarVistas(v)]));
  } catch {}
}
