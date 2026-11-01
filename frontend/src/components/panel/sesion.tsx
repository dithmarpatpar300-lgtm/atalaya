"use client";

import { useEffect, useState } from "react";
import { HiloFase } from "@/components/ui/hilo-fase";
import { api } from "@/lib/api";
import { horas, horasQueFaltan, paqueteInicial, uno } from "@/lib/avance";
import { hora } from "@/lib/formato";
import { efecto, loQueSiguio, pedidoInicial, type ResumenSesion } from "@/lib/sesion";

const tarjeta = "rounded-tarjeta border border-borde bg-superficie p-5";

const TIPO = {
  pedido: { texto: "pedido", clase: "bg-acento-suave text-acento-fuerte" },
  permiso: { texto: "permiso", clase: "bg-ok-suave text-ok" },
} as const;

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
        <div className="esqueleto h-40 rounded-control" aria-label="Cargando la sesión" />
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
            Tus pedidos
          </h2>
          <span className="text-ayuda text-tenue">Una sesión no tiene encargo: tiene conversaciones contigo</span>
        </div>
        {r.pedidos.length === 0 ? (
          <p className="m-0 text-suave">Todavía ningún pedido en esta sesión.</p>
        ) : (
          <div role="listbox" aria-label="Pedidos" className="flex max-h-[18rem] flex-col gap-0.5 overflow-y-auto">
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
                  <span className={`rounded-chip text-center text-ayuda max-sm:hidden ${TIPO[p.tipo].clase}`}>{TIPO[p.tipo].texto}</span>
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
            <dt className="text-tenue">Qué siguió</dt>
            <dd className="m-0 truncate" title={loQueSiguio(sel)}>
              {loQueSiguio(sel)}
            </dd>
            <dt className="text-tenue">Etapas</dt>
            <dd className="m-0 truncate text-suave" title={sel.etapas.join(" · ")}>
              {sel.etapas.length ? sel.etapas.join(" · ") : "sin plan"}
            </dd>
            <dt className="text-tenue">Respondió</dt>
            <dd className="m-0 truncate text-suave" title={sel.respuesta}>
              {sel.respuesta || "—"}
            </dd>
          </dl>
        )}
      </section>

      <section aria-labelledby="fase-t" className={`flex min-w-0 flex-col gap-3 ${tarjeta}`}>
        <div className="flex items-baseline justify-between gap-3">
          <h2 id="fase-t" className="m-0 text-encabezado font-semibold">
            {f ? `Avance de la fase ${f.fase.replace(/^F/, "")}` : "Avance del proyecto"}
          </h2>
          {f && (
            <button type="button" onClick={() => alAbrirDoc(f.ruta)} className="dato text-ayuda text-tenue hover:text-texto" title={`Abrir ${f.ruta}`}>
              {f.fase}/AVANCE.md
            </button>
          )}
        </div>
        {f ? (
          <>
            <div className="flex items-baseline gap-3">
              <span className="text-[44px] leading-none font-semibold tracking-[-0.02em]">{uno(f.calculado)} %</span>
              <span className="text-rotulo text-suave">
                {horas(Math.round(((f.horas * f.calculado) / 100) * 10) / 10)} de {horas(f.horas)} h
              </span>
            </div>
            <HiloFase fase={f} elegido={pq} alElegir={setPaquete} />
            <p className="m-0 truncate text-rotulo" title={f.ahora}>
              Ahora: {f.ahora || "—"}
            </p>
            <p className="m-0 text-ayuda text-tenue">
              Faltan unas {horas(Math.round(horasQueFaltan(f) * 10) / 10)} h de lo estimado. El % lo calcula Atalaya, pesado por horas.
            </p>
            {f.avisos.length > 0 && (
              <p className="m-0 truncate text-ayuda text-aviso" title={f.avisos.join("; ")}>
                ⚠ {f.avisos[0]}
                {f.avisos.length > 1 ? ` (y ${f.avisos.length - 1} más)` : ""}
              </p>
            )}
          </>
        ) : (
          <p className="m-0 text-suave">
            Este proyecto no tiene <span className="dato">docs/trazabilidad/F&lt;n&gt;/AVANCE.md</span>: la bitácora del coordinador. Sin ella, el avance solo se puede calcular a mano.
          </p>
        )}
      </section>

      <section aria-labelledby="equipo-t" className={`flex min-w-0 flex-col gap-2 @3xl:col-span-2 ${tarjeta}`}>
        <div className="flex flex-wrap items-baseline justify-between gap-3">
          <h2 id="equipo-t" className="m-0 text-encabezado font-semibold">
            El equipo que lanzó
          </h2>
          <span className="text-ayuda text-tenue">{r.equipo.length ? `${r.equipo.length} agentes y llamadas a Codex` : "Nadie todavía"}</span>
        </div>
        {r.equipo.length > 0 && (
          <div className="relative pl-6">
            <span aria-hidden="true" className="absolute top-0 bottom-5 left-[7px] w-px bg-borde-fuerte" />
            {r.equipo.map((m) => (
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
                <span className="truncate text-ayuda text-suave max-sm:hidden">{m.estado.toLowerCase()}</span>
              </button>
            ))}
          </div>
        )}
      </section>

      <section aria-labelledby="cuota-t" className={`flex min-w-0 flex-col gap-2 ${tarjeta}`}>
        <h2 id="cuota-t" className="m-0 text-encabezado font-semibold">
          Cuota, Atalaya y documentos
        </h2>
        <dl className="m-0 grid grid-cols-[minmax(0,1fr)_auto] gap-x-3 gap-y-1.5 text-[13px]">
          <dt className="text-suave">Lecturas de la cuota</dt>
          <dd className="dato m-0 text-right">
            {r.lecturas_cuota}
            {r.ultima_cuota ? `, la última ${hora(new Date(r.ultima_cuota))}` : ""}
          </dd>
          <dt className="text-suave">Consultas a Atalaya</dt>
          <dd className="dato m-0 text-right">{r.consultas}</dd>
          <dt className="text-suave">Commits</dt>
          <dd className="dato m-0 text-right">{r.commits.length}</dd>
        </dl>
        {r.commits.length > 0 && (
          <ul aria-label="Commits" className="m-0 flex list-none flex-wrap gap-1.5 p-0">
            {r.commits.map((c) => (
              <li key={c.paso}>
                <button
                  type="button"
                  onClick={() => alIrAPaso(c.paso)}
                  title={c.orden}
                  className="dato h-7 rounded-control border border-borde px-2 text-ayuda hover:border-borde-fuerte"
                >
                  commit {hora(new Date(c.hora))}
                </button>
              </li>
            ))}
          </ul>
        )}
        <span className="mt-1 border-t border-borde pt-2 text-ayuda text-tenue">Documentos de estado que editó</span>
        {r.documentos.length === 0 ? (
          <span className="text-ayuda text-suave">Ninguno.</span>
        ) : (
          <ul className="m-0 flex list-none flex-col gap-1 p-0">
            {r.documentos.map((d) => (
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
                <span className="text-right text-suave">{d.ediciones === 1 ? "1 vez" : `${d.ediciones} veces`}</span>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Lo que la sesión te preguntó y lo que respondiste (la bandeja de Decisiones, solo de esta sesión). */}
      {(r.decisiones?.length ?? 0) > 0 && (
        // A todo el ancho (las tres columnas de la vista): sin hueco al lado y con sitio para la pregunta. Filas de alto
        // fijo y columnas fijas, como los pedidos; la respuesta lleva tope de ancho y se recorta (lo que cambia en vivo no mueve el diseño).
        <section aria-labelledby="decisiones-t" className={`flex min-w-0 flex-col gap-2 @3xl:col-span-3 ${tarjeta}`}>
          <h2 id="decisiones-t" className="m-0 text-encabezado font-semibold">
            Lo que te preguntó <span className="dato text-ayuda font-normal text-tenue">{r.decisiones!.length}</span>
          </h2>
          <ul className="m-0 flex list-none flex-col gap-1 p-0">
            {r.decisiones!.slice(0, 8).map((d) => {
              const respuesta = d.estado === "pendiente" ? "te espera" : d.preguntas.map((q) => q.respuesta).filter(Boolean).join(" · ") || "respondida";
              return (
                <li key={d.id}>
                  <button
                    type="button"
                    disabled={d.paso < 0}
                    onClick={() => alIrAPaso(d.paso)}
                    title={d.paso < 0 ? undefined : "Ir a ese paso en la actividad"}
                    className="grid h-10 w-full grid-cols-[3.5rem_minmax(0,1fr)_minmax(0,22rem)] items-center gap-3 rounded-control bg-hundido px-3 text-left text-rotulo hover:bg-elevado disabled:hover:bg-hundido max-sm:grid-cols-[3rem_minmax(0,1fr)]"
                  >
                    <span className="dato text-tenue">{hora(new Date(d.hora))}</span>
                    <span className="min-w-0 truncate" title={d.preguntas.map((q) => q.texto).join(" · ")}>
                      {d.preguntas[0]?.texto}
                      {d.preguntas.length > 1 && <span className="text-tenue"> · y {d.preguntas.length - 1} más</span>}
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
        </section>
      )}
    </>
  );
}
