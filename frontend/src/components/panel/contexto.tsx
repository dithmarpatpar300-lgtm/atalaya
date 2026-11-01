"use client";

import { type KeyboardEvent, type MouseEvent, useEffect, useId, useRef, useState } from "react";
import { type Agente, api, type ContextoAgente } from "@/lib/api";
import { decimal, t, tServidor } from "@/lib/i18n";
import { hora } from "@/lib/formato";
import { tokensCortos } from "@/lib/hilos";
import { repartirRotulos } from "@/lib/rotulos";

type Pestana = "contexto" | "lleno" | "cache" | "peticiones";

/** Un decimal siempre (coma en español, punto en inglés): «108,0 k». Para la lectura del gráfico, que no debe bailar (lo que cambia en vivo no mueve el diseño). */
const fijo = (n: number) => decimal(n / 1000, 1) + " k";
const pct = (n: number, de: number) => (de ? decimal((n / de) * 100, 1) : decimal(0, 1)) + " %";

function Totales({ a }: { a: Agente }) {
  const tk = a.tokens;
  const celdas = [
    { k: t("Entrada sin caché"), v: tokensCortos(tk.entrada), nota: t("tokens que no estaban en caché"), color: "bg-texto" },
    { k: t("Caché escrita"), v: tokensCortos(tk.cache_escrita), nota: t("lo nuevo de cada petición"), color: "bg-acento" },
    { k: t("Caché leída"), v: tokensCortos(tk.cache_leida), nota: t("el prefijo reutilizado"), color: "bg-borde-campo" },
    { k: t("Salida"), v: tokensCortos(tk.salida), nota: t("respuestas y llamadas"), color: "bg-opcional" },
  ];
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(10rem,1fr))] gap-2.5">
      {celdas.map((c) => (
        <div key={c.k} className="flex flex-col gap-1 rounded-panel border border-borde px-3.5 py-3">
          <span className="flex items-center gap-2 text-ayuda text-suave">
            <span aria-hidden="true" className={`size-2 rounded-[2px] ${c.color}`} />
            {c.k}
          </span>
          <span className="dato text-[18px]">{c.v}</span>
          <span className="truncate text-[11.5px] text-tenue">{c.nota}</span>
        </div>
      ))}
    </div>
  );
}

