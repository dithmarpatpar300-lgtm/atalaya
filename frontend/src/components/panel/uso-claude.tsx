"use client";

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { IconoRecargar } from "@/components/ui/iconos";
import { api, type EstadoCuotas } from "@/lib/api";
import { cuando, porcentaje } from "@/lib/formato";
import { t } from "@/lib/i18n";
import {
  camino,
  escala,
  etiquetaCorta,
  etiquetaVentana,
  filasTabla,
  frescura,
  marcasTiempo,
  masCercano,
  PERIODOS,
  pctOGuion,
  proximaConsulta,
  type Reinicio,
  type Serie,
  separarEtiquetas,
  seriesDe,
  type Tono,
  trazos,
  type UsoClaude,
  vistaVentana,
} from "@/lib/uso-claude";

// Los colores de serie del gráfico, validados con el script de dataviz (todas las parejas, claro y oscuro): los dos de
// serie del panel (globals.css) y un naranja. El color sigue a la ventana: 5 horas, la semana, y la de cada modelo.
const TRAZO = ["stroke-serie-claude", "stroke-serie-codex", "stroke-[#c2410c] oscuro:stroke-[#d96a24]"] as const;
const RELLENO = ["fill-serie-claude", "fill-serie-codex", "fill-[#c2410c] oscuro:fill-[#d96a24]"] as const;
const TONO: Record<Tono, string> = { normal: "text-suave", aviso: "text-aviso", error: "text-error", tenue: "text-tenue" };

const CLAVE = "atalaya:uso-claude";
const ALTO = 176;
const M = { izq: 42, der: 78, arriba: 10, abajo: 22 };

function leerHoras(): number {
  try {
    const h = (JSON.parse(localStorage.getItem(CLAVE) ?? "{}") as { horas?: number }).horas;
    return PERIODOS.some((p) => p.horas === h) ? h! : 168;
  } catch {
    return 168;
  }
}

/**
 * La cuota de Claude según /usage: la pide al montar, cuando cambian las Cuotas (el aviso SSE «cuotas» las vuelve a
 * pedir), al cambiar el periodo y, por su cuenta, al minuto o justo después del próximo reinicio. `uso` es null
 * mientras no hay respuesta o si este servidor no tiene /api/uso-claude: entonces la franja queda como estaba.
 */
export function useUsoClaude(cuotas: EstadoCuotas | null) {
  const [horas, setHorasEstado] = useState(168);
  const [uso, setUso] = useState<UsoClaude | null>(null);
  const [horasDe, setHorasDe] = useState(168);
  const [fallo, setFallo] = useState(false);
  const [pidiendo, setPidiendo] = useState(false);
  const [espera, setEspera] = useState(0);
  const [vuelta, setVuelta] = useState(0);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- el periodo guardado se lee después de hidratar
    setHorasEstado(leerHoras());
  }, []);

  useEffect(() => {
    const c = new AbortController();
    api
      .usoClaude(horas, c.signal)
      .then((u) => {
        setUso(u);
        setHorasDe(horas);
        setFallo(false);
      })
      .catch((e: unknown) => {
        if (e instanceof DOMException && e.name === "AbortError") return;
        setFallo(true);
      });
    return () => c.abort();
  }, [horas, cuotas, vuelta]);

  useEffect(() => {
    if (!uso) return;
    const s = proximaConsulta(uso, Date.now() / 1000);
    const id = window.setTimeout(() => setVuelta((v) => v + 1), s * 1000);
    return () => window.clearTimeout(id);
  }, [uso]);

  const setHoras = useCallback((h: number) => {
    setHorasEstado(h);
    try {
      localStorage.setItem(CLAVE, JSON.stringify({ horas: h }));
    } catch {}
  }, []);

  const leer = useCallback(() => {
    setPidiendo(true);
    api
      .leerUsoClaude()
      .then(({ uso: u, proximaEn }) => {
        setEspera(proximaEn);
        // El historial que se ve es el del periodo elegido: se queda hasta la consulta que sigue.
        if (u) setUso((previo) => ({ ...u, historial: previo?.historial ?? u.historial }));
        setVuelta((v) => v + 1);
      })
      .catch(() => setFallo(true))
      .finally(() => setPidiendo(false));
  }, []);

  // Sin respuesta todavía, o un servidor sin /api/uso-claude (404): la franja de Cuotas, tal como estaba.
  const sinFuente = fallo && !uso;
  return { uso: sinFuente ? null : uso, horas, horasDe, setHoras, fallo, pidiendo, espera, leer };
}

