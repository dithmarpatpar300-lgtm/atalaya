/**
 * La lógica pura del gráfico «Contexto por petición»: el rango visible, la escala vertical,
 * las etapas y el carril de las compactaciones con detección de choques. Sin React, para probarla.
 */

/** Los rangos del selector. «compactacion»: desde la última compactación (con unas peticiones de antes, para ver la caída). */
export type Rango = "compactacion" | "ultimas" | "todas";

/** Cuántas peticiones muestra «Últimas». */
export const ULTIMAS = 200;
/** Peticiones de antes de la compactación que se dejan a la vista, para que se vea el salto. */
export const MARGEN_COMPACTACION = 8;
/** Con menos peticiones que estas tras la compactación, el rango «compactacion» casi no tiene curva: no es la opción por omisión. */
const MINIMO_TRAS_COMPACTAR = 10;

type ConPeticion = { peticion: number };

/** Las compactaciones que caen dentro de la serie: su primera petición posterior existe y tiene una anterior. */
export function compactacionesEnSerie<T extends ConPeticion>(comps: T[], n: number): T[] {
  return comps.filter((k) => k.peticion > 0 && k.peticion < n);
}

/** Los rangos que tienen sentido para esta serie, en el orden del selector. */
export function rangosPosibles(n: number, comps: ConPeticion[]): Rango[] {
  const r: Rango[] = [];
  if (compactacionesEnSerie(comps, n).length > 0) r.push("compactacion");
  if (n > ULTIMAS) r.push("ultimas");
  r.push("todas");
  return r;
}

/**
 * El rango por omisión: lo que hay en el contexto ahora. Si hubo una compactación con curva detrás, desde ella;
 * si no, las últimas {@link ULTIMAS} en una serie larga; si no, todas. Así las compactaciones viejas no quedan
 * como líneas pegadas al borde: se ven al cambiar de rango o en la tabla.
 */
export function rangoPorOmision(n: number, comps: ConPeticion[]): Rango {
  const dentro = compactacionesEnSerie(comps, n);
  const ult = dentro[dentro.length - 1];
  if (ult && n - ult.peticion >= MINIMO_TRAS_COMPACTAR) return "compactacion";
  return n > ULTIMAS ? "ultimas" : "todas";
}

/** El primer índice visible de la serie para un rango (el último es siempre n − 1: lo nuevo entra por la derecha). */
export function desdeDeRango(rango: Rango, n: number, comps: ConPeticion[]): number {
  if (rango === "ultimas") return Math.max(0, n - ULTIMAS);
  if (rango === "compactacion") {
    const dentro = compactacionesEnSerie(comps, n);
    const ult = dentro[dentro.length - 1];
    if (ult) return Math.max(0, ult.peticion - MARGEN_COMPACTACION);
  }
  return 0;
}

/** Una compactación se dibuja solo si su caída cae dentro del rango: nunca recortada en el borde. */
export function compactacionVisible(peticion: number, desde: number, n: number): boolean {
  return peticion > desde && peticion < n;
}

/**
 * La escala vertical: un tope «redondo» por encima del máximo (con un 8 % de aire) y sus marcas,
 * con pasos de 1, 2, 2,5 o 5 por potencia de diez. Devuelve el tope y las marcas sin el cero.
 */
export function escalaY(maximo: number): { tope: number; ticks: number[] } {
  const m = Math.max(1, maximo * 1.08);
  // De los pasos redondos que dan entre 3 y 5 marcas, el de tope más bajo: menos aire sobre la curva.
  const pot = 10 ** Math.floor(Math.log10(m / 3));
  const pasos = [0.1, 1, 10].flatMap((e) => [1, 2, 2.5, 5].map((f) => f * pot * e));
  let paso = pasos[pasos.length - 1];
  let tope = Infinity;
  for (const p of pasos) {
    const k = Math.ceil(m / p);
    if (k >= 3 && k <= 5 && k * p < tope) [paso, tope] = [p, k * p];
  }
  const ticks: number[] = [];
  for (let v = paso; v <= tope + paso / 1e6; v += paso) ticks.push(Math.round(v));
  return { tope, ticks };
}

/** Un tramo de la serie con la misma etapa (o el mismo turno, si no tiene nombre). `hasta` es exclusivo. */
export type Tramo = { desde: number; hasta: number; turno: number; etapa: string };

/** Parte la serie en tramos donde cambia la etapa o empieza un turno. */
export function tramos(ps: { turno: number; etapa: string }[]): Tramo[] {
  const r: Tramo[] = [];
  ps.forEach((p, i) => {
    const ult = r[r.length - 1];
    if (ult && ult.etapa === p.etapa && ult.turno === p.turno) ult.hasta = i + 1;
    else r.push({ desde: i, hasta: i + 1, turno: p.turno, etapa: p.etapa });
  });
  return r;
}

/** Un grupo del carril: sus miembros (índices de entrada), y dónde va y cuánto mide, en píxeles. */
export type Grupo = { miembros: number[]; left: number; ancho: number };

/**
 * Coloca fichas en un carril de una sola línea sin que se toquen nunca. Cada ficha va centrada en su `x` y se
 * corre para no salirse del carril. Si dos chocan (menos de `hueco` px entre ellas), se funden en un grupo cuyo
 * ancho da `anchoGrupo` (por ejemplo «3 compactaciones»), centrado en el medio de sus miembros; se repite hasta
 * que no queda ningún choque. Ninguna ficha es más ancha que el carril.
 */
export function colocarEnCarril(
  items: { x: number; ancho: number }[],
  carril: number,
  anchoGrupo: (miembros: number[]) => number,
  hueco = 6,
): Grupo[] {
  if (carril <= 0 || items.length === 0) return [];
  const orden = items.map((_, i) => i).sort((a, b) => items[a].x - items[b].x);
  let grupos: number[][] = orden.map((i) => [i]);
  const colocar = (miembros: number[]): Grupo => {
    const ancho = Math.min(carril, miembros.length === 1 ? items[miembros[0]].ancho : anchoGrupo(miembros));
    const xs = miembros.map((i) => items[i].x);
    const centro = (Math.min(...xs) + Math.max(...xs)) / 2;
    const left = Math.max(0, Math.min(carril - ancho, centro - ancho / 2));
    return { miembros, left, ancho };
  };
  for (;;) {
    const puestos = grupos.map(colocar);
    const j = puestos.findIndex((g, k) => k > 0 && puestos[k - 1].left + puestos[k - 1].ancho + hueco > g.left);
    if (j < 0) return puestos;
    grupos = [...grupos.slice(0, j - 1), [...grupos[j - 1], ...grupos[j]], ...grupos.slice(j + 1)];
  }
}