/** El gráfico del contexto por petición: etapas, compactaciones y una lectura que sigue al puntero o a las flechas. */
function Grafico({ c }: { c: ContextoAgente }) {
  const base = useId();
  const [hov, setHov] = useState<number | null>(null);
  // El ancho real del gráfico, para repartir los rótulos en píxeles y no en porcentaje.
  const caja = useRef<HTMLDivElement>(null);
  const [ancho, setAncho] = useState(0);
  const hayPeticiones = c.peticiones.length > 0;
  useEffect(() => {
    const el = caja.current;
    if (!el) return;
    const ro = new ResizeObserver(([e]) => setAncho(Math.round(e.contentRect.width)));
    ro.observe(el);
    return () => ro.disconnect();
  }, [hayPeticiones]);
  const ps = c.peticiones;
  const n = ps.length;
  const tope = Math.max(...ps.map((p) => p.contexto), ...c.compactaciones.map((k) => k.antes), 1) * 1.1;
  const X = (i: number) => (n > 1 ? (i / (n - 1)) * 600 : 300);
  const Y = (v: number) => 160 - (v / tope) * 150;
  const linea = ps.map((p, i) => `${i ? "L" : "M"} ${X(i).toFixed(1)} ${Y(p.contexto).toFixed(1)}`).join(" ");

  // Etapas: una línea donde cambia (o donde empieza un turno). Los rótulos se reparten en
  // píxeles para que nunca se monten; el que no cabe se lee al pasar el puntero.
  const cortes: { i: number; texto: string }[] = [];
  ps.forEach((p, i) => {
    if (i === 0) return;
    const q = ps[i - 1];
    if (p.etapa !== q.etapa || p.turno !== q.turno) cortes.push({ i, texto: p.etapa || t("Turno {n}", { n: p.turno }) });
  });
  const rotulos = repartirRotulos(
    cortes.map((k) => (X(k.i) / 600) * ancho),
    ancho,
  );

  const h = hov === null ? null : Math.max(0, Math.min(n - 1, hov));
  const p = h === null ? null : ps[h];
  const delta = p && h ? p.contexto - ps[h - 1].contexto : 0;

  function mover(e: MouseEvent<HTMLDivElement>) {
    const r = e.currentTarget.getBoundingClientRect();
    const i = Math.round(Math.max(0, Math.min(1, (e.clientX - r.left) / r.width)) * (n - 1));
    if (i !== hov) setHov(i);
  }
  function tecla(e: KeyboardEvent<HTMLDivElement>) {
    const actual = hov ?? n - 1;
    const paso = { ArrowLeft: -1, ArrowRight: 1, PageDown: 10, PageUp: -10 }[e.key];
    let i: number | null = null;
    if (paso) i = actual + paso;
    else if (e.key === "Home") i = 0;
    else if (e.key === "End") i = n - 1;
    else if (e.key === "Escape") {
      setHov(null);
      return;
    }
    if (i === null) return;
    e.preventDefault();
    setHov(Math.max(0, Math.min(n - 1, i)));
  }

  if (n === 0) return <p className="m-0 text-rotulo text-suave">{t("Todavía no hay peticiones.")}</p>;
  const xPct = h === null ? 0 : (X(h) / 600) * 100;
  return (
    <div className="flex min-w-0 flex-col gap-2">
      <div className="flex justify-between text-ayuda text-suave">
        <span>{n === 1 ? t("Contexto por petición (1 petición)") : t("Contexto por petición ({n} peticiones)", { n })}</span>
        <span className="dato">{t("máximo {valor}", { valor: tokensCortos(Math.max(...ps.map((q) => q.contexto))) })}</span>
      </div>
      <div
        ref={caja}
        tabIndex={0}
        role="img"
        aria-label={t("Contexto por petición, de {de} a {a}. Pasa el puntero o usa las flechas para ver cada petición.", { de: tokensCortos(ps[0].contexto), a: tokensCortos(ps[n - 1].contexto) })}
        aria-describedby={`${base}-lectura`}
        onMouseMove={mover}
        onMouseLeave={() => setHov(null)}
        onFocus={() => setHov(n - 1)}
        onBlur={() => setHov(null)}
        onKeyDown={tecla}
        className="relative h-40 cursor-crosshair rounded-panel border border-borde bg-hundido"
      >
        <svg aria-hidden="true" width="100%" height="160" viewBox="0 0 600 160" preserveAspectRatio="none" className="block rounded-panel">
          <path d="M0 40H600M0 80H600M0 120H600" stroke="var(--borde)" strokeWidth="1" fill="none" vectorEffect="non-scaling-stroke" />
          {cortes.map((k) => (
            <path key={`e${k.i}`} d={`M ${X(k.i)} 0 V 160`} stroke="var(--borde-fuerte)" strokeWidth="1" strokeDasharray="3 3" fill="none" vectorEffect="non-scaling-stroke" />
          ))}
          {c.compactaciones.map((k, i) => (
            <path key={`c${i}`} d={`M ${X(Math.max(0, k.peticion - 0.5))} 0 V 160`} stroke="var(--aviso)" strokeWidth="2" fill="none" vectorEffect="non-scaling-stroke" />
          ))}
          <path d={`${linea} L ${X(n - 1)} 160 L ${X(0)} 160 Z`} fill="var(--acento-suave)" opacity="0.9" />
          <path d={linea} stroke="var(--acento)" strokeWidth="2" fill="none" vectorEffect="non-scaling-stroke" strokeLinejoin="round" />
        </svg>
        {rotulos.map((r) => (
          <span
            key={`r${cortes[r.k].i}`}
            className="pointer-events-none absolute top-1.5 truncate rounded-chip bg-hundido/80 px-1 text-[11px] text-tenue"
            style={{ left: r.left, maxWidth: r.max }}
          >
            {cortes[r.k].texto}
          </span>
        ))}
        {c.compactaciones.map((k, i) => {
          // Una sola línea, y dentro del gráfico aunque la compactación caiga junto a un borde.
          const x = (X(Math.max(0, k.peticion - 0.5)) / 600) * 100;
          const mover = x > 85 ? "translateX(calc(-100% - 4px))" : x < 15 ? "translateX(4px)" : "translateX(-50%)";
          return (
          <span
            key={`cr${i}`}
            className="dato pointer-events-none absolute bottom-1.5 rounded-chip bg-aviso-suave px-1 text-[10.5px] whitespace-nowrap text-aviso"
            style={{ left: `${x}%`, transform: mover }}
          >
            {tokensCortos(k.antes)} → {tokensCortos(k.despues)}
          </span>
          );
        })}
        {p && h !== null && (
          <>
            <span aria-hidden="true" className="pointer-events-none absolute inset-y-0 w-px bg-acento opacity-55" style={{ left: `${xPct}%` }} />
            <span
              aria-hidden="true"
              className="pointer-events-none absolute size-2.5 rounded-full bg-acento shadow-[0_0_0_4px_var(--acento-suave)]"
              style={{ left: `calc(${xPct}% - 5px)`, top: Y(p.contexto) - 5 }}
            />
            {/* La ficha va al costado de la guía, nunca sobre el punto, y tiene ancho fijo. */}
            <div
              aria-hidden="true"
              className="capa pointer-events-none absolute top-2 z-10 flex w-[13.5rem] flex-col gap-1 rounded-panel px-3 py-2.5 text-ayuda"
              style={{ left: `${xPct}%`, transform: xPct > 50 ? "translateX(calc(-100% - 14px))" : "translateX(14px)" }}
            >
              <div className="flex justify-between text-suave">
                <span>{t("Petición {i} de {n}", { i: h + 1, n })}</span>
                <span className="dato">{hora(new Date(p.hora))}</span>
              </div>
              <div className="dato text-base text-texto">
                {fijo(p.contexto)} <span className="text-ayuda text-suave">· {pct(p.contexto, c.ventana)}</span>
              </div>
              <div className="flex justify-between text-suave">
                <span>{t("Desde la anterior")}</span>
                <span className="dato text-texto">{h ? `${delta >= 0 ? "+" : "−"}${fijo(Math.abs(delta))}` : t("primera")}</span>
              </div>
              <div className="flex justify-between text-suave">
                <span>{t("Salida de la respuesta")}</span>
                <span className="dato text-opcional">↑ {fijo(p.salida)}</span>
              </div>
              <div className="flex justify-between gap-2 text-suave">
                <span>{t("Etapa")}</span>
                <span className="truncate text-acento-fuerte">{p.etapa || t("turno {n}", { n: p.turno })}</span>
              </div>
            </div>
          </>
        )}
      </div>
      {/* La lectura: una línea de alto fijo, columnas de ancho fijo (lo que cambia en vivo no mueve el diseño). */}
      <div className="flex h-6 items-center overflow-hidden rounded-chip bg-elevado/55 px-2.5 text-ayuda whitespace-nowrap text-tenue">
        {p && h !== null ? (
          <>
            <span className="dato w-[8.5rem] shrink-0">{t("Petición {i} de {n}", { i: String(h + 1).padStart(String(n).length, " "), n })}</span>
            <span className="dato w-[3.25rem] shrink-0 text-suave">{hora(new Date(p.hora))}</span>
            <span className="dato w-[9.5rem] shrink-0 text-texto">
              {fijo(p.contexto)} · {pct(p.contexto, c.ventana)}
            </span>
            <span className="dato w-[4.75rem] shrink-0 text-texto">{h ? `${delta >= 0 ? "+" : "−"}${fijo(Math.abs(delta))}` : t("primera")}</span>
            <span className="min-w-0 flex-1 truncate text-acento-fuerte">{p.resumen || p.etapa}</span>
          </>
        ) : (
          <span className="min-w-0 flex-1 truncate">{t("Pasa el puntero por la curva, o enfócala y usa las flechas, para ver cada petición.")}</span>
        )}
      </div>
      <span id={`${base}-lectura`} aria-live="polite" className="sr-only">
        {p && h !== null ? t("Petición {i} a las {hora}: {contexto}, {pct} de la ventana. {resumen}", { i: h + 1, hora: hora(new Date(p.hora)), contexto: fijo(p.contexto), pct: pct(p.contexto, c.ventana), resumen: p.resumen }) : ""}
      </span>
      <p className="m-0 h-[18px] truncate text-ayuda text-tenue" title={notaCompactaciones(c)}>
        {notaCompactaciones(c)}
      </p>
    </div>
  );
}

