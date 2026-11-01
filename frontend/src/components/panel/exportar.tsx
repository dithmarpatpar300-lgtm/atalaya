"use client";

import { useEffect, useRef, useState } from "react";
import { IconoCerrar } from "@/components/ui/iconos";
import { api, type OpcionesExportar } from "@/lib/api";

const SECCIONES: { id: keyof Omit<OpcionesExportar, "salidas">; nombre: string; nota: string }[] = [
  { id: "encargo", nombre: "Encargo", nota: "la asignación y el texto que recibió" },
  { id: "resumen", nombre: "Resumen y entregables", nota: "estado, bitácora, informe y acta" },
  { id: "archivos", nombre: "Archivos tocados", nota: "con sus líneas sumadas y quitadas" },
  { id: "pasos", nombre: "Pasos", nota: "uno por línea, con su hora y su tipo" },
  { id: "diffs", nombre: "Diffs completos", nota: "cada edición como bloque diff" },
  { id: "tokens", nombre: "Tokens y contexto", nota: "totales, caché y el contexto al final" },
];

const SALIDAS: { id: OpcionesExportar["salidas"]; nombre: string }[] = [
  { id: "recortadas", nombre: "Recortadas a 40 líneas" },
  { id: "completas", nombre: "Completas" },
  { id: "ninguna", nombre: "Sin salidas" },
];

const CLAVE = "atalaya:exportar";
const POR_OMISION: OpcionesExportar = { encargo: true, resumen: true, archivos: true, pasos: true, diffs: false, tokens: true, salidas: "recortadas" };

function leerOpciones(): OpcionesExportar {
  try {
    return { ...POR_OMISION, ...(JSON.parse(localStorage.getItem(CLAVE) ?? "{}") as Partial<OpcionesExportar>) };
  } catch {
    return POR_OMISION;
  }
}

const kb = (n: number) => `${(n / 1024).toFixed(1).replace(".", ",")} KB`;

/**
 * Exportar un agente como Markdown: el servidor lo arma con las secciones
 * elegidas y los secretos tapados; aquí se ve, se copia o se descarga. El panel no escribe nada en el disco.
 */
