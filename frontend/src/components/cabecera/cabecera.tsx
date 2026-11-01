"use client";

import type { ReactNode } from "react";
import { IconoBuscar } from "@/components/ui/iconos";
import { Marca } from "@/components/ui/marca";
import type { ProyectoResumen } from "@/lib/api";
import type { Conexion } from "@/lib/vivo";
import { MenuTema } from "./menu-tema";
import { SelectorProyecto } from "./selector-proyecto";

export type Seccion = "proyectos" | "decisiones" | "agentes" | "linea" | "avance" | "salud" | "estadisticas" | "datapedia" | "consultas";

const TEXTO_CONEXION: Record<Conexion, string> = { conectando: "Conectando", "en vivo": "En vivo", reconectando: "Reconectando" };

/** Ctrl en Windows y Linux, ⌘ en Mac; al prerenderizar, Ctrl. */
function teclaMod(): string {
  if (typeof navigator === "undefined") return "Ctrl";
  return /Mac|iPhone|iPad/.test(navigator.platform) ? "⌘" : "Ctrl";
}

/**
 * La cabecera de una sola fila (52 px), de vidrio: la marca, el proyecto como migas, las secciones como pestañas con
 * subrayado, la puerta a la paleta de órdenes, la luz de la conexión en vivo y el tema. En un teléfono las pestañas
 * bajan a una segunda fila y la búsqueda queda en su icono.
 */
export function Cabecera({
  proyectos,
  proyecto,
  alElegirProyecto,
  conexion,
  seccion,
  cuentas,
  alAbrirPaleta,
  campana,
}: {
  proyectos: ProyectoResumen[];
  proyecto: string;
  alElegirProyecto: (p: string) => void;
  conexion: Conexion;
  seccion: Seccion;
  cuentas: Record<Seccion, number | null>;
  alAbrirPaleta: () => void;
  // La campana de las alertas, junto al tema.
  campana: ReactNode;
}) {
  const mod = teclaMod();
  const pestanas: { id: Seccion; nombre: string }[] = [
    { id: "proyectos", nombre: "Proyectos" },
    { id: "decisiones", nombre: "Decisiones" },
    { id: "agentes", nombre: "Agentes" },
    { id: "linea", nombre: "Línea" },
    { id: "avance", nombre: "Avance" },
    { id: "salud", nombre: "Salud" },
    { id: "estadisticas", nombre: "Estadísticas" },
    { id: "datapedia", nombre: "Datapedia" },
    { id: "consultas", nombre: "Consultas" },
  ];
  return (
    <header className="vidrio sticky top-0 z-30 border-b border-borde">
      <div className="flex min-h-[52px] flex-wrap items-center gap-x-2.5 gap-y-1 px-4 pt-2 sm:gap-x-3.5 sm:px-5 xl:flex-nowrap xl:pt-0">
        <span className="flex items-center gap-2.5 text-encabezado font-semibold tracking-[-0.01em] whitespace-nowrap">
          <Marca />
          {/* Entre 1280 y 1800 px queda el logo: las nueve pestañas necesitan el sitio (un navegador con barra lateral, como Zen, deja ~1680 px en una pantalla de 1920). */}
          <span className="max-sm:sr-only xl:max-[1799px]:sr-only">Atalaya</span>
        </span>
        <span aria-hidden="true" className="text-xl font-light text-borde-fuerte">
          /
        </span>
        <SelectorProyecto proyectos={proyectos} valor={proyecto} alCambiar={alElegirProyecto} />

        {/* Desde 1280 px, una sola fila que nunca se parte: si falta sitio, las pestañas se desplazan de lado y los botones de la derecha no se mueven. Más angosto, las pestañas bajan a su propia fila. */}
        <nav aria-label="Secciones" className="order-last -mx-1 flex w-full gap-1 self-stretch overflow-x-auto xl:order-none xl:mx-0 xl:ml-1.5 xl:w-auto xl:min-w-0 xl:shrink xl:[scrollbar-width:none]">
          {pestanas.map((p) => {
            const actual = p.id === seccion;
            return (
              <a
                key={p.id}
                href={`#${p.id}`}
                aria-current={actual ? "page" : undefined}
                className={`relative flex min-h-10 shrink-0 items-center gap-1.5 px-2.5 whitespace-nowrap xl:px-2 min-[1800px]:px-2.5 transition-[color,box-shadow] duration-200 ease-[cubic-bezier(0.34,1.56,0.64,1)] ${
                  actual ? "font-semibold text-texto shadow-[inset_0_-2px_0_var(--acento)]" : "text-suave hover:text-texto"
                }`}
              >
                {p.nombre}
                {/* Lo que te espera se ve siempre, como insignia sobre la esquina: no ocupa sitio en la fila, así la
                    pestaña queda centrada sobre su raya y que aparezca no corre las demás (lo que cambia en vivo no mueve el diseño). */}
                {p.id === "decisiones"
                  ? !!cuentas.decisiones && (
                      <span
                        aria-label={`${cuentas.decisiones} esperándote`}
                        className="dato pointer-events-none absolute top-0.5 right-0 min-w-4 translate-x-1/3 rounded-full bg-aviso px-1 text-center text-[10px] leading-4 font-semibold text-fondo"
                      >
                        {cuentas.decisiones}
                      </span>
                    )
                  : cuentas[p.id] !== null && <span className="dato text-[11px] font-normal text-tenue max-[1799px]:hidden">{cuentas[p.id]}</span>}
              </a>
            );
          })}
        </nav>

        <span className="flex-1" />

        <button
          type="button"
          onClick={alAbrirPaleta}
          aria-haspopup="dialog"
          aria-keyshortcuts={mod === "⌘" ? "Meta+K" : "Control+K"}
          aria-label="Buscar agentes, proyectos, documentos o acciones"
          className="flex h-[34px] items-center gap-2 rounded-control border border-borde bg-elevado/55 px-2.5 text-tenue transition-colors duration-150 hover:border-borde-fuerte hover:text-suave min-[1800px]:w-64 shrink-0"
        >
          <IconoBuscar className="size-[15px] shrink-0" />
          <span className="hidden flex-1 truncate text-left min-[1800px]:inline">Buscar agentes, trazas, documentos</span>
          <kbd className="dato hidden rounded-[5px] border border-borde px-1.5 text-[11px] md:inline">{mod} K</kbd>
        </button>

        <span className="flex items-center gap-2 text-ayuda text-suave" aria-live="polite">
          <span
            aria-hidden="true"
            className={`size-[7px] rounded-full ${
              conexion === "en vivo" ? "brillo bg-ok text-(--brillo-ok)" : conexion === "reconectando" ? "pulso bg-aviso" : "bg-borde-campo"
            }`}
          />
          {/* Entre 1280 y 1800 px las nueve pestañas no dejan sitio: queda la luz, y el texto para el lector de pantalla. */}
          <span className="max-md:sr-only xl:max-[1799px]:sr-only">{TEXTO_CONEXION[conexion]}</span>
        </span>

        {campana}
        <MenuTema />
      </div>
    </header>
  );
}