function notaCompactaciones(c: ContextoAgente): string {
  if (c.compactaciones.length === 0) return t("Sin compactaciones. Cuando ocurren, se marcan en ámbar con los tokens de antes y después.");
  const k = c.compactaciones[c.compactaciones.length - 1];
  const vars = { hora: hora(new Date(k.hora)), antes: tokensCortos(k.antes), despues: tokensCortos(k.despues), motivo: k.motivo === "auto" ? t("automática") : t("pedida") };
  return c.compactaciones.length === 1
    ? t("Una compactación; la última a las {hora}: {antes} → {despues} ({motivo}).", vars)
    : t("{n} compactaciones; la última a las {hora}: {antes} → {despues} ({motivo}).", { ...vars, n: c.compactaciones.length });
}

function Medidor({ c }: { c: ContextoAgente }) {
  const ult = c.peticiones[c.peticiones.length - 1];
  if (!ult) return null;
  const libre = Math.max(0, c.ventana - ult.contexto);
  const partes = [
    { k: t("Prefijo reutilizado (caché leída)"), v: ult.cache_leida, color: "bg-borde-campo", borde: "" },
    { k: t("Nuevo en esta petición (caché escrita)"), v: ult.cache_escrita, color: "bg-acento", borde: "" },
    { k: t("Entrada sin caché"), v: ult.entrada, color: "bg-texto", borde: "" },
    { k: t("Libre"), v: libre, color: "bg-transparent", borde: "border border-borde-fuerte" },
  ];
  return (
    <div className="flex min-w-0 flex-col gap-3">
      <div className="flex items-baseline justify-between">
        <span className="font-semibold">{t("{pct} usado", { pct: pct(ult.contexto, c.ventana) })}</span>
        <span className="dato text-rotulo text-suave">{t("{valor} de {max}", { valor: tokensCortos(ult.contexto), max: c.ventana === 1_000_000 ? "1 M" : "200 k" })}</span>
      </div>
      <div role="img" aria-label={t("Contexto de la última petición: {valor} de {max}", { valor: tokensCortos(ult.contexto), max: tokensCortos(c.ventana) })} className="flex h-2.5 gap-0.5 overflow-hidden rounded-full bg-elevado">
        {partes.slice(0, 3).map((p) => (p.v > 0 ? <span key={p.k} className={`h-2.5 ${p.color}`} style={{ width: `${(p.v / c.ventana) * 100}%`, minWidth: 2 }} /> : null))}
      </div>
      <ul className="m-0 flex list-none flex-col gap-1.5 p-0">
        {partes.map((p) => (
          <li key={p.k} className="grid grid-cols-[0.75rem_minmax(0,1fr)_4.5rem_3.5rem] items-center gap-2 text-rotulo">
            <span aria-hidden="true" className={`size-[9px] rounded-[2px] ${p.color} ${p.borde}`} />
            <span className="truncate">{p.k}</span>
            <span className="dato text-right">{tokensCortos(p.v)}</span>
            <span className="dato text-right text-suave">{pct(p.v, c.ventana)}</span>
          </li>
        ))}
      </ul>
      <p className="m-0 text-ayuda text-tenue">
        {t("De la última petición a la API. La ventana se infiere ({detalle}). El reparto que muestra Claude Code (herramientas del sistema, MCP, memoria, skills) no queda en el transcrito: aquí no se inventa.", {
          detalle: c.ventana === 1_000_000 ? t("una petición pasó de 200 k") : t("ninguna pasó de 200 k"),
        })}
      </p>
    </div>
  );
}

