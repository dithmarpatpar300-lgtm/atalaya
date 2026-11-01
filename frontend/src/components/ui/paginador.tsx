"use client";

import { type ReactNode, useEffect, useRef, useState } from "react";
import { t } from "@/lib/i18n";
import { paginasVisibles, ventanaViva } from "@/lib/paginas";
import { IconoChevron } from "./iconos";
import { Segmentado } from "./segmentado";

/** Casillas del mismo tamaño: números y «…» se alinean en una rejilla. */
const CASILLA = "dato inline-grid h-7 min-w-7 place-items-center rounded-control px-1.5 text-ayuda transition-colors duration-150";
const FLECHA = "border border-borde text-texto not-disabled:hover:bg-elevado disabled:cursor-not-allowed disabled:text-tenue disabled:opacity-50";

export type PorPagina = { valor: number; opciones: readonly number[]; alCambiar: (n: number) => void };

/**
 * El pie de una lista por páginas. A la izquierda, qué se ve («26–50 de 509»); a la derecha, el tamaño de la página
 * (si hay `porPagina`) y los números entre las flechas ‹ › (lib/paginas.ts). Con una sola página no hay números. Si
 * llegó algo mientras mirabas otra página, `nuevos` lo dice y lleva a la primera.
 */
export function Paginador({
  pagina,
  paginas,
  desde,
  hasta,
  total,
  alIr,
  porPagina,
  nuevos = 0,
  alVerNuevos,
  nombre,
  arriba = false,
}: {
  /** Encima de la lista en vez de al pie (la raya va abajo). */
  arriba?: boolean;
  /** Desde 1. */
  pagina: number;
  paginas: number;
  /** Posiciones visibles, desde 1 (0 = no hay nada). */
  desde: number;
  hasta: number;
  total: number;
  alIr: (pagina: number) => void;
  porPagina?: PorPagina;
  nuevos?: number;
  alVerNuevos?: () => void;
  /** Qué lista es, para el lector de pantalla («Páginas de El equipo que lanzó»). */
  nombre: string;
}) {
  return (
    <div className={`flex flex-wrap items-center gap-x-4 gap-y-2 border-borde ${arriba ? "mb-2 border-b pb-2.5" : "border-t pt-2.5"}`}>
      {/* Cada parte del renglón mide 32 px (lo del selector «Por página»): con o sin él, todo pie mide lo mismo y dos pies
          vecinos quedan con la raya a la misma altura. */}
      <p className="dato m-0 mr-auto flex h-8 items-center gap-2 text-ayuda whitespace-nowrap text-suave" aria-live="polite">
        {total > 0 ? t("{desde}–{hasta} de {total}", { desde, hasta, total }) : ""}
        {nuevos > 0 && alVerNuevos && (
          <button type="button" onClick={alVerNuevos} className="rounded-chip bg-acento-suave px-2 text-acento-fuerte hover:bg-acento hover:text-sobre-acento">
            {nuevos === 1 ? t("1 nuevo") : t("{n} nuevos", { n: nuevos })}
          </button>
        )}
      </p>

      {porPagina && porPagina.opciones.length > 1 && (
        <div className="flex h-8 items-center gap-2">
          <span aria-hidden="true" className="text-ayuda whitespace-nowrap text-tenue">
            {t("Por página")}
          </span>
          <div className="dato" style={{ width: `${porPagina.opciones.length * 2.75}rem` }}>
            <Segmentado
              etiqueta={t("Por página")}
              opciones={porPagina.opciones.map((n) => ({ valor: String(n), nombre: String(n) }))}
              valor={String(porPagina.valor)}
              alCambiar={(v) => Number(v) !== porPagina.valor && porPagina.alCambiar(Number(v))}
            />
          </div>
        </div>
      )}

      {paginas > 1 && (
        <nav aria-label={t("Páginas de {lista}", { lista: nombre })} className="flex h-8 items-center">
          <ul className="m-0 flex list-none items-center gap-0.5 p-0">
            <li>
              <button type="button" aria-label={t("Página anterior")} disabled={pagina <= 1} onClick={() => alIr(pagina - 1)} className={`${CASILLA} ${FLECHA}`}>
                <IconoChevron className="size-3.5 rotate-180" />
              </button>
            </li>
            {paginasVisibles(pagina, paginas).map((c) =>
              c.tipo === "pagina" ? (
                <li key={c.numero}>
                  <button
                    type="button"
                    aria-current={c.numero === pagina ? "page" : undefined}
                    aria-label={t("Página {n}", { n: c.numero })}
                    onClick={() => c.numero !== pagina && alIr(c.numero)}
                    className={`${CASILLA} ${c.numero === pagina ? "bg-acento font-semibold text-sobre-acento" : "text-suave hover:bg-elevado hover:text-texto"}`}
                  >
                    {c.numero}
                  </button>
                </li>
              ) : (
                <li key={c.lado} aria-hidden="true" className={`${CASILLA} text-tenue`}>
                  …
                </li>
              ),
            )}
            <li>
              <button type="button" aria-label={t("Página siguiente")} disabled={pagina >= paginas} onClick={() => alIr(pagina + 1)} className={`${CASILLA} ${FLECHA}`}>
                <IconoChevron className="size-3.5" />
              </button>
            </li>
          </ul>
        </nav>
      )}
    </div>
  );
}

