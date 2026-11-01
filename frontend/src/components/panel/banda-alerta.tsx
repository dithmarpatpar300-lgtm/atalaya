"use client";

import { useEffect, useState } from "react";
import { BandaTarjeta } from "@/components/ui/tarjeta-alerta";
import { marcasDe } from "@/lib/alertas";
import type { Alerta, Regla } from "@/lib/api";

/**
 * Una regla rota que no espera (un error sin ver: la frontera de archivos o un agente que pasó su presupuesto),
 * flotando bajo la cabecera. Flota y no empuja la página (lo que cambia en vivo no mueve el diseño). Tiene la misma
 * geometría que la tarjeta del centro de alertas: el icono en la primera línea del título, el filo y el contorno del
 * nivel, y el texto en una línea.
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
      className="aparecer-menu fixed left-1/2 z-20 w-[min(60rem,calc(100vw-1.5rem))] -translate-x-1/2 rounded-panel shadow-[0_12px_32px_rgb(0_0_0/0.25)]"
    >
      <BandaTarjeta alerta={alerta} marcas={marcasDe(alerta, reglas)} alVer={alVer} alVisto={alVisto} />
    </div>
  );
}
