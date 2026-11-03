"use client";

import { useState } from "react";
import type { CeldaMapa, MapaSesion as Mapa } from "@/lib/api";
import { diaYMes, hora } from "@/lib/formato";
import { decimal, t } from "@/lib/i18n";
import { cambiosDeDia, marcoMapa, opacidadCelda } from "@/lib/lotes";

/** Lo que dice una barra, al pasar por encima y en su etiqueta. */
function detalle(c: CeldaMapa): { titulo: string; nota: string } {
  const f = new Date(c.hora);
  const marcas = [t("{pct} % ediciones", { pct: decimal(c.edita * 100, 0) })];
  if (c.pedidos) marcas.push(c.pedidos === 1 ? t("1 pedido") : t("{n} pedidos", { n: c.pedidos }));
  if (c.errores) marcas.push(c.errores === 1 ? t("1 error") : t("{n} errores", { n: c.errores }));
  return { titulo: t("{dia} {hora}, {n} pasos", { dia: diaYMes(f), hora: hora(f), n: c.pasos }), nota: marcas.join(", ") };
}

/**
 * Toda la sesión, de arriba abajo, en barras con el mismo número de pasos: más intensa, más ediciones; a la izquierda,
 * los pedidos; a la derecha, los errores; el marco es lo que se ve en la lista. Un clic lleva a ese tramo. Mide lo que la
 * lista, sin pasar de la ventana: siempre se ve entero. El teclado llega a lo mismo por el paginador.
 */
export function MapaSesion({ mapa, inicio, fin, alto, alIr }: { mapa: Mapa; inicio: number; fin: number; alto: number; alIr: (n: number) => void }) {
  const [sobre, setSobre] = useState<number | null>(null);
  const { celdas, por_celda: porCelda } = mapa;
  const paso = 100 / celdas.length;
  const marco = marcoMapa(inicio, fin, porCelda * celdas.length);
  const activa = sobre !== null ? celdas[sobre] : null;
  const info = activa && detalle(activa);
  return (
    <aside
      aria-label={t("Mapa de toda la sesión: cada barra son {n} pasos", { n: porCelda })}
      className="relative w-[92px] shrink-0 border-l border-borde bg-hundido"
    >
      <div className="sticky top-[68px]" style={{ height: `max(18rem, min(${alto}px, calc(100dvh - 84px)))` }} onMouseLeave={() => setSobre(null)}>
        {cambiosDeDia(celdas).map((i) => (
          <span key={i} aria-hidden="true" className="dato pointer-events-none absolute left-2 text-[10.5px] leading-none text-tenue" style={{ top: `calc(${i * paso}% + 1px)` }}>
            {diaYMes(new Date(celdas[i].hora))}
          </span>
        ))}
        {celdas.map((c, i) => {
          const d = detalle(c);
          return (
            <button
              key={c.n}
              type="button"
              tabIndex={-1}
              aria-label={`${d.titulo}. ${d.nota}`}
              onClick={() => alIr(c.n)}
              onMouseEnter={() => setSobre(i)}
              className="absolute inset-x-0 block"
              style={{ top: `${i * paso}%`, height: `${paso}%` }}
            >
              <span
                className="absolute inset-y-px left-[48px] w-[26px] rounded-[2px] bg-acento"
                style={{ opacity: sobre === i ? 1 : opacidadCelda(c.edita), minHeight: 2 }}
              />
              {c.pedidos > 0 && <span className="absolute inset-y-px left-[41px] w-[5px] rounded-[1px] bg-texto" style={{ minHeight: 2 }} />}
              {c.errores > 0 && <span className="absolute top-1/2 left-[77px] size-[5px] -translate-y-1/2 rounded-full bg-error" />}
            </button>
          );
        })}
        {/* El marco no atrapa el clic: debajo siguen las barras. */}
        <div
          aria-hidden="true"
          className="pointer-events-none absolute left-[37px] w-12 rounded-[5px] border-[1.5px] border-texto"
          style={{ top: `${marco.top}%`, height: `${marco.alto}%`, minHeight: 8 }}
        />
        {info && sobre !== null && (
          <div
            aria-hidden="true"
            className="pointer-events-none absolute right-full z-10 mr-1.5 -translate-y-1/2 rounded-control border border-borde bg-superficie px-2 py-1 text-ayuda whitespace-nowrap shadow-[0_2px_8px_rgb(0_0_0/0.18)]"
            style={{ top: `${Math.min(96, Math.max(4, (sobre + 0.5) * paso))}%` }}
          >
            <div className="dato text-texto">{info.titulo}</div>
            <div className="text-suave">{info.nota}</div>
          </div>
        )}
      </div>
    </aside>
  );
}
