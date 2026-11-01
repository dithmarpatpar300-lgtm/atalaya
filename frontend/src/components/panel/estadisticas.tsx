"use client";

import { useEffect, useState } from "react";
import { api } from "@/lib/api";
import { cuando, tokens } from "@/lib/formato";
import { decimal, idioma, t, tServidor } from "@/lib/i18n";
import { FONDO_PROVEEDOR } from "@/lib/proveedor";
import { MensajeNivel } from "@/components/ui/tarjeta-alerta";
import {
  ancho,
  celdaTexto,
  cifra,
  type ClaseEstadistica,
  escala,
  filasTabla,
  type InformeEstadisticas,
  lectura,
  type MedidaEstadisticas,
  medidaDe,
  miles,
  type PeriodoEstadisticas,
} from "@/lib/estadisticas";

const PERIODOS: { id: PeriodoEstadisticas; nombre: (fase?: string) => string }[] = [
  { id: "semana", nombre: () => t("Semana") },
  { id: "mes", nombre: () => t("Mes") },
  { id: "fase", nombre: (f) => (f ? t("Fase {n}", { n: f.replace(/^F/, "") }) : t("Fase")) },
];

const medidas = (): { id: MedidaEstadisticas; nombre: string }[] => [
  { id: "minutos", nombre: t("Minutos") },
  { id: "salida", nombre: t("Tokens de salida") },
  { id: "nuevos", nombre: t("Tokens nuevos") },
];

const CLAVE = "atalaya:estadisticas";

function leerAjustes(): { periodo: PeriodoEstadisticas; medida: MedidaEstadisticas } {
  try {
    const a = JSON.parse(localStorage.getItem(CLAVE) ?? "{}") as { periodo?: string; medida?: string };
    return {
      periodo: a.periodo === "mes" || a.periodo === "fase" ? a.periodo : "semana",
      medida: a.medida === "salida" || a.medida === "nuevos" ? a.medida : "minutos",
    };
  } catch {
    return { periodo: "semana", medida: "minutos" };
  }
}

const claveDe = (c: ClaseEstadistica) => `${c.clase}/${c.proveedor}`;
const TONO = { ok: "text-ok", aviso: "text-aviso", suave: "text-suave" } as const;

function Segmentado<T extends string>({ etiqueta, opciones, valor, alElegir }: { etiqueta: string; opciones: { id: T; nombre: string }[]; valor: T; alElegir: (v: T) => void }) {
  return (
    <div role="group" aria-label={etiqueta} className="flex gap-0.5 rounded-control bg-elevado p-[3px]">
      {opciones.map((o) => {
        const on = o.id === valor;
        return (
          <button
            key={o.id}
            type="button"
            aria-pressed={on}
            onClick={() => alElegir(o.id)}
            className={`h-[30px] rounded-chip px-3 text-rotulo whitespace-nowrap transition-colors duration-150 ${
              on ? "bg-superficie font-medium text-texto shadow-sm" : "text-suave hover:text-texto"
            }`}
          >
            {o.nombre}
          </button>
        );
      })}
    </div>
  );
}

/**
 * ¿Cuánto cuesta cada clase de tarea?: medianas y p90 por clase de
 * AGENTES.md §2, contra el presupuesto de sus asignaciones y contra la configuración que manda la matriz (§3 y §14.2).
 * Solo cuentan los agentes ya entregados. Sirve para recalibrar la matriz y decidir qué va a Codex.
 */
