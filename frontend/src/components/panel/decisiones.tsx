"use client";

import { useEffect, useState } from "react";
import { Fantasmas, Paginador, usePaginaViva } from "@/components/ui/paginador";
import { api } from "@/lib/api";
import { agrupar, type Decision, eleccion, elegida, hace, type RespuestaDecisiones, tituloDecision } from "@/lib/decisiones";
import { cuando } from "@/lib/formato";
import { t } from "@/lib/i18n";
import { useAhora } from "@/lib/vivo";
import { Responder } from "./responder";

// «Tienda Demo» → TD; «Atalaya» → AT; «Cuotas» → CU.
const TAMANOS_RESPONDIDAS = [10, 25] as const;

function sigla(proyecto: string): string {
  const ps = proyecto.split(/[\s_-]+/).filter(Boolean);
  return (ps.length > 1 ? ps[0][0] + ps[1][0] : proyecto.slice(0, 2)).toUpperCase();
}

function Fila({ d, on, alElegir, ahora }: { d: Decision; on: boolean; alElegir: () => void; ahora: number }) {
  const respondida = d.estado === "respondida";
  return (
    <button
      type="button"
      aria-pressed={on}
      onClick={alElegir}
      onFocus={alElegir}
      className={`grid w-full items-center gap-3 rounded-panel border px-3.5 text-left transition-colors ${
        respondida ? "h-14 grid-cols-[2.25rem_minmax(0,1fr)_minmax(0,10rem)]" : "h-[76px] grid-cols-[2.25rem_minmax(0,1fr)_5.5rem]"
      } ${on ? "border-acento bg-seleccion" : "border-borde bg-superficie hover:border-borde-fuerte"}`}
    >
      <span className="dato grid h-9 w-9 place-items-center rounded-control bg-elevado text-ayuda font-semibold">{sigla(d.proyecto)}</span>
      <span className="flex min-w-0 flex-col">
        <span className={`truncate ${respondida ? "" : "font-semibold"}`} title={tituloDecision(d)}>
          {tituloDecision(d)}
        </span>
        {!respondida && (
          <span className="truncate text-ayuda text-suave" title={d.sesion}>
            {d.proyecto} · {d.sesion}
          </span>
        )}
      </span>
      {respondida ? (
        <span className="flex min-w-0 flex-col items-end">
          <span className="max-w-full min-w-0 truncate rounded-chip bg-ok-suave px-2 py-0.5 text-ayuda text-ok" title={eleccion(d)}>
            {eleccion(d) || t("respondida")}
          </span>
          <span className="dato text-[11px] text-tenue">{cuando(new Date(d.hora))}</span>
        </span>
      ) : (
        <span className="flex flex-col items-end">
          <span className="dato text-ayuda text-aviso">{hace(d.hora, ahora)}</span>
          <span className="text-ayuda text-tenue">{d.tipo === "atalaya" ? t("por Atalaya") : d.tipo === "botones" ? t("botones") : d.preguntas.length === 1 ? t("1 pregunta") : t("{n} preguntas", { n: d.preguntas.length })}</span>
        </span>
      )}
    </button>
  );
}

/**
 * Lo que te espera: cada sesión que se detuvo a preguntarte algo, con la pregunta, sus opciones y la
 * recomendada; debajo, lo que ya respondiste. Solo lectura: se responde en la app de Claude Code o en el móvil.
 */
