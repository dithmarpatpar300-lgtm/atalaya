"use client";

import { useEffect, useId, useState } from "react";
import { IconoCerrar, IconoCheck } from "@/components/ui/iconos";
import { api } from "@/lib/api";
import { t, tServidor } from "@/lib/i18n";
import {
  archivosQueTiene,
  chipNivel,
  cifraAvance,
  COLOR_TRAMO,
  cuentas,
  type FiltroPortafolio,
  filtrar,
  fraseFalta,
  hiloEnTexto,
  lineaFase,
  type ProyectoPortafolio,
  proyectoElegido,
  saludHoy,
  tramosHilo,
} from "@/lib/portafolio";

const FILTROS: { id: FiltroPortafolio; nombre: string }[] = [
  { id: "todos", nombre: "Todos" },
  { id: "vivos", nombre: "Con agentes vivos" },
  { id: "falta", nombre: "Les falta algo" },
];

const VACIO: Record<FiltroPortafolio, string> = {
  todos:
    "Todavía no hay proyectos registrados. Pídele a Claude Code, dentro de un proyecto, «registra este proyecto en Atalaya» (la skill atalaya-proyecto), o agrega su línea a proyectos.json.",
  vivos: "Ningún proyecto tiene agentes vivos ahora.",
  falta: "A ningún proyecto le falta nada del contrato.",
};
// Las frases de arriba son claves del diccionario: se pasan por t() al dibujarlas.

/**
 * La portada «Tus proyectos»: una tarjeta por proyecto de proyectos.json, con su avance, su
 * hilo, su contrato, la salud de hoy y las consultas a Atalaya; a la derecha, el contrato del elegido archivo por
 * archivo y la acción siguiente. Vuelve a pedir el portafolio con cada `version`; mientras llega, se queda lo anterior.
 */
export function Proyectos({ version, alAbrirProyecto }: { version: number; alAbrirProyecto: (nombre: string, seccion?: string) => void }) {
  const [ps, setPs] = useState<ProyectoPortafolio[] | null>(null);
  const [error, setError] = useState(false);
  const [filtro, setFiltro] = useState<FiltroPortafolio>("todos");
  const [elegido, setElegido] = useState<string | null>(null);
  const idFiltro = useId();

  useEffect(() => {
    const c = new AbortController();
    api
      .portafolio(c.signal)
      .then((r) => {
        setPs(r);
        setError(false);
      })
      .catch((e: unknown) => {
        if (!(e instanceof DOMException && e.name === "AbortError")) setError(true);
      });
    return () => c.abort();
  }, [version]);

  const todos = ps ?? [];
  const n = cuentas(todos);
  const visibles = filtrar(todos, filtro);
  const sel = proyectoElegido(visibles, elegido);

  return (
    <main id="contenido" tabIndex={-1} aria-label={t("Tus proyectos")} className="relative mx-auto flex w-full max-w-[1520px] flex-col gap-5 p-4 sm:p-5">
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex min-w-0 flex-[1_1_28rem] flex-col gap-1">
          <h1 className="m-0 text-titulo font-semibold">{t("Tus proyectos")}</h1>
          <p className="m-0 max-w-3xl text-suave">
            {t("Todos tus proyectos de un vistazo. Cada tarjeta sale de los archivos de su proyecto; donde falta un archivo, lo dice en vez de suponer.")}
          </p>
        </div>
        <span title={t("Llega con la skill atalaya-proyecto")} className="inline-flex">
          <button
            type="button"
            disabled
            title={t("Llega con la skill atalaya-proyecto")}
            className="h-(--alto-control) cursor-not-allowed rounded-control border border-borde bg-superficie px-3.5 text-rotulo font-medium text-tenue"
          >
            {t("Proyecto nuevo")}
          </button>
        </span>
      </div>

      {error && <p className="m-0 rounded-panel bg-error-suave px-4 py-3 text-error">{t("No se pudo leer el portafolio. Se reintenta con el próximo aviso.")}</p>}

      <div className="flex flex-wrap items-center gap-2.5">
        <span id={idFiltro} className="text-rotulo text-suave">
          {t("Mostrar")}
        </span>
        <div role="group" aria-labelledby={idFiltro} className="flex max-w-full flex-wrap gap-0.5 rounded-control bg-elevado p-[3px]">
          {FILTROS.map((f) => {
            const on = f.id === filtro;
            return (
              <button
                key={f.id}
                type="button"
                aria-pressed={on}
                onClick={() => setFiltro(f.id)}
                className={`flex h-[30px] shrink-0 items-center gap-1.5 rounded-chip px-3 text-rotulo whitespace-nowrap transition-colors duration-150 ${
                  on ? "bg-superficie font-medium text-texto shadow-sm" : "text-suave hover:text-texto"
                }`}
              >
                {t(f.nombre)}
                <span className="dato inline-block min-w-[2ch] text-right text-ayuda text-tenue">{ps ? n[f.id] : "–"}</span>
              </button>
            );
          })}
        </div>
      </div>

      <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_380px]">
        <section aria-label={t("Proyectos")} className="min-w-0">
          {!ps ? (
            <div className="grid gap-3 sm:grid-cols-2" aria-hidden="true">
              {[0, 1, 2, 3].map((i) => (
                <div key={i} className="esqueleto h-[196px] rounded-tarjeta" />
              ))}
            </div>
          ) : visibles.length === 0 ? (
            <p className="m-0 flex h-[196px] items-center justify-center rounded-tarjeta border border-dashed border-borde px-4 text-center text-suave">{t(VACIO[filtro])}</p>
          ) : (
            <ul className="m-0 grid list-none gap-3 p-0 sm:grid-cols-2">
              {visibles.map((p) => (
                <li key={p.nombre} className="min-w-0">
                  <Tarjeta p={p} on={sel?.nombre === p.nombre} alElegir={() => setElegido(p.nombre)} />
                </li>
              ))}
            </ul>
          )}
        </section>

        <aside aria-label={t("Contrato del proyecto elegido")} aria-live="polite" className="min-w-0 lg:sticky lg:top-[68px]">
          {sel ? <Contrato p={sel} alAbrir={(s) => alAbrirProyecto(sel.nombre, s)} /> : <div className={ps ? "hidden" : "esqueleto h-[420px] rounded-tarjeta"} aria-hidden="true" />}
        </aside>
      </div>
    </main>
  );
}

