"use client";

import { COLOR_HILO, type FaseAvance, horas, type Paquete, tonoEstado, tramos } from "@/lib/avance";

/**
 * El hilo de la fase: una línea partida en paquetes, cada tramo tan ancho como sus horas, verde lo hecho,
 * ámbar lo avanzado de lo que está en curso y gris lo que falta. Al pasar el puntero o enfocar un tramo, su paquete
 * se lee en un hueco de una línea con columnas fijas debajo (regla: lo que cambia en vivo no mueve el diseño; nada flota ni mueve el diseño).
 */
export function HiloFase({
  fase,
  elegido,
  alElegir,
  alto = 14,
  etiquetas = false,
}: {
  fase: Pick<FaseAvance, "paquetes">;
  elegido: Paquete | null;
  alElegir: (id: string) => void;
  alto?: number;
  etiquetas?: boolean;
}) {
  const ts = tramos(fase);
  const tono = elegido ? tonoEstado(elegido.estado) : null;
  return (
    <div className="flex flex-col gap-2">
      <div role="group" aria-label="Paquetes de la fase: pasa el puntero o enfoca un tramo" className="flex gap-0.5" style={{ height: alto }}>
        {ts.map((t) => {
          const on = elegido?.id === t.paquete.id;
          return (
            <button
              key={t.paquete.id}
              type="button"
              aria-label={t.aria}
              aria-pressed={on}
              onMouseEnter={() => alElegir(t.paquete.id)}
              onFocus={() => alElegir(t.paquete.id)}
              onClick={() => alElegir(t.paquete.id)}
              className="relative min-w-0 overflow-hidden rounded-[3px] transition-[filter] duration-100 hover:brightness-125 focus-visible:brightness-125 motion-reduce:transition-none"
              style={{ flex: `${t.peso} 1 0`, background: t.fondo, boxShadow: on ? "0 0 0 2px var(--superficie), 0 0 0 3px var(--texto)" : undefined }}
            >
              {t.relleno > 0 && <span aria-hidden="true" className="absolute inset-y-0 left-0" style={{ width: `${t.relleno}%`, background: COLOR_HILO.curso }} />}
            </button>
          );
        })}
      </div>
      {etiquetas && (
        <div aria-hidden="true" className="flex gap-0.5">
          {ts.map((t) => (
            <span
              key={t.paquete.id}
              className={`dato min-w-0 truncate text-center text-[11px] ${elegido?.id === t.paquete.id ? "text-texto" : "text-tenue"}`}
              style={{ flex: `${t.peso} 1 0` }}
            >
              {t.paquete.id}
            </span>
          ))}
        </div>
      )}
      <div aria-live="polite" className="dato grid h-[30px] grid-cols-[3rem_minmax(0,1fr)_3rem_5.5rem_3rem] items-center gap-2 rounded-control bg-hundido px-2.5 text-[12.5px]">
        {elegido ? (
          <>
            <b className="font-semibold">{elegido.id}</b>
            <span className="truncate font-sans" title={elegido.que}>
              {elegido.que || "—"}
            </span>
            <span className="text-right text-suave">{horas(elegido.horas)} h</span>
            <span className={tono?.clase}>{tono?.palabra}</span>
            <span className="text-right">{elegido.pct} %</span>
          </>
        ) : (
          <span className="col-span-full font-sans text-tenue">Sin paquetes en la tabla.</span>
        )}
      </div>
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-ayuda text-suave">
        <Clave color={COLOR_HILO.hecho} texto="hecho" />
        <Clave color={COLOR_HILO.curso} texto="avanzado en curso" />
        <Clave color={COLOR_HILO.falta} texto="lo que falta" />
        <span className="ml-auto text-tenue">el ancho de cada tramo son sus horas</span>
      </div>
    </div>
  );
}

function Clave({ color, texto }: { color: string; texto: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span aria-hidden="true" className="size-2.5 rounded-[2px]" style={{ background: color }} />
      {texto}
    </span>
  );
}
