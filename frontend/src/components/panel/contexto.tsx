"use client";

import { type KeyboardEvent, type MouseEvent, useEffect, useId, useRef, useState } from "react";
import { type Agente, api, type ContextoAgente } from "@/lib/api";
import { decimal, t, tServidor } from "@/lib/i18n";
import { hora } from "@/lib/formato";
import { tokensCortos } from "@/lib/hilos";
import { repartirRotulos } from "@/lib/rotulos";
import {
  colocarEnCarril,
  compactacionVisible,
  desdeDeRango,
  escalaY,
  type Rango,
  rangoPorOmision,
  rangosPosibles,
  tramos,
  ULTIMAS,
} from "@/lib/contexto-grafico";

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

/** Alto fijo de cada pieza del gráfico, en píxeles: nada se mueve al llegar datos en vivo. */
const ALTO_CARRIL = 20;
const ALTO_CURVA = 160;
/** Ancho de una ficha del carril de compactaciones: cifras de ancho fijo (≈ 0,62 em a 10,5 px), el relleno y la marca. */
const anchoFicha = (texto: string) => Math.ceil(texto.length * 6.6) + 20;

type Compactacion = ContextoAgente["compactaciones"][number];

const motivo = (k: Compactacion) => (k.motivo === "auto" ? t("automática") : t("pedida"));
const textoCompactacion = (k: Compactacion) => `${tokensCortos(k.antes)} → ${tokensCortos(k.despues)}`;
const detalleCompactacion = (k: Compactacion) => `${hora(new Date(k.hora))} · ${textoCompactacion(k)} (${motivo(k)})`;

/**
 * El gráfico del contexto por petición. Tres franjas de alto fijo: arriba el carril de las etapas, al medio la curva,
 * abajo el carril de las compactaciones; los rótulos nunca entran en la curva ni se tocan entre sí (lib/contexto-grafico).
 * Muestra un rango (por omisión, desde la última compactación) y tiene una vista de tabla con todo.
 */
