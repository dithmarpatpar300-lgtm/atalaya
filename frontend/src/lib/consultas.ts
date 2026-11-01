/**
 * La lógica pura de la vista Consultas (components/panel/consultas.tsx): qué consultas caen en el periodo, los
 * filtros del registro, los días sin consultas del gráfico y la escala de las barras. Sin React, para probarla con
 * node --test.
 */
import type { Consulta, FilaAhorro } from "./api";

export type Filtro = "todas" | "ahorro" | "releyo" | "repetidas" | "fallos";

/** Cada filtro del registro: la misma regla para la cuenta del botón y para la lista. */
export const PASA: Record<Filtro, (c: Consulta) => boolean> = {
  todas: () => true,
  ahorro: (c) => (c.ahorro_tokens ?? 0) > 0,
  releyo: (c) => c.releyo === true,
  repetidas: (c) => c.repetida === true,
  fallos: (c) => c.error,
};

/** Las consultas desde hace `dias` días, la misma ventana que el resumen del servidor (ahora − dias). */
export function delPeriodo(lista: readonly Consulta[], dias: number, ahora: number): Consulta[] {
  const desde = ahora - dias * 86_400_000;
  return lista.filter((c) => {
    const h = Date.parse(c.hora);
    return Number.isFinite(h) && h >= desde;
  });
}

/** Las cuentas de cada filtro sobre la misma lista. */
export function cuentas(lista: readonly Consulta[]): Record<Filtro, number> {
  const out = { todas: 0, ahorro: 0, releyo: 0, repetidas: 0, fallos: 0 } as Record<Filtro, number>;
  for (const c of lista) for (const f of Object.keys(PASA) as Filtro[]) if (PASA[f](c)) out[f]++;
  return out;
}

/** «2026-10-31» en la hora local, la clave de los días del servidor (Hora.Local()). */
export function claveDia(f: Date): string {
  return `${f.getFullYear()}-${String(f.getMonth() + 1).padStart(2, "0")}-${String(f.getDate()).padStart(2, "0")}`;
}

const vacia = (clave: string): FilaAhorro => ({
  clave,
  n: 0,
  bytes: 0,
  alternativa_bytes: 0,
  alternativa_tokens: 0,
  ahorro_bytes: 0,
  ahorro_tokens: 0,
  releyo: 0,
  repetidas: 0,
  errores: 0,
});

/**
 * Un día por columna, del más viejo a hoy, con los días sin consultas en cero: el gráfico tiene siempre `dias`
 * columnas (con 14 días no se corre cuando llega una consulta de un día nuevo) y un hueco se ve como hueco.
 */
export function diasSeguidos(filas: readonly FilaAhorro[] | null, dias: number, ahora: number): FilaAhorro[] {
  const porClave = new Map((filas ?? []).map((f) => [f.clave, f]));
  const hoy = new Date(ahora);
  const out: FilaAhorro[] = [];
  for (let i = dias - 1; i >= 0; i--) {
    const d = new Date(hoy.getFullYear(), hoy.getMonth(), hoy.getDate() - i);
    const k = claveDia(d);
    out.push(porClave.get(k) ?? vacia(k));
  }
  return out;
}

/**
 * El tope de un eje: el máximo redondeado hacia arriba a 1, 2 o 5 por una potencia de 10 (34 → 50, 15 191 → 20 000),
 * para que la raya de arriba lleve una cifra redonda. Nunca 0, para no dividir por cero.
 */
export function topeRedondo(max: number): number {
  if (!(max > 0)) return 1;
  const p = 10 ** Math.floor(Math.log10(max));
  for (const m of [1, 2, 5, 10]) if (m * p >= max) return m * p;
  return 10 * p;
}

/** Ancho de una barra en % de `tope`, con un mínimo visible para lo que no es cero (1 de 15 000 se ve). */
export function ancho(v: number, tope: number): number {
  if (!(v > 0) || !(tope > 0)) return 0;
  return Math.max(1.5, Math.min(100, (v / tope) * 100));
}
