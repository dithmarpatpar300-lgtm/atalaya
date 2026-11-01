"use client";

import type { ReactNode } from "react";
import { COLOR, detalleBarra, type Marca, NOMBRE_NIVEL, porcentajeDe } from "@/lib/alertas";
import type { Alerta, NivelAlerta } from "@/lib/api";
import { hora } from "@/lib/formato";
import { t, tServidor } from "@/lib/i18n";

/*
 * La tarjeta de una alerta, compartida por el centro de alertas (la campana), la banda flotante y los mensajes de los
 * paneles. Una sola geometría: una rejilla de dos columnas, la del icono de ancho fijo (1,25 rem) y la del contenido.
 * El icono va en una caja del alto de la primera línea del título, así su centro cae en el centro de esa línea; debajo,
 * en la columna del contenido y alineados con el título, el cuerpo, los datos y las acciones.
 *
 * La severidad nunca va solo en el color: la dice la forma del icono (octógono, triángulo, círculo), la etiqueta en
 * palabra y el borde (error: filo de 6 px y contorno del nivel; aviso: filo de 4 px y contorno del nivel; info: sin filo,
 * contorno neutro). El filo es del color sólido del nivel, que pasa 3:1 contra la superficie en claro y en oscuro.
 *
 * Lo que cambia en vivo (la hora, el texto, la cifra) tiene su hueco fijo: una línea o dos con puntos suspensivos, el
 * texto entero en `title` y las cifras en columnas de ancho fijo (lo que cambia en vivo no mueve el diseño).
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
export function IconoNivel({ nivel, className = "size-4" }: { nivel: NivelAlerta; className?: string }) {
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

/** El borde de cada nivel: el contorno y el filo de la izquierda (su ancho también distingue el nivel). */
const BORDE: Record<NivelAlerta, { contorno: string; filo: string }> = {
  error: { contorno: "border-error/45", filo: "w-1.5 bg-error" },
  aviso: { contorno: "border-aviso/40", filo: "w-1 bg-aviso" },
  info: { contorno: "border-borde-fuerte", filo: "" },
};

/** El filo de nivel, pegado al borde izquierdo; la tarjeta lo recorta con su radio. Info no lleva. */
function Filo({ nivel }: { nivel: NivelAlerta }) {
  const filo = BORDE[nivel].filo;
  return filo ? <span aria-hidden="true" className={`absolute inset-y-0 left-0 ${filo}`} /> : null;
}

/**
 * La columna del icono: ancho fijo y el alto de la primera línea del título (`alto`), con el icono centrado en ella.
 * Así el icono queda alineado con el título aunque el título o el cuerpo ocupen más líneas.
 */
function ColumnaIcono({ nivel, alto = "h-5" }: { nivel: NivelAlerta; alto?: string }) {
  return (
    <span aria-hidden="true" className={`flex w-5 items-center justify-center ${alto} ${COLOR[nivel].texto}`}>
      <IconoNivel nivel={nivel} />
    </span>
  );
}

/** El marco común: la rejilla de dos columnas, el contorno y el filo del nivel. */
const MARCO = "relative grid grid-cols-[1.25rem_minmax(0,1fr)] gap-x-2.5 overflow-hidden rounded-panel border";

/** El nivel en palabra, con su color. */
export function ChipNivel({ nivel }: { nivel: NivelAlerta }) {
  const c = COLOR[nivel];
  return (
    <span className={`shrink-0 rounded-chip border px-1.5 text-[11px] leading-[16px] font-medium ${c.texto} ${c.fondo} ${c.borde}`}>
      {t(NOMBRE_NIVEL[nivel])}
    </span>
  );
}

/** La hora en su columna de ancho fijo. */
function HoraAlerta({ alerta }: { alerta: Alerta }) {
  return (
    <time dateTime={alerta.hora} className="dato w-10 shrink-0 text-right text-ayuda text-suave">
      {alerta.hora ? hora(new Date(alerta.hora)) : ""}
    </time>
  );
}

/**
 * Regla, nivel y hora en una línea de alto fijo (h-5). El título toma lo que queda y se corta con puntos suspensivos;
 * la etiqueta y la hora no se encogen. Con `agente`, el nombre del agente va en la misma línea con un mínimo propio,
 * para que un título largo no lo borre (la banda); sin él, la tarjeta lo pone en su línea de abajo.
 */