function Lleno({ c, alIrAPaso }: { c: ContextoAgente; alIrAPaso: (n: number) => void }) {
  const total = c.lleno.reduce((s, l) => s + l.tokens, 0);
  const max = Math.max(1, ...c.lleno.map((l) => l.tokens));
  const color = ["bg-hilo-1", "bg-acento", "bg-opcional", "bg-ok", "bg-borde-campo", "bg-hilo-2", "bg-hilo-4"];
  return (
    <div className="flex flex-wrap gap-6">
      <div className="flex min-w-0 flex-[1_1_20rem] flex-col gap-2.5">
        <span className="text-ayuda text-suave">{t("Qué llenó el contexto (resultados de herramientas ≈ bytes ÷ 4, y lo que generó la IA)")}</span>
        {c.lleno.map((l, i) => (
          <div key={l.categoria} className="grid grid-cols-[minmax(0,11rem)_minmax(0,1fr)_4.5rem_2.75rem] items-center gap-2.5 text-rotulo">
            <span className="truncate">{t(l.categoria)}</span>
            <span className="h-2 rounded-full bg-elevado">
              <span className={`block h-2 rounded-full ${color[i % color.length]}`} style={{ width: `${(l.tokens / max) * 100}%` }} />
            </span>
            <span className="dato text-right">{tokensCortos(l.tokens)}</span>
            <span className="dato text-right text-suave">{Math.round((l.tokens / total) * 100)} %</span>
          </div>
        ))}
      </div>
      <div className="flex min-w-0 flex-[1_1_20rem] flex-col gap-1.5">
        <span className="text-ayuda text-suave">{t("Los cinco resultados más pesados (llevan a su paso)")}</span>
        {c.pesados.map((p) => (
          <button
            key={p.n}
            type="button"
            onClick={() => alIrAPaso(p.n)}
            className="grid min-h-10 grid-cols-[2.75rem_minmax(0,1fr)_4rem] items-center gap-2.5 rounded-control border border-borde px-2.5 text-left hover:border-borde-fuerte hover:bg-elevado"
          >
            <span className="dato text-ayuda text-tenue">{hora(new Date(p.hora))}</span>
            <span className="truncate text-rotulo">
              {t(p.tipo)} · {p.resumen}
            </span>
            <span className="dato text-right text-rotulo">{tokensCortos(p.tokens)}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

function Cache({ a }: { a: Agente }) {
  const tk = a.tokens;
  const total = tk.cache_leida + tk.cache_escrita + tk.entrada;
  const acierto = total ? (tk.cache_leida / total) * 100 : 0;
  return (
    <div className="flex flex-wrap items-center gap-6">
      <div className="flex shrink-0 flex-col gap-1">
        <span className="dato text-cifra-grande font-medium tracking-tight">{decimal(acierto, 1)} %</span>
        <span className="text-rotulo text-suave">{t("de la entrada salió de la caché")}</span>
      </div>
      <div className="flex min-w-0 flex-[1_1_22rem] flex-col gap-2.5">
        <div
          role="img"
          aria-label={t("Caché leída {leida}, caché escrita {escrita}, sin caché {entrada}", { leida: tokensCortos(tk.cache_leida), escrita: tokensCortos(tk.cache_escrita), entrada: tokensCortos(tk.entrada) })}
          className="flex h-3 gap-0.5 overflow-hidden rounded-full"
        >
          <span className="bg-borde-campo" style={{ width: `${(tk.cache_leida / Math.max(1, total)) * 100}%` }} />
          <span className="bg-acento" style={{ width: `${(tk.cache_escrita / Math.max(1, total)) * 100}%`, minWidth: 3 }} />
          <span className="bg-texto" style={{ width: `${(tk.entrada / Math.max(1, total)) * 100}%`, minWidth: 3 }} />
        </div>
        <p className="m-0 text-rotulo text-suave">
          {t("De cada 100 tokens de entrada, {leidos} se leyeron de la caché (el prefijo que no cambió), {escritos} se escribieron nuevos y casi ninguno entró sin caché. Por eso {tokens} leídos no pesan como {tokens} nuevos.", {
            leidos: Math.round(acierto),
            escritos: Math.round((tk.cache_escrita / Math.max(1, total)) * 100),
            tokens: tokensCortos(tk.cache_leida),
          })}
        </p>
      </div>
    </div>
  );
}

function Peticiones({ c }: { c: ContextoAgente }) {
  const [orden, setOrden] = useState<"recientes" | "pesadas">("recientes");
  const filas = (orden === "recientes" ? [...c.peticiones].reverse() : [...c.peticiones].sort((x, y) => y.cache_escrita + y.entrada - (x.cache_escrita + x.entrada))).slice(0, 10);
  return (
    <div className="flex flex-col gap-2.5">
      <div role="group" aria-label={t("Orden")} className="flex gap-0.5 self-start rounded-control bg-elevado p-[3px]">
        {(
          [
            ["recientes", t("Las últimas")],
            ["pesadas", t("Las que más contexto nuevo escribieron")],
          ] as const
        ).map(([id, nombre]) => (
          <button
            key={id}
            type="button"
            aria-pressed={orden === id}
            onClick={() => setOrden(id)}
            className={`h-7 rounded-chip px-2.5 text-ayuda ${orden === id ? "bg-superficie text-texto shadow-[0_1px_2px_rgb(0_0_0/0.18)]" : "text-suave hover:text-texto"}`}
          >
            {nombre}
          </button>
        ))}
      </div>
      <div className="scroll-interno overflow-x-auto rounded-panel border border-borde">
        <table className="w-full border-collapse text-rotulo">
          <thead>
            <tr className="bg-elevado text-left text-suave">
              <th className="px-3 py-2 font-medium">{t("Hora")}</th>
              <th className="px-2 py-2 font-medium">{t("Petición")}</th>
              <th className="px-2 py-2 text-right font-medium">{t("Contexto")}</th>
              <th className="px-2 py-2 text-right font-medium">{t("Caché escrita")}</th>
              <th className="px-3 py-2 text-right font-medium">{t("Salida")}</th>
            </tr>
          </thead>
          <tbody>
            {filas.map((p, i) => (
              <tr key={`${p.hora}-${i}`} className="border-t border-borde">
                <td className="dato px-3 py-2 text-tenue">{hora(new Date(p.hora))}</td>
                <td className="max-w-0 truncate px-2 py-2">{p.resumen || p.etapa || "—"}</td>
                <td className="dato px-2 py-2 text-right">{tokensCortos(p.contexto)}</td>
                <td className="dato px-2 py-2 text-right">{tokensCortos(p.cache_escrita)}</td>
                <td className="dato px-3 py-2 text-right">{tokensCortos(p.salida)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

/**
 * Contexto y tokens: los totales y cuatro vistas. Contexto: el medidor de la última
 * petición y la curva por petición, con etapas y compactaciones. Qué lo llenó: por herramienta y los resultados más
 * pesados. Caché: cuánto se reutilizó. Por petición: las últimas o las más pesadas.
 */
export function ContextoTokens({ a, version, alIrAPaso }: { a: Agente; version: string; alIrAPaso: (n: number) => void }) {
  const base = useId();
  const [pestana, setPestana] = useState<Pestana>("contexto");
  const [c, setC] = useState<ContextoAgente | null>(null);
  const [error, setError] = useState<string | null>(null);
  const esCodex = a.proveedor === "Codex";

  useEffect(() => {
    if (esCodex) return;
    const control = new AbortController();
    api
      .contexto(a.id, control.signal)
      .then((x) => {
        setC(x);
        setError(null);
      })
      .catch((e: Error) => {
        if (e.name !== "AbortError") setError(tServidor(e.message));
      });
    return () => control.abort();
  }, [a.id, version, esCodex]);

  const pestanas: [Pestana, string][] = [
    ["contexto", t("Contexto")],
    ["lleno", t("Qué lo llenó")],
    ["cache", t("Caché")],
    ["peticiones", t("Por petición")],
  ];
  return (
    <section aria-labelledby={`${base}-t`} className="flex flex-col gap-4 rounded-tarjeta border border-borde bg-superficie px-6 py-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id={`${base}-t`} className="m-0 text-encabezado font-semibold">
          {t("Contexto y tokens")}
        </h2>
        {!esCodex && (
          <div role="tablist" aria-label={t("Vistas de tokens")} className="flex gap-0.5 rounded-control bg-elevado p-[3px]">
            {pestanas.map(([id, nombre]) => (
              <button
                key={id}
                type="button"
                role="tab"
                id={`${base}-${id}`}
                aria-selected={pestana === id}
                aria-controls={`${base}-panel`}
                onClick={() => setPestana(id)}
                className={`h-7 rounded-chip px-3 text-rotulo transition-colors duration-150 ${
                  pestana === id ? "bg-superficie text-texto shadow-[0_1px_2px_rgb(0_0_0/0.18)]" : "text-suave hover:text-texto"
                }`}
              >
                {nombre}
              </button>
            ))}
          </div>
        )}
      </div>
      <Totales a={a} />
      {esCodex ? (
        <p className="m-0 text-rotulo text-suave">
          {a.interactiva
            ? t("Codex informa sus tokens al terminar cada turno: no hay serie por petición. Los pasos de la sesión, con cada orden y su salida, están en la actividad.")
            : t("Codex informa sus tokens una vez, al terminar cada turno: no hay serie por petición. Los pasos de la llamada, con cada orden y su salida, están en la actividad.")}
        </p>
      ) : error ? (
        <p className="m-0 text-rotulo text-error">{t("No se pudo leer el contexto: {error}", { error })}</p>
      ) : !c ? (
        <div className="esqueleto h-40 rounded-panel" aria-label={t("Cargando el contexto")} />
      ) : (
        <div id={`${base}-panel`} role="tabpanel" aria-labelledby={`${base}-${pestana}`}>
          {pestana === "contexto" && (
            <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.6fr)]">
              <Medidor c={c} />
              <Grafico c={c} />
            </div>
          )}
          {pestana === "lleno" && <Lleno c={c} alIrAPaso={alIrAPaso} />}
          {pestana === "cache" && <Cache a={a} />}
          {pestana === "peticiones" && <Peticiones c={c} />}
        </div>
      )}
    </section>
  );
}
