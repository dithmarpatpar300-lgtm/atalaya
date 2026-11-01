"use client";

import { useEffect, useRef, useState } from "react";
import { IconoCerrar } from "@/components/ui/iconos";
import { Markdown } from "@/components/ui/markdown";
import { type Agente, api, type Detalle, type EstadoAgente } from "@/lib/api";
import { idioma, t, tServidor } from "@/lib/i18n";
import { COLOR_PROVEEDOR } from "@/lib/proveedor";
import { nombreAgente, titulos } from "@/lib/arbol";
import { archivosTocados, linaje, type NodoLinaje, posicionHito } from "@/lib/encargo";
import { duracion, hora, minutosDe } from "@/lib/formato";
import { coloresDeHilo } from "@/lib/hilos";
import { useAhora } from "@/lib/vivo";
import { ContextoTokens } from "./contexto";
import { Exportar } from "./exportar";
import { VistaSesion } from "./sesion";

const ESTADO: Record<EstadoAgente, { clase: string; punto: string }> = {
  Trabajando: { clase: "bg-ok-suave text-ok", punto: "pulso bg-ok" },
  "En espera": { clase: "bg-aviso-suave text-aviso", punto: "bg-aviso" },
  Entregado: { clase: "bg-elevado text-suave", punto: "bg-borde-campo" },
  Inactiva: { clase: "bg-elevado text-suave", punto: "bg-borde-campo" },
};

const tarjeta = "rounded-tarjeta border border-borde bg-superficie px-6 py-5";
const subtitulo = "m-0 text-rotulo font-semibold text-suave";

export function Documento({ proyecto, ruta, alCerrar }: { proyecto: string; ruta: string; alCerrar: () => void }) {
  const dialogo = useRef<HTMLDialogElement>(null);
  const [texto, setTexto] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  useEffect(() => {
    dialogo.current?.showModal();
    api
      .doc(proyecto, ruta)
      .then((d) => setTexto(d.texto))
      .catch((e: Error) => setError(e.message));
  }, [proyecto, ruta]);
  return (
    <dialog
      ref={dialogo}
      onClose={alCerrar}
      aria-labelledby="doc-titulo"
      className="capa m-auto w-[min(56rem,calc(100vw-2rem))] rounded-tarjeta p-0 text-texto backdrop:bg-velo"
    >
      <div className="flex items-center gap-3 border-b border-borde px-5 py-3">
        <h2 id="doc-titulo" className="dato m-0 min-w-0 flex-1 truncate text-rotulo font-medium">
          {ruta}
        </h2>
        <button type="button" onClick={() => dialogo.current?.close()} aria-label={t("Cerrar")} className="grid size-8 place-items-center rounded-control hover:bg-elevado">
          <IconoCerrar className="size-4" />
        </button>
      </div>
      <div className="scroll-interno max-h-[75vh] overflow-y-auto p-5">
        {error ? (
          <p className="m-0 text-error">{t("No se pudo leer: {error}", { error: tServidor(error) })}</p>
        ) : texto === null ? (
          <div className="esqueleto h-24 rounded-panel" aria-label={t("Cargando")} />
        ) : (
          <Markdown texto={texto} ruta={ruta} />
        )}
      </div>
    </dialog>
  );
}