export function CabeceraAlerta({ alerta, agente = false }: { alerta: Alerta; agente?: boolean }) {
  const titulo = tServidor(alerta.titulo);
  return (
    <div className="flex h-5 min-w-0 items-center gap-2">
      <h3 className={`m-0 min-w-0 truncate text-rotulo font-semibold ${agente ? "" : "flex-1"}`} title={titulo}>
        {titulo}
      </h3>
      <ChipNivel nivel={alerta.nivel} />
      {agente && (
        <span className="min-w-16 flex-1 truncate text-ayuda text-tenue" title={`${alerta.nombre} · ${alerta.proyecto}`}>
          {alerta.nombre}
        </span>
      )}
      <HoraAlerta alerta={alerta} />
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
    <div className="flex min-w-0 flex-col gap-1">
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
        <div className="flex h-4 min-w-0 items-center gap-3 @min-[22rem]:pr-[5rem] text-[11px] leading-4 text-tenue">
          <span className="dato min-w-0 flex-1 truncate" title={medida}>
            {medida}
          </span>
          {marcas[0] && (
            <span className="flex min-w-0 shrink items-center gap-1.5">
              <span aria-hidden="true" className="h-2.5 w-[3px] shrink-0 rounded-full bg-texto" />
              <span className="truncate" title={marcas[0].nombre}>
                {marcas[0].nombre}
              </span>
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
    <p className="dato m-0 h-6 min-w-0 truncate rounded-chip bg-hundido px-2 text-ayuda leading-6 text-suave" title={texto}>
      {texto || "—"}
    </p>
  );
}

/** El origen del umbral: el documento de la regla o «Umbral por omisión». */
export function fuenteDe(alerta: Alerta): string {
  return alerta.fuente ? t("Regla: {fuente}", { fuente: alerta.fuente }) : t("Umbral por omisión");
}

/** Los botones de la alerta: «Ver el agente» y «Visto» (o «Vista», en el mismo hueco). Nunca se encogen ni se parten. */
function AccionesAlerta({ alerta, vista = false, alVer, alVisto, fuerte = false }: { alerta: Alerta; vista?: boolean; alVer: () => void; alVisto: () => void; fuerte?: boolean }) {
  // En la banda, «Visto» va con el color del nivel (salvo info, que no lo tiene sólido) para que se lea como la acción.
  const visto = fuerte && alerta.nivel !== "info" ? `${COLOR[alerta.nivel].barra} font-medium text-superficie hover:opacity-90` : "bg-elevado hover:bg-borde";
  return (
    <span className="flex shrink-0 items-center gap-2">
      <button
        type="button"
        onClick={alVer}
        className="h-8 shrink-0 rounded-control border border-borde-fuerte px-2.5 text-rotulo whitespace-nowrap hover:bg-elevado"
      >
        {t("Ver el agente")}
      </button>
      {vista ? (
        <span className="grid h-8 w-[3.75rem] shrink-0 place-items-center text-ayuda text-tenue">{t("Vista")}</span>
      ) : (
        <button type="button" onClick={alVisto} className={`h-8 w-[3.75rem] shrink-0 rounded-control text-rotulo ${visto}`}>
          {t("Visto")}
        </button>
      )}
    </span>
  );
}

/**
 * La tarjeta del centro de alertas. Fila 1: icono y título con su nivel y su hora; debajo, en la columna del título,
 * el agente, el cuerpo (dos líneas fijas), los datos y el pie con la fuente y los botones. Una vista no se atenúa (bajaría
 * el contraste del texto): pierde el contorno de color, toma el fondo hundido y cambia «Visto» por «Vista», sin mover el
 * pie.
 */
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
  const fuente = fuenteDe(alerta);
  const texto = tServidor(alerta.texto);
  return (
    <article
      aria-label={`${t(NOMBRE_NIVEL[alerta.nivel])}: ${tServidor(alerta.titulo)}, ${alerta.nombre}`}
      className={`@container ${MARCO} shrink-0 gap-y-2 py-2.5 pr-3 pl-3.5 transition-colors duration-150 ${
        vista ? "border-borde bg-hundido" : `${BORDE[alerta.nivel].contorno} bg-superficie`
      }`}
    >
      <Filo nivel={alerta.nivel} />
      <ColumnaIcono nivel={alerta.nivel} />
      <div className="flex min-w-0 flex-col gap-0.5">
        <CabeceraAlerta alerta={alerta} />
        <p className="m-0 h-4 truncate text-ayuda text-tenue" title={`${alerta.nombre} · ${alerta.proyecto}`}>
          {alerta.nombre}
          {alerta.proyecto && alerta.proyecto !== alerta.nombre && <span> · {alerta.proyecto}</span>}
        </p>
      </div>
      <p className="col-start-2 m-0 line-clamp-2 h-[2.25rem] min-w-0 text-rotulo break-words" title={texto}>
        {texto}
      </p>
      <div className="col-start-2 min-w-0">{alerta.max > 0 ? <BarraAlerta alerta={alerta} marcas={marcas} /> : <DatoAlerta alerta={alerta} />}</div>
      {/* En un ancho de teléfono la fuente sube a su propia línea y los botones quedan juntos a la derecha. */}
      <div className="col-start-2 flex min-w-0 flex-wrap items-center justify-end gap-x-2 gap-y-1.5 border-t border-borde pt-2">
        <span className="h-4 min-w-0 basis-full truncate text-ayuda text-tenue @min-[22rem]:flex-1 @min-[22rem]:basis-0" title={fuente}>
          {fuente}
        </span>
        <AccionesAlerta alerta={alerta} vista={vista} alVer={alVer} alVisto={alVisto} />
      </div>
    </article>
  );
}

/**
 * La misma alerta en una banda flotante: la rejilla de la tarjeta en una o dos filas. El icono va en la primera línea
 * del título, como en la tarjeta (antes se centraba en todo el alto y se desfasaba al partirse en dos filas).
 */
export function BandaTarjeta({ alerta, marcas, alVer, alVisto }: { alerta: Alerta; marcas: readonly Marca[]; alVer: () => void; alVisto: () => void }) {
  const texto = tServidor(alerta.texto);
  const fuente = fuenteDe(alerta);
  return (
    <div className={`${MARCO} ${BORDE[alerta.nivel].contorno} items-start bg-superficie py-2.5 pr-3 pl-3.5`}>
      <Filo nivel={alerta.nivel} />
      <ColumnaIcono nivel={alerta.nivel} />
      <div className="flex min-w-0 flex-wrap items-center gap-x-5 gap-y-2">
        <div className="flex min-w-0 flex-[1_1_18rem] flex-col gap-0.5">
          <CabeceraAlerta alerta={alerta} agente />
          <p className="m-0 h-[1.125rem] truncate text-rotulo" title={`${texto} · ${fuente}`}>
            {texto}
            <span className="text-tenue"> · {fuente}</span>
          </p>
        </div>
        {alerta.max > 0 && (
          <div className="w-[min(14rem,100%)] shrink-0">
            <BarraAlerta alerta={alerta} marcas={marcas} detalle={false} />
          </div>
        )}
        <span className="ml-auto">
          <AccionesAlerta alerta={alerta} alVer={alVer} alVisto={alVisto} fuerte />
        </span>
      </div>
    </div>
  );
}

/**
 * Un mensaje de un panel (no se pudo leer, un aviso de la datapedia) con la misma geometría: el icono del nivel en la
 * primera línea y el texto, que se parte en las líneas que haga falta, alineado a su derecha. El nivel también va en
 * palabra para el lector de pantalla.
 */
export function MensajeNivel({ nivel, children, className = "" }: { nivel: NivelAlerta; children: ReactNode; className?: string }) {
  const c = COLOR[nivel];
  return (
    <div className={`${MARCO} ${BORDE[nivel].contorno} ${c.fondo} py-2.5 pr-3 pl-3.5 ${className}`}>
      <Filo nivel={nivel} />
      <ColumnaIcono nivel={nivel} alto="h-[1.125rem]" />
      <p className={`m-0 min-w-0 text-rotulo break-words ${nivel === "info" ? "text-texto" : c.texto}`}>
        <span className="sr-only">{t(NOMBRE_NIVEL[nivel])}: </span>
        {children}
      </p>
    </div>
  );
}
