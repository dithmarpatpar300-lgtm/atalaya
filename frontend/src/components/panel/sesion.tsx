"use client";

import { useEffect, useMemo, useState } from "react";
import { HiloFase } from "@/components/ui/hilo-fase";
import { Paginador, RegionEstable, usePaginaViva } from "@/components/ui/paginador";
import { api } from "@/lib/api";
import { t, tServidor } from "@/lib/i18n";
import { horas, horasQueFaltan, paqueteInicial, uno } from "@/lib/avance";
import { hora } from "@/lib/formato";
import { efecto, loQueSiguio, pedidoInicial, type ResumenSesion } from "@/lib/sesion";

const tarjeta = "rounded-tarjeta border border-borde bg-superficie p-5";

const TIPO = {
  pedido: { texto: "pedido", clase: "bg-acento-suave text-acento-fuerte" },
  permiso: { texto: "permiso", clase: "bg-ok-suave text-ok" },
} as const;

// Cuántos por página en cada lista: filas de 48 px en el equipo; fichas de commit (cuatro por línea) y rutas en la
// tarjeta angosta; las decisiones, de 44 px, a todo el ancho. La tarjeta angosta lleva un solo tamaño por lista: sin
// «Por página» su pie cabe en una línea, y su alto (4 líneas de commits, 8 rutas) queda a la par del equipo de 10 filas,
// que está al lado: las dos tarjetas miden lo mismo y sus pies, pegados al fondo, quedan en la misma línea.
const TAMANOS_EQUIPO = [10, 25, 50] as const;
const TAMANOS_COMMITS = [16] as const;
const TAMANOS_DOCS = [8] as const;
const TAMANOS_DECISIONES = [8, 25] as const;

/**
 * La sesión principal: una sesión no tiene encargo ni bitácora, así que en su lugar van tus
 * pedidos y lo que siguió a cada uno, el equipo que lanzó, el avance de la fase (la bitácora del coordinador) y lo
 * que tocó de la cuota, de Atalaya y de los documentos de estado. Va dentro de la rejilla del detalle del agente.
 */