function Tarjeta({ p, on, alElegir }: { p: ProyectoPortafolio; on: boolean; alElegir: () => void }) {
  const av = cifraAvance(p.avance);
  const fase = lineaFase(p);
  const salud = saludHoy(p.salud);
  const tiene = archivosQueTiene(p);
  const tramos = tramosHilo(p.avance);
  const vivosTexto = p.vivos === 1 ? t("1 vivo") : t("{n} vivos", { n: p.vivos });
  const consultas = p.consultas === 1 ? t("1 consulta a Atalaya hoy") : t("{n} consultas a Atalaya hoy", { n: p.consultas });
  const contrato = t("{n} de {total}", { n: tiene, total: p.contrato.length });
  return (
    <button
      type="button"
      aria-pressed={on}
      onClick={alElegir}
      aria-label={t("{nombre}. {fase}. {nivel}. Avance {avance}. Contrato {contrato}. Salud hoy {salud}. {consultas}.{vivos}", {
        nombre: p.nombre,
        fase,
        nivel: chipNivel(p.nivel),
        avance: av.cifra === "—" ? av.nota : `${av.cifra}, ${av.nota}`,
        contrato,
        salud: salud.cifra === "—" ? t("sin agentes") : t("{cifra} en regla", { cifra: salud.cifra }),
        consultas,
        vivos: p.vivos > 0 ? ` ${vivosTexto}.` : "",
      })}
      className={`flex h-[196px] w-full flex-col justify-between rounded-tarjeta border bg-superficie p-4 text-left transition-colors duration-150 ${
        on ? "border-acento shadow-[0_0_0_1px_var(--acento)]" : "border-borde hover:border-borde-fuerte"
      }`}
    >
      <span className="flex w-full min-w-0 items-start gap-3">
        <span aria-hidden="true" className="dato flex size-8 shrink-0 items-center justify-center rounded-control bg-elevado text-ayuda font-semibold text-texto">
          {p.sigla}
        </span>
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-encabezado leading-5 font-semibold" title={p.nombre}>
            {p.nombre}
          </span>
          <span className="truncate text-ayuda leading-4 text-suave" title={fase}>
            {fase}
          </span>
        </span>
        <span className="flex shrink-0 flex-col items-end gap-1">
          <span
            className={`flex h-[22px] items-center rounded-chip px-2 text-ayuda whitespace-nowrap ${p.nivel > 0 ? "bg-elevado text-suave" : "bg-aviso-suave text-aviso"}`}
          >
            {chipNivel(p.nivel)}
          </span>
          <span className={`flex h-4 items-center gap-1.5 text-ayuda whitespace-nowrap text-ok ${p.vivos > 0 ? "" : "invisible"}`}>
            <span aria-hidden="true" className="brillo size-1.5 rounded-full bg-current" />
            <span className="dato">{vivosTexto}</span>
          </span>
        </span>
      </span>

      <span className="flex h-9 w-full min-w-0 items-baseline gap-2.5">
        <span className={`text-[30px] leading-9 font-semibold tracking-[-0.01em] [font-variant-numeric:proportional-nums] ${p.avance ? "text-texto" : "text-tenue"}`}>
          {av.cifra}
        </span>
        <span className="min-w-0 truncate text-rotulo text-suave" title={p.avance ? t("{nota}, calculado de {ruta}", { nota: av.nota, ruta: p.avance.ruta }) : av.nota}>
          {av.nota}
        </span>
      </span>

      <span aria-hidden="true" title={hiloEnTexto(p.avance)} className="flex h-1.5 w-full gap-[2px]">
        {tramos.map((tr) => (
          <span key={tr.tipo} className="h-full min-w-[3px] rounded-full" style={{ flex: `${tr.peso} 1 0`, background: COLOR_TRAMO[tr.tipo] }} />
        ))}
      </span>

      <span className="grid w-full grid-cols-4 gap-2">
        <Dato rotulo={t("Contrato")} valor={contrato} titulo={t("{n} de {total} archivos del contrato", { n: tiene, total: p.contrato.length })} />
        <Dato rotulo={t("Salud hoy")} valor={salud.cifra} titulo={salud.titulo} />
        <Dato
          rotulo={t("Consultas")}
          valor={String(p.consultas)}
          titulo={p.consultas === 1 ? t("1 consulta a Atalaya por MCP desde la medianoche") : t("{n} consultas a Atalaya por MCP desde la medianoche", { n: p.consultas })}
        />
        <Dato
          rotulo={t("Te esperan")}
          valor={String(p.decisiones ?? 0)}
          titulo={p.decisiones === 1 ? t("1 pregunta de sus sesiones sin responder") : t("{n} preguntas de sus sesiones sin responder", { n: p.decisiones ?? 0 })}
          aviso={(p.decisiones ?? 0) > 0}
        />
      </span>
    </button>
  );
}

