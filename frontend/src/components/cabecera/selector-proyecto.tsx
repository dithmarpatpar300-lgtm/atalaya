"use client";

import { type KeyboardEvent, useCallback, useId, useRef, useState } from "react";
import { IconoArribaAbajo, IconoCheck } from "@/components/ui/iconos";
import type { ProyectoResumen } from "@/lib/api";
import { haceCuanto } from "@/lib/formato";
import { useCerrarFuera } from "@/lib/use-fuera";
import { useAhora } from "@/lib/vivo";
import { t } from "@/lib/i18n";

/** «Tienda Demo» → «TD», «Atalaya» → «AT». */
export function iniciales(nombre: string): string {
  const partes = nombre.split(/[\s_-]+/).filter(Boolean);
  return (partes.length > 1 ? partes[0][0] + partes[1][0] : nombre.slice(0, 2)).toUpperCase();
}

// Cada proyecto con su tinta, siempre la misma para el mismo nombre.
const TINTAS = [
  "bg-acento-suave text-acento-fuerte",
  "bg-opcional-suave text-opcional",
  "bg-ok-suave text-ok",
  "bg-aviso-suave text-aviso",
  "bg-elevado text-suave",
];
export function tinta(nombre: string): string {
  let h = 0;
  for (const c of nombre) h = (h * 31 + c.charCodeAt(0)) >>> 0;
  return TINTAS[h % TINTAS.length];
}

function Avatar({ nombre, className = "" }: { nombre: string; className?: string }) {
  return (
    <span aria-hidden="true" className={`grid shrink-0 place-items-center rounded-chip text-[11px] font-bold ${tinta(nombre)} ${className}`}>
      {iniciales(nombre)}
    </span>
  );
}

function subtitulo(p: ProyectoResumen, ahora: number): string {
  const cuantos = p.activos ? (p.activos === 1 ? t("1 activo") : t("{n} activos", { n: p.activos })) : (p.agentes === 1 ? t("1 agente") : t("{n} agentes", { n: p.agentes }));
  return p.ultimo && ahora ? `${cuantos} · ${haceCuanto(p.ultimo, ahora)}` : cuantos;
}

/**
 * El proyecto como migas (`/ Tienda Demo ▾`, como el selector de equipo de Vercel). Abre un buscador con la lista:
 * cada proyecto con su luz (agentes activos), cuántos y hace cuánto trabajaron. Es el patrón «combobox» de la
 * WAI-ARIA: el foco queda en el buscador, las flechas mueven la opción activa (`aria-activedescendant`), Intro elige,
 * Escape cierra y devuelve el foco al botón.
 */
