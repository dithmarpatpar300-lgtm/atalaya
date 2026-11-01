// La diferencia entre dos versiones de un archivo de texto, para la vista previa de la configuración
// (components/panel/configurar.tsx).

export type LineaDiferencia = { signo: "+" | "−" | " "; texto: string };

/**
 * Las líneas que se quitan (−) y se agregan (+), cada una en su lugar: la subsecuencia común más larga, como un diff.
 * Las líneas en blanco no cuentan. Los archivos que pasan por aquí son chicos (un registro, una configuración).
 */
export function diferencia(antes: string, despues: string): LineaDiferencia[] {
  const a = antes.split("\n").filter((l) => l.trim());
  const d = despues.split("\n").filter((l) => l.trim());
  // largo[i][j]: la subsecuencia común más larga entre a[i:] y d[j:].
  const largo = Array.from({ length: a.length + 1 }, () => new Array<number>(d.length + 1).fill(0));
  for (let i = a.length - 1; i >= 0; i--) {
    for (let j = d.length - 1; j >= 0; j--) {
      largo[i][j] = a[i] === d[j] ? largo[i + 1][j + 1] + 1 : Math.max(largo[i + 1][j], largo[i][j + 1]);
    }
  }
  const out: LineaDiferencia[] = [];
  let i = 0;
  let j = 0;
  while (i < a.length || j < d.length) {
    if (i < a.length && j < d.length && a[i] === d[j]) {
      out.push({ signo: " ", texto: a[i] });
      i++;
      j++;
    } else if (i < a.length && (j === d.length || largo[i + 1][j] >= largo[i][j + 1])) {
      // En un empate, primero lo que se quita y después lo que se agrega, como lo muestra git.
      out.push({ signo: "−", texto: a[i++] });
    } else {
      out.push({ signo: "+", texto: d[j++] });
    }
  }
  return out;
}