export type UsoClaudeVivo = ReturnType<typeof useUsoClaude>;

function Frescura({ u, ahora, vivo, className = "" }: { u: UsoClaude; ahora: number; vivo: UsoClaudeVivo; className?: string }) {
  const f = frescura(u, ahora, vivo.espera, vivo.fallo);
  return (
    <span className={`dato truncate text-ayuda whitespace-nowrap ${TONO[f.tono]} ${className}`} title={f.titulo}>
      {ahora > 0 ? f.texto : " "}
    </span>
  );
}

function BotonLeer({ u, vivo, ahora }: { u: UsoClaude; vivo: UsoClaudeVivo; ahora: number }) {
  const leyendo = vivo.pidiendo || u.estado === "leyendo";
  const esperando = vivo.espera > 0 && ahora > 0 && ahora < vivo.espera;
  return (
    <button
      type="button"
      onClick={vivo.leer}
      disabled={leyendo || esperando}
      aria-label={t("Leer la cuota de Claude ahora con /usage")}
      title={t("Leer la cuota de Claude ahora con /usage")}
      className="flex h-7 shrink-0 items-center gap-1.5 rounded-control border border-borde px-2.5 text-ayuda text-suave hover:border-borde-fuerte hover:bg-elevado hover:text-texto disabled:cursor-not-allowed disabled:text-tenue"
    >
      <IconoRecargar className={`size-3.5 ${leyendo ? "animate-spin" : ""}`} />
      {t("Leer ahora")}
    </button>
  );
}

/**
 * La línea de Claude en la franja, con /usage: cada ventana con su barra, su % y, debajo, su cuenta atrás; a la
 * derecha, la frescura y «Leer ahora». Todo en huecos de ancho y alto fijos: una lectura nueva no mueve nada.
 */
export function FranjaClaude({ u, ahora, vivo }: { u: UsoClaude; ahora: number; vivo: UsoClaudeVivo }) {
  return (
    <div className="flex min-w-0 basis-full flex-wrap items-center gap-x-3 gap-y-1.5">
      <span aria-hidden="true" className="size-2 shrink-0 rounded-[2px] bg-acento" />
      <span className="w-14 shrink-0 font-semibold">Claude</span>
      <span className="flex min-w-0 basis-full flex-wrap gap-x-3 gap-y-1.5 max-sm:order-last sm:flex-1 sm:basis-0 sm:flex-nowrap">
        {u.ventanas.length ? (
          u.ventanas.map((v, i) => {
            const vista = vistaVentana(v, ahora);
            const nombre = etiquetaVentana(v.clave, v.etiqueta);
            return (
              <span key={v.clave} className="flex min-w-0 flex-[1_1_9rem] flex-col gap-0.5 max-sm:basis-full" title={vista.titulo || undefined}>
                <span className="flex min-w-0 items-center gap-2">
                  <span className="shrink-0 text-ayuda whitespace-nowrap text-suave">{etiquetaCorta(v.clave, v.etiqueta)}</span>
                  <span
                    role="meter"
                    aria-label={t("Claude, {ventana}: {pct} usado", { ventana: nombre, pct: vista.pct === null ? "—" : porcentaje(vista.pct) })}
                    aria-valuemin={0}
                    aria-valuemax={100}
                    aria-valuenow={vista.pct ?? 0}
                    aria-valuetext={vista.pct === null ? vista.cuenta : porcentaje(vista.pct)}
                    className="h-[5px] min-w-10 flex-1 rounded-full bg-elevado"
                  >
                    {vista.pct !== null && (
                      <span className="block h-[5px] rounded-full bg-acento" style={{ width: `${vista.pct}%`, opacity: i === 0 ? 1 : 0.6 }} />
                    )}
                  </span>
                  <span className={`dato w-11 shrink-0 text-right text-ayuda ${vista.pct === null ? "text-tenue" : ""}`}>
                    {vista.pct === null ? "—" : porcentaje(vista.pct)}
                  </span>
                </span>
                <span className={`dato h-4 truncate text-[11.5px] leading-4 ${vista.pct === null ? "text-aviso" : "text-tenue"}`}>{vista.cuenta || " "}</span>
              </span>
            );
          })
        ) : (
          <span className="flex h-[38px] min-w-0 flex-1 items-center text-ayuda text-tenue">{t("Sin lectura de /usage todavía.")}</span>
        )}
      </span>
      <span className="flex shrink-0 items-center gap-2 max-sm:ml-auto">
        <Frescura u={u} ahora={ahora} vivo={vivo} className="w-[12.5rem] text-right" />
        <BotonLeer u={u} vivo={vivo} ahora={ahora} />
      </span>
    </div>
  );
}

