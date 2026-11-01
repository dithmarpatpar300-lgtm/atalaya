"use client";

import { useEffect, useState } from "react";
import { BarraAlerta, CabeceraAlerta, CarrilNivel, fuenteDe } from "@/components/ui/tarjeta-alerta";
import { COLOR, marcasDe } from "@/lib/alertas";
import type { Alerta, Regla } from "@/lib/api";

/**
 * Una regla rota que no espera (un error sin ver: la frontera de archivos o un agente que pasó su presupuesto),
 * flotando bajo la cabecera. Flota y no empuja la página: lo que cambia en vivo no mueve el diseño (lo que cambia en vivo no mueve el diseño). Tiene
 * la misma forma que la tarjeta del centro de alertas: el carril de nivel a la izquierda y el texto en una línea.
 */
export function BandaAlerta({
  alerta,
  reglas,
  alVer,
  alVisto,
}: {
  alerta: Alerta;
  reglas: readonly Regla[];
  alVer: () => void;
  alVisto: () => void;
}) {
  const c = COLOR[alerta.nivel];
  const linea = `${alerta.texto} · ${fuenteDe(alerta)}`;
  // Bajo la cabecera, que en un ancho de teléfono se parte en dos filas y pasa de 60 px: se mide y se sigue.
  const [top, setTop] = useState(60);
  useEffect(() => {
    const cab = document.querySelector("header");
    if (!cab) return;
    const medir = () => setTop(Math.round(cab.getBoundingClientRect().height) + 8);
    medir();
    const ro = new ResizeObserver(medir);
    ro.observe(cab);
    return () => ro.disconnect();
  }, []);
  return (
    <div
      role="alert"
      style={{ top }}
      className={`aparecer-menu fixed left-1/2 z-20 flex w-[min(60rem,calc(100vw-1.5rem))] -translate-x-1/2 overflow-hidden rounded-panel border bg-superficie shadow-[0_12px_32px_rgb(0_0_0/0.25)] ${c.borde}`}
    >
      <CarrilNivel nivel={alerta.nivel} centrado />
      <div className="flex min-w-0 flex-1 flex-wrap items-center gap-x-5 gap-y-2 px-3.5 py-2.5">
        <div className="flex min-w-0 flex-[1_1_20rem] flex-col gap-0.5">
          <CabeceraAlerta alerta={alerta} />
          <p className="m-0 h-[1.125rem] truncate text-rotulo" title={linea}>
            {alerta.texto}
            <span className="text-tenue"> · {fuenteDe(alerta)}</span>
          </p>
        </div>
        {alerta.max > 0 && (
          <div className="w-[min(14rem,100%)] shrink-0">
            <BarraAlerta alerta={alerta} marcas={marcasDe(alerta, reglas)} detalle={false} />
          </div>
        )}
        <span className="ml-auto flex shrink-0 gap-2">
          <button type="button" onClick={alVer} className={`h-8 rounded-control border px-3 text-rotulo whitespace-nowrap hover:bg-elevado ${c.borde}`}>
            Ver el agente
          </button>
          <button type="button" onClick={alVisto} className={`h-8 rounded-control px-3 text-rotulo font-medium hover:opacity-90 ${alerta.nivel === "info" ? "bg-elevado text-texto" : `${c.barra} text-superficie`}`}>
            Visto
          </button>
        </span>
      </div>
    </div>
  );
}