export function Estadisticas({ proyecto, version, alAbrirAgente }: { proyecto: string; version: number; alAbrirAgente: (id: string) => void }) {
  const [periodo, setPeriodo] = useState<PeriodoEstadisticas>("semana");
  const [medida, setMedida] = useState<MedidaEstadisticas>("minutos");
  const [inf, setInf] = useState<InformeEstadisticas | null>(null);
  const [error, setError] = useState(false);
  const [elegida, setElegida] = useState<string | null>(null);

  useEffect(() => {
    const a = leerAjustes();
    // eslint-disable-next-line react-hooks/set-state-in-effect -- los ajustes guardados se leen después de hidratar
    setPeriodo(a.periodo);
    setMedida(a.medida);
  }, []);

  useEffect(() => {
    if (!proyecto) return;
    const c = new AbortController();
    api
      .estadisticas(proyecto, periodo, c.signal)
      .then((r) => {
        setInf(r);
        setError(false);
      })
      .catch((e: unknown) => {
        if (!(e instanceof DOMException && e.name === "AbortError")) setError(true);
      });
    return () => c.abort();
  }, [proyecto, periodo, version]);

  function guardar(p: PeriodoEstadisticas, m: MedidaEstadisticas) {
    try {
      localStorage.setItem(CLAVE, JSON.stringify({ periodo: p, medida: m }));
    } catch {}
  }

  const clases = inf?.clases ?? [];
  const max = escala(clases, medida);
  const sel = clases.find((c) => claveDe(c) === elegida) ?? clases.find((c) => c.n >= 5) ?? clases[0];
  const lec = sel ? lectura(sel, medida) : null;
  const tot = inf?.totales;
  const opusAlto = tot ? tot.meta_opus > 0 && tot.opus_pct >= tot.meta_opus * 100 : false;
  const kpis = tot
    ? [
        {
          rotulo: t("Agentes"),
          valor: String(tot.agentes),
          nota:
            tot.codex === 1
              ? t("{claude} de Claude y 1 llamada a Codex, ya entregados", { claude: tot.claude })
              : t("{claude} de Claude y {codex} llamadas a Codex, ya entregados", { claude: tot.claude, codex: tot.codex }),
        },
        { rotulo: t("Minutos de agente"), valor: decimal(tot.minutos, 0), nota: t("suma de todos, con solapes") },
        { rotulo: t("Salida"), valor: miles(tot.salida), nota: t("tokens que escribieron los modelos") },
        { rotulo: t("Caché leída"), valor: `${decimal(tot.cache_pct)} %`, nota: t("de la entrada; {nuevos} tokens nuevos", { nuevos: miles(tot.nuevos) }) },
        {
          rotulo: t("En Opus"),
          valor: `${decimal(tot.opus_pct)} %`,
          nota: tot.meta_opus
            ? t("meta: menos del {pct} % ({fuente})", { pct: decimal(tot.meta_opus * 100, 0), fuente: tServidor(tot.fuente_opus ?? "") })
            : t("§12 no fija una meta"),
          color: tot.meta_opus ? (opusAlto ? "text-aviso" : "text-ok") : "",
        },
      ]
    : null;
  const filas = filasTabla(clases);
  const celda = new Map((inf?.celdas ?? []).map((c) => [`${c.clase}/${c.modelo}`, c]));

  return (
    <main id="contenido" tabIndex={-1} aria-label={t("Estadísticas")} className="relative mx-auto flex w-full max-w-[1520px] flex-col gap-5 p-5">
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex min-w-0 flex-[1_1_32rem] flex-col gap-1">
          <h1 className="m-0 text-titulo font-semibold">{t("¿Cuánto cuesta cada clase de tarea?")}</h1>
          <p className="m-0 max-w-3xl text-suave">
            {t("Medianas por clase de AGENTS.md §2, contra el presupuesto de sus asignaciones y la matriz §3. Sirve para recalibrarla y para decidir qué va a Codex.")}
          </p>
        </div>
        <Segmentado
          etiqueta={t("Periodo")}
          opciones={PERIODOS.map((p) => ({ id: p.id, nombre: p.nombre(inf?.fase) }))}
          valor={periodo}
          alElegir={(p) => {
            setPeriodo(p);
            guardar(p, medida);
          }}
        />
      </div>

      {error && !inf && <MensajeNivel nivel="error">{t("No se pudieron leer las estadísticas. Se reintenta con el próximo aviso.")}</MensajeNivel>}

      <div className="grid grid-cols-[repeat(auto-fill,minmax(13rem,1fr))] gap-3">
        {kpis
          ? kpis.map((k) => (
              <div key={k.rotulo} className="flex h-[96px] flex-col gap-0.5 rounded-panel border border-borde bg-superficie p-3.5">
                <span className="text-ayuda text-suave">{k.rotulo}</span>
                <span className={`dato text-[22px] leading-8 ${k.color ?? ""}`}>{k.valor}</span>
                <span className="truncate text-ayuda text-tenue" title={k.nota}>
                  {k.nota}
                </span>
              </div>
            ))
          : [0, 1, 2, 3, 4].map((i) => <div key={i} className="esqueleto h-[96px] rounded-panel" aria-hidden="true" />)}
      </div>

      <section aria-label={t("Mediana por clase")} className="flex flex-col gap-3 rounded-tarjeta border border-borde bg-superficie p-4">
        <div className="flex flex-wrap items-center gap-2.5">
          <h2 className="m-0 text-encabezado font-semibold">{t("Mediana por clase")}</h2>
          <span className="flex-1" />
          <Segmentado
            etiqueta={t("Medida")}
            opciones={medidas()}
            valor={medida}
            alElegir={(m) => {
              setMedida(m);
              guardar(periodo, m);
            }}
          />
        </div>

        {inf && clases.length === 0 && <p className="m-0 px-1 py-4 text-suave">{t("Ningún agente entregado en este periodo. Prueba con «Mes» o con la fase.")}</p>}
        <ul className="m-0 flex list-none flex-col gap-1 p-0">
          {clases.map((c) => {
            const v = medidaDe(c, medida);
            const on = sel && claveDe(c) === claveDe(sel);
            return (
              <li key={claveDe(c)}>
                <button
                  type="button"
                  aria-pressed={!!on}
                  onMouseEnter={() => setElegida(claveDe(c))}
                  onFocus={() => setElegida(claveDe(c))}
                  onClick={() => setElegida(claveDe(c))}
                  aria-label={t("{clase} {nombre}, {proveedor}: mediana {mediana}, p90 {p90}, {n} agentes", {
                    clase: c.clase,
                    nombre: c.nombre,
                    proveedor: c.proveedor,
                    mediana: cifra(medida, v.mediana),
                    p90: cifra(medida, v.p90),
                    n: c.n,
                  })}
                  className={`grid h-9 w-full grid-cols-[3.5rem_minmax(0,13rem)_minmax(0,1fr)_6.5rem] items-center gap-3 rounded-control px-2 text-left max-sm:grid-cols-[3rem_minmax(0,1fr)_5.5rem] ${
                    on ? "bg-seleccion" : "hover:bg-elevado/60"
                  }`}
                >
                  <span className="dato flex items-center gap-1.5 font-semibold">
                    <span aria-hidden="true" className={`size-2 shrink-0 rounded-full ${FONDO_PROVEEDOR[c.proveedor]}`} />
                    {c.clase}
                  </span>
                  <span className="truncate text-ayuda text-suave max-sm:hidden" title={`${c.nombre} · ${c.proveedor}`}>
                    {c.nombre}
                    {c.proveedor === "Codex" ? " · Codex" : ""}
                  </span>
                  <span aria-hidden="true" className="relative h-3.5 rounded-chip bg-elevado">
                    <span className={`absolute inset-y-0 left-0 rounded-chip opacity-30 ${FONDO_PROVEEDOR[c.proveedor]}`} style={{ width: ancho(v.p90, max) }} />
                    <span className={`absolute inset-y-0 left-0 rounded-chip ${FONDO_PROVEEDOR[c.proveedor]}`} style={{ width: ancho(v.mediana, max) }} />
                    {medida === "minutos" && <span className="absolute -inset-y-1 w-0.5 bg-texto" style={{ left: ancho(c.presupuesto, max) }} />}
                  </span>
                  <span className="dato text-right">{cifra(medida, v.mediana)}</span>
                </button>
              </li>
            );
          })}
        </ul>

        <div className="flex flex-wrap gap-x-4 gap-y-1 text-ayuda text-suave">
          <span className="flex items-center gap-1.5">
            <span aria-hidden="true" className="h-2.5 w-3 rounded-[2px] bg-serie-claude" />
            Claude
          </span>
          <span className="flex items-center gap-1.5">
            <span aria-hidden="true" className="h-2.5 w-3 rounded-[2px] bg-serie-codex" />
            Codex
          </span>
          <span className="flex items-center gap-1.5">
            <span aria-hidden="true" className="h-2.5 w-3 rounded-[2px] bg-suave opacity-30" />
            {t("hasta el p90")}
          </span>
          {medida === "minutos" && (
            <span className="flex items-center gap-1.5">
              <span aria-hidden="true" className="h-3.5 w-0.5 bg-texto" />
              {t("presupuesto (las asignaciones, o el tope {tope})", { tope: tServidor(inf?.tope ?? "") })}
            </span>
          )}
        </div>

        {/* La lectura de la clase elegida: alto fijo y columnas fijas, para que nada salte al pasar el puntero (lo que cambia en vivo no mueve el diseño). */}
        <div className="flex flex-col gap-0.5 rounded-control bg-elevado px-3 py-2">
          <div className="dato flex h-5 items-center overflow-hidden text-[12.5px] whitespace-nowrap">
            {sel ? (
              <>
                <span className="w-24 shrink-0 font-semibold">
                  {sel.clase}
                  {sel.proveedor === "Codex" ? " · Codex" : ""}
                </span>
                <span className="w-16 shrink-0 text-suave">n = {sel.n}</span>
                <span className="w-44 shrink-0">{t("mediana {v}", { v: cifra(medida, medidaDe(sel, medida).mediana) })}</span>
                <span className="w-40 shrink-0 text-suave">p90 {cifra(medida, medidaDe(sel, medida).p90)}</span>
                <span className="min-w-0 truncate text-suave" title={sel.matriz_fuente ? tServidor(sel.matriz_fuente) : undefined}>
                  {sel.matriz ? t("matriz {matriz} · {en} de {n}", { matriz: sel.matriz, en: sel.en_matriz, n: sel.n }) : t("sin matriz")}
                </span>
              </>
            ) : (
              <span className="text-tenue">{t("Pasa el puntero por una clase.")}</span>
            )}
          </div>
          <span className={`h-5 truncate text-ayuda leading-5 ${lec ? TONO[lec.tono] : ""}`} title={lec?.texto}>
            {lec?.texto}
          </span>
        </div>

        {/* Los agentes de la clase elegida: de la cifra a quién la hizo. Alto fijo con su propio desplazamiento (lo que cambia en vivo no mueve el diseño). */}
        {sel && (sel.agentes?.length ?? 0) > 0 && (
          <div className="flex flex-col gap-1.5">
            <span className="text-ayuda text-suave">
              {t("Los agentes de {clase}", { clase: sel.clase })}
              {sel.proveedor === "Codex" ? " · Codex" : ""} <span className="dato text-tenue">{sel.agentes!.length}</span>
            </span>
            <ul className="scroll-interno m-0 flex h-[13.5rem] list-none flex-col gap-1 overflow-y-auto p-0">
              {sel.agentes!.map((x) => {
                const pasa = x.presupuesto > 0 && x.minutos > x.presupuesto;
                return (
                  <li key={x.id}>
                    <button
                      type="button"
                      onClick={() => alAbrirAgente(x.id)}
                      className="grid h-9 w-full grid-cols-[minmax(0,1fr)_minmax(0,11rem)_5.5rem_5rem] items-center gap-3 rounded-control bg-hundido px-3 text-left text-rotulo hover:bg-elevado max-sm:grid-cols-[minmax(0,1fr)_5rem]"
                    >
                      <span className="flex min-w-0 items-center gap-2">
                        <span aria-hidden="true" className={`size-2 shrink-0 rounded-full ${FONDO_PROVEEDOR[x.proveedor]}`} />
                        <span className="truncate" title={x.nombre}>
                          {x.nombre}
                        </span>
                      </span>
                      <span className="dato truncate text-ayuda text-suave max-sm:hidden">{x.modelo}</span>
                      <span className={`dato text-right ${pasa ? "text-aviso" : ""}`} title={pasa ? t("Pasó de su presupuesto de {min} min", { min: decimal(x.presupuesto, 0) }) : undefined}>
                        {decimal(x.minutos)} min
                      </span>
                      <span className="dato text-right text-ayuda text-tenue max-sm:hidden">{cuando(new Date(x.fin))}</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          </div>
        )}
      </section>

      <section aria-label={t("Clase por modelo")} className="overflow-hidden rounded-tarjeta border border-borde bg-superficie">
        <div className="overflow-x-auto">
          <table className="w-full min-w-[40rem] border-collapse text-rotulo">
            <caption className="px-4 pt-3.5 pb-1.5 text-left">
              <span className="font-semibold">{t("Minutos medianos por clase y modelo")}</span>{" "}
              <span className="text-ayuda text-tenue">{t("(entre paréntesis, cuántos agentes; en ámbar, fuera de la matriz)")}</span>
            </caption>
            <thead>
              <tr className="text-right text-ayuda text-suave">
                <th scope="col" className="px-4 py-2 text-left font-medium">
                  {t("Clase")}
                </th>
                {(inf?.modelos ?? []).map((m) => (
                  <th key={m} scope="col" className="dato px-3 py-2 font-medium whitespace-nowrap">
                    {m}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {filas.map((cl) => (
                <tr key={cl} className="border-t border-borde">
                  <th scope="row" className="dato px-4 py-2 text-left font-semibold">
                    {cl}
                  </th>
                  {(inf?.modelos ?? []).map((m) => {
                    const c = celda.get(`${cl}/${m}`);
                    const fuera = c && c.matriz && !c.en_matriz;
                    return (
                      <td
                        key={m}
                        title={
                          c
                            ? c.matriz
                              ? c.en_matriz
                                ? t("La matriz manda {matriz}", { matriz: c.matriz })
                                : t("Fuera de la matriz: manda {matriz}", { matriz: c.matriz })
                              : t("La matriz no fija esta clase")
                            : undefined
                        }
                        className={`dato px-3 py-2 text-right whitespace-nowrap ${!c ? "text-tenue" : fuera ? "text-aviso" : ""}`}
                      >
                        {celdaTexto(c)}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {inf?.matriz && <p className="m-0 border-t border-borde px-4 py-2.5 text-ayuda text-tenue">{t("La matriz se lee de {matriz} (§3 para Claude, §14.2 para Codex); no se copia al panel.", { matriz: inf.matriz })}</p>}
      </section>
      {inf && (
        <section aria-label={t("Por semana")} className="overflow-hidden rounded-tarjeta border border-borde bg-superficie">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[48rem] border-collapse text-rotulo">
              <caption className="px-4 pt-3.5 pb-1.5 text-left">
                <span className="block font-semibold">{t("Por semana")}</span>
                <span className="text-ayuda text-tenue">
                  {inf.dias_diario === 1
                    ? t("Las últimas 8 semanas, de lunes a domingo, sin importar el periodo de arriba. Lo que los transcritos ya no tienen sale del diario (1 día guardado).")
                    : t("Las últimas 8 semanas, de lunes a domingo, sin importar el periodo de arriba. Lo que los transcritos ya no tienen sale del diario ({n} días guardados).", { n: inf.dias_diario })}
                </span>
              </caption>
              <thead>
                <tr className="text-right text-ayuda text-suave">
                  {[t("Semana"), t("Agentes"), "Claude", "Codex", "Opus", t("Minutos"), t("Mediana"), t("Tokens nuevos")].map((titulo, i) => (
                    <th key={titulo} scope="col" className={`px-3 py-2 font-medium whitespace-nowrap ${i === 0 ? "text-left" : ""}`}>{titulo}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {inf.semanas.map((s, i) => (
                  <tr key={s.desde} className="h-9 border-t border-borde">
                    <th scope="row" className="px-3 text-left font-semibold whitespace-nowrap">{i === inf.semanas.length - 1 ? t("esta semana") : idioma() === "en" ? `${s.desde.slice(5, 7)}/${s.desde.slice(8, 10)}` : `${s.desde.slice(8, 10)}/${s.desde.slice(5, 7)}`}</th>
                    {[s.agentes, s.claude, s.codex, s.opus, decimal(s.minutos), `${decimal(s.mediana)} min`, tokens(s.nuevos)].map((valor, j) => (
                      <td key={j} className="dato px-3 text-right whitespace-nowrap">{s.agentes === 0 ? "—" : valor}</td>
                    ))}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </main>
  );
}