/**
 * Guarda el alto más grande que tuvo la lista mientras hay varias páginas: la última página, más corta, no sube el pie
 * ni lo que hay debajo; cambiar de página no mueve la página. Se olvida al cambiar el ancho, el tamaño de página o la
 * lista (`clave`), y con una sola página no reserva nada.
 */
export function RegionEstable({ activa, clave, className, children }: { activa: boolean; clave: string; className?: string; children: ReactNode }) {
  const interior = useRef<HTMLDivElement>(null);
  const [m, setM] = useState({ clave: "", ancho: 0, alto: 0 });
  useEffect(() => {
    const el = interior.current;
    if (!el || !activa) return;
    const ro = new ResizeObserver(() => {
      const ancho = Math.round(el.offsetWidth);
      const alto = el.offsetHeight;
      setM((p) => (p.clave === clave && p.ancho === ancho ? (alto > p.alto ? { clave, ancho, alto } : p) : { clave, ancho, alto }));
    });
    ro.observe(el);
    return () => ro.disconnect();
  }, [activa, clave]);
  return (
    <div className={className} style={{ minHeight: activa && m.clave === clave ? m.alto : undefined }}>
      {/* flow-root: el margen de la primera fila queda dentro de lo medido. */}
      <div ref={interior} className="flow-root">
        {children}
      </div>
    </div>
  );
}

function leerTamano(almacen: string, opciones: readonly number[], defecto: number): number {
  try {
    const n = Number(localStorage.getItem(`atalaya:por-pagina:${almacen}`));
    return opciones.includes(n) ? n : defecto;
  } catch {
    return defecto;
  }
}

/**
 * Una lista viva por páginas, con lo nuevo en la página 1 (lib/paginas.ts, ventanaViva). El tamaño de página se
 * recuerda en este navegador por lista (`almacen`); la página vuelve a la 1 al cambiar `reinicio` (otra sesión) o el
 * tamaño. Devuelve lo visible y las propiedades del pie; `mostrar` es falso mientras la lista quepa en la página más
 * chica (sin pie, nada que paginar).
 */
export function usePaginaViva<T>(
  elementos: readonly T[],
  clave: (e: T) => string,
  { almacen, opciones, defecto, reinicio }: { almacen: string; opciones: readonly number[]; defecto: number; reinicio: string },
) {
  const [tamano, setTamano] = useState(() => leerTamano(almacen, opciones, defecto));
  const [estado, setEstado] = useState<{ reinicio: string; pagina: number; foto: string[] | null }>({ reinicio, pagina: 1, foto: null });
  const e = estado.reinicio === reinicio ? estado : { reinicio, pagina: 1, foto: null };
  const v = ventanaViva(elementos, clave, e.pagina, tamano, e.foto);

  function alIr(n: number) {
    // Al salir de la 1 se congela el orden de ahora; al volver a ella, se suelta.
    setEstado({ reinicio, pagina: n, foto: n === 1 ? null : (e.foto ?? elementos.map(clave)) });
  }

  const pie = {
    pagina: v.pagina,
    paginas: v.paginas,
    desde: v.desde,
    hasta: v.hasta,
    total: v.total,
    nuevos: v.nuevos,
    alIr,
    alVerNuevos: () => alIr(1),
    porPagina: {
      valor: tamano,
      opciones,
      alCambiar: (n: number) => {
        setTamano(n);
        try {
          localStorage.setItem(`atalaya:por-pagina:${almacen}`, String(n));
        } catch {}
        setEstado({ reinicio, pagina: 1, foto: null });
      },
    },
  };
  return { visibles: v.visibles, pie, mostrar: elementos.length > Math.min(...opciones), claveAlto: `${reinicio}:${tamano}` };
}