export function SelectorProyecto({
  proyectos,
  valor,
  alCambiar,
}: {
  proyectos: ProyectoResumen[];
  valor: string;
  alCambiar: (p: string) => void;
}) {
  const ahora = useAhora();
  const base = useId();
  const [abierto, setAbierto] = useState(false);
  const [texto, setTexto] = useState("");
  const [activa, setActiva] = useState(0);
  const boton = useRef<HTMLButtonElement>(null);
  const caja = useRef<HTMLDivElement>(null);
  const cerrar = useCallback(() => setAbierto(false), []);
  useCerrarFuera(abierto, caja, boton, cerrar);

  const q = texto.trim().toLowerCase();
  const lista = proyectos.filter((p) => !q || p.nombre.toLowerCase().includes(q));
  const actual = proyectos.find((p) => p.nombre === valor);
  const indice = Math.min(activa, Math.max(0, lista.length - 1));

  function abrir() {
    setTexto("");
    setActiva(Math.max(0, proyectos.findIndex((p) => p.nombre === valor)));
    setAbierto(true);
  }

  function elegir(nombre: string) {
    if (nombre !== valor) alCambiar(nombre);
    setAbierto(false);
    boton.current?.focus();
  }

  function alTecla(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown") setActiva(Math.min(lista.length - 1, indice + 1));
    else if (e.key === "ArrowUp") setActiva(Math.max(0, indice - 1));
    else if (e.key === "Home") setActiva(0);
    else if (e.key === "End") setActiva(lista.length - 1);
    else if (e.key === "Enter" && lista[indice]) elegir(lista[indice].nombre);
    else if (e.key === "Tab") setAbierto(false);
    else return;
    if (e.key !== "Tab") e.preventDefault();
  }

  return (
    <div className="relative">
      <button
        ref={boton}
        type="button"
        onClick={() => (abierto ? setAbierto(false) : abrir())}
        aria-haspopup="dialog"
        aria-expanded={abierto}
        aria-label={t("Proyecto: {nombre}. Cambiar de proyecto", { nombre: valor || t("ninguno") })}
        className={`flex h-[34px] max-w-[11rem] items-center gap-2 rounded-control border pr-2 pl-[5px] transition-colors duration-150 sm:max-w-[18rem] ${
          abierto ? "border-acento bg-elevado" : "border-borde hover:border-borde-fuerte hover:bg-elevado"
        }`}
      >
        {valor && <Avatar nombre={valor} className="size-6" />}
        <span className="truncate font-medium">{valor || t("Proyecto")}</span>
        {actual && actual.activos > 0 && (
          <span className="dato hidden shrink-0 rounded-chip bg-ok-suave px-1.5 text-[11px] text-ok sm:inline xl:max-[1799px]:hidden">
            {actual.activos === 1 ? t("1 activo") : t("{n} activos", { n: actual.activos })}
          </span>
        )}
        <IconoArribaAbajo className="size-3.5 shrink-0 text-suave" />
      </button>

      {abierto && (
        <div ref={caja} role="dialog" aria-label={t("Elegir proyecto")} className="capa aparecer-menu absolute top-[42px] left-0 z-40 w-[22rem] max-w-[calc(100vw-2rem)] rounded-panel p-1.5">
          <label htmlFor={`${base}-q`} className="sr-only">
            {t("Buscar proyecto")}
          </label>
          <input
            id={`${base}-q`}
            autoFocus
            role="combobox"
            aria-expanded="true"
            aria-controls={`${base}-lista`}
            aria-activedescendant={lista[indice] ? `${base}-o${indice}` : undefined}
            aria-autocomplete="list"
            value={texto}
            onChange={(e) => {
              setTexto(e.target.value);
              setActiva(0);
            }}
            onKeyDown={alTecla}
            placeholder={t("Buscar proyecto…")}
            autoComplete="off"
            spellCheck={false}
            className="mb-1 h-9 w-full rounded-control border border-borde bg-elevado px-2.5 text-cuerpo placeholder:text-tenue focus-visible:outline-none"
          />
          <ul id={`${base}-lista`} role="listbox" aria-label={t("Proyectos")} className="scroll-interno m-0 max-h-80 list-none overflow-y-auto p-0">
            {lista.map((p, i) => {
              const elegido = p.nombre === valor;
              return (
                <li
                  key={p.nombre}
                  id={`${base}-o${i}`}
                  role="option"
                  aria-selected={elegido}
                  onPointerMove={() => setActiva(i)}
                  onClick={() => elegir(p.nombre)}
                  className={`grid min-h-12 cursor-pointer grid-cols-[1.625rem_minmax(0,1fr)_auto] items-center gap-x-2.5 rounded-control px-2 py-1.5 ${
                    i === indice ? "bg-elevado" : ""
                  }`}
                >
                  <Avatar nombre={p.nombre} className="size-[26px]" />
                  <span className="flex min-w-0 flex-col">
                    <span className={`truncate ${elegido ? "font-semibold" : "font-medium"}`}>{p.nombre}</span>
                    <span className="truncate text-ayuda text-suave">{subtitulo(p, ahora)}</span>
                  </span>
                  <span className="flex items-center gap-2">
                    <span aria-hidden="true" className={`size-[7px] rounded-full ${p.activos ? "bg-ok" : "bg-borde-fuerte"}`} />
                    <span aria-hidden="true" className="grid size-4 place-items-center text-acento">
                      {elegido && <IconoCheck className="size-4" />}
                    </span>
                  </span>
                </li>
              );
            })}
          </ul>
          {lista.length === 0 && <p className="m-0 px-2 py-3 text-rotulo text-suave">{t("Ningún proyecto se llama así.")}</p>}
          <div className="mt-1 flex justify-between border-t border-borde px-2 pt-2 pb-1 text-ayuda text-tenue">
            <span>
              {proyectos.length === 1 ? t("1 proyecto con agentes") : t("{n} proyectos con agentes", { n: proyectos.length })}
            </span>
            <span className="dato">{t("↑↓ Intro")}</span>
          </div>
        </div>
      )}
    </div>
  );
}
