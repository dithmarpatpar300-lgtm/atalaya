"use client";

import { useState } from "react";
import { t } from "@/lib/i18n";
import { paginasVisibles, relleno, sinRepetir, ventanaViva } from "@/lib/paginas";
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
        {/* De ancho fijo, el del contador más largo («509–509 de 509»): «1–16» y «17–32» no cambian el ancho y el pie de
            una tarjeta angosta no pasa a dos líneas en unas páginas sí y en otras no. */}
        <span style={{ minWidth: `${t("{desde}–{hasta} de {total}", { desde: total, hasta: total, total }).length}ch` }}>
          {total > 0 ? t("{desde}–{hasta} de {total}", { desde, hasta, total }) : ""}
        </span>
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
        // Ancho reservado para la ventana más ancha de números (a lo más 9 casillas y las flechas): pasar de «1 2 3 … 99» a
        // «1 … 4 5 6 7 8 … 99» no ensancha el pie ni lo hace saltar de línea.
        <nav aria-label={t("Páginas de {lista}", { lista: nombre })} className="flex h-8 items-center justify-end" style={{ minWidth: `${(Math.min(paginas, 9) + 2) * 30}px` }}>
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
 * Filas fantasma: completan una página corta hasta su tamaño, con la misma geometría que las reales (`className` da el
 * alto y el margen), tenues, fuera del lector de pantalla y sin foco. Así toda página mide lo mismo sin medir nada: el
 * pie no se mueve nunca, tampoco en la última página ni mientras llegan datos en vivo (antes se reservaba el alto más
 * grande visto, y una página con más alto le dejaba un hueco a las demás).
 */
export function Fantasmas({ n, className, como = "div" }: { n: number; className: string; como?: "div" | "li" }) {
  const Etiqueta = como;
  return Array.from({ length: n }, (_, i) => <Etiqueta key={`\u0000fantasma-${i}`} aria-hidden="true" data-fantasma="" className={`${className} opacity-40`} />);
}

/** Lo mismo en una tabla: una fila de `columnas` celdas vacías del alto de las reales. */
export function FantasmasDeTabla({ n, columnas, className }: { n: number; columnas: number; className: string }) {
  return Array.from({ length: n }, (_, i) => (
    <tr key={`\u0000fantasma-${i}`} aria-hidden="true" data-fantasma="" className="border-t border-borde">
      <td colSpan={columnas} className="p-0">
        <div className={`${className} opacity-40`} />
      </td>
    </tr>
  ));
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
  if (estado.reinicio !== reinicio) setEstado(e);
  const v = ventanaViva(elementos, clave, e.pagina, tamano, e.foto);

  function alIr(n: number) {
    // Al salir de la 1 se congela el orden de ahora; al volver a ella, se suelta.
    setEstado({ reinicio, pagina: n, foto: n === 1 ? null : (e.foto ?? sinRepetir(elementos, clave).map(clave)) });
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
  const mostrar = elementos.length > Math.min(...opciones);
  // Las fantasma solo con el pie a la vista: sin pie no hay páginas que cambiar.
  return { visibles: v.visibles, pie, mostrar, faltan: mostrar ? relleno(v.visibles.length, tamano, v.paginas) : 0 };
}