/** El detalle de Claude: cada ventana con su reinicio, el historial en un gráfico y su tabla. */
export function DetalleClaude({ u, ahora, vivo }: { u: UsoClaude; ahora: number; vivo: UsoClaudeVivo }) {
  return (
    <div className="flex min-w-0 flex-col gap-3.5 rounded-panel border border-borde p-4 lg:col-span-2">
      <div className="flex flex-wrap items-center gap-2.5">
        <span aria-hidden="true" className="size-2.5 rounded-[2px] bg-acento" />
        <span className="font-semibold">Claude</span>
        <span className="text-ayuda text-suave">{t("por /usage")}</span>
        <span className="flex-1" />
        <Frescura u={u} ahora={ahora} vivo={vivo} className="w-[14rem] text-right" />
      </div>
      <div className="grid gap-x-8 gap-y-3 md:grid-cols-2 xl:grid-cols-3">
        {u.ventanas.map((v) => {
          const vista = vistaVentana(v, ahora);
          const nombre = etiquetaVentana(v.clave, v.etiqueta);
          return (
            <div key={v.clave} className="grid grid-cols-[minmax(0,1fr)_4rem] items-center gap-x-3 gap-y-1">
              <span className="flex min-w-0 items-center gap-2">
                <span className="truncate text-rotulo">{nombre}</span>
                {v.activa && (
                  <span className="shrink-0 rounded-chip bg-acento-suave px-1.5 text-ayuda text-acento-fuerte" title={t("La ventana que manda ahora")}>
                    {t("la que limita")}
                  </span>
                )}
              </span>
              <span className={`dato text-right ${vista.pct === null ? "text-tenue" : ""}`}>{vista.pct === null ? "—" : porcentaje(vista.pct)}</span>
              <div
                className="col-span-2 h-1.5 rounded-full bg-elevado"
                role="meter"
                aria-label={t("{ventana}: {pct} usado", { ventana: nombre, pct: vista.pct === null ? "—" : porcentaje(vista.pct) })}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-valuenow={vista.pct ?? 0}
                aria-valuetext={vista.pct === null ? vista.cuenta : porcentaje(vista.pct)}
              >
                {vista.pct !== null && <div className="h-1.5 rounded-full bg-acento" style={{ width: `${vista.pct}%` }} />}
              </div>
              <span className={`dato col-span-2 h-4 truncate text-ayuda ${vista.pct === null ? "text-aviso" : "text-suave"}`} title={vista.titulo || undefined}>
                {vista.titulo || vista.cuenta || " "}
              </span>
            </div>
          );
        })}
      </div>
      {u.extra_activo && <p className="m-0 text-ayuda text-suave">{t("El uso extra está activado: al llegar al límite, sigue con cargo aparte.")}</p>}
      <Grafico u={u} vivo={vivo} ahora={ahora} />
    </div>
  );
}

function Muestra({ s }: { s: Serie }) {
  return (
    <svg aria-hidden="true" width="16" height="8" className="shrink-0">
      <line x1="1" x2="15" y1="4" y2="4" className={TRAZO[s.color]} strokeWidth="2" strokeLinecap="round" strokeDasharray={s.discontinua ? "4 3" : undefined} />
    </svg>
  );
}

/**
 * El historial: una línea por ventana sobre el tiempo, de 0 a 100 %, con los reinicios marcados. El puntero o las
 * flechas mueven un cursor que lee la hora y el % de cada ventana; la tabla dice lo mismo sin el gráfico.
 */
