"use client";

import type { ReactNode } from "react";
import { COLOR, detalleBarra, type Marca, NOMBRE_NIVEL, porcentajeDe } from "@/lib/alertas";
import type { Alerta, NivelAlerta } from "@/lib/api";
import { hora } from "@/lib/formato";
import { t, tServidor } from "@/lib/i18n";

/*
 * La tarjeta de una alerta, compartida por el centro de alertas (la campana) y la banda flotante. A la izquierda, el
 * carril de nivel: una columna de ancho fijo a todo el alto, con el fondo suave del nivel y su icono. Lo que cambia en
 * vivo (la hora, el texto, la cifra) tiene su hueco fijo: una línea o dos con puntos suspensivos, el texto entero en
 * `title` y las cifras en columnas de ancho fijo (lo que cambia en vivo no mueve el diseño).
 */

function Trazo({ className, children }: { className?: string; children: ReactNode }) {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      focusable="false"
      className={className}
    >
      {children}
    </svg>
  );
}

/** El icono de cada nivel: error, un octógono (alto); aviso, un triángulo; info, un círculo con la «i». */
export function IconoNivel({ nivel, className = "size-[18px]" }: { nivel: NivelAlerta; className?: string }) {
  if (nivel === "error")
    return (
      <Trazo className={className}>
        <path d="M7.9 2.5h8.2l5.4 5.4v8.2l-5.4 5.4H7.9l-5.4-5.4V7.9z" />
        <path d="M12 7.5v5M12 16.5h.01" />
      </Trazo>
    );
  if (nivel === "aviso")
    return (
      <Trazo className={className}>
        <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0z" />
        <path d="M12 9v4M12 17h.01" />
      </Trazo>
    );
  return (
    <Trazo className={className}>
      <circle cx="12" cy="12" r="9.5" />
      <path d="M12 11v5.5M12 7.5h.01" />
    </Trazo>
  );
}

/** La columna de nivel: ancho fijo, a todo el alto de la tarjeta; la tarjeta la recorta con su radio. */
export function CarrilNivel({ nivel, centrado = false }: { nivel: NivelAlerta; centrado?: boolean }) {
  const c = COLOR[nivel];
  return (
    <div aria-hidden="true" className={`flex w-10 shrink-0 justify-center border-r ${c.carril} ${c.linea} ${centrado ? "items-center" : "items-start pt-[13px]"}`}>
      <span className={c.texto}>
        <IconoNivel nivel={nivel} />
      </span>
    </div>
  );
}

/** El nivel en palabra, con su color. */
export function ChipNivel({ nivel }: { nivel: NivelAlerta }) {
  const c = COLOR[nivel];
  return (
    <span className={`shrink-0 rounded-chip border px-1.5 text-[11px] leading-[16px] font-medium ${c.texto} ${c.fondo} ${c.borde}`}>
      {t(NOMBRE_NIVEL[nivel])}
    </span>
  );
}

/** Regla, nivel, agente y hora, en una línea de alto fijo; la hora en su columna de ancho fijo. */
export function CabeceraAlerta({ alerta }: { alerta: Alerta }) {
  return (
    <div className="flex h-5 min-w-0 items-center gap-2">
      <h3 className="m-0 min-w-0 truncate text-rotulo font-semibold" title={tServidor(alerta.titulo)}>
        {tServidor(alerta.titulo)}
      </h3>
      <ChipNivel nivel={alerta.nivel} />
      <span className="min-w-0 flex-1 truncate text-ayuda text-tenue" title={`${alerta.nombre} · ${alerta.proyecto}`}>
        {alerta.nombre}
      </span>
      <time dateTime={alerta.hora} className="dato w-10 shrink-0 text-right text-ayuda text-suave">
        {alerta.hora ? hora(new Date(alerta.hora)) : ""}
      </time>
    </div>
  );
}

/**
 * La barra de lo que mide la regla, con la cifra en su columna de ancho fijo («88,0 %») y las marcas del umbral. Debajo,
 * en una línea fija, la medida en sus unidades y qué es la marca. La cifra va en texto: la barra es solo dibujo.
 */