/** El linaje con los mismos hilos que la línea de tiempo: cada hijo cuelga de su padre con un codo curvo. */
function Linaje({ nodos, alAbrir }: { nodos: NodoLinaje[]; alAbrir: (id: string) => void }) {
  const ALTO = 34;
  const X = (nivel: number) => 7 + nivel * 18;
  const cy = (i: number) => i * ALTO + ALTO / 2;
  return (
    <div className="relative">
      <ul className="m-0 list-none p-0">
        {nodos.map((nd) => (
          <li key={nd.id} className="flex h-[34px] min-w-0 items-center gap-2" style={{ paddingLeft: X(nd.nivel) + 12 }}>
            {/* El nombre nunca se corta; la nota cede el espacio. */}
            {nd.yo ? (
              <span className="shrink-0 text-rotulo font-semibold whitespace-nowrap">{nd.nombre}</span>
            ) : (
              <button type="button" onClick={() => alAbrir(nd.id)} className="shrink-0 text-rotulo whitespace-nowrap text-suave underline-offset-2 hover:text-texto hover:underline">
                {nd.nombre}
              </button>
            )}
            <span className="min-w-0 truncate text-ayuda text-tenue">{nd.nota}</span>
          </li>
        ))}
      </ul>
      <svg aria-hidden="true" width="64" height={nodos.length * ALTO} className="pointer-events-none absolute top-0 left-0 overflow-visible">
        {nodos.map((nd, i) =>
          nd.padre < 0 ? null : (
            <path
              key={`p${nd.id}`}
              d={`M ${X(nodos[nd.padre].nivel)} ${cy(nd.padre) + 7} V ${cy(i) - 9} Q ${X(nodos[nd.padre].nivel)} ${cy(i)} ${X(nodos[nd.padre].nivel) + 9} ${cy(i)} H ${X(nd.nivel) - 6}`}
              fill="none"
              stroke={nd.proveedor === "Codex" ? COLOR_PROVEEDOR.Codex : "var(--riel)"}
              strokeWidth="1.5"
              strokeDasharray={nd.proveedor === "Codex" ? "4 3" : undefined}
              strokeLinecap="round"
            />
          ),
        )}
        {nodos.map((nd, i) => {
          const color = COLOR_PROVEEDOR[nd.proveedor];
          return (
            <circle
              key={`c${nd.id}`}
              cx={X(nd.nivel)}
              cy={cy(i)}
              r={nd.nivel === 0 ? 5 : 4}
              fill={nd.activo || nd.yo ? color : "var(--superficie)"}
              stroke={color}
              strokeWidth="1.5"
              className={nd.yo && nd.activo ? "pulso" : undefined}
            />
          );
        })}
      </svg>
    </div>
  );
}

function Entregable({ nombre, ruta, estado, bien, alAbrir }: { nombre: string; ruta: string; estado: string; bien: boolean; alAbrir: () => void }) {
  return (
    <li className="flex min-h-9 min-w-0 items-center gap-2.5">
      <span className={`shrink-0 rounded-chip px-[7px] py-px text-ayuda font-medium whitespace-nowrap ${bien ? "bg-ok-suave text-ok" : "bg-aviso-suave text-aviso"}`}>{estado}</span>
      <button type="button" onClick={alAbrir} className="flex min-w-0 flex-col text-left">
        <span className="text-rotulo hover:underline">{nombre}</span>
        <span className="dato truncate text-[11.5px] text-tenue">{ruta}</span>
      </button>
    </li>
  );
}

/**
 * El encabezado, el encargo detallado, el avance y el contexto del agente. La
 * actividad va aparte, a todo el ancho (panel.tsx).
 */
