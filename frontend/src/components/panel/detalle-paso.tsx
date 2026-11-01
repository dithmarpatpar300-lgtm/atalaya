"use client";

import type { ReactNode } from "react";
import { Markdown } from "@/components/ui/markdown";
import type { DetallePaso } from "@/lib/api";
import { Diff } from "./diff";

/**
 * Dos maneras de leer un paso (v2.2: «Truncado» se quitó porque no aportaba):
 * - Estilizado: lo que entiende una persona. El diff con números, la orden con su stdout y su stderr aparte, la
 *   lectura de un archivo con sus números de línea y las respuestas de la IA con su Markdown.
 * - RAW: la entrada y el resultado de la herramienta tal como están en el transcrito (JSON), sin interpretar.
 */
export type Modo = "estilizado" | "crudo";

// Los colores de terminal (ANSI) no se interpretan aquí: se quitan.
const ANSI = new RegExp(String.fromCharCode(27) + "\\[[0-9;?]*[ -/]*[@-~]", "g");
const sinAnsi = (s: string) => s.replace(ANSI, "");

function Bloque({ texto, tono = "normal", rotulo }: { texto: string; tono?: "normal" | "suave" | "error" | "orden"; rotulo?: string }) {
  const color = { normal: "text-texto", suave: "text-suave", error: "text-error bg-error-suave", orden: "text-texto" }[tono];
  return (
    <div className="border-b border-borde last:border-0">
      {rotulo && <div className={`px-3.5 pt-2 text-ayuda font-medium ${tono === "error" ? "text-error" : "text-tenue"}`}>{rotulo}</div>}
      <pre className={`dato scroll-interno m-0 max-h-[32rem] overflow-auto px-3.5 py-2.5 text-ayuda leading-relaxed break-words whitespace-pre-wrap ${color}`}>
        {tono === "orden" && <span className="text-tenue select-none">$ </span>}
        {texto.replace(/\n+$/, "")}
      </pre>
    </div>
  );
}

/** La salida de Read viene como «   12→texto» o «12\ttexto»: el número va a su propia columna. */
function Lectura({ texto }: { texto: string }) {
  const filas = texto.replace(/\n+$/, "").split("\n");
  const numeradas = filas.filter((l) => /^\s*\d+(\t|→)/.test(l)).length > filas.length / 2;
  if (!numeradas) return <Bloque texto={texto} tono="suave" />;
  return (
    <div className="scroll-interno max-h-[32rem] overflow-auto">
      <table className="dato w-full border-collapse text-ayuda leading-relaxed">
        <tbody>
          {filas.map((l, i) => {
            const m = /^\s*(\d+)(?:\t|→)(.*)$/.exec(l);
            return (
              <tr key={i}>
                <td className="w-px border-r border-borde px-2 text-right align-top whitespace-nowrap text-tenue select-none">{m?.[1] ?? ""}</td>
                <td className="px-3 break-all whitespace-pre-wrap text-suave">{m ? m[2] : l}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function Prosa({ texto }: { texto: string }) {
  return (
    <div className="p-3.5 text-rotulo">
      <Markdown texto={texto} ruta="" />
    </div>
  );
}

function Estilizado({ d }: { d: DetallePaso }): ReactNode {
  if (d.tipo === "Responde") return <Prosa texto={d.salida} />;
  if (d.tipo === "Pide") return <Prosa texto={d.orden} />;
  if (d.diff && d.diff.length > 0) {
    return (
      <>
        <Diff archivo={d.archivo} diff={d.diff} />
        {d.error && <Bloque texto={d.salida} tono="error" rotulo="Error" />}
      </>
    );
  }
  if (d.tipo === "Ejecuta") {
    const hayPartes = d.stdout !== undefined || d.stderr !== undefined;
    return (
      <>
        <Bloque texto={d.orden} tono="orden" />
        {d.interrumpido && <p className="m-0 border-b border-borde bg-aviso-suave px-3.5 py-1.5 text-ayuda text-aviso">Interrumpida antes de terminar.</p>}
        {hayPartes ? (
          <>
            {d.stdout ? <Bloque texto={sinAnsi(d.stdout)} tono="suave" rotulo="stdout" /> : null}
            {d.stderr ? <Bloque texto={sinAnsi(d.stderr)} tono="error" rotulo="stderr" /> : null}
            {!d.stdout && !d.stderr && <Bloque texto={sinAnsi(d.salida) || "Sin salida"} tono="suave" />}
          </>
        ) : (
          <Bloque texto={sinAnsi(d.salida) || "Sin salida"} tono={d.error ? "error" : "suave"} />
        )}
      </>
    );
  }
  if (d.tipo === "Lee") {
    return (
      <>
        <Bloque texto={d.orden} />
        <Lectura texto={d.salida} />
      </>
    );
  }
  return (
    <>
      {d.orden && d.orden !== "—" && <Bloque texto={d.orden} />}
      <Bloque texto={sinAnsi(d.salida) || "Sin salida"} tono={d.error ? "error" : "suave"} />
    </>
  );
}

function Crudo({ d }: { d: DetallePaso }) {
  const entrada = d.entrada_cruda ?? (d.tipo === "Pide" ? d.orden : undefined);
  const salida = d.salida_cruda ?? (d.tipo === "Responde" ? d.salida : undefined);
  return (
    <>
      <Bloque texto={entrada ?? d.orden ?? ""} rotulo={d.tipo === "Pide" ? "Mensaje" : "Entrada (input de la herramienta)"} />
      {(salida ?? d.salida) && <Bloque texto={salida ?? d.salida} tono="suave" rotulo={d.tipo === "Responde" ? "Texto" : "Resultado (tool_result)"} />}
    </>
  );
}

export function CuerpoPaso({ d, modo }: { d: DetallePaso; modo: Modo }) {
  return modo === "crudo" ? <Crudo d={d} /> : <Estilizado d={d} />;
}
