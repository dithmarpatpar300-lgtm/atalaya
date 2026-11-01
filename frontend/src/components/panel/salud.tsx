"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { api } from "@/lib/api";
import { hora } from "@/lib/formato";
import { t, tServidor } from "@/lib/i18n";
import { Documento } from "./detalle-agente";
import {
  type Celda,
  type CarpetaTemporal,
  COLOR_MARCA,
  type FilaProceso,
  filaInicial,
  hallazgos,
  type InformeProceso,
  MARCA,
  palabraMarca,
  type Periodo,
  peso,
  puntoDe,
  resumen,
} from "@/lib/salud";

const PERIODOS: { id: Periodo; nombre: (fase?: string) => string }[] = [
  { id: "hoy", nombre: () => t("Hoy") },
  { id: "fase", nombre: (f) => (f ? t("Fase {n}", { n: f.replace(/^F/, "") }) : t("Fase")) },
  { id: "todo", nombre: () => t("Todo") },
];

const CLAVE = "atalaya:salud";

function leerPeriodo(): Periodo {
  try {
    const p = localStorage.getItem(CLAVE);
    return p === "fase" || p === "todo" ? p : "hoy";
  } catch {
    return "hoy";
  }
}

const CHIP_TEMPORAL: Record<CarpetaTemporal["estado"], string> = {
  "en uso": "bg-aviso-suave text-aviso",
  "agente entregado": "bg-error-suave text-error",
  "sin agente": "bg-elevado text-suave",
};

/**
 * La salud del proceso: cada subagente y cada llamada a Codex contra las
 * reglas de AGENTS.md, con la ruta que prueba cada falta, y los temporales que siguen en el disco. El panel solo
 * lee: no corrige ni borra nada, dice dónde mirar.
 */