export function DetalleAgente({
  detalle,
  cargando,
  agentes,
  alAbrirAgente,
  alVerTraza,
  alIrAPaso,
}: {
  detalle: Detalle | null;
  cargando: boolean;
  agentes: Agente[];
  alAbrirAgente: (id: string) => void;
  alVerTraza: (a: Agente) => void;
  alIrAPaso: (n: number) => void;
}) {
  const ahora = useAhora();
  const [doc, setDoc] = useState<string | null>(null);
  const [exportando, setExportando] = useState(false);
  if (!detalle) {
    return (
      <div className="grid min-w-0 flex-1 gap-5">
        <section className={tarjeta}>
          {cargando ? (
            <div className="esqueleto h-16 rounded-control" aria-label={t("Cargando")} />
          ) : (
            <p className="m-0 text-suave">{t("Elige un agente de la lista para ver su encargo, su avance y lo que hace paso a paso.")}</p>
          )}
        </section>
      </div>
    );
  }
  const a = detalle.agente;
  const est = ESTADO[a.estado];
  const pct = a.avance && a.avance.porcentaje >= 0 ? a.avance.porcentaje : a.estado === "Entregado" ? 100 : null;
  const fin = a.estado === "Trabajando" && ahora > 0 ? ahora : Date.parse(a.ultimo) / 1000;
  const transcurrido = Math.max(0, fin - Date.parse(a.inicio) / 1000);
  const presupuesto = minutosDe(a.presupuesto);
  const minutos = Math.round(transcurrido / 60);
  const escala = presupuesto ?? Math.max(1, transcurrido / 60);
  const pasado = presupuesto !== null && minutos > presupuesto;
  const padre = a.padre ? agentes.find((x) => x.id === a.padre) : undefined;
  const titulo = titulos(a);
  // De dónde nació: la sesión por su nombre (o su id, si no está en la lista); una llamada a Codex sin sesión lo dice.
  const clase = a.interactiva ? t("sesión interactiva de Codex") : a.tipo === "codex" ? t("llamada") : a.tipo === "subagente" ? t("subagente") : t("sesión");
  const idPadre = padre ? padre.id.slice(0, 4) : a.padre ? a.padre.slice(0, 4) : "";
  const deSesion = padre
    ? padre.tipo === "sesion"
      ? t("{clase} de la sesión {nombre} {id}…", { clase, nombre: nombreAgente(padre), id: idPadre })
      : t("{clase} de {nombre} {id}…", { clase, nombre: nombreAgente(padre), id: idPadre })
    : a.padre
      ? t("{clase} de la sesión {id}…", { clase, id: idPadre })
      : "";
  const origen = [a.modelo, a.effort && `effort ${a.effort}`, deSesion || (a.tipo === "codex" && !a.interactiva ? t("llamada sin sesión enlazada") : clase), a.inicio && t("desde {hora}", { hora: hora(new Date(a.inicio)) })]
    .filter(Boolean)
    .join(" · ");
  const turnos = Math.max(0, ...detalle.pasos.map((p) => p.turno));

  // La ficha: de la línea «Asignación» si la hay; si no (una sesión), lo que se sabe de ella.
  const ficha: { k: string; v: string; dato?: boolean }[] = a.asignacion
    ? [
        { k: t("Nivel"), v: a.nivel || "—" },
        { k: t("Clase"), v: a.clase_detalle || a.clase || "—" },
        { k: t("Rol"), v: a.rol_asignado || a.rol.toLowerCase() },
        { k: t("Modelo"), v: a.modelo_asignado || a.modelo || "—", dato: true },
        { k: "Effort", v: a.effort || "—" },
        { k: t("Presupuesto"), v: a.presupuesto || "—" },
        { k: "Base", v: a.base || "—", dato: true },
        { k: t("Cuota al lanzar"), v: a.cuota || "—" },
      ]
    : [
        { k: t("Proveedor"), v: `${a.proveedor} · ${clase}` },
        { k: t("Modelo"), v: [a.modelo, a.effort && `effort ${a.effort}`].filter(Boolean).join(", ") || "—", dato: true },
        { k: t("Clase"), v: a.clase || "—" },
        { k: "Base", v: a.base || "—", dato: true },
        { k: t("Proyecto"), v: a.proyecto || "—" },
        { k: t("Inicio"), v: a.inicio ? new Date(a.inicio).toLocaleString(idioma() === "en" ? "en-US" : "es-PE", { dateStyle: "medium", timeStyle: "short" }) : "—" },
        { k: t("Turnos"), v: turnos ? String(turnos) : "—" },
        { k: t("Pasos"), v: String(a.pasos) },
      ];

  const nodos = linaje(a, agentes);
  const archivos = archivosTocados(detalle.pasos, a.nombrados);
  const colores = coloresDeHilo(detalle.pasos);
  const hayFrontera = (a.nombrados?.length ?? 0) > 0;
  const hitos = a.avance?.hitos ?? [];

  return (
    <div className="@container w-full min-w-0 flex-1">
      <div className="grid grid-cols-1 gap-5 @3xl:grid-cols-3">
        {/* Encabezado */}
        <section aria-labelledby="titulo" className={`col-span-full flex flex-wrap items-center gap-x-5 gap-y-3 ${tarjeta}`}>
          <div className="flex min-w-0 flex-[1_1_22rem] flex-col gap-1.5">
            <div className="flex flex-wrap items-center gap-2.5">
              <h1 id="titulo" className="m-0 min-w-0 truncate text-titulo font-semibold whitespace-nowrap" title={titulo.principal}>
                {titulo.principal}
              </h1>
              {titulo.clase && <span className="dato rounded-chip border border-borde px-[7px] text-ayuda text-suave">{titulo.clase}</span>}
              <span className={`inline-flex items-center gap-1.5 rounded-chip px-2 py-0.5 text-ayuda font-medium ${est.clase}`}>
                <span aria-hidden="true" className={`size-1.5 rounded-full ${est.punto}`} />
                {t(a.estado)}
              </span>
            </div>
            <p className="m-0 truncate whitespace-nowrap text-suave" title={titulo.completo}>
              {titulo.alterno}
            </p>
            <p className="dato m-0 truncate text-ayuda whitespace-nowrap text-tenue" title={origen}>
              {origen}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            {a.proveedor === "Claude" && (
              <>
                <button type="button" onClick={() => alVerTraza(a)} className="h-9 rounded-control border border-borde-fuerte px-3.5 font-medium hover:bg-elevado">
                  {t("Ver la traza")}
                </button>
                <button type="button" onClick={() => setExportando(true)} className="h-9 rounded-control border border-borde-fuerte px-3.5 font-medium hover:bg-elevado">
                  {t("Exportar")}
                </button>
              </>
            )}
            {/* Una sesión (también la interactiva de Codex) no tiene informe ni bitácora: su avance se abre desde la
                tarjeta de la fase. */}
            {a.tipo !== "sesion" && !a.interactiva && (
              <>
                <button
                  type="button"
                  disabled={!a.informe}
                  title={a.informe || t("El encargo no nombra un informe")}
                  onClick={() => setDoc(a.informe)}
                  className="h-9 rounded-control border border-borde-fuerte px-3.5 font-medium hover:bg-elevado disabled:cursor-not-allowed disabled:text-tenue"
                >
                  {t("Ver informe")}
                </button>
                <button
                  type="button"
                  disabled={!a.bitacora}
                  title={a.bitacora || t("Sin bitácora")}
                  onClick={() => setDoc(a.bitacora)}
                  className="h-9 rounded-control bg-acento px-3.5 font-medium text-sobre-acento hover:bg-acento-fuerte disabled:cursor-not-allowed disabled:bg-elevado disabled:text-tenue"
                >
                  {t("Abrir bitácora")}
                </button>
              </>
            )}
          </div>
        </section>

        {/* Una sesión no tiene encargo ni bitácora: su vista propia. */}
        {a.tipo === "sesion" && <VistaSesion id={a.id} version={`${a.ultimo}-${a.pasos}`} alAbrirAgente={alAbrirAgente} alAbrirDoc={setDoc} alIrAPaso={alIrAPaso} />}

        {a.tipo !== "sesion" && (
          <>
        {/* Encargo */}
        <section aria-labelledby="encargo" className={`flex min-w-0 flex-col gap-5 @3xl:col-span-2 ${tarjeta}`}>
          <div className="flex flex-wrap items-baseline justify-between gap-3">
            <h2 id="encargo" className="m-0 text-encabezado font-semibold">
              {t("Encargo")}
            </h2>
            <span className="text-ayuda text-tenue">{a.asignacion ? t("Leído de la línea «Asignación» del encargo") : t("Una sesión: sin línea «Asignación»")}</span>
          </div>
          {/* Las celdas llenan sus filas (4 + 4, o 3 + 3 + 2 estiradas): nunca queda un hueco. */}
          <dl className="m-0 flex flex-wrap gap-px overflow-hidden rounded-panel border border-borde bg-borde">
            {ficha.map((f) => (
              <div key={f.k} className="min-w-[9.5rem] flex-[1_1_calc(25%-1px)] bg-superficie px-3 py-2.5">
                <dt className="text-[11.5px] text-tenue">{f.k}</dt>
                <dd className={`m-0 mt-0.5 [overflow-wrap:anywhere] ${f.dato ? "dato text-rotulo" : "text-[13.5px]"}`}>{f.v}</dd>
              </div>
            ))}
          </dl>

          <div className="flex flex-wrap gap-6">
            {(nodos.length > 1 || a.tipo === "codex") && (
              <div className="flex min-w-0 flex-[1_1_15rem] flex-col gap-2">
                <h3 className={subtitulo}>{t("Linaje")}</h3>
                <Linaje nodos={nodos} alAbrir={alAbrirAgente} />
              </div>
            )}
            {(a.bitacora || a.informe || a.acta) && (
              <div className="flex min-w-0 flex-[1_1_15rem] flex-col gap-2">
                <h3 className={subtitulo}>{t("Entregables")}</h3>
                <ul className="m-0 flex list-none flex-col gap-1 p-0">
                  {a.bitacora && (
                    <Entregable
                      nombre={t("Bitácora")}
                      ruta={a.bitacora}
                      estado={pct === 100 ? t("Entregada") : a.avance ? `${a.avance.porcentaje} %` : t("Sin leer")}
                      bien={pct === 100}
                      alAbrir={() => setDoc(a.bitacora)}
                    />
                  )}
                  {a.informe && (
                    <Entregable nombre={t("Informe")} ruta={a.informe} estado={a.informe_existe ? t("Escrito") : t("Pendiente")} bien={!!a.informe_existe} alAbrir={() => setDoc(a.informe)} />
                  )}
                  {a.acta && (
                    <Entregable
                      nombre={t("Acta de temporales")}
                      ruta={a.acta}
                      estado={a.acta_estado === "cerrada" ? t("Cerrada") : t("Abierta")}
                      bien={a.acta_estado === "cerrada"}
                      alAbrir={() => setDoc(a.acta!)}
                    />
                  )}
                </ul>
              </div>
            )}
          </div>

          {archivos.length > 0 && (
            <div className="flex flex-col gap-2">
              <h3 className={subtitulo}>
                {t("Archivos que tocó")}{" "}
                <span className="font-normal text-tenue">
                  · {hayFrontera ? t("el encargo nombra {n}", { n: a.nombrados!.length }) : t("el encargo no nombra archivos")}
                </span>
              </h3>
              <div className="scroll-interno overflow-x-auto rounded-panel border border-borde">
                {/* Columnas fijas para las cifras; el nombre del archivo se lleva el resto del ancho. */}
                <table className="w-full table-fixed border-collapse text-rotulo">
                  <thead>
                    <tr className="bg-elevado text-left text-suave">
                      <th className="px-3 py-2 font-medium">{t("Archivo")}</th>
                      <th className="w-[4.5rem] px-2 py-2 text-right font-medium">{t("Leído")}</th>
                      <th className="w-[5rem] px-2 py-2 text-right font-medium">{t("Buscado")}</th>
                      <th className="w-[4.75rem] px-2 py-2 text-right font-medium">{t("Editado")}</th>
                      {hayFrontera && <th className="w-[8.5rem] px-3 py-2 font-medium">{t("En el encargo")}</th>}
                    </tr>
                  </thead>
                  <tbody>
                    {archivos.slice(0, 12).map((f) => (
                      <tr key={f.hilo} className="border-t border-borde">
                        <td className="px-3 py-2">
                          <span className="flex min-w-0 items-center gap-2" title={f.hilo}>
                            <span aria-hidden="true" className="size-2 shrink-0 rounded-full" style={{ background: colores.get(f.hilo) }} />
                            <span className="dato truncate text-[12.5px]">{f.nombre}</span>
                          </span>
                        </td>
                        <td className="dato px-2 py-2 text-right">{f.lee || "—"}</td>
                        <td className="dato px-2 py-2 text-right">{f.busca || "—"}</td>
                        <td className="dato px-2 py-2 text-right">{f.edita || "—"}</td>
                        {hayFrontera && (
                          <td className="px-3 py-2">
                            <span className={`rounded-chip px-[7px] py-px text-ayuda ${f.nombrado ? "bg-ok-suave text-ok" : "bg-elevado text-suave"}`}>
                              {f.nombrado ? t("nombrado") : t("no nombrado")}
                            </span>
                          </td>
                        )}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {archivos.length > 12 && <span className="text-ayuda text-tenue">{t("Y {n} archivos más en la actividad.", { n: archivos.length - 12 })}</span>}
            </div>
          )}

          {a.encargo && (
            <details className="rounded-panel border border-borde">
              <summary className="cursor-pointer px-3 py-2 text-rotulo text-suave select-none hover:text-texto">{t("El encargo completo")}</summary>
              <div className="scroll-interno max-h-96 overflow-y-auto border-t border-borde p-3.5 text-rotulo">
                <Markdown texto={a.encargo} ruta="" />
              </div>
            </details>
          )}
        </section>

        {/* Avance */}
        <section aria-labelledby="avance" className={`flex min-w-0 flex-col gap-3.5 ${tarjeta}`}>
          <h2 id="avance" className="m-0 text-encabezado font-semibold">
            {t("Avance")}
          </h2>
          <div className="flex items-baseline gap-2.5">
            <span className="dato text-[44px] leading-none font-medium tracking-[-0.04em]">{pct === null ? "—" : `${pct} %`}</span>
            <span className="text-rotulo text-suave">
              {a.interactiva
                ? a.estado === "Trabajando"
                  ? t("turno en curso")
                  : a.estado === "En espera"
                    ? t("esperando al usuario")
                    : t("sesión terminada")
                : a.tipo === "codex" ? (a.estado === "Trabajando" ? t("llamada en curso") : t("llamada terminada")) : a.avance ? t("según la bitácora") : t("sin bitácora de avance")}
            </span>
          </div>
          <div className="flex flex-col gap-1.5">
            <div className="flex justify-between text-ayuda text-suave">
              <span>{t("Tiempo")}</span>
              <span className={`dato ${pasado ? "text-aviso" : ""}`}>{presupuesto ? t("{valor} de {max} min", { valor: minutos, max: presupuesto }) : duracion(transcurrido)}</span>
            </div>
            <div className="relative h-2.5 rounded-full bg-elevado">
              <div
                className={`h-2.5 rounded-full ${pasado ? "bg-aviso" : a.estado === "Trabajando" ? "bg-acento" : "bg-borde-campo"}`}
                style={{ width: `${Math.min(100, (transcurrido / 60 / escala) * 100)}%` }}
              />
              {hitos.map((h, i) => {
                const x = posicionHito(h.hora, a.inicio, escala);
                return x === null ? null : (
                  <span
                    key={i}
                    title={`${h.hora} · ${h.texto}`}
                    className="absolute -top-[3px] h-4 w-0.5 rounded-[1px] bg-texto opacity-55"
                    style={{ left: `calc(${(x * 100).toFixed(1)}% - 1px)` }}
                  />
                );
              })}
            </div>
            <div className="flex justify-between text-[11.5px] text-tenue">
              <span>{hitos.length ? t("Hitos de la bitácora marcados") : t("Sin hitos")}</span>
              <span className="dato">{presupuesto ? `${presupuesto} min` : ""}</span>
            </div>
          </div>
          {a.avance?.desfase && (
            <p className="m-0 truncate text-ayuda text-aviso" title={a.avance.desfase}>
              {t("La bitácora no actualizó su primera línea: {detalle}", { detalle: tServidor(a.avance.desfase) })}
            </p>
          )}
          {hitos.length > 0 && (
            <ol className="m-0 flex list-none flex-col gap-2 border-t border-borde p-0 pt-3">
              {hitos.slice(-6).map((h, i) => {
                const m = /^(\d{1,3})\s*%\s*·\s*(.*)$/.exec(h.texto);
                return (
                  <li key={i} className="grid grid-cols-[2.5rem_3.25rem_minmax(0,1fr)] gap-2 text-[12.5px]">
                    <span className="dato text-tenue">{h.hora}</span>
                    {/* Ancho para «100 %» en una línea: el % nunca baja solo a la siguiente. */}
                    <span className="dato whitespace-nowrap text-acento">{m ? `${m[1]} %` : ""}</span>
                    <span className="text-suave">{m ? m[2] : h.texto}</span>
                  </li>
                );
              })}
            </ol>
          )}
        </section>
          </>
        )}

        <div className="col-span-full min-w-0">
          <ContextoTokens a={a} version={`${a.ultimo}-${a.pasos}`} alIrAPaso={alIrAPaso} />
        </div>

        {doc && <Documento proyecto={a.proyecto} ruta={doc} alCerrar={() => setDoc(null)} />}
        {exportando && <Exportar id={a.id} nombre={titulo.principal} alCerrar={() => setExportando(false)} />}
      </div>
    </div>
  );
}