export function VistaSesion({
  id,
  version,
  alAbrirAgente,
  alAbrirDoc,
  alIrAPaso,
}: {
  id: string;
  version: string;
  alAbrirAgente: (id: string) => void;
  alAbrirDoc: (ruta: string) => void;
  /** Lleva la actividad de esta sesión a un paso (un commit, una decisión). */
  alIrAPaso: (n: number) => void;
}) {
  const [r, setR] = useState<ResumenSesion | null>(null);
  const [pedido, setPedido] = useState<number | null>(null);
  const [paquete, setPaquete] = useState<string | null>(null);

  // Las listas que crecen con la sesión, por páginas (components/ui/paginador.tsx): lo más nuevo en la página 1.
  // El servidor da el equipo y los commits del más viejo al más nuevo; aquí se ven al revés.
  const equipo = useMemo(() => [...(r?.equipo ?? [])].reverse(), [r]);
  const commits = useMemo(() => [...(r?.commits ?? [])].reverse(), [r]);
  const pEquipo = usePaginaViva(equipo, (m) => m.id, { almacen: "equipo", opciones: TAMANOS_EQUIPO, defecto: 10, reinicio: id });
  const pCommits = usePaginaViva(commits, (c) => String(c.paso), { almacen: "commits", opciones: TAMANOS_COMMITS, defecto: 16, reinicio: id });
  const pDocs = usePaginaViva(r?.documentos ?? [], (d) => d.ruta, { almacen: "documentos", opciones: TAMANOS_DOCS, defecto: 8, reinicio: id });
  const pDecisiones = usePaginaViva(r?.decisiones ?? [], (d) => d.id, { almacen: "decisiones-sesion", opciones: TAMANOS_DECISIONES, defecto: 8, reinicio: id });

  // Se vuelve a pedir con cada paso nuevo de la sesión; mientras llega, se queda lo anterior (nada salta).
  useEffect(() => {
    const c = new AbortController();
    api
      .sesion(id, c.signal)
      .then(setR)
      .catch(() => {});
    return () => c.abort();
  }, [id, version]);

  if (!r) {
    return (
      <section className={`@3xl:col-span-3 ${tarjeta}`}>
        <div className="esqueleto h-40 rounded-control" aria-label={t("Cargando la sesión")} />
      </section>
    );
  }
  const sel = pedidoInicial(r.pedidos, pedido);
  const f = r.avance;
  const pq = f ? paqueteInicial(f, paquete) : null;

  return (
    <>
      <section aria-labelledby="pedidos-t" className={`flex min-w-0 flex-col gap-2 @3xl:col-span-2 ${tarjeta}`}>
        <div className="flex flex-wrap items-baseline justify-between gap-3">
          <h2 id="pedidos-t" className="m-0 text-encabezado font-semibold">
            {t("Tus pedidos")}
          </h2>
          <span className="text-ayuda text-tenue">{t("Una sesión no tiene encargo: tiene conversaciones contigo")}</span>
        </div>
        {r.pedidos.length === 0 ? (
          <p className="m-0 text-suave">{t("Todavía ningún pedido en esta sesión.")}</p>
        ) : (
          <div role="listbox" aria-label={t("Pedidos")} className="flex max-h-[18rem] flex-col gap-0.5 overflow-y-auto">
            {r.pedidos.map((p) => {
              const on = sel?.n === p.n;
              return (
                <button
                  key={p.n}
                  type="button"
                  role="option"
                  aria-selected={on}
                  onClick={() => setPedido(p.n)}
                  className={`grid h-10 shrink-0 grid-cols-[4rem_5rem_minmax(0,1fr)_10rem] items-center gap-3 rounded-control px-3 text-left max-sm:grid-cols-[3.5rem_minmax(0,1fr)] ${
                    on ? "bg-seleccion" : "hover:bg-elevado/60"
                  }`}
                >
                  <span className="dato text-ayuda text-tenue">{hora(new Date(p.hora))}</span>
                  <span className={`rounded-chip text-center text-ayuda max-sm:hidden ${TIPO[p.tipo].clase}`}>{t(TIPO[p.tipo].texto)}</span>
                  <span className="truncate" title={p.texto}>
                    {p.texto}
                  </span>
                  <span className="truncate text-right text-ayuda text-suave max-sm:hidden">{efecto(p)}</span>
                </button>
              );
            })}
          </div>
        )}
        {sel && (
          <dl aria-live="polite" className="m-0 grid h-[88px] grid-cols-[6rem_minmax(0,1fr)] content-center gap-x-3 gap-y-1 rounded-control bg-hundido px-3 text-[13px]">
            <dt className="text-tenue">{t("Qué siguió")}</dt>
            <dd className="m-0 truncate" title={loQueSiguio(sel)}>
              {loQueSiguio(sel)}
            </dd>
            <dt className="text-tenue">{t("Etapas")}</dt>
            <dd className="m-0 truncate text-suave" title={sel.etapas.join(" · ")}>
              {sel.etapas.length ? sel.etapas.join(" · ") : t("sin plan")}
            </dd>
            <dt className="text-tenue">{t("Respondió")}</dt>
            <dd className="m-0 truncate text-suave" title={sel.respuesta}>
              {sel.respuesta || "—"}
            </dd>
          </dl>
        )}
      </section>

      <section aria-labelledby="fase-t" className={`flex min-w-0 flex-col gap-3 ${tarjeta}`}>
        <div className="flex items-baseline justify-between gap-3">
          <h2 id="fase-t" className="m-0 text-encabezado font-semibold">
            {f ? t("Avance de la fase {n}", { n: f.fase.replace(/^F/, "") }) : t("Avance del proyecto")}
          </h2>
          {f && (
            <button type="button" onClick={() => alAbrirDoc(f.ruta)} className="dato text-ayuda text-tenue hover:text-texto" title={t("Abrir {ruta}", { ruta: f.ruta })}>
              {f.fase}/AVANCE.md
            </button>
          )}
        </div>
        {f ? (
          <>
            <div className="flex items-baseline gap-3">
              <span className="text-[44px] leading-none font-semibold tracking-[-0.02em]">{uno(f.calculado)} %</span>
              <span className="text-rotulo text-suave">
                {t("{hechas} de {total} h", { hechas: horas(Math.round(((f.horas * f.calculado) / 100) * 10) / 10), total: horas(f.horas) })}
              </span>
            </div>
            <HiloFase fase={f} elegido={pq} alElegir={setPaquete} />
            <p className="m-0 truncate text-rotulo" title={f.ahora}>
              {t("Ahora: {ahora}", { ahora: f.ahora || "—" })}
            </p>
            <p className="m-0 text-ayuda text-tenue">
              {t("Faltan unas {horas} h de lo estimado. El % lo calcula Atalaya, pesado por horas.", { horas: horas(Math.round(horasQueFaltan(f) * 10) / 10) })}
            </p>
            {f.avisos.length > 0 && (
              <p className="m-0 truncate text-ayuda text-aviso" title={f.avisos.map(tServidor).join("; ")}>
                ⚠ {tServidor(f.avisos[0])}
                {f.avisos.length > 1 ? ` ${t("(y {n} más)", { n: f.avisos.length - 1 })}` : ""}
              </p>
            )}
          </>
        ) : (
          <p className="m-0 text-suave">
            {t("Este proyecto no tiene {ruta}: la bitácora del coordinador. Sin ella, el avance solo se puede calcular a mano.", { ruta: "docs/trazabilidad/F<n>/AVANCE.md" })}
          </p>
        )}
      </section>

      <section aria-labelledby="equipo-t" className={`flex min-w-0 flex-col gap-2 @3xl:col-span-2 ${tarjeta}`}>
        <div className="flex flex-wrap items-baseline justify-between gap-3">
          <h2 id="equipo-t" className="m-0 text-encabezado font-semibold">
            {t("El equipo que lanzó")}
          </h2>
          <span className="text-ayuda text-tenue">{r.equipo.length ? t("{n} agentes y llamadas a Codex", { n: r.equipo.length }) : t("Nadie todavía")}</span>
        </div>
        {r.equipo.length > 0 && (
          <RegionEstable activa={pEquipo.mostrar && pEquipo.pie.paginas > 1} clave={pEquipo.claveAlto}>
          <div className="relative pl-6">
            <span aria-hidden="true" className="absolute top-0 bottom-5 left-[7px] w-px bg-borde-fuerte" />
            {pEquipo.visibles.map((m) => (
              <button
                key={m.id}
                type="button"
                onClick={() => alAbrirAgente(m.id)}
                className="relative mt-1 grid h-11 w-full grid-cols-[minmax(0,1fr)_11rem_3.5rem_7rem] items-center gap-3 rounded-control bg-hundido px-3 text-left hover:bg-elevado max-sm:grid-cols-[minmax(0,1fr)_3.5rem]"
              >
                <span aria-hidden="true" className="absolute top-1/2 -left-[17px] h-px w-[15px] bg-borde-fuerte" />
                <span className="flex min-w-0 items-center gap-2">
                  <span aria-hidden="true" className={`size-2 shrink-0 rounded-full ${m.proveedor === "Codex" ? "bg-serie-codex" : "bg-serie-claude"}`} />
                  <span className="truncate" title={`${m.nombre} · ${m.tarea}`}>
                    <b className="font-semibold">{m.nombre}</b> <span className="text-suave">{m.tarea}</span>
                  </span>
                </span>
                <span className="dato truncate text-ayuda max-sm:hidden">{[m.modelo.replace(/^claude-/, ""), m.effort].filter(Boolean).join("/")}</span>
                <span className="dato text-ayuda text-tenue">{hora(new Date(m.inicio))}</span>
                <span className="truncate text-ayuda text-suave max-sm:hidden">{t(m.estado).toLowerCase()}</span>
              </button>
            ))}
          </div>
          </RegionEstable>
        )}
        {/* Pegado al fondo: queda en la misma línea que el último pie de la tarjeta de al lado. */}
        {pEquipo.mostrar && (
          <div className="mt-auto">
            <Paginador {...pEquipo.pie} nombre={t("El equipo que lanzó")} />
          </div>
        )}
      </section>

      <section aria-labelledby="cuota-t" className={`flex min-w-0 flex-col gap-2 ${tarjeta}`}>
        <h2 id="cuota-t" className="m-0 text-encabezado font-semibold">
          {t("Cuota, Atalaya y documentos")}
        </h2>
        <dl className="m-0 grid grid-cols-[minmax(0,1fr)_auto] gap-x-3 gap-y-1.5 text-[13px]">
          <dt className="text-suave">{t("Lecturas de la cuota")}</dt>
          <dd className="dato m-0 text-right">
            {r.ultima_cuota ? t("{n}, la última {hora}", { n: r.lecturas_cuota, hora: hora(new Date(r.ultima_cuota)) }) : r.lecturas_cuota}
          </dd>
          <dt className="text-suave">{t("Consultas a Atalaya")}</dt>
          <dd className="dato m-0 text-right">{r.consultas}</dd>
          <dt className="text-suave">Commits</dt>
          <dd className="dato m-0 text-right">{r.commits.length}</dd>
        </dl>
        {r.commits.length > 0 && (
          <RegionEstable activa={pCommits.mostrar && pCommits.pie.paginas > 1} clave={pCommits.claveAlto}>
          {/* Cuatro por línea, de ancho fijo: la cifra «Commits» de arriba ya dice qué son, la ficha solo su hora. */}
          <ul aria-label="Commits" className="m-0 grid list-none grid-cols-4 gap-1.5 p-0">
            {pCommits.visibles.map((c) => (
              <li key={c.paso}>
                <button
                  type="button"
                  onClick={() => alIrAPaso(c.paso)}
                  title={c.orden}
                  aria-label={`commit ${hora(new Date(c.hora))}`}
                  className="dato h-7 w-full rounded-control border border-borde text-ayuda hover:border-borde-fuerte"
                >
                  {hora(new Date(c.hora))}
                </button>
              </li>
            ))}
          </ul>
          </RegionEstable>
        )}
        {pCommits.mostrar && <Paginador {...pCommits.pie} nombre="Commits" />}
        <span className="mt-1 border-t border-borde pt-2 text-ayuda text-tenue">{t("Documentos de estado que editó")}</span>
        {r.documentos.length === 0 ? (
          <span className="text-ayuda text-suave">{t("Ninguno.")}</span>
        ) : (
          <RegionEstable activa={pDocs.mostrar && pDocs.pie.paginas > 1} clave={pDocs.claveAlto}>
          <ul className="m-0 flex list-none flex-col gap-1 p-0">
            {pDocs.visibles.map((d) => (
              <li key={d.ruta} className="grid grid-cols-[minmax(0,1fr)_4.5rem] gap-2 text-[12.5px]">
                {/* Uno del proyecto se abre aquí; uno de fuera (la memoria de Claude Code) queda como texto. */}
                {/^[A-Za-z]:|^[~/\\]/.test(d.ruta) ? (
                  <span className="dato truncate" title={d.ruta}>
                    {d.ruta}
                  </span>
                ) : (
                  <button type="button" onClick={() => alAbrirDoc(d.ruta)} title={d.ruta} className="dato truncate text-left text-acento underline-offset-2 hover:underline">
                    {d.ruta}
                  </button>
                )}
                <span className="text-right text-suave">{d.ediciones === 1 ? t("1 vez") : t("{n} veces", { n: d.ediciones })}</span>
              </li>
            ))}
          </ul>
          </RegionEstable>
        )}
        {pDocs.mostrar && (
          <div className="mt-auto">
            <Paginador {...pDocs.pie} nombre={t("Documentos de estado que editó")} />
          </div>
        )}
      </section>

      {/* Lo que la sesión te preguntó y lo que respondiste (la bandeja de Decisiones, solo de esta sesión). */}
      {(r.decisiones?.length ?? 0) > 0 && (
        // A todo el ancho (las tres columnas de la vista): sin hueco al lado y con sitio para la pregunta. Filas de alto
        // fijo y columnas fijas, como los pedidos; la respuesta lleva tope de ancho y se recorta (lo que cambia en vivo no mueve el diseño).
        <section aria-labelledby="decisiones-t" className={`flex min-w-0 flex-col gap-2 @3xl:col-span-3 ${tarjeta}`}>
          <h2 id="decisiones-t" className="m-0 text-encabezado font-semibold">
            {t("Lo que te preguntó")} <span className="dato text-ayuda font-normal text-tenue">{r.decisiones!.length}</span>
          </h2>
          <RegionEstable activa={pDecisiones.mostrar && pDecisiones.pie.paginas > 1} clave={pDecisiones.claveAlto}>
          <ul className="m-0 flex list-none flex-col gap-1 p-0">
            {pDecisiones.visibles.map((d) => {
              const respuesta = d.estado === "pendiente" ? t("te espera") : d.preguntas.map((q) => q.respuesta).filter(Boolean).join(" · ") || t("respondida");
              return (
                <li key={d.id}>
                  <button
                    type="button"
                    disabled={d.paso < 0}
                    onClick={() => alIrAPaso(d.paso)}
                    title={d.paso < 0 ? undefined : t("Ir a ese paso en la actividad")}
                    className="grid h-10 w-full grid-cols-[3.5rem_minmax(0,1fr)_minmax(0,22rem)] items-center gap-3 rounded-control bg-hundido px-3 text-left text-rotulo hover:bg-elevado disabled:hover:bg-hundido max-sm:grid-cols-[3rem_minmax(0,1fr)]"
                  >
                    <span className="dato text-tenue">{hora(new Date(d.hora))}</span>
                    <span className="min-w-0 truncate" title={d.preguntas.map((q) => q.texto).join(" · ")}>
                      {d.preguntas[0]?.texto}
                      {d.preguntas.length > 1 && <span className="text-tenue"> · {t("y {n} más", { n: d.preguntas.length - 1 })}</span>}
                    </span>
                    <span className="flex min-w-0 justify-end max-sm:hidden">
                      <span
                        className={`max-w-full min-w-0 truncate rounded-chip px-2 py-0.5 text-ayuda ${d.estado === "pendiente" ? "bg-aviso-suave text-aviso" : "bg-ok-suave text-ok"}`}
                        title={respuesta}
                      >
                        {respuesta}
                      </span>
                    </span>
                  </button>
                </li>
              );
            })}
          </ul>
          </RegionEstable>
          {pDecisiones.mostrar && <Paginador {...pDecisiones.pie} nombre={t("Lo que te preguntó")} />}
        </section>
      )}
    </>
  );
}