export function Decisiones({ version, alAbrirSesion }: { version: number; alAbrirSesion: (d: Decision) => void }) {
  const [resp, setResp] = useState<RespuestaDecisiones | null>(null);
  const [error, setError] = useState(false);
  const [elegida_, setElegida] = useState<string | null>(null);
  const [respondiendo, setRespondiendo] = useState<string | null>(null);
  const ahora = useAhora() * 1000;

  useEffect(() => {
    const c = new AbortController();
    api
      .decisiones(c.signal)
      .then((r) => {
        setResp(r);
        setError(false);
      })
      .catch((e: unknown) => {
        if (!(e instanceof DOMException && e.name === "AbortError")) setError(true);
      });
    return () => c.abort();
  }, [version]);

  const g = agrupar(resp?.decisiones ?? []);
  const todas = [...g.esperando, ...g.sinRespuesta, ...g.respondidas];
  // La primera vez se fija la de arriba: una decisión nueva que llega mientras lees no cambia lo que miras (lo que cambia en vivo no mueve el diseño).
  const primera = todas[0]?.id;
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- elección por omisión cuando llega la lista
    if (!elegida_ && primera) setElegida(primera);
  }, [elegida_, primera]);
  const sel = todas.find((d) => d.id === elegida_) ?? todas[0];
  // Las respondidas se juntan (14 días de todas las sesiones): por páginas, con las nuevas en la primera.
  const pResp = usePaginaViva(g.respondidas, (d) => d.id, { almacen: "decisiones-respondidas", opciones: TAMANOS_RESPONDIDAS, defecto: 10, reinicio: "todas" });

  const lista = (titulo: string, ds: Decision[]) =>
    ds.length > 0 && (
      <section aria-label={titulo} className="flex flex-col gap-2">
        <h2 className="m-0 text-rotulo font-semibold text-suave">
          {titulo} · <span className="dato">{ds.length}</span>
        </h2>
        {ds.map((d) => (
          <Fila key={d.id} d={d} on={sel?.id === d.id} alElegir={() => setElegida(d.id)} ahora={ahora} />
        ))}
      </section>
    );

  return (
    <main id="contenido" tabIndex={-1} aria-label={t("Decisiones")} className="relative mx-auto flex w-full max-w-[1520px] flex-col gap-4 p-5">
      <div className="flex flex-col gap-1">
        <h1 className="m-0 text-titulo font-semibold">{t("Lo que te espera")}</h1>
        <p className="m-0 max-w-3xl text-suave">
          {t("De todos los proyectos: cada sesión que se detuvo a preguntarte algo, con la pregunta, las opciones y la que recomendó. Las que llegan por Atalaya se responden aquí, por pasos, con texto e imágenes; las de la app, en la app o el móvil.")}
        </p>
      </div>

      {error && !resp && <p className="m-0 rounded-panel bg-error-suave px-4 py-3 text-error">{t("No se pudieron leer las decisiones. Se reintenta con el próximo aviso.")}</p>}
      {!resp && !error && <div className="esqueleto h-40 rounded-tarjeta" aria-hidden="true" />}
      {resp && todas.length === 0 && <p className="m-0 rounded-tarjeta border border-borde bg-superficie px-5 py-6 text-suave">{t("Ninguna sesión te preguntó nada en los últimos 14 días.")}</p>}

      {todas.length > 0 && (
        <div className="flex flex-wrap items-start gap-4">
          <div className="flex min-w-0 flex-[1_1_32rem] flex-col gap-4">
            {g.esperando.length === 0 && <p className="m-0 rounded-panel bg-ok-suave px-4 py-3 text-ok">{t("Nada te espera ahora.")}</p>}
            {lista(t("Esperándote"), g.esperando)}
            {lista(t("Sin respuesta, de hace más de un día"), g.sinRespuesta)}
            {g.respondidas.length > 0 && (
              <section aria-label={t("Respondidas")} className="flex flex-col gap-2">
                <h2 className="m-0 text-rotulo font-semibold text-suave">
                  {t("Respondidas")} · <span className="dato">{g.respondidas.length}</span>
                </h2>
                {/* Filas de 56 px; la última página se completa con fantasma del mismo alto y el pie no se mueve. */}
                <div className="flex flex-col gap-2">
                  {pResp.visibles.map((d) => (
                    <Fila key={d.id} d={d} on={sel?.id === d.id} alElegir={() => setElegida(d.id)} ahora={ahora} />
                  ))}
                  <Fantasmas n={pResp.faltan} className="h-14 rounded-panel border border-borde" />
                </div>
                {pResp.mostrar && <Paginador {...pResp.pie} nombre={t("Respondidas")} />}
              </section>
            )}
          </div>

          {sel && (
            <aside aria-label={t("Decisión elegida")} className="sticky top-[68px] flex min-h-[28rem] min-w-0 flex-[1_1_26rem] flex-col gap-3 rounded-tarjeta border border-borde-fuerte bg-superficie p-5">
              <span className="flex min-w-0 items-center gap-2 text-ayuda text-suave">
                <span className="dato rounded-chip bg-elevado px-1.5">{sigla(sel.proyecto)}</span>
                <span className="truncate" title={`${sel.proyecto} · ${sel.sesion}`}>
                  {sel.proyecto} · {sel.sesion} · {cuando(new Date(sel.hora))}
                </span>
              </span>
              {sel.preguntas.map((q, i) => (
                <div key={i} className="flex flex-col gap-2 pt-1">
                  {q.encabezado && <span className="self-start rounded-chip bg-acento-suave px-2 text-ayuda font-semibold text-acento-fuerte">{q.encabezado}</span>}
                  <p className="m-0 text-[15.5px] leading-snug font-semibold">{q.texto}</p>
                  {q.opciones.map((o, j) => {
                    const es = elegida(q, o);
                    return (
                      <div key={`${j}:${o.etiqueta}`} className={`flex flex-col gap-0.5 rounded-control border px-3 py-2.5 ${es ? "border-ok bg-ok-suave" : "border-borde bg-hundido"}`}>
                        <span className="flex flex-wrap items-center gap-2 font-medium">
                          {o.etiqueta}
                          {o.recomendada && <span className="rounded-chip bg-acento-suave px-1.5 text-ayuda font-normal text-acento-fuerte">{t("recomendada")}</span>}
                          {es && <span className="rounded-chip bg-ok-suave px-1.5 text-ayuda font-normal text-ok">{t("elegiste esta")}</span>}
                        </span>
                        {o.nota && <span className="text-ayuda text-suave">{o.nota}</span>}
                      </div>
                    );
                  })}
                  {q.respuesta && !q.opciones.some((o) => elegida(q, o)) && (
                    <div className="rounded-control border border-ok bg-ok-suave px-3 py-2.5">
                      <span className="text-ayuda text-ok">{t("Respondiste con tus palabras")}</span>
                      <p className="m-0 line-clamp-4" title={q.respuesta}>
                        {q.respuesta}
                      </p>
                    </div>
                  )}
                </div>
              ))}
              <span className="flex-1" />
              {sel.contexto && (
                <div className="flex flex-col gap-0.5 rounded-control bg-hundido px-3 py-2.5 text-rotulo">
                  <span className="text-tenue">{t("Lo que te contó antes de preguntar")}</span>
                  <span className="line-clamp-3" title={sel.contexto}>
                    {sel.contexto}
                  </span>
                </div>
              )}
              <div className="flex flex-wrap items-center gap-3">
                {/* Las que llegaron por `preguntar` se responden aquí; las de la app, en la app. */}
                {sel.tipo === "atalaya" && sel.estado === "pendiente" && (
                  <button
                    type="button"
                    onClick={() => setRespondiendo(sel.id.replace(/^atalaya~/, ""))}
                    className="h-9 rounded-control bg-acento px-4 font-medium text-sobre-acento hover:bg-acento-fuerte"
                  >
                    {t("Responder aquí")}
                  </button>
                )}
                {sel.agente && (
                  <button type="button" onClick={() => alAbrirSesion(sel)} className="h-9 rounded-control border border-borde-fuerte px-3.5 font-medium hover:bg-elevado">
                    {t("Abrir la sesión en esa pregunta")}
                  </button>
                )}
                <span className="text-ayuda text-tenue">
                  {sel.estado === "respondida" && sel.respondida
                    ? t("Respondida: {hora}", { hora: cuando(new Date(sel.respondida)) })
                    : t("La sesión está {estado}", { estado: sel.estado_sesion ? t(sel.estado_sesion).toLowerCase() : t("sin dato") })}
                </span>
              </div>
            </aside>
          )}
        </div>
      )}
      {respondiendo && <Responder id={respondiendo} alCerrar={() => setRespondiendo(null)} />}
    </main>
  );
}