function Grafico({ c }: { c: ContextoAgente }) {
  const base = useId();
  const [hov, setHov] = useState<number | null>(null);
  const [elegido, setElegido] = useState<Rango | null>(null);
  const [vista, setVista] = useState<"grafico" | "tabla">("grafico");
  // El ancho real de la curva, para repartir los rótulos en píxeles y no en porcentaje.
  const caja = useRef<HTMLDivElement>(null);
  const [ancho, setAncho] = useState(0);
  const ps = c.peticiones;
  const n = ps.length;
  const hay = n > 0;
  useEffect(() => {
    const el = caja.current;
    if (!el) return;
    const ro = new ResizeObserver(([e]) => setAncho(Math.round(e.contentRect.width)));
    ro.observe(el);
    return () => ro.disconnect();
  }, [hay, vista]);

  if (n === 0) return <p className="m-0 text-rotulo text-suave">{t("Todavía no hay peticiones.")}</p>;

  // El rango: el que eligió la persona si sigue teniendo sentido; si no, el de omisión (cambia solo al compactarse).
  const posibles = rangosPosibles(n, c.compactaciones);
  const rango = elegido && posibles.includes(elegido) ? elegido : rangoPorOmision(n, c.compactaciones);
  const desde = desdeDeRango(rango, n, c.compactaciones);
  const m = n - desde;
  const vistas = ps.slice(desde);
  const { tope, ticks } = escalaY(Math.max(...vistas.map((p) => p.contexto)));
  // X en el viewBox de 600 y en píxeles; el índice es el de la serie completa.
  const X = (i: number) => (m > 1 ? ((i - desde) / (m - 1)) * 600 : 300);
  const Xpx = (i: number) => (X(i) / 600) * ancho;
  const Y = (v: number) => ALTO_CURVA - (v / tope) * (ALTO_CURVA - 4);
  const linea = vistas.map((p, j) => `${j ? "L" : "M"} ${X(desde + j).toFixed(1)} ${Y(p.contexto).toFixed(1)}`).join(" ");

  // Etapas: un corte donde cambia la etapa o empieza un turno, dentro del rango.
  const cortes = tramos(ps)
    .filter((tr) => tr.desde > desde)
    .map((tr) => ({ i: tr.desde, texto: tr.etapa || t("Turno {n}", { n: tr.turno }) }));
  // Los que no caben con al menos 64 px se quitan (no quedan en «Panel: enlac…»): su nombre está en la ficha y en la tabla.
  const rotulos = repartirRotulos(
    cortes.map((k) => Xpx(k.i)),
    ancho,
    84,
    8,
    64,
  );

  // Compactaciones: solo las que caen dentro del rango, en su carril, agrupadas si chocan.
  const visibles = c.compactaciones.filter((k) => compactacionVisible(k.peticion, desde, n));
  const fuera = c.compactaciones.length - visibles.length;
  const xCompacta = (k: Compactacion) => X(k.peticion - 0.5);
  const grupos = colocarEnCarril(
    visibles.map((k) => ({ x: (xCompacta(k) / 600) * ancho, ancho: anchoFicha(textoCompactacion(k)) })),
    ancho,
    (ms) => anchoFicha(t("{n} compact.", { n: ms.length })),
  );

  const h = hov === null ? null : Math.max(desde, Math.min(n - 1, hov));
  const p = h === null ? null : ps[h];
  const delta = p && h ? p.contexto - ps[h - 1].contexto : 0;
  const compactadaAqui = h === null ? undefined : c.compactaciones.find((k) => k.peticion === h);

  function mover(e: MouseEvent<HTMLDivElement>) {
    const r = e.currentTarget.getBoundingClientRect();
    const i = desde + Math.round(Math.max(0, Math.min(1, (e.clientX - r.left) / r.width)) * (m - 1));
    if (i !== hov) setHov(i);
  }
  function tecla(e: KeyboardEvent<HTMLDivElement>) {
    const actual = h ?? n - 1;
    const paso = { ArrowLeft: -1, ArrowRight: 1, PageDown: 10, PageUp: -10 }[e.key];
    let i: number | null = null;
    if (paso) i = actual + paso;
    else if (e.key === "Home") i = desde;
    else if (e.key === "End") i = n - 1;
    else if (e.key === "Escape") {
      setHov(null);
      return;
    }
    if (i === null) return;
    e.preventDefault();
    setHov(Math.max(desde, Math.min(n - 1, i)));
  }
  function elegir(r: Rango) {
    setElegido(r);
    setHov(null);
  }

  const nombreRango: Record<Rango, string> = {
    compactacion: t("Desde la compactación"),
    ultimas: t("Últimas {n}", { n: ULTIMAS }),
    todas: t("Todas"),
  };
  const xPct = h === null ? 0 : (X(h) / 600) * 100;
  const fmt = (v: number) => (v >= 1_000_000 ? decimal(v / 1_000_000, v % 1_000_000 ? 2 : 0) + " M" : Math.round(v / 1000) + " k");
  const boton = (activo: boolean) =>
    `h-7 shrink-0 rounded-chip px-2.5 text-ayuda whitespace-nowrap ${activo ? "bg-superficie text-texto shadow-[0_1px_2px_rgb(0_0_0/0.18)]" : "text-suave hover:text-texto"}`;

  return (
    // Los colores de serie, validados con el script de dataviz sobre el fondo hundido (claro y oscuro).
    <div className="flex min-w-0 flex-col gap-2 [--serie-compacta:#8a5a00] [--serie-contexto:#5b21b6] oscuro:[--serie-compacta:#c08a0c] oscuro:[--serie-contexto:#8b5cf6]">
      <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1.5">
        <span className="text-ayuda text-suave">
          {n === 1 ? t("Contexto por petición (1 petición)") : t("Contexto por petición ({n} peticiones)", { n })}
        </span>
        <div className="flex flex-wrap items-center gap-1.5">
          {posibles.length > 1 && (
            <div role="group" aria-label={t("Rango del gráfico")} className="flex gap-0.5 rounded-control bg-elevado p-[3px]">
              {posibles.map((r) => (
                <button key={r} type="button" aria-pressed={rango === r} onClick={() => elegir(r)} className={boton(rango === r)}>
                  {nombreRango[r]}
                </button>
              ))}
            </div>
          )}
          <div role="group" aria-label={t("Ver como")} className="flex gap-0.5 rounded-control bg-elevado p-[3px]">
            <button type="button" aria-pressed={vista === "grafico"} onClick={() => setVista("grafico")} className={boton(vista === "grafico")}>
              {t("Gráfico")}
            </button>
            <button type="button" aria-pressed={vista === "tabla"} onClick={() => setVista("tabla")} className={boton(vista === "tabla")}>
              {t("Tabla")}
            </button>
          </div>
        </div>
      </div>

      {vista === "tabla" ? (
        <TablaContexto c={c} />
      ) : (
        <div className="grid grid-cols-[2.75rem_minmax(0,1fr)] gap-x-1.5">
          {/* Carril de etapas: fuera de la curva, una línea, sin choques. El nombre entero va en `title`. */}
          <span aria-hidden="true" />
          <div className="relative overflow-hidden" style={{ height: ALTO_CARRIL }} data-carril="etapas">
            {rotulos.map((r) => (
              <span
                key={`r${cortes[r.k].i}`}
                title={cortes[r.k].texto}
                data-rotulo="etapa"
                className="absolute top-0.5 truncate border-l border-dashed border-borde-fuerte pl-1 text-[11px] leading-4 text-tenue"
                style={{ left: r.left, maxWidth: r.max }}
              >
                {cortes[r.k].texto}
              </span>
            ))}
          </div>

          {/* Eje vertical: a la izquierda, fuera de la curva. */}
          <div aria-hidden="true" className="dato relative text-right text-[10.5px] text-tenue" style={{ height: ALTO_CURVA }}>
            {ticks.map((v) => (
              <span key={v} className="absolute right-0 leading-3" style={{ top: Y(v) - 6 }}>
                {fmt(v)}
              </span>
            ))}
          </div>
          <div
            ref={caja}
            tabIndex={0}
            role="img"
            aria-label={t("Contexto por petición, de {de} a {a}. Pasa el puntero o usa las flechas para ver cada petición.", { de: tokensCortos(vistas[0].contexto), a: tokensCortos(ps[n - 1].contexto) })}
            aria-describedby={`${base}-lectura`}
            onMouseMove={mover}
            onMouseLeave={() => setHov(null)}
            onFocus={() => setHov(n - 1)}
            onBlur={() => setHov(null)}
            onKeyDown={tecla}
            className="relative cursor-crosshair rounded-panel border border-borde bg-hundido"
            style={{ height: ALTO_CURVA }}
          >
            <svg aria-hidden="true" width="100%" height={ALTO_CURVA} viewBox={`0 0 600 ${ALTO_CURVA}`} preserveAspectRatio="none" className="block rounded-panel">
              <path d={ticks.map((v) => `M0 ${Y(v).toFixed(1)}H600`).join("")} stroke="var(--borde)" strokeWidth="1" fill="none" vectorEffect="non-scaling-stroke" />
              {rotulos.map((r) => (
                <path key={`e${cortes[r.k].i}`} d={`M ${X(cortes[r.k].i)} 0 V ${ALTO_CURVA}`} stroke="var(--borde-fuerte)" strokeWidth="1" strokeDasharray="3 3" fill="none" vectorEffect="non-scaling-stroke" />
              ))}
              <path d={`${linea} L ${X(n - 1)} ${ALTO_CURVA} L ${X(desde)} ${ALTO_CURVA} Z`} fill="var(--acento-suave)" opacity="0.9" />
              {visibles.map((k) => (
                <path key={`c${k.hora}`} d={`M ${xCompacta(k)} 0 V ${ALTO_CURVA}`} stroke="var(--serie-compacta)" strokeWidth="2" fill="none" vectorEffect="non-scaling-stroke" />
              ))}
              <path d={linea} stroke="var(--serie-contexto)" strokeWidth="2" fill="none" vectorEffect="non-scaling-stroke" strokeLinejoin="round" />
            </svg>
            {p && h !== null && (
              <>
                <span aria-hidden="true" className="pointer-events-none absolute inset-y-0 w-px bg-acento opacity-55" style={{ left: `${xPct}%` }} />
                <span
                  aria-hidden="true"
                  className="pointer-events-none absolute size-2.5 rounded-full bg-[var(--serie-contexto)] shadow-[0_0_0_2px_var(--fondo-hundido),0_0_0_5px_var(--acento-suave)]"
                  style={{ left: `calc(${xPct}% - 5px)`, top: Y(p.contexto) - 5 }}
                />
                {/* La ficha va al costado de la guía, nunca sobre el punto, y tiene ancho y filas fijos. */}
                <div
                  aria-hidden="true"
                  className="capa pointer-events-none absolute top-1.5 z-10 flex w-[13.5rem] flex-col gap-1 rounded-panel px-3 py-2 text-ayuda"
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
                    <span className="dato text-texto">↑ {fijo(p.salida)}</span>
                  </div>
                  <div className="flex justify-between gap-2 text-suave">
                    <span>{t("Compactación")}</span>
                    <span className="dato truncate text-texto">{compactadaAqui ? textoCompactacion(compactadaAqui) : "—"}</span>
                  </div>
                  <div className="flex justify-between gap-2 text-suave">
                    <span>{t("Etapa")}</span>
                    <span className="truncate text-texto">{p.etapa || t("turno {n}", { n: p.turno })}</span>
                  </div>
                </div>
              </>
            )}
          </div>

          {/* Carril de compactaciones: fichas que no se tocan; las que chocan se agrupan y el detalle va en `title` y en la tabla. */}
          <span aria-hidden="true" />
          <div className="relative overflow-hidden" style={{ height: ALTO_CARRIL }} data-carril="compactaciones">
            {grupos.map((g) => {
              const ks = g.miembros.map((j) => visibles[j]);
              const texto = ks.length === 1 ? textoCompactacion(ks[0]) : t("{n} compact.", { n: ks.length });
              return (
                <span
                  key={ks[0].hora}
                  title={ks.map(detalleCompactacion).join("\n")}
                  data-rotulo="compactacion"
                  className="dato absolute top-[3px] flex h-[15px] items-center gap-1 overflow-hidden rounded-chip bg-elevado px-1 text-[10.5px] whitespace-nowrap text-suave"
                  style={{ left: g.left, width: g.ancho }}
                >
                  <span aria-hidden="true" className="h-2.5 w-[3px] shrink-0 rounded-[1px] bg-[var(--serie-compacta)]" />
                  <span className="truncate">{texto}</span>
                </span>
              );
            })}
          </div>

          {/* Eje horizontal: la hora del principio y del final del rango. */}
          <span aria-hidden="true" />
          <div aria-hidden="true" className="dato flex h-4 justify-between text-[10.5px] leading-4 text-tenue">
            <span>{hora(new Date(vistas[0].hora))}</span>
            <span>{hora(new Date(ps[n - 1].hora))}</span>
          </div>
        </div>
      )}

      {/* La leyenda: la identidad nunca va solo en el color. */}
      <ul className="m-0 flex list-none flex-wrap gap-x-3.5 gap-y-1 p-0 text-ayuda text-suave">
        <li className="flex items-center gap-1.5">
          <span aria-hidden="true" className="h-0.5 w-3.5 rounded-full bg-[var(--serie-contexto)]" />
          {t("Contexto de cada petición")}
        </li>
        <li className="flex items-center gap-1.5">
          <span aria-hidden="true" className="h-2.5 w-[3px] rounded-[1px] bg-[var(--serie-compacta)]" />
          {t("Compactación (antes → después)")}
        </li>
        <li className="flex items-center gap-1.5">
          <span aria-hidden="true" className="h-2.5 border-l border-dashed border-borde-fuerte" />
          {t("Cambio de etapa o de turno")}
        </li>
      </ul>

      {/* La lectura: una línea de alto fijo, columnas de ancho fijo (lo que cambia en vivo no mueve el diseño). */}
      <div className="flex h-6 items-center overflow-hidden rounded-chip bg-elevado/55 px-2.5 text-ayuda whitespace-nowrap text-tenue">
        {p && h !== null ? (
          <>
            <span className="dato w-[8.5rem] shrink-0">{t("Petición {i} de {n}", { i: String(h + 1).padStart(String(n).length, " "), n })}</span>
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
      <p className="m-0 h-[18px] truncate text-ayuda text-tenue" title={notaCompactaciones(c, vista === "grafico" ? fuera : 0)}>
        {notaCompactaciones(c, vista === "grafico" ? fuera : 0)}
      </p>
    </div>
  );
}

/**
 * La tabla del gráfico, con todo lo que el rango no muestra: cada etapa con su tramo de peticiones y cada compactación,
 * también las de antes de la serie. Lo más reciente arriba; alto fijo con desplazamiento interno.
 */
function TablaContexto({ c }: { c: ContextoAgente }) {
  const ps = c.peticiones;
  type Fila = { orden: number; clave: string; tipo: "etapa" | "compactacion"; hora: string; nombre: string; tramo: string; valor: string };
  const filas: Fila[] = [
    ...tramos(ps).map((tr) => ({
      orden: tr.desde,
      clave: `e${tr.desde}`,
      tipo: "etapa" as const,
      hora: ps[tr.desde].hora,
      nombre: tr.etapa || t("Turno {n}", { n: tr.turno }),
      tramo: tr.hasta - tr.desde === 1 ? String(tr.desde + 1) : `${tr.desde + 1}–${tr.hasta}`,
      valor: `${tokensCortos(ps[tr.desde].contexto)} → ${tokensCortos(ps[tr.hasta - 1].contexto)}`,
    })),
    ...c.compactaciones.map((k) => ({
      // Antes de su primera petición posterior; las de antes de la serie quedan al fondo.
      orden: k.peticion - 0.5,
      clave: `c${k.hora}`,
      tipo: "compactacion" as const,
      hora: k.hora,
      nombre: `${t("Compactación")} (${motivo(k)})`,
      // Entre la petición k y la k + 1 (contando desde 1).
      tramo: k.peticion <= 0 ? t("antes de la serie") : `${k.peticion} | ${k.peticion + 1}`,
      valor: textoCompactacion(k),
    })),
  ].sort((a, b) => b.orden - a.orden);
  return (
    <div className="scroll-interno overflow-auto rounded-panel border border-borde" style={{ height: ALTO_CARRIL * 2 + ALTO_CURVA + 16 }}>
      <table className="w-full border-collapse text-rotulo">
        <caption className="sr-only">{t("Etapas y compactaciones de la sesión")}</caption>
        <thead className="sticky top-0 bg-elevado">
          <tr className="text-left text-suave">
            <th className="px-3 py-1.5 font-medium">{t("Hora")}</th>
            <th className="px-2 py-1.5 font-medium">{t("Etapa o compactación")}</th>
            <th className="px-2 py-1.5 text-right font-medium">{t("Peticiones")}</th>
            <th className="px-3 py-1.5 text-right font-medium">{t("Contexto")}</th>
          </tr>
        </thead>
        <tbody>
          {filas.map((f) => (
            <tr key={f.clave} className="border-t border-borde">
              <td className="dato px-3 py-1.5 whitespace-nowrap text-tenue">{hora(new Date(f.hora))}</td>
              <td className="max-w-0 px-2 py-1.5">
                <span className="flex min-w-0 items-center gap-1.5">
                  {f.tipo === "compactacion" && <span aria-hidden="true" className="h-2.5 w-[3px] shrink-0 rounded-[1px] bg-[var(--serie-compacta)]" />}
                  <span className="truncate" title={f.nombre}>
                    {f.nombre}
                  </span>
                </span>
              </td>
              <td className="dato px-2 py-1.5 text-right whitespace-nowrap text-suave">{f.tramo}</td>
              <td className="dato px-3 py-1.5 text-right whitespace-nowrap">{f.valor}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function notaCompactaciones(c: ContextoAgente, fuera: number): string {
  if (c.compactaciones.length === 0) return t("Sin compactaciones. Cuando ocurren, se marcan en ámbar con los tokens de antes y después.");
  const k = c.compactaciones[c.compactaciones.length - 1];
  const vars = { hora: hora(new Date(k.hora)), antes: tokensCortos(k.antes), despues: tokensCortos(k.despues), motivo: motivo(k) };
  const texto =
    c.compactaciones.length === 1
      ? t("Una compactación; la última a las {hora}: {antes} → {despues} ({motivo}).", vars)
      : t("{n} compactaciones; la última a las {hora}: {antes} → {despues} ({motivo}).", { ...vars, n: c.compactaciones.length });
  return fuera > 0 ? `${texto} ${fuera === 1 ? t("Una queda fuera del rango: está en la tabla.") : t("{n} quedan fuera del rango: están en la tabla.", { n: fuera })}` : texto;
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