function Dato({ rotulo, valor, titulo, aviso = false }: { rotulo: string; valor: string; titulo: string; aviso?: boolean }) {
  return (
    <span className="flex min-w-0 flex-col" title={titulo}>
      <span className="truncate text-ayuda text-tenue">{rotulo}</span>
      <span className={`dato truncate text-cuerpo leading-5 ${aviso ? "font-semibold text-aviso" : "text-texto"}`}>{valor}</span>
    </span>
  );
}

function Contrato({ p, alAbrir }: { p: ProyectoPortafolio; alAbrir: (seccion?: string) => void }) {
  const tiene = archivosQueTiene(p);
  const frase = fraseFalta(p);
  return (
    <div className="flex flex-col gap-3.5 rounded-tarjeta border border-borde bg-superficie p-4">
      <div className="flex min-w-0 flex-col gap-0.5">
        <h2 className="m-0 truncate text-encabezado font-semibold" title={p.nombre}>
          {p.nombre}
        </h2>
        <span className="dato truncate text-ayuda text-suave" title={p.ruta}>
          {p.ruta}
        </span>
      </div>

      <div className="flex flex-col gap-1.5">
        <div className="flex h-5 items-center justify-between gap-2">
          <h3 className="m-0 text-rotulo font-medium text-suave">{t("Contrato de trazabilidad")}</h3>
          <span className="dato text-ayuda text-tenue">{t("{n} de {total}", { n: tiene, total: p.contrato.length })}</span>
        </div>
        <ul className="m-0 flex list-none flex-col overflow-hidden rounded-panel border border-borde p-0">
          {p.contrato.map((c) => (
            <li key={c.id} className="grid h-8 grid-cols-[4.75rem_minmax(0,1fr)] items-center gap-2 border-b border-borde px-2.5 last:border-b-0">
              {c.esta ? (
                <span className="flex items-center gap-1 text-ayuda font-medium text-ok">
                  <IconoCheck className="size-3.5 shrink-0" />
                  {t("está")}
                </span>
              ) : (
                <span className="flex items-center gap-1 text-ayuda font-medium text-error">
                  <IconoCerrar className="size-3.5 shrink-0" />
                  {t("falta")}
                </span>
              )}
              <span className={`dato truncate text-[12.5px] ${c.esta ? "text-texto" : "text-suave"}`} title={c.ruta === c.nombre ? c.ruta : `${tServidor(c.nombre)} · ${c.ruta}`}>
                {c.esta && c.id === "avance" ? c.ruta : tServidor(c.nombre)}
              </span>
            </li>
          ))}
        </ul>
      </div>

      <p className="m-0 line-clamp-2 h-[42px] text-cuerpo text-texto" title={frase}>
        {frase}
      </p>

      <button
        type="button"
        onClick={() => alAbrir()}
        className="h-(--alto-control) rounded-control bg-acento px-4 text-rotulo font-medium text-sobre-acento transition-colors duration-150 hover:bg-acento-fuerte"
      >
        {t("Abrir el proyecto")}
      </button>
      {/* El mismo proyecto en las otras secciones: el avance, la salud de hoy, su línea y lo que te pregunta. */}
      <nav aria-label={t("{nombre} en otras secciones", { nombre: p.nombre })} className="grid grid-cols-2 gap-1.5">
        {[
          ["avance", t("Avance")],
          ["salud", t("Salud")],
          ["linea", t("Línea")],
          ["decisiones", (p.decisiones ?? 0) > 0 ? t("Te esperan {n}", { n: p.decisiones ?? 0 }) : t("Decisiones")],
        ].map(([s, nombre]) => (
          <button
            key={s}
            type="button"
            onClick={() => alAbrir(s)}
            className={`h-8 truncate rounded-control border px-2.5 text-rotulo transition-colors hover:bg-elevado ${s === "decisiones" && (p.decisiones ?? 0) > 0 ? "border-aviso text-aviso" : "border-borde"}`}
          >
            {nombre}
          </button>
        ))}
      </nav>
    </div>
  );
}
