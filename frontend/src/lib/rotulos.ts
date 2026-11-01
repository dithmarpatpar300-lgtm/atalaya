/** Un rótulo colocado: `k` es su índice en la lista de entrada; `left` y `max`, en píxeles. */
export type Rotulo = { k: number; left: number; max: number };

/**
 * Reparte los rótulos de una fila (las etapas del gráfico) para que nunca se monten.
 * Recorre de izquierda a derecha y pone uno nuevo solo si queda a `minimo` px del anterior;
 * cada rótulo puede ocupar hasta el siguiente (o el borde) menos `hueco`, y se corta con «…».
 * Los que quedarían con menos de `angosto` px se quitan: su etapa se lee al pasar el puntero.
 */
export function repartirRotulos(xs: number[], ancho: number, minimo = 84, hueco = 8, angosto = 40, tope = 200): Rotulo[] {
  if (ancho <= 0) return [];
  const elegidos: { k: number; left: number }[] = [];
  let ultimo = -Infinity;
  xs.forEach((x, k) => {
    if (x - ultimo < minimo || ancho - x < angosto + hueco) return;
    elegidos.push({ k, left: x });
    ultimo = x;
  });
  return elegidos
    .map((r, j) => {
      const fin = j + 1 < elegidos.length ? elegidos[j + 1].left : ancho;
      return { ...r, max: Math.min(tope, fin - r.left - hueco) };
    })
    .filter((r) => r.max >= angosto);
}