export function Salud({ proyecto, version, alAbrirAgente }: { proyecto: string; version: number; alAbrirAgente: (id: string, paso?: number) => void }) {
  const [periodo, setPeriodo] = useState<Periodo>("hoy");
  const [inf, setInf] = useState<InformeProceso | null>(null);
  const [error, setError] = useState(false);
  const [elegida, setElegida] = useState<string | null>(null);
  const filasRef = useRef(new Map<string, HTMLButtonElement>());
  const registrar = useCallback((id: string, b: HTMLButtonElement | null) => {
    if (b) filasRef.current.set(id, b);
    else filasRef.current.delete(id);
  }, []);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- el periodo guardado se lee después de hidratar
    setPeriodo(leerPeriodo());
  }, []);

  // Se vuelve a pedir con cada aviso (agentes o temporales). Mientras llega, se queda lo anterior: nada salta.
  useEffect(() => {
    if (!proyecto) return;
    const c = new AbortController();
    api
      .proceso(proyecto, periodo, c.signal)
      .then((r) => {
        setInf(r);
        setError(false);
      })
      .catch((e: unknown) => {
        if (!(e instanceof DOMException && e.name === "AbortError")) setError(true);
      });
    return () => c.abort();
  }, [proyecto, periodo, version]);

  function elegirPeriodo(p: Periodo) {
    setPeriodo(p);
    try {
      localStorage.setItem(CLAVE, p);
    } catch {}
  }

  const filas = inf?.filas ?? [];
  const actual = filaInicial(filas, elegida);
  const sel = filas.find((f) => f.id === actual) ?? null;
  const r = inf ? resumen(inf) : null;
  const titulo = {
    hoy: t("hoy"),
    fase: inf?.fase ? t("fase {n}", { n: inf.fase.replace(/^F/, "") }) : t("la fase en curso"),
    todo: t("todo lo leído"),
  }[periodo];

  function alTecla(e: React.KeyboardEvent) {
    const i = filas.findIndex((f) => f.id === actual);
    let j = i;
    if (e.key === "ArrowDown") j = Math.min(filas.length - 1, i + 1);
    else if (e.key === "ArrowUp") j = Math.max(0, i - 1);
    else if (e.key === "Home") j = 0;
    else if (e.key === "End") j = filas.length - 1;
    else return;
    e.preventDefault();
    const f = filas[j];
    if (!f) return;
    setElegida(f.id);
    filasRef.current.get(f.id)?.focus();
  }

  const kpis = r
    ? [
        { rotulo: t("En regla"), valor: t("{n} de {total}", { n: r.enRegla, total: r.agentes }), nota: t("agentes que cumplen todo lo que les aplica"), color: "text-ok" },
        {
          rotulo: t("Faltas"),
          valor: String(r.faltas),
          nota: r.faltas
            ? r.conFaltas === 1
              ? t("en 1 agente: {reglas}", { reglas: r.reglasConFaltas.join(", ") })
              : t("en {n} agentes: {reglas}", { n: r.conFaltas, reglas: r.reglasConFaltas.join(", ") })
            : t("ninguna"),
          color: r.faltas ? "text-error" : "text-ok",
        },
        {
          rotulo: t("A medias"),
          valor: String(r.medias),
          nota: !r.medias
            ? t("ninguna")
            : r.mediasActivos === r.medias
              ? t("de agentes que siguen trabajando")
              : r.mediasActivos === 0
                ? t("de agentes que ya entregaron")
                : t("{n} de agentes que siguen trabajando", { n: r.mediasActivos }),
          color: r.medias ? "text-aviso" : "text-ok",
        },
        {
          rotulo: t("Temporales en el disco"),
          valor: String(r.temporales),
          nota: r.temporales ? (r.sinActaCerrada ? t("{n} sin acta cerrada", { n: r.sinActaCerrada }) : t("todos con su acta cerrada")) : t("ninguno"),
          color: r.sinActaCerrada ? "text-aviso" : "text-ok",
        },
      ]
    : null;

  return (
    <main id="contenido" tabIndex={-1} aria-label={t("Salud del proceso")} className="relative mx-auto flex w-full max-w-[1520px] flex-col gap-5 p-5">
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex min-w-0 flex-[1_1_32rem] flex-col gap-1">
          <h1 className="m-0 text-titulo font-semibold">{t("Salud del proceso · {periodo}", { periodo: titulo })}</h1>
          <p className="m-0 max-w-3xl text-suave">
            {t("Cada agente contra las reglas de AGENTS.md. El panel solo lee los archivos: no corrige nada, te dice dónde mirar.")}
          </p>
        </div>
        <div role="group" aria-label={t("Periodo")} className="flex gap-0.5 rounded-control bg-elevado p-[3px]">
          {PERIODOS.map((p) => {
            const on = p.id === periodo;
            return (
              <button
                key={p.id}
                type="button"
                aria-pressed={on}
                onClick={() => elegirPeriodo(p.id)}
                className={`h-[30px] min-w-16 rounded-chip px-3 text-rotulo whitespace-nowrap transition-colors duration-150 ${
                  on ? "bg-superficie font-medium text-texto shadow-sm" : "text-suave hover:text-texto"
                }`}
              >
                {p.nombre(inf?.fase)}
              </button>
            );
          })}
        </div>
      </div>

      {error && !inf && <p className="m-0 rounded-panel bg-error-suave px-4 py-3 text-error">{t("No se pudo leer la salud del proceso. Se reintenta con el próximo aviso.")}</p>}

      <div className="grid grid-cols-[repeat(auto-fill,minmax(14rem,1fr))] gap-3">
        {kpis
          ? kpis.map((k) => (
              <div key={k.rotulo} className="flex h-[96px] flex-col gap-0.5 rounded-panel border border-borde bg-superficie p-3.5">
                <span className="text-ayuda text-suave">{k.rotulo}</span>
                <span className={`dato text-[22px] leading-8 ${k.color}`}>{k.valor}</span>
                <span className="truncate text-ayuda text-tenue" title={k.nota}>
                  {k.nota}
                </span>
              </div>
            ))
          : [0, 1, 2, 3].map((i) => <div key={i} className="esqueleto h-[96px] rounded-panel" aria-hidden="true" />)}
      </div>

      <div className="flex flex-col items-start gap-5 xl:flex-row">
        <section aria-label={t("Agentes y reglas")}className="w-full min-w-0 overflow-hidden rounded-tarjeta border border-borde bg-superficie xl:flex-[999_1_0]">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[52rem] table-fixed border-collapse text-rotulo">
              <thead>
                <tr className="text-left text-ayuda text-suave">
                  <th scope="col" className="px-4 py-3 font-medium">
                    {t("Agente")}
                  </th>
                  {(inf?.reglas ?? []).map((g) => (
                    <th
                      key={g.id}
                      scope="col"
                      title={`${tServidor(g.nota)}${g.umbral ? ` (${tServidor(g.umbral)})` : ""} · ${g.fuente ? tServidor(g.fuente) : g.id === "frontera" ? t("no está escrita: se vigila igual, por omisión") : t("no está escrita: no se exige")}`}
                      className="w-[6.5rem] px-1 py-3 text-center font-medium"
                    >
                      <span className="block truncate">{tServidor(g.nombre)}</span>
                      <span className={`dato block truncate text-[10.5px] font-normal ${g.omision ? "text-aviso" : "text-tenue"}`}>
                        {g.fuente ? tServidor(g.fuente) : g.id === "frontera" ? t("por omisión") : t("no escrita")}
                      </span>
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody onKeyDown={alTecla}>
                {inf === null ? (
                  <tr>
                    <td colSpan={8} className="p-3">
                      <div className="esqueleto h-12 rounded-control" aria-label={t("Cargando")} />
                    </td>
                  </tr>
                ) : filas.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="px-4 py-6 text-suave">
                      {t("Ningún subagente ni llamada a Codex en este periodo. Prueba con «{fase}» o «Todo».", { fase: PERIODOS[1].nombre(inf.fase) })}
                    </td>
                  </tr>
                ) : (
                  filas.map((f) => <FilaSalud key={f.id} f={f} elegida={f.id === actual} alElegir={() => setElegida(f.id)} registrar={registrar} />)
                )}
              </tbody>
            </table>
          </div>
          <p className="m-0 border-t border-borde px-4 py-2.5 text-ayuda text-tenue">
            {t("✓ cumple · ✕ falta · ! a medias · ? sin dato (los archivos no lo dicen) · — no aplica (Codex no lleva bitácora ni acta: su informe y REGISTRO.md hacen ese papel)")}
          </p>
        </section>

        <aside aria-label={t("Detalle del agente")} className="flex w-full min-w-0 flex-col gap-3 rounded-tarjeta border border-borde-fuerte bg-superficie p-4 xl:sticky xl:top-[68px] xl:w-[25rem] xl:shrink-0">
          {sel ? <Detalle f={sel} proyecto={proyecto} alAbrirAgente={alAbrirAgente} /> : <p className="m-0 text-suave">{t("Elige un agente para ver qué le falta y dónde mirarlo.")}</p>}
        </aside>
      </div>

      <section aria-label={t("Temporales que siguen en el disco")} className="flex flex-col gap-2.5 rounded-tarjeta border border-borde bg-superficie p-4">
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1">
          <h2 className="m-0 text-encabezado font-semibold">{t("Temporales que siguen en el disco")}</h2>
          <span className="dato truncate text-ayuda text-tenue" title={(inf?.lugares ?? []).join(" · ")}>
            {(inf?.lugares ?? []).join(" · ")}
          </span>
        </div>
        <span className="text-ayuda text-tenue">{t("Las carpetas que los agentes dejaron en el disco. El panel no borra: el borrado sigue siendo del agente o tuyo.")}</span>
        {inf && inf.temporales.length === 0 && <p className="m-0 rounded-panel bg-ok-suave px-3 py-2.5 text-ok">{t("Ninguna carpeta temporal en el disco.")}</p>}
        {(inf?.temporales ?? []).map((c) => (
          <div key={c.ruta} className="grid min-h-14 grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 rounded-panel bg-elevado px-3 py-2 sm:grid-cols-[minmax(0,1fr)_6rem_auto_9.5rem]">
            <span className="flex min-w-0 flex-col">
              <span className="dato truncate" title={c.ruta}>
                {c.ruta}
              </span>
              <span className="truncate text-ayuda text-suave" title={actaDe(c)}>
                {actaDe(c)}
              </span>
            </span>
            <span className="dato text-right text-ayuda text-suave max-sm:hidden">{peso(c.bytes)}</span>
            <span className="dato text-ayuda whitespace-nowrap text-tenue max-sm:hidden">
              {c.parciales ? t("{n} arch. · {p} a medias", { n: c.archivos, p: c.parciales }) : t("{n} arch.", { n: c.archivos })}
            </span>
            <span className="flex justify-end">
              {c.agente ? (
                <button
                  type="button"
                  onClick={() => alAbrirAgente(c.agente!)}
                  title={c.agente_nombre ? t("Ver {nombre}", { nombre: c.agente_nombre }) : t("Ver el agente")}
                  className={`rounded-chip px-2 py-0.5 text-ayuda font-medium whitespace-nowrap hover:underline ${CHIP_TEMPORAL[c.estado]}`}
                >
                  {t(c.estado)}
                </button>
              ) : (
                <span className={`rounded-chip px-2 py-0.5 text-ayuda font-medium whitespace-nowrap ${CHIP_TEMPORAL[c.estado]}`}>{t(c.estado)}</span>
              )}
            </span>
          </div>
        ))}
      </section>
    </main>
  );
}

function actaDe(c: CarpetaTemporal): string {
  if (!c.acta) return c.agente_nombre ? t("Sin acta en temporales/ · {quien}", { quien: c.agente_nombre }) : t("Sin acta en temporales/");
  const estado = c.acta_estado ? t(c.acta_estado) : "";
  return c.agente_nombre
    ? t("Acta {estado}: {acta} · {quien}", { estado, acta: c.acta, quien: c.agente_nombre })
    : t("Acta {estado}: {acta}", { estado, acta: c.acta });
}

function FilaSalud({
  f,
  elegida,
  alElegir,
  registrar,
}: {
  f: FilaProceso;
  elegida: boolean;
  alElegir: () => void;
  registrar: (id: string, b: HTMLButtonElement | null) => void;
}) {
  return (
    <tr className={`border-t border-borde ${elegida ? "bg-seleccion" : "hover:bg-elevado/50"}`}>
      <th scope="row" className="p-0 text-left font-normal">
        <button
          ref={(b) => registrar(f.id, b)}
          type="button"
          tabIndex={elegida ? 0 : -1}
          aria-pressed={elegida}
          onClick={alElegir}
          className="flex h-[52px] w-full items-center gap-2.5 px-4 text-left"
        >
          <span aria-hidden="true" className={`size-2 shrink-0 rounded-full ${puntoDe(f)}`} />
          <span className="flex min-w-0 flex-col">
            <span className="truncate font-medium" title={f.tarea ? `${f.nombre} · ${f.tarea}` : f.nombre}>
              {f.nombre}
            </span>
            <span className="dato truncate text-[11.5px] text-tenue">
              {f.clase || f.proveedor} · {f.activo ? t("trabajando") : hora(new Date(f.inicio))}
            </span>
          </span>
        </button>
      </th>
      {f.celdas.map((c) => (
        <td key={c.regla} className="px-1 text-center">
          <span
            title={`${tServidor(c.titulo)}. ${tServidor(c.texto)}`}
            aria-label={`${tServidor(c.regla)}: ${palabraMarca(c.marca)}`}
            className={`inline-grid h-6 min-w-7 place-items-center rounded-chip px-1.5 font-semibold ${COLOR_MARCA[c.marca].fondo} ${COLOR_MARCA[c.marca].texto}`}
          >
            {MARCA[c.marca].signo}
          </span>
        </td>
      ))}
    </tr>
  );
}

/**
 * Lo que prueba una falta, como enlace: el paso del transcrito (abre al agente en ese paso) o el documento del proyecto
 * (lo abre aquí). Una ruta fuera del proyecto (un transcrito sin paso, una carpeta temporal) queda como texto.
 */
function Prueba({ c, alPaso, alDoc }: { c: Celda; alPaso: (id: string, paso: number) => void; alDoc: (ruta: string) => void }) {
  if (c.agente && (c.paso ?? -1) >= 0) {
    return (
      <button type="button" onClick={() => alPaso(c.agente!, c.paso!)} className="self-start text-left text-ayuda text-acento underline-offset-2 hover:underline">
        {t("Ver el paso {n} en la actividad", { n: c.paso ?? 0 })}
      </button>
    );
  }
  if (!c.ruta) return null;
  if (/^docs\/.+\.md$/.test(c.ruta) || /^[\w.-]+\.md$/.test(c.ruta)) {
    return (
      <button type="button" onClick={() => alDoc(c.ruta!)} className="dato self-start truncate text-left text-ayuda text-acento underline-offset-2 hover:underline" title={c.ruta}>
        {c.ruta}
      </button>
    );
  }
  return (
    <span className="dato truncate text-ayuda text-suave" title={c.ruta}>
      {c.ruta}
    </span>
  );
}

function Detalle({ f, proyecto, alAbrirAgente }: { f: FilaProceso; proyecto: string; alAbrirAgente: (id: string, paso?: number) => void }) {
  const [doc, setDoc] = useState<string | null>(null);
  const hs = hallazgos(f);
  return (
    <>
      <header className="flex items-start gap-3">
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-encabezado font-semibold" title={f.nombre}>
            {f.nombre}
          </span>
          <span className="truncate text-ayuda text-suave" title={f.tarea}>
            {f.tarea || f.clase} · {t(f.estado.toLowerCase())} {f.activo ? "" : hora(new Date(f.ultimo))}
          </span>
        </span>
        <button
          type="button"
          onClick={() => alAbrirAgente(f.id)}
          className="h-(--alto-control-compacto) shrink-0 rounded-control bg-acento px-3 text-rotulo font-medium whitespace-nowrap text-sobre-acento hover:bg-acento-fuerte"
        >
          {t("Ver el agente →")}
        </button>
      </header>
      {hs.map((c) => (
        <div key={c.regla} className={`flex flex-col gap-1 rounded-panel px-3 py-2.5 ${COLOR_MARCA[c.marca].fondo}`}>
          <span className={`flex items-center gap-2 font-medium ${COLOR_MARCA[c.marca].texto}`}>
            <span aria-hidden="true" className="dato">
              {MARCA[c.marca].signo}
            </span>
            {tServidor(c.titulo)}
            <span className="sr-only">({palabraMarca(c.marca)})</span>
          </span>
          <span className="text-rotulo">{tServidor(c.texto)}</span>
          <Prueba c={c} alPaso={(id, n) => alAbrirAgente(id, n)} alDoc={setDoc} />
        </div>
      ))}
      {doc && <Documento proyecto={proyecto} ruta={doc} alCerrar={() => setDoc(null)} />}
      {hs.length === 0 && <p className="m-0 rounded-panel bg-ok-suave px-3 py-2.5 text-ok">{t("Cumple todas las reglas que le aplican.")}</p>}
      <ul className="m-0 flex list-none flex-col gap-1 border-t border-borde p-0 pt-2.5">
        {f.celdas
          .filter((c) => c.marca === "cumple" || c.marca === "no_aplica")
          .map((c) => (
            <li key={c.regla} className="grid grid-cols-[1.25rem_minmax(0,1fr)] gap-1.5 text-ayuda text-suave" title={c.ruta ?? undefined}>
              <span aria-hidden="true" className={`dato ${COLOR_MARCA[c.marca].texto}`}>
                {MARCA[c.marca].signo}
              </span>
              <span className="truncate">
                {tServidor(c.titulo)}. {tServidor(c.texto)}
              </span>
            </li>
          ))}
      </ul>
    </>
  );
}
