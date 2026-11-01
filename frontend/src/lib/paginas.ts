/**
 * La lógica pura del paginador (components/ui/paginador.tsx): la primera y la
 * última página siempre a la vista. Con más de 7 páginas, siempre 7 casillas: la actual con una vecina a cada lado y
 * «…» donde faltan al menos dos páginas; con 7 o menos, todas. Las flechas ‹ › las pone el componente.
 */
export type Casilla = { tipo: "pagina"; numero: number } | { tipo: "hueco"; lado: "izquierda" | "derecha" };

/** Cuántas casillas se muestran con más de 7 páginas: siempre las mismas. */
export const CASILLAS = 7;

export function paginasVisibles(actual: number, total: number): Casilla[] {
  if (total < 1) return [];
  const p = Math.min(Math.max(1, actual), total);
  const pag = (numero: number): Casilla => ({ tipo: "pagina", numero });
  const rango = (a: number, b: number) => Array.from({ length: b - a + 1 }, (_, i) => pag(a + i));
  if (total <= CASILLAS) return rango(1, total);
  const izq: Casilla = { tipo: "hueco", lado: "izquierda" };
  const der: Casilla = { tipo: "hueco", lado: "derecha" };
  if (p <= 4) return [...rango(1, 5), der, pag(total)];
  if (p >= total - 3) return [pag(1), izq, ...rango(total - 4, total)];
  return [pag(1), izq, pag(p - 1), pag(p), pag(p + 1), der, pag(total)];
}

/** Ancho de casilla según las cifras de la última página, nunca menos de 28 px. */
export function anchoCasilla(paginas: number): string {
  return `max(1.75rem, calc(${String(Math.max(1, paginas)).length}ch + 0.5rem))`;
}

/** Cuántas páginas hacen falta para `elementos` de a `porPagina` (al menos una). */
export function cuantasPaginas(elementos: number, porPagina: number): number {
  return Math.max(1, Math.ceil(elementos / porPagina));
}

/** La página dentro de [1, total]. */
export const acotar = (pagina: number, total: number) => Math.min(Math.max(1, Math.round(pagina) || 1), Math.max(1, total));

/**
 * Lo que se ve de una lista viva cuya primera página es la de lo nuevo (lo más reciente arriba, como la Auditoría).
 * En la página 1 la lista es la de ahora: lo que llega se ve. En otra página, el orden se congela en `foto` (las
 * claves tal como estaban al salir de la 1): lo que llega no corre las filas ni te cambia de página, y cada fila se
 * pinta con su dato de ahora (por clave). Lo que llegó después de la foto se cuenta en `nuevos`. Lo que ya no está, sale.
 */
export function ventanaViva<T>(
  elementos: readonly T[],
  clave: (e: T) => string,
  pagina: number,
  porPagina: number,
  foto: readonly string[] | null,
): { visibles: T[]; desde: number; hasta: number; total: number; paginas: number; pagina: number; nuevos: number } {
  // Defensa: una clave repetida no multiplica filas (cada clave sale una vez, la primera). Con claves repetidas la foto
  // devolvía el mismo elemento por cada aparición y React, con dos hijos de igual `key`, dejaba nodos huérfanos.
  const unicos = sinRepetir(elementos, clave);
  let orden: T[] = unicos;
  let nuevos = 0;
  if (pagina > 1 && foto) {
    const porClave = new Map(unicos.map((e) => [clave(e), e]));
    const enFoto = new Set(foto);
    orden = [...enFoto].map((k) => porClave.get(k)).filter((e): e is T => e !== undefined);
    nuevos = unicos.reduce((n, e) => n + (enFoto.has(clave(e)) ? 0 : 1), 0);
  }
  const total = orden.length;
  const paginas = cuantasPaginas(total, porPagina);
  const p = acotar(pagina, paginas);
  const inicio = (p - 1) * porPagina;
  const visibles = orden.slice(inicio, inicio + porPagina);
  return { visibles, desde: visibles.length ? inicio + 1 : 0, hasta: inicio + visibles.length, total, paginas, pagina: p, nuevos };
}

/** Cada clave una vez, en el orden de llegada (se queda la primera). Sin repetidas devuelve la misma lista. */
export function sinRepetir<T>(elementos: readonly T[], clave: (e: T) => string): T[] {
  const vistas = new Set<string>();
  const out = elementos.filter((e) => {
    const k = clave(e);
    if (vistas.has(k)) return false;
    vistas.add(k);
    return true;
  });
  return out.length === elementos.length ? (elementos as T[]) : out;
}

/**
 * Cuántas filas fantasma completan una página corta: con varias páginas, toda página mide `porPagina` filas y el pie no
 * se mueve nunca (ni en la última, ni en la primera de la actividad, ni mientras carga). Con una sola, nada.
 */
export function relleno(visibles: number, porPagina: number, paginas: number): number {
  return paginas > 1 ? Math.max(0, porPagina - visibles) : 0;
}

/**
 * Páginas ancladas a una marca, para una lista que crece por el final (la actividad: lo viejo arriba, lo nuevo abajo).
 * `ancla` es cuántos elementos había al abrir (o al volver a la última página): la última página empieza `porPagina`
 * antes de la marca y se queda con todo lo que llegó después (crece por abajo, como hasta ahora, sin correr nada); las
 * anteriores van de a `porPagina` hacia atrás y la primera es la corta, la de lo más viejo. Así la página que se ve al
 * abrir siempre está llena, y lo que llega nunca cambia lo que muestra otra página.
 */
export function ventanaAnclada(
  total: number,
  ancla: number,
  porPagina: number,
  pagina: number,
): { inicio: number; fin: number; paginas: number; pagina: number } {
  const base = Math.min(Math.max(0, ancla), total);
  const paginas = cuantasPaginas(base, porPagina);
  const p = acotar(pagina, paginas);
  const inicio = Math.max(0, base - (paginas - p + 1) * porPagina);
  const fin = p === paginas ? total : base - (paginas - p) * porPagina;
  return { inicio, fin, paginas, pagina: p };
}

/** La página (desde 1) de la ventana anclada que contiene la posición `i` (desde 0). */
export function paginaDe(i: number, total: number, ancla: number, porPagina: number): number {
  const base = Math.min(Math.max(0, ancla), total);
  const paginas = cuantasPaginas(base, porPagina);
  if (i >= base - porPagina) return paginas;
  return Math.max(1, paginas - Math.ceil((base - porPagina - i) / porPagina));
}