function Grafico({ u, vivo, ahora }: { u: UsoClaude; vivo: UsoClaudeVivo; ahora: number }) {
  const caja = useRef<HTMLDivElement>(null);
  const [ancho, setAncho] = useState(0);
  const [cursor, setCursor] = useState(-1);
  const [tabla, setTabla] = useState(false);
  useEffect(() => {
    const el = caja.current;
    if (!el) return;
    const ro = new ResizeObserver(() => setAncho(el.clientWidth));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

  const horas = vivo.horasDe;
  // El fin del periodo avanza por minutos, no por segundos: el gráfico no se rehace a cada tic del reloj.
  const minuto = Math.ceil((ahora || u.tomada_en || (u.historial.at(-1)?.t ?? 0)) / 60) * 60;
  const fin = minuto;
  const ini = fin - horas * 3600;
  const series = useMemo(() => seriesDe(u), [u]);
  const claves = useMemo(() => series.map((s) => s.clave), [series]);
  const { tramos, reinicios } = useMemo(() => trazos(u.historial, claves, ini, fin), [u.historial, claves, ini, fin]);
  const pts = useMemo(() => u.historial.filter((p) => p.t >= ini && p.t <= fin).sort((a, b) => a.t - b.t), [u.historial, ini, fin]);
  // En 7 y 30 días, la de 5 horas se reinicia decenas de veces: se marcan solo los reinicios semanales (la bajada de la
  // línea de 5 horas ya se ve, y el cursor la nombra).
  const marcados = horas > 24 ? reinicios.filter((r) => r.claves.some((c) => c !== "session")) : reinicios;
  const ancPlot = Math.max(10, ancho - M.izq - M.der);
  const altPlot = ALTO - M.arriba - M.abajo;
  const e = escala(ini, fin, ancPlot, altPlot);
  const ticks = marcasTiempo(ini, fin, horas);
  const actual = cursor >= 0 && cursor < pts.length ? pts[cursor] : null;

  // Las etiquetas del final de cada línea: el último % leído, en texto, sin pisarse.
  const finales = series
    .map((s) => {
      const tr = tramos.get(s.clave) ?? [];
      const ult = tr.at(-1)?.at(-1);
      return ult ? { s, x: e.x(ult[0]), v: ult[1] } : null;
    })
    .filter((f): f is { s: Serie; x: number; v: number } => !!f);
  const ysFinales = separarEtiquetas(
    finales.map((f) => e.y(f.v)),
    13,
    altPlot,
  );

  function mover(ev: React.PointerEvent<SVGRectElement>) {
    const r = ev.currentTarget.getBoundingClientRect();
    setCursor(masCercano(pts, e.t(ev.clientX - r.left)));
  }

  function tecla(ev: React.KeyboardEvent) {
    if (!pts.length) return;
    const paso = ev.shiftKey ? 10 : 1;
    let n = cursor;
    if (ev.key === "ArrowLeft") n = cursor < 0 ? pts.length - 1 : Math.max(0, cursor - paso);
    else if (ev.key === "ArrowRight") n = cursor < 0 ? pts.length - 1 : Math.min(pts.length - 1, cursor + paso);
    else if (ev.key === "Home") n = 0;
    else if (ev.key === "End") n = pts.length - 1;
    else if (ev.key === "Escape") n = -1;
    else return;
    ev.preventDefault();
    setCursor(n);
  }

  const reinicioCerca = (p: { t: number }): Reinicio | undefined => reinicios.find((r) => Math.abs(r.t - p.t) <= 15 * 60);
  const lectura = actual
    ? [
        cuando(new Date(actual.t * 1000)),
        ...series.map((s) => `${s.etiqueta} ${pctOGuion(actual.v[s.clave])}`),
        ...(reinicioCerca(actual) ? [t("reinicio")] : []),
      ].join(" · ")
    : "";
  const xCursor = actual ? e.x(actual.t) : 0;
  const izqTip = actual ? (xCursor > ancPlot / 2 ? M.izq + xCursor - 12 - 216 : M.izq + xCursor + 12) : 0;
  const filas = tabla ? filasTabla(u.historial, ini, fin, horas) : [];

  return (
    <div className="flex flex-col gap-2 border-t border-borde pt-3">
      <div className="flex flex-wrap items-center gap-x-4 gap-y-2">
        <h3 className="m-0 text-rotulo font-semibold">{t("Historial")}</h3>
        <ul aria-label={t("Leyenda")} className="m-0 flex min-w-0 flex-1 list-none flex-wrap gap-x-4 gap-y-1 p-0 text-ayuda text-suave max-sm:order-last max-sm:basis-full">
          {series.map((s) => (
            <li key={s.clave} className="flex items-center gap-1.5">
              <Muestra s={s} />
              {s.etiqueta}
            </li>
          ))}
          <li className="flex items-center gap-1.5">
            <svg aria-hidden="true" width="10" height="12" className="shrink-0">
              <line x1="5" x2="5" y1="0" y2="12" className="stroke-borde-fuerte" strokeWidth="1.5" strokeDasharray="2 2" />
            </svg>
            {horas > 24 ? t("reinicio semanal") : t("reinicio")}
          </li>
        </ul>
        <div role="group" aria-label={t("Periodo")} className="flex gap-0.5 rounded-control bg-elevado p-[3px]">
          {PERIODOS.map((p) => {
            const on = vivo.horas === p.horas;
            return (
              <button
                key={p.horas}
                type="button"
                aria-pressed={on}
                onClick={() => {
                  setCursor(-1);
                  vivo.setHoras(p.horas);
                }}
                className={`h-[26px] rounded-chip px-2.5 text-ayuda whitespace-nowrap transition-colors duration-150 ${
                  on ? "bg-superficie font-medium text-texto shadow-sm" : "text-suave hover:text-texto"
                }`}
              >
                {t(p.rotulo)}
              </button>
            );
          })}
        </div>
        <button
          type="button"
          aria-pressed={tabla}
          aria-controls="uso-claude-tabla"
          onClick={() => setTabla((v) => !v)}
          className="h-7 rounded-control border border-borde px-2.5 text-ayuda text-suave hover:border-borde-fuerte hover:bg-elevado hover:text-texto"
        >
          {tabla ? t("Ocultar la tabla") : t("Ver como tabla")}
        </button>
      </div>

      <div
        ref={caja}
        tabIndex={0}
        role="group"
        aria-roledescription={t("gráfico")}
        aria-label={t("Historial de la cuota de Claude. Flechas para moverse por las lecturas; Esc para salir.")}
        onKeyDown={tecla}
        onBlur={() => setCursor(-1)}
        className="relative rounded-control focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-acento"
        style={{ height: ALTO }}
      >
        {ancho > 0 && (
          <svg aria-hidden="true" width={ancho} height={ALTO} className="block overflow-visible">
            <g transform={`translate(${M.izq},${M.arriba})`}>
              {[0, 25, 50, 75, 100].map((g) => (
                <g key={g}>
                  <line x1={0} x2={ancPlot} y1={e.y(g)} y2={e.y(g)} className={g === 0 ? "stroke-borde-fuerte" : "stroke-borde"} strokeDasharray={g === 0 ? undefined : "2 4"} />
                  {g % 50 === 0 && (
                    <text x={-6} y={e.y(g)} dy="0.32em" textAnchor="end" className="dato fill-tenue text-[11px]">
                      {g} %
                    </text>
                  )}
                </g>
              ))}
              {ticks.map((tk, i) => {
                const x = e.x(tk.t);
                const ancla = x < 24 ? "start" : x > ancPlot - 24 && i === ticks.length - 1 ? "end" : "middle";
                return (
                  <g key={tk.t}>
                    <line x1={x} x2={x} y1={altPlot} y2={altPlot + 4} className="stroke-borde-fuerte" />
                    <text x={x} y={altPlot + 16} textAnchor={ancla} className="dato fill-tenue text-[11px]">
                      {tk.rotulo}
                    </text>
                  </g>
                );
              })}
              {marcados.map((r) => (
                <line key={r.t} x1={e.x(r.t)} x2={e.x(r.t)} y1={0} y2={altPlot} className="stroke-borde-fuerte" strokeWidth="1.5" strokeDasharray="2 2" />
              ))}
              {series.map((s) => {
                const tr = tramos.get(s.clave) ?? [];
                return (
                  <g key={s.clave}>
                    <path
                      d={camino(tr, e.x, e.y)}
                      fill="none"
                      className={TRAZO[s.color]}
                      strokeWidth="2"
                      strokeLinejoin="round"
                      strokeLinecap="round"
                      strokeDasharray={s.discontinua ? "5 4" : undefined}
                    />
                    {tr
                      .filter((x) => x.length === 1)
                      .map(([[tt, v]]) => (
                        <circle key={tt} cx={e.x(tt)} cy={e.y(v)} r={2.5} className={RELLENO[s.color]} />
                      ))}
                  </g>
                );
              })}
              {finales.map((f, i) => (
                <g key={f.s.clave}>
                  <line x1={ancPlot + 6} x2={ancPlot + 14} y1={ysFinales[i]} y2={ysFinales[i]} className={TRAZO[f.s.color]} strokeWidth="2" strokeLinecap="round" />
                  <text x={ancPlot + 18} y={ysFinales[i]} dy="0.32em" className="dato fill-suave text-[11px]">
                    {porcentaje(f.v)}
                  </text>
                </g>
              ))}
              {actual && (
                <g>
                  <line x1={xCursor} x2={xCursor} y1={0} y2={altPlot} className="stroke-suave" strokeWidth="1" />
                  {series.map((s) => {
                    const v = actual.v[s.clave];
                    if (typeof v !== "number") return null;
                    return <circle key={s.clave} cx={xCursor} cy={e.y(v)} r={4.5} strokeWidth="2" className={`${RELLENO[s.color]} stroke-superficie`} />;
                  })}
                </g>
              )}
              {/* El área que lee el puntero: todo el trazado, más grande que cualquier marca. */}
              <rect
                x={0}
                y={0}
                width={ancPlot}
                height={altPlot}
                fill="transparent"
                className="cursor-crosshair"
                onPointerMove={mover}
                onPointerDown={mover}
                onPointerLeave={() => setCursor(-1)}
              />
            </g>
          </svg>
        )}
        {pts.length === 0 && (
          <p className="pointer-events-none absolute inset-0 m-0 grid place-items-center text-ayuda text-tenue">{t("Todavía no hay lecturas de /usage en este periodo.")}</p>
        )}
        {actual && (
          <div
            aria-hidden="true"
            className="pointer-events-none absolute top-1 flex w-[216px] flex-col gap-1 rounded-control border border-borde bg-capa px-3 py-2 text-ayuda shadow-[var(--sombra-capa)]"
            style={{ left: Math.max(0, izqTip) }}
          >
            <span className="dato font-semibold text-texto">{cuando(new Date(actual.t * 1000))}</span>
            {series.map((s) => (
              <span key={s.clave} className="flex items-center gap-1.5 text-suave">
                <Muestra s={s} />
                <span className="min-w-0 flex-1 truncate">{s.etiqueta}</span>
                <span className="dato text-texto">{pctOGuion(actual.v[s.clave])}</span>
              </span>
            ))}
            {(() => {
              const r = reinicioCerca(actual);
              if (!r) return null;
              const nombres = r.claves.map((c) => series.find((s) => s.clave === c)?.etiqueta ?? c).join(", ");
              return (
                <span className="text-tenue">
                  {r.exacto
                    ? t("Reinicio a las {hora}: {ventanas}", { hora: cuando(new Date(r.t * 1000)), ventanas: nombres })
                    : t("Reinicio entre las {de} y las {a}: {ventanas}", { de: cuando(new Date(r.desde * 1000)), a: cuando(new Date(r.t * 1000)), ventanas: nombres })}
                </span>
              );
            })()}
          </div>
        )}
      </div>
      <p className="sr-only" aria-live="polite">
        {lectura}
      </p>

      {tabla && (
        <div id="uso-claude-tabla" className="max-h-64 overflow-auto rounded-control border border-borde">
          <table className="w-full border-collapse text-ayuda">
            <caption className="sr-only">{t("Historial de la cuota de Claude: la última lectura de cada tramo, de la más nueva a la más vieja.")}</caption>
            <thead className="sticky top-0 bg-superficie">
              <tr className="text-left text-suave">
                <th scope="col" className="px-3 py-1.5 font-medium">
                  {t("Hora")}
                </th>
                {series.map((s) => (
                  <th key={s.clave} scope="col" className="px-3 py-1.5 text-right font-medium">
                    {s.etiqueta}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {filas.map((p) => (
                <tr key={p.t} className="border-t border-borde">
                  <th scope="row" className="dato px-3 py-1 text-left font-normal">
                    {cuando(new Date(p.t * 1000))}
                  </th>
                  {series.map((s) => (
                    <td key={s.clave} className="dato px-3 py-1 text-right">
                      {pctOGuion(p.v[s.clave])}
                    </td>
                  ))}
                </tr>
              ))}
              {filas.length === 0 && (
                <tr>
                  <td colSpan={series.length + 1} className="px-3 py-2 text-tenue">
                    {t("Todavía no hay lecturas de /usage en este periodo.")}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