export function Exportar({ id, nombre, alCerrar }: { id: string; nombre: string; alCerrar: () => void }) {
  const dialogo = useRef<HTMLDialogElement>(null);
  const [op, setOp] = useState<OpcionesExportar>(POR_OMISION);
  const [md, setMd] = useState<{ nombre: string; archivo: string; markdown: string } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [copiado, setCopiado] = useState(false);

  useEffect(() => {
    dialogo.current?.showModal();
    // eslint-disable-next-line react-hooks/set-state-in-effect -- las opciones guardadas se leen después de hidratar
    setOp(leerOpciones());
  }, []);

  useEffect(() => {
    const c = new AbortController();
    api
      .exportar(id, op, c.signal)
      .then((r) => {
        setMd(r);
        setError(null);
      })
      .catch((e: unknown) => {
        if (!(e instanceof DOMException && e.name === "AbortError")) setError(e instanceof Error ? e.message : "no se pudo armar");
      });
    return () => c.abort();
  }, [id, op]);

  function cambiar(o: OpcionesExportar) {
    setOp(o);
    setCopiado(false);
    try {
      localStorage.setItem(CLAVE, JSON.stringify(o));
    } catch {}
  }

  async function copiar() {
    if (!md) return;
    try {
      await navigator.clipboard.writeText(md.markdown);
      setCopiado(true);
    } catch {
      setError("El navegador no dejó copiar; usa Descargar.");
    }
  }

  function descargar() {
    if (!md) return;
    const url = URL.createObjectURL(new Blob([md.markdown], { type: "text/markdown;charset=utf-8" }));
    const a = document.createElement("a");
    a.href = url;
    a.download = md.archivo;
    a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }

  const bytes = md ? new TextEncoder().encode(md.markdown).length : 0;

  return (
    <dialog
      ref={dialogo}
      onClose={alCerrar}
      aria-labelledby="exportar-titulo"
      className="capa m-auto w-[min(66rem,calc(100vw-2rem))] rounded-tarjeta p-0 text-texto backdrop:bg-velo"
    >
      <div className="flex items-center gap-3 border-b border-borde px-5 py-3.5">
        <div className="flex min-w-0 flex-1 flex-col">
          <h2 id="exportar-titulo" className="m-0 truncate text-encabezado font-semibold">
            Exportar {md?.nombre ?? nombre}
          </h2>
          <span className="text-ayuda text-suave">Un Markdown para adjuntar a la trazabilidad.</span>
        </div>
        <button type="button" onClick={() => dialogo.current?.close()} aria-label="Cerrar" className="grid size-9 place-items-center rounded-control hover:bg-elevado">
          <IconoCerrar className="size-4" />
        </button>
      </div>

      <div className="flex flex-wrap">
        <div className="flex flex-[1_1_18rem] flex-col gap-4 border-borde p-5 md:border-r">
          <fieldset className="m-0 flex flex-col gap-1 border-0 p-0">
            <legend className="pb-1.5 font-semibold">Qué incluir</legend>
            {SECCIONES.map((s) => (
              <label key={s.id} className={`flex cursor-pointer items-start gap-2.5 rounded-control px-2.5 py-2 ${op[s.id] ? "bg-seleccion" : "hover:bg-elevado/60"}`}>
                <input type="checkbox" checked={op[s.id]} onChange={() => cambiar({ ...op, [s.id]: !op[s.id] })} className="mt-0.5 size-4 accent-[var(--acento)]" />
                <span className="flex flex-col">
                  <span className="font-medium">{s.nombre}</span>
                  <span className="text-ayuda text-suave">{s.nota}</span>
                </span>
              </label>
            ))}
          </fieldset>
          <fieldset className="m-0 flex flex-col gap-0.5 border-0 p-0" disabled={!op.pasos}>
            <legend className="pb-1.5 font-semibold">Salidas de las órdenes</legend>
            {SALIDAS.map((s) => (
              <label key={s.id} className={`flex cursor-pointer items-center gap-2.5 rounded-control px-2.5 py-1.5 ${op.pasos ? "" : "text-tenue"}`}>
                <input type="radio" name="salidas" checked={op.salidas === s.id} onChange={() => cambiar({ ...op, salidas: s.id })} className="size-4 accent-[var(--acento)]" />
                {s.nombre}
              </label>
            ))}
          </fieldset>
        </div>

        <div className="flex min-w-0 flex-[999_1_30rem] flex-col gap-2 p-5">
          <span className="flex h-5 items-center gap-2.5">
            <span className="font-semibold">Vista previa</span>
            <span className="dato truncate text-ayuda text-suave">{md ? `${md.archivo} · ${kb(bytes)}` : ""}</span>
          </span>
          <pre className="dato scroll-interno m-0 h-[min(29rem,55vh)] overflow-auto rounded-control border border-borde bg-fondo px-3.5 py-3 text-[12.5px] leading-relaxed [overflow-wrap:anywhere] whitespace-pre-wrap">
            {error ? <span className="text-error">No se pudo armar: {error}</span> : md ? md.markdown : "Armando…"}
          </pre>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2.5 border-t border-borde px-5 py-3.5">
        <span className="flex-[1_1_20rem] text-ayuda text-suave">Los secretos se tapan con la misma regla del panel. El panel no escribe nada en el disco.</span>
        <span role="status" className="w-20 text-right text-ayuda text-ok">
          {copiado ? "Copiado" : ""}
        </span>
        <button type="button" disabled={!md} onClick={copiar} className="h-9 rounded-control border border-borde-fuerte px-3.5 font-medium hover:bg-elevado disabled:text-tenue">
          Copiar
        </button>
        <button
          type="button"
          disabled={!md}
          onClick={descargar}
          className="h-9 rounded-control bg-acento px-4 font-medium text-sobre-acento hover:bg-acento-fuerte disabled:bg-elevado disabled:text-tenue"
        >
          Descargar .md
        </button>
      </div>
    </dialog>
  );
}
