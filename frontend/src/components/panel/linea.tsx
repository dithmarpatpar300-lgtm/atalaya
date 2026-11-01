"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { api } from "@/lib/api";
import { t } from "@/lib/i18n";
import { MensajeNivel } from "@/components/ui/tarjeta-alerta";
import {
  cuotaEn,
  curva,
  enParaleloCon,
  type FilaOrdenada,
  hhmm,
  marcas,
  minutos,
  nombreDia,
  ordenar,
  paralelo,
  pct,
  type RespuestaLinea,
  tramo,
} from "@/lib/linea";

const CLAVE = "atalaya:linea";
const ALTO_CUOTA = 64;
const ALTO_PARALELO = 44;
// Los colores de serie validados para gráficos (globals.css); la sesión va en gris, que no es una serie.
const BARRA = {
  Claude: "border-serie-claude bg-serie-claude/25",
  Codex: "border-serie-codex bg-serie-codex/25",
  sesion: "border-borde-fuerte bg-elevado",
} as const;
const PUNTO = { Claude: "bg-serie-claude", Codex: "bg-serie-codex", sesion: "bg-borde-fuerte" } as const;
const RAYADO = "repeating-linear-gradient(135deg, var(--aviso) 0 3px, transparent 3px 7px)";
const TEXTO_DIARIO = t("Su transcrito ya se borró: la barra sale del diario, sin cruces ni pasos.");

function leerSesiones(): boolean {
  try {
    return (JSON.parse(localStorage.getItem(CLAVE) ?? "{}") as { sesiones?: boolean }).sesiones !== false;
  } catch {
    return true;
  }
}

const tipoColor = (f: FilaOrdenada) => (f.tipo === "sesion" ? "sesion" : f.proveedor);

/**
 * La línea de tiempo global: cada agente del proyecto es una barra sobre el reloj
 * del día. Arriba, la cuota de 5 h de Claude y Codex según Cuotas; abajo, cuántos trabajaban a la vez. Rayado, lo que
 * pasó del tope de AGENTS.md; una marca roja, cuando editó un archivo que otro agente vivo también editaba.
 */
