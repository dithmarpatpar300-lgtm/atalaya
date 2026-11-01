/**
 * La lógica pura del paginador (components/ui/paginador.tsx): la primera y la
 * última página siempre a la vista, la actual con dos a cada lado y «…» como separador que no se pulsa, solo donde
 * quedan dos páginas o más fuera de vista (si queda una sola, se muestra su número). Con 99 páginas y la 6:
 * 1 … 4 5 6 7 8 … 99; con la 3: 1 2 3 4 5 … 99. Las flechas ‹ › las pone el componente.
 */
export type Casilla = { tipo: "pagina"; numero: number } | { tipo: "hueco"; lado: "izquierda" | "derecha" };

export const VECINAS = 2;

export function paginasVisibles(actual: number, total: number): Casilla[] {
  if (total < 1) return [];
  const pagina = Math.min(Math.max(1, actual), total);
  let inicio = Math.max(1, pagina - VECINAS);
  let fin = Math.min(total, pagina + VECINAS);
  if (inicio === 3) inicio = 2;
  if (fin === total - 2) fin = total - 1;

  const casillas: Casilla[] = [];
  if (inicio > 1) casillas.push({ tipo: "pagina", numero: 1 });
  if (inicio > 2) casillas.push({ tipo: "hueco", lado: "izquierda" });
  for (let n = inicio; n <= fin; n++) casillas.push({ tipo: "pagina", numero: n });
  if (fin < total - 1) casillas.push({ tipo: "hueco", lado: "derecha" });
  if (fin < total) casillas.push({ tipo: "pagina", numero: total });
  return casillas;
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
  let orden: T[] = elementos as T[];
  let nuevos = 0;
  if (pagina > 1 && foto) {
    const porClave = new Map(elementos.map((e) => [clave(e), e]));
    const enFoto = new Set(foto);
    orden = foto.map((k) => porClave.get(k)).filter((e): e is T => e !== undefined);
    nuevos = elementos.reduce((n, e) => n + (enFoto.has(clave(e)) ? 0 : 1), 0);
  }
  const total = orden.length;
  const paginas = cuantasPaginas(total, porPagina);
  const p = acotar(pagina, paginas);
  const inicio = (p - 1) * porPagina;
  const visibles = orden.slice(inicio, inicio + porPagina);
  return { visibles, desde: visibles.length ? inicio + 1 : 0, hasta: inicio + visibles.length, total, paginas, pagina: p, nuevos };
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