export function BarraAlerta({ alerta, marcas = [], detalle = true }: { alerta: Alerta; marcas?: readonly Marca[]; detalle?: boolean }) {
  const c = COLOR[alerta.nivel];
  const ancho = Math.max(0, Math.min(100, (alerta.valor / alerta.max) * 100));
  const medida = detalleBarra(alerta);
  return (
    <div className="flex flex-col gap-1">
      <div className="flex h-5 items-center gap-3">
        <span aria-hidden="true" className="relative h-2 min-w-0 flex-1 rounded-full bg-borde">
          <span className="absolute inset-0 overflow-hidden rounded-full">
            <span className={`block h-full rounded-full transition-[width] duration-300 ${c.barra}`} style={{ width: `${ancho}%` }} />
          </span>
          {marcas.map((m) => (
            <span
              key={m.nombre}
              title={m.nombre}
              className="absolute -top-[3px] h-3.5 w-[3px] -translate-x-1/2 rounded-full bg-texto ring-2 ring-superficie"
              style={{ left: `${m.pos * 100}%` }}
            />
          ))}
        </span>
        <span className="dato w-[4.25rem] shrink-0 text-right text-rotulo font-semibold" title={medida}>
          {porcentajeDe(alerta)}
        </span>
      </div>
      {detalle && (
        <div className="flex h-4 min-w-0 items-center gap-3 @min-[19rem]:pr-[5rem] text-[11px] leading-4 text-tenue">
          <span className="dato min-w-0 flex-1 truncate" title={medida}>
            {medida}
          </span>
          {marcas[0] && (
            <span className="flex shrink-0 items-center gap-1.5">
              <span aria-hidden="true" className="h-2.5 w-[3px] rounded-full bg-texto" />
              {marcas[0].nombre}
            </span>
          )}
        </div>
      )}
    </div>
  );
}

/** Sin barra: el archivo de la frontera o el dato corto de la regla, en una línea fija. */
function DatoAlerta({ alerta }: { alerta: Alerta }) {
  const texto = alerta.archivo ?? tServidor(alerta.chip);
  return (
    <p className="dato m-0 h-5 truncate rounded-chip bg-hundido px-2 text-ayuda leading-5 text-suave" title={texto}>
      {texto}
    </p>
  );
}

/** El origen del umbral: el documento de la regla o «Umbral por omisión». */
export function fuenteDe(alerta: Alerta): string {
  return alerta.fuente ? t("Regla: {fuente}", { fuente: alerta.fuente }) : t("Umbral por omisión");
}

/** La tarjeta del centro de alertas. Una vista se atenúa y cambia «Visto» por «Vista», sin mover el pie. */
export function TarjetaAlerta({
  alerta,
  marcas,
  vista,
  alVer,
  alVisto,
}: {
  alerta: Alerta;
  marcas: readonly Marca[];
  vista: boolean;
  alVer: () => void;
  alVisto: () => void;
}) {
  const c = COLOR[alerta.nivel];
  const fuente = fuenteDe(alerta);
  return (
    <article
      aria-label={`${t(NOMBRE_NIVEL[alerta.nivel])}: ${tServidor(alerta.titulo)}, ${alerta.nombre}`}
      className={`flex shrink-0 overflow-hidden rounded-panel border bg-superficie transition-opacity duration-150 ${c.borde} ${vista ? "opacity-60" : ""}`}
    >
      <CarrilNivel nivel={alerta.nivel} />
      <div className="@container flex min-w-0 flex-1 flex-col gap-2 px-3 pt-2.5 pb-2.5">
        <CabeceraAlerta alerta={alerta} />
        <p className="m-0 line-clamp-2 h-[2.25rem] text-rotulo" title={tServidor(alerta.texto)}>
          {tServidor(alerta.texto)}
        </p>
        {alerta.max > 0 ? <BarraAlerta alerta={alerta} marcas={marcas} /> : <DatoAlerta alerta={alerta} />}
        {/* En un ancho de teléfono la fuente sube a su propia línea y los botones quedan juntos a la derecha. */}
        <div className="border-t border-borde pt-2">
          <div className="flex flex-wrap items-center justify-end gap-x-2 gap-y-1.5">
            <span className="h-4 min-w-0 basis-full truncate text-ayuda text-tenue @min-[19rem]:flex-1 @min-[19rem]:basis-0" title={fuente}>
              {fuente}
            </span>
            <button
              type="button"
              onClick={alVer}
              className="h-8 shrink-0 rounded-control border border-borde px-2.5 text-rotulo whitespace-nowrap hover:border-borde-fuerte hover:bg-elevado"
            >
              {t("Ver el agente")}
            </button>
            {vista ? (
              <span className="grid h-8 w-[3.75rem] shrink-0 place-items-center text-ayuda text-tenue">{t("Vista")}</span>
            ) : (
              <button type="button" onClick={alVisto} className="h-8 w-[3.75rem] shrink-0 rounded-control bg-elevado text-rotulo hover:bg-borde">
                {t("Visto")}
              </button>
            )}
          </div>
        </div>
      </div>
    </article>
  );
}
