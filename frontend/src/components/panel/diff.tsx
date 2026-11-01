"use client";

import type { Trozo } from "@/lib/api";

export function cuentaDiff(diff: Trozo[]): { mas: number; menos: number } {
  let mas = 0;
  let menos = 0;
  for (const t of diff)
    for (const l of t.lineas) {
      if (l.startsWith("+")) mas++;
      else if (l.startsWith("-")) menos++;
    }
  return { mas, menos };
}

interface Fila {
  viejo: number | null;
  nuevo: number | null;
  signo: " " | "+" | "-";
  texto: string;
}

/** Numera las líneas de un trozo: las de contexto avanzan las dos cuentas, las quitadas la vieja, las puestas la nueva. */
function filas(t: Trozo): Fila[] {
  const conNumeros = t.viejo_desde > 0 || t.nuevo_desde > 0;
  let v = t.viejo_desde;
  let n = t.nuevo_desde;
  return t.lineas.map((l) => {
    const signo = (l[0] === "+" || l[0] === "-" ? l[0] : " ") as Fila["signo"];
    const texto = l.slice(1);
    if (!conNumeros) return { viejo: null, nuevo: null, signo, texto };
    if (signo === "+") return { viejo: null, nuevo: n++, signo, texto };
    if (signo === "-") return { viejo: v++, nuevo: null, signo, texto };
    return { viejo: v++, nuevo: n++, signo, texto };
  });
}

const FONDO = { "+": "bg-ok-suave", "-": "bg-error-suave", " ": "" } as const;
const SIGNO = { "+": "text-ok", "-": "text-error", " ": "text-tenue" } as const;

/**
 * Un diff como lo muestra Claude Code: las líneas quitadas en rojo y las puestas en verde, con los números viejos y
 * nuevos a la izquierda. Los números salen del structuredPatch que guarda el transcrito; si no está (un Edit que
 * todavía no termina), el diff se calcula de la llamada y va sin números.
 */
export function Diff({ archivo, diff }: { archivo?: string; diff: Trozo[] }) {
  const { mas, menos } = cuentaDiff(diff);
  const trozos = diff.map((t) => ({ t, fs: filas(t) }));
  return (
    <div className="overflow-hidden">
      {archivo && (
        <div className="flex items-center gap-2 border-b border-borde px-3 py-2 text-ayuda">
          <span className="dato min-w-0 flex-1 truncate text-texto">{archivo}</span>
          <span className="dato shrink-0 rounded-chip bg-ok-suave px-1.5 text-ok">+{mas}</span>
          <span className="dato shrink-0 rounded-chip bg-error-suave px-1.5 text-error">−{menos}</span>
        </div>
      )}
      <div className="scroll-interno max-h-[32rem] overflow-auto">
        <table className="dato w-full border-collapse text-ayuda leading-relaxed">
          {trozos.map(({ t, fs }, i) => (
              <tbody key={i}>
                {(t.viejo_desde > 0 || t.nuevo_desde > 0) && (
                  <tr>
                    <td colSpan={4} className="bg-elevado px-3 py-1 text-tenue">
                      @@ −{t.viejo_desde},{t.viejo_lineas} +{t.nuevo_desde},{t.nuevo_lineas} @@
                    </td>
                  </tr>
                )}
                {fs.map((f, j) => (
                  <tr key={j} className={FONDO[f.signo]}>
                    <td className="w-px border-r border-borde px-2 text-right whitespace-nowrap text-tenue select-none">{f.viejo ?? ""}</td>
                    <td className="w-px border-r border-borde px-2 text-right whitespace-nowrap text-tenue select-none">{f.nuevo ?? ""}</td>
                    <td className={`w-px pl-2 select-none ${SIGNO[f.signo]}`}>{f.signo === " " ? "" : f.signo === "-" ? "−" : "+"}</td>
                    <td className="px-2 break-all whitespace-pre-wrap">{f.texto}</td>
                  </tr>
                ))}
              </tbody>
          ))}
        </table>
      </div>
    </div>
  );
}