export function Linea({ proyecto, version, alAbrirAgente }: { proyecto: string; version: number; alAbrirAgente: (id: string) => void }) {
  const [dia, setDia] = useState("");
  const [respCruda, setResp] = useState<RespuestaLinea | null>(null);
  // Mientras llega la del proyecto nuevo, la del anterior no se muestra como si fuera de este.
  const resp = respCruda && respCruda.proyecto === proyecto ? respCruda : null;
  const [error, setError] = useState(false);
  const [sesiones, setSesiones] = useState(true);
  const [elegida, setElegida] = useState<string | null>(null);
  const [cursor, setCursor] = useState<number | null>(null);
  const plot = useRef<HTMLDivElement>(null);
  // Las filas tienen alto máximo y se desplazan solas: con muchos agentes el reloj no empuja todo hacia abajo. El ancho
  // de su barra de desplazamiento (0 si no la hay, o si el sistema la superpone) se descuenta del reloj, las horas,
  // «A la vez» y el cursor, para que las barras sigan alineadas con las horas.
  const lista = useRef<HTMLUListElement>(null);
  const [barra, setBarra] = useState(0);
  useEffect(() => {
    const el = lista.current;
    if (!el) return;
    const ro = new ResizeObserver(() => setBarra(el.offsetWidth - el.clientWidth));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- el ajuste guardado se lee después de hidratar
    setSesiones(leerSesiones());
  }, []);

  useEffect(() => {
    if (!proyecto) return;
    const c = new AbortController();
    api
      .linea(proyecto, dia, c.signal)
      .then((r) => {
        setResp(r);
        setError(false);
      })
      .catch((e: unknown) => {
        if (!(e instanceof DOMException && e.name === "AbortError")) setError(true);
      });
    return () => c.abort();
  }, [proyecto, dia, version]);

  const filas = useMemo(() => resp ? ordenar(resp.filas, sesiones) : [], [resp, sesiones]);
  const [ini, fin] = resp ? tramo(filas, Date.parse(resp.desde), Date.parse(resp.hasta)) : [0, 1];
  const ticks = resp ? marcas(ini, fin) : [];
  const vez = useMemo(() => resp ? paralelo(filas, ini, fin) : [], [resp, filas, ini, fin]);
  const maxVez = useMemo(() => Math.max(1, ...vez.map((v) => v.n)), [vez]);
  // La primera vez se fija la del primer agente: un agente nuevo que llega en vivo no cambia lo que miras (lo que cambia en vivo no mueve el diseño).
  const primera = (filas.find((f) => f.tipo !== "sesion") ?? filas[0])?.id;
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- elección por omisión cuando llega la lista
    if (!elegida && primera) setElegida(primera);
  }, [elegida, primera]);
  const sel = filas.find((f) => f.id === elegida) ?? filas.find((f) => f.tipo !== "sesion") ?? filas[0];
  const hoy = resp ? resp.ahora.slice(0, 10) : "";
  const p = (ms: number) => `${pct(ms, ini, fin)}%`;
  const claude = resp?.cuota.claude;
  const codex = resp?.cuota.codex;
  const hayCuota = !!(curva(claude, ini, fin, ALTO_CUOTA) || curva(codex, ini, fin, ALTO_CUOTA));

  function mover(e: React.PointerEvent) {
    const r = plot.current?.getBoundingClientRect();
    if (!r) return;
    const x = (e.clientX - r.left) / r.width;
    setCursor(x < 0 || x > 1 ? null : ini + x * (fin - ini));
  }

  function alternarSesiones() {
    const v = !sesiones;
    setSesiones(v);
    try {
      localStorage.setItem(CLAVE, JSON.stringify({ sesiones: v }));
    } catch {}
  }

  const lecturaCursor = (() => {
    if (cursor === null) return null;
    const c1 = cuotaEn(claude, cursor);
    const c2 = cuotaEn(codex, cursor);
    const n = filas.filter((f) => f.tipo !== "sesion" && f.a <= cursor && f.b >= cursor).length;
    return { hora: hhmm(cursor), claude: c1, codex: c2, n };
  })();

  return (
    <main id="contenido" tabIndex={-1} aria-label={t("Línea de tiempo")} className="relative mx-auto flex w-full max-w-[1520px] flex-col gap-4 p-5">
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex min-w-0 flex-[1_1_32rem] flex-col gap-1">
          <h1 className="m-0 text-titulo font-semibold">
            {resp && filas.length ? t("{dia}, de {de} a {a}", { dia: nombreDia(resp.dia, hoy), de: hhmm(ini), a: hhmm(fin) }) : t("Línea de tiempo")}
          </h1>
          <p className="m-0 max-w-3xl text-suave">{t("Cada agente es una barra sobre el reloj. Arriba, la cuota de la ventana de 5 h; abajo, cuántos trabajaban a la vez.")}</p>
        </div>
        {resp && resp.dias.length > 0 && (
          <div role="group" aria-label={t("Día")} className="flex max-w-full gap-0.5 overflow-x-auto rounded-control bg-elevado p-[3px]">
            {resp.dias.slice(0, 7).map((d) => {
              const on = d === resp.dia;
              return (
                <button
                  key={d}
                  type="button"
                  aria-pressed={on}
                  onClick={() => setDia(d === hoy ? "" : d)}
                  className={`h-[30px] shrink-0 rounded-chip px-3 text-rotulo whitespace-nowrap transition-colors duration-150 ${
                    on ? "bg-superficie font-medium text-texto shadow-sm" : "text-suave hover:text-texto"
                  }`}
                >
                  {nombreDia(d, hoy)}
                </button>
              );
            })}
            {resp.dias.length > 7 && (
              <input
                type="date"
                aria-label={t("Otro día")}
                min={resp.dias.reduce((a, b) => (a < b ? a : b))}
                max={hoy}
                value={resp.dias.slice(0, 7).includes(resp.dia) ? "" : resp.dia}
                onChange={(e) => setDia(e.target.value === hoy ? "" : e.target.value)}
                className={`h-[30px] shrink-0 rounded-chip px-3 text-rotulo ${
                  resp.dias.slice(0, 7).includes(resp.dia) ? "bg-elevado text-suave" : "bg-superficie font-medium text-texto shadow-sm"
                }`}
              />
            )}
          </div>
        )}
        <button
          type="button"
          aria-pressed={sesiones}
          onClick={alternarSesiones}
          className="h-9 rounded-control border border-borde bg-superficie px-3 text-rotulo transition-colors hover:border-borde-fuerte"
        >
          {sesiones ? t("Ocultar las sesiones") : t("Mostrar las sesiones")}
        </button>
      </div>

      {error && !resp && <MensajeNivel nivel="error">{t("No se pudo leer la línea de tiempo. Se reintenta con el próximo aviso.")}</MensajeNivel>}

      <section aria-label={t("Agentes sobre el reloj")} className="rounded-tarjeta border border-borde bg-superficie p-4">
        {/* La lectura del puntero: alto fijo y columnas fijas, para que nada salte al moverlo (lo que cambia en vivo no mueve el diseño). */}
        <div className="dato mb-2 flex h-5 items-center gap-0 overflow-hidden text-[12.5px] whitespace-nowrap">
          {lecturaCursor ? (
            <>
              <span className="w-14 shrink-0 font-semibold">{lecturaCursor.hora}</span>
              <span className="flex w-36 shrink-0 items-center gap-1.5">
                <span aria-hidden="true" className="h-0.5 w-3 bg-serie-claude" />
                Claude {lecturaCursor.claude === null ? "—" : `${Math.round(lecturaCursor.claude)} %`}
              </span>
              <span className="flex w-36 shrink-0 items-center gap-1.5">
                <span aria-hidden="true" className="h-0.5 w-3 bg-serie-codex" />
                Codex {lecturaCursor.codex === null ? "—" : `${Math.round(lecturaCursor.codex)} %`}
              </span>
              <span className="text-suave">
                {lecturaCursor.n === 1 ? t("1 agente a la vez") : t("{n} agentes a la vez", { n: lecturaCursor.n })}
              </span>
            </>
          ) : (
            <span className="text-tenue">{t("Pasa el puntero por el reloj para leer la cuota y cuántos trabajaban.")}</span>
          )}
        </div>

        <div className="overflow-x-auto">
          <div className="relative min-w-[52rem] overflow-x-clip" onPointerMove={mover} onPointerLeave={() => setCursor(null)}>
            {/* La cuota de 5 h. */}
            <div className="grid h-[72px] grid-cols-[14rem_minmax(0,1fr)] items-end gap-3" style={{ paddingRight: barra }}>
              <span className="flex flex-col pb-1 text-ayuda text-suave">
                <span>{t("Cuota, ventana de 5 h")}</span>
                {!hayCuota && resp && <span className="text-tenue">{t("sin lecturas de Cuotas este día")}</span>}
              </span>
              <div ref={plot} className="relative border-b border-borde" style={{ height: ALTO_CUOTA }}>
                <svg aria-hidden="true" width="100%" height={ALTO_CUOTA} viewBox={`0 0 1000 ${ALTO_CUOTA}`} preserveAspectRatio="none" className="block overflow-visible">
                  <line x1="0" x2="1000" y1={ALTO_CUOTA / 2} y2={ALTO_CUOTA / 2} className="stroke-borde" strokeDasharray="3 3" vectorEffect="non-scaling-stroke" />
                  <polyline points={curva(claude, ini, fin, ALTO_CUOTA)} fill="none" className="stroke-serie-claude" strokeWidth="2" vectorEffect="non-scaling-stroke" />
                  <polyline points={curva(codex, ini, fin, ALTO_CUOTA)} fill="none" className="stroke-serie-codex" strokeWidth="2" vectorEffect="non-scaling-stroke" />
                </svg>
                <span className="dato absolute top-[calc(50%-9px)] left-0.5 text-[11px] text-tenue">50 %</span>
              </div>
            </div>

            {/* Las horas. */}
            <div className="grid h-6 grid-cols-[14rem_minmax(0,1fr)] gap-3" style={{ paddingRight: barra }}>
              <span />
              <div className="relative">
                {ticks.map((tk, i) => (
                  <span
                    key={tk}
                    className="dato absolute top-1 text-[11.5px] text-tenue"
                    style={{ left: p(tk), transform: pct(tk, ini, fin) < 1 ? "none" : i === ticks.length - 1 && pct(tk, ini, fin) > 97 ? "translateX(-100%)" : "translateX(-50%)" }}
                  >
                    {hhmm(tk)}
                  </span>
                ))}
              </div>
            </div>

            {resp && filas.length === 0 && <p className="m-0 border-t border-borde px-1 py-6 text-suave">{t("Ningún agente este día en {proyecto}.", { proyecto: resp.proyecto })}</p>}
            {!resp && !error && [0, 1, 2, 3].map((i) => <div key={i} className="esqueleto my-1 h-[30px] rounded-control" aria-hidden="true" />)}

            <ul ref={lista} className="m-0 max-h-[min(34rem,65vh)] list-none overflow-y-auto overscroll-contain p-0">
              {filas.map((f) => {
                const on = sel?.id === f.id;
                const color = tipoColor(f);
                const tope = resp ? f.a + resp.tope_min * 60_000 : f.b;
                const exceso = f.excede && f.b > tope ? `${((f.b - tope) / Math.max(1, f.b - f.a)) * 100}%` : null;
                return (
                  <li key={f.id} className="grid h-[34px] grid-cols-[14rem_minmax(0,1fr)] items-center gap-3 border-t border-borde">
                    <span className="flex min-w-0 items-center gap-2" style={{ paddingLeft: f.nivel ? 14 : 0 }}>
                      <span aria-hidden="true" className={`size-2 shrink-0 rounded-full ${PUNTO[color]}`} />
                      <span className={`truncate text-rotulo ${f.tipo === "sesion" ? "text-suave" : ""}`} title={`${f.nombre}${f.clase ? ` · ${f.clase}` : ""}`}>
                        {f.nombre}
                      </span>
                      {f.diario && <span className="shrink-0 text-ayuda text-tenue" title={TEXTO_DIARIO}>{t("diario")}</span>}
                    </span>
                    <div className="relative h-full">
                      {ticks.map((tk) => (
                        <span key={tk} aria-hidden="true" className="absolute inset-y-0 w-px bg-borde/70" style={{ left: p(tk) }} />
                      ))}
                      <button
                        type="button"
                        aria-pressed={on}
                        aria-label={
                          f.vivo
                            ? t("{nombre}, de {de} a {a}, sigue", { nombre: f.nombre, de: hhmm(f.a), a: hhmm(f.b) })
                            : t("{nombre}, de {de} a {a}", { nombre: f.nombre, de: hhmm(f.a), a: hhmm(f.b) })
                        }
                        onFocus={() => setElegida(f.id)}
                        onClick={() => setElegida(f.id)}
                        onDoubleClick={() => {
                          if (!f.diario) alAbrirAgente(f.id);
                        }}
                        className={`absolute top-[7px] h-5 min-w-1.5 overflow-hidden rounded-chip border p-0 ${BARRA[color]} ${on ? "border-2" : ""} ${f.a < ini ? "rounded-l-none border-l-0" : ""} ${f.vivo || f.b > fin ? "rounded-r-none" : ""}`}
                        style={{ left: p(f.a), width: `calc(${pct(f.b, ini, fin) - pct(f.a, ini, fin)}% )` }}
                      >
                        {exceso && <span aria-hidden="true" className="absolute inset-y-0 right-0" style={{ width: exceso, background: RAYADO }} />}
                      </button>
                      {f.choques.map((c) => {
                        const ms = Date.parse(c.hora);
                        if (ms < ini || ms > fin) return null;
                        const texto = t("{hora} · editó {ruta}, que {otro} también editaba", { hora: hhmm(ms), ruta: c.ruta, otro: c.otro });
                        const abrir = t("{texto}. Abrir a {otro}", { texto, otro: c.otro });
                        // La clave lleva la contraparte: el servidor da un cruce por cada otro agente, con igual ruta y hora.
                        // La marca abre al otro agente; su área de clic es más ancha que la raya (≥ 12 px).
                        return c.otro_id ? (
                          <button
                            key={`${c.ruta}|${c.hora}|${c.otro_id ?? ""}|${c.otro}`}
                            type="button"
                            title={abrir}
                            aria-label={abrir}
                            onClick={() => alAbrirAgente(c.otro_id!)}
                            className="group absolute inset-y-0 w-3 -translate-x-1/2"
                            style={{ left: p(ms) }}
                          >
                            <span aria-hidden="true" className="absolute inset-y-1 left-1/2 w-[3px] -translate-x-1/2 rounded-full bg-error group-hover:w-[5px]" />
                          </button>
                        ) : (
                          <span key={`${c.ruta}|${c.hora}|${c.otro_id ?? ""}|${c.otro}`} title={texto} className="absolute top-1 bottom-1 w-[3px] -translate-x-1/2 rounded-full bg-error" style={{ left: p(ms) }} />
                        );
                      })}
                    </div>
                  </li>
                );
              })}
            </ul>

            {/* A la vez. */}
            {filas.length > 0 && (
              <div className="grid grid-cols-[14rem_minmax(0,1fr)] items-end gap-3 border-t border-borde-fuerte pt-1.5" style={{ paddingRight: barra }}>
                <span className="pb-1 text-ayuda text-suave">{t("A la vez (sin las sesiones)")}</span>
                <div aria-hidden="true" className="relative" style={{ height: ALTO_PARALELO }}>
                  {vez.map((v) =>
                    v.n ? (
                      <span
                        key={v.t}
                        className="absolute bottom-0 rounded-t-[2px] bg-suave/45"
                        style={{ left: p(v.t), width: `calc(${pct(v.t + 5 * 60_000, ini, fin) - pct(v.t, ini, fin)}% - 1px)`, height: (v.n / maxVez) * ALTO_PARALELO }}
                      />
                    ) : null,
                  )}
                  <span className="dato absolute top-0 left-0.5 text-[11px] text-tenue">{maxVez}</span>
                </div>
              </div>
            )}

            {/* La hora del puntero, sobre todo el reloj. */}
            {cursor !== null && (
              <span aria-hidden="true" className="pointer-events-none absolute inset-y-0 left-[calc(14rem+0.75rem)]" style={{ right: barra }}>
                <span className="absolute inset-y-0 w-px bg-texto/50" style={{ left: p(cursor) }} />
              </span>
            )}
          </div>
        </div>

        <div className="mt-3 flex flex-wrap gap-x-4 gap-y-1 text-ayuda text-suave">
          <span className="flex items-center gap-1.5">
            <span aria-hidden="true" className={`h-2.5 w-4 rounded-[3px] border ${BARRA.Claude}`} />
            Claude
          </span>
          <span className="flex items-center gap-1.5">
            <span aria-hidden="true" className={`h-2.5 w-4 rounded-[3px] border ${BARRA.Codex}`} />
            Codex
          </span>
          <span className="flex items-center gap-1.5">
            <span aria-hidden="true" className={`h-2.5 w-4 rounded-[3px] border ${BARRA.sesion}`} />
            {t("sesión")}
          </span>
          <span className="flex items-center gap-1.5" title={resp?.fuente ? t("La regla está en {fuente}", { fuente: resp.fuente }) : t("Tope por omisión: AGENTS.md no lo escribe")}>
            <span aria-hidden="true" className="h-2.5 w-4 rounded-[3px]" style={{ background: RAYADO }} />
            {t("pasado de {min} min", { min: resp?.tope_min ?? 30 })}
          </span>
          <span className="flex items-center gap-1.5">
            <span aria-hidden="true" className="h-3 w-[3px] rounded-full bg-error" />
            {t("editó un archivo que otro agente vivo también editaba")}
          </span>
        </div>
      </section>

      {/* El agente elegido: alto fijo (lo que cambia en vivo no mueve el diseño). */}
      <section aria-label={t("Agente elegido")}className="grid min-h-[76px] grid-cols-[repeat(auto-fill,minmax(11rem,1fr))] items-center gap-3 rounded-tarjeta border border-borde-fuerte bg-superficie px-4 py-3">
        {sel ? (
          <>
            <span className="flex min-w-0 flex-col">
              <span className="text-ayuda text-tenue">{sel.tipo === "sesion" ? t("Sesión") : sel.tipo === "codex" ? t("Llamada a Codex") : t("Agente")}</span>
              <span className="truncate font-semibold" title={sel.nombre}>
                {sel.nombre}
              </span>
            </span>
            <span className="flex min-w-0 flex-col">
              <span className="text-ayuda text-tenue">{t("Clase y modelo")}</span>
              <span className="dato truncate" title={sel.clase}>
                {sel.clase || "—"}
              </span>
            </span>
            <span className="flex flex-col">
              <span className="text-ayuda text-tenue">{t("De a")}</span>
              <span className="dato">
                {hhmm(sel.a)} – {sel.vivo ? t("sigue") : hhmm(sel.b)}
              </span>
            </span>
            <span className="flex flex-col">
              <span className="text-ayuda text-tenue">{t("Duración")}</span>
              <span className={`dato ${sel.excede ? "text-aviso" : ""}`}>
                {minutos(sel.b - sel.a)}
                {sel.excede ? ` · ${t("pasó de {min}", { min: resp?.tope_min ?? 30 })}` : ""}
              </span>
            </span>
            <span className="flex min-w-0 flex-col">
              <span className="text-ayuda text-tenue">{t("En paralelo con")}</span>
              <span className="truncate" title={enParaleloCon(sel, filas).join(", ")}>
                {enParaleloCon(sel, filas).join(", ") || t("ninguno")}
              </span>
            </span>
            {sel.diario ? (
              <span className="truncate text-ayuda text-tenue" title={TEXTO_DIARIO}>{TEXTO_DIARIO}</span>
            ) : (
              <button
                type="button"
                onClick={() => alAbrirAgente(sel.id)}
                className="h-9 justify-self-start rounded-control border border-borde bg-elevado px-3 text-rotulo transition-colors hover:border-borde-fuerte"
              >
                {sel.tipo === "sesion" ? t("Abrir la sesión") : t("Abrir el agente")}
              </button>
            )}
          </>
        ) : (
          <span className="text-tenue">{t("Elige una barra para ver al agente.")}</span>
        )}
      </section>
    </main>
  );
}
