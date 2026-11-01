"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { IconoCampana } from "@/components/ui/iconos";
import { Segmentado } from "@/components/ui/segmentado";
import { TarjetaAlerta } from "@/components/ui/tarjeta-alerta";
import { marcasDe } from "@/lib/alertas";
import type { Alerta, EstadoAvisos, Regla } from "@/lib/api";
import { cuando } from "@/lib/formato";
import { useCerrarFuera } from "@/lib/use-fuera";
import { t, tServidor } from "@/lib/i18n";

type Filtro = "sinver" | "todas";

/**
 * La campana de la cabecera y su centro de alertas. Cuenta las alertas sin ver; al abrirla,
 * cada alerta dice su regla, de qué documento salió el umbral y qué agente la causa. Abajo, las reglas vigiladas y los
 * controles de los avisos de Windows de Atalaya. Lo que se marca como visto se guarda en este navegador.
 */
export function Campana({
  alertas,
  reglas,
  vistas,
  avisos,
  alMarcarVista,
  alMarcarTodas,
  alCambiarAvisos,
  alProbarAviso,
  alAbrirAgente,
}: {
  alertas: Alerta[];
  reglas: Regla[];
  vistas: ReadonlySet<string>;
  avisos: EstadoAvisos | null;
  alMarcarVista: (id: string) => void;
  alMarcarTodas: () => void;
  alCambiarAvisos: (c: { decisiones?: boolean; alertas?: boolean }) => Promise<void>;
  alProbarAviso: () => Promise<void>;
  alAbrirAgente: (id: string) => void;
}) {
  const [abierto, setAbierto] = useState(false);
  // Dónde se abre: bajo la campana y siempre dentro de la ventana (en un ancho mediano la cabecera se parte en dos
  // filas y la campana puede quedar a la izquierda).
  const [pos, setPos] = useState({ top: 60, left: 12, width: 432 });
  const [filtro, setFiltro] = useState<Filtro>("sinver");
  const [guardando, setGuardando] = useState(false);
  const [estadoAviso, setEstadoAviso] = useState("");
  const boton = useRef<HTMLButtonElement>(null);
  const caja = useRef<HTMLDivElement>(null);
  const cerrar = useCallback(() => setAbierto(false), []);
  useCerrarFuera(abierto, caja, boton, cerrar);

  useEffect(() => {
    if (!abierto) return;
    window.addEventListener("resize", cerrar);
    return () => window.removeEventListener("resize", cerrar);
  }, [abierto, cerrar]);

  function alternar() {
    if (abierto) {
      setAbierto(false);
      return;
    }
    const r = boton.current?.getBoundingClientRect();
    if (r) {
      const width = Math.min(432, window.innerWidth - 24);
      setPos({ top: r.bottom + 8, left: Math.max(12, Math.min(r.right - width, window.innerWidth - width - 12)), width });
    }
    setAbierto(true);
  }

  const sinVer = alertas.filter((a) => !vistas.has(a.id));
  const lista = filtro === "sinver" ? sinVer : alertas;
  const grave = sinVer.some((a) => a.nivel === "error");
  const ultimo = avisos?.enviados?.[0];
  const estado = estadoAviso || (ultimo
    ? t("Último: {hora} · {titulo}{error}", { hora: cuando(new Date(ultimo.hora)), titulo: tServidor(ultimo.titulo), error: ultimo.error ? ` · ${t("Error: {error}", { error: tServidor(ultimo.error) })}` : "" })
    : t("Todavía no salió ningún aviso."));

  async function ejecutarAviso(accion: () => Promise<void>, prueba = false) {
    setGuardando(true);
    setEstadoAviso(prueba ? t("Enviando…") : "");
    try {
      await accion();
      setEstadoAviso(prueba ? t("Llegó la prueba: mírala en la esquina de la pantalla.") : "");
    } catch (e) {
      setEstadoAviso(e instanceof Error ? e.message : String(e));
    } finally {
      setGuardando(false);
    }
  }

  return (
    <div className="relative">
      <button
        ref={boton}
        type="button"
        onClick={alternar}
        aria-haspopup="dialog"
        aria-expanded={abierto}
        aria-label={sinVer.length ? t("Alertas: {n} sin ver", { n: sinVer.length }) : t("Alertas: nada sin ver")}
        title={t("Alertas de las reglas")}
        className={`relative grid size-[34px] place-items-center rounded-full border transition-colors duration-150 ${
          abierto ? "border-acento bg-elevado text-texto" : "border-borde text-suave hover:border-borde-fuerte hover:bg-elevado hover:text-texto"
        }`}
      >
        <IconoCampana className="size-[16px]" />
        {sinVer.length > 0 && (
          <span
            aria-hidden="true"
            className={`dato absolute -top-1.5 -right-1.5 grid h-[18px] min-w-[18px] place-items-center rounded-full px-1 text-[11px] font-semibold ${
              grave ? "bg-error text-superficie" : "bg-aviso text-superficie"
            }`}
          >
            {sinVer.length > 9 ? "9+" : sinVer.length}
          </span>
        )}
      </button>

      {abierto && (
        <div
          ref={caja}
          role="dialog"
          aria-label={t("Alertas")}
          className="capa aparecer-menu fixed z-40 flex flex-col rounded-panel"
          style={{ top: pos.top, left: pos.left, width: pos.width, maxHeight: `calc(100vh - ${pos.top + 12}px)` }}
        >
          <div className="flex items-center gap-3 border-b border-borde p-3">
            <h2 className="m-0 flex-1 text-encabezado font-semibold">{t("Alertas")}</h2>
            <div className="w-44">
              <Segmentado
                etiqueta={t("Qué alertas")}
                valor={filtro}
                alCambiar={setFiltro}
                opciones={[
                  { valor: "sinver", nombre: t("Sin ver"), cuenta: sinVer.length },
                  { valor: "todas", nombre: t("Todas"), cuenta: alertas.length },
                ]}
              />
            </div>
          </div>

          {/* Las tarjetas, las reglas y las notificaciones van en el mismo desplazamiento: en un teléfono las reglas ya no se
              comen el alto y las tarjetas caben enteras. */}
          <div className="scroll-interno flex min-h-0 flex-1 flex-col gap-2 overflow-x-hidden overflow-y-auto p-2">
            {lista.length === 0 ? (
              <p className="m-0 px-2 py-6 text-center text-rotulo text-suave">
                {alertas.length === 0 ? t("Ninguna regla rota ahora.") : t("Nada sin ver. Las vistas siguen en «Todas».")}
              </p>
            ) : (
              lista.map((a) => (
                <TarjetaAlerta
                  key={a.id}
                  alerta={a}
                  marcas={marcasDe(a, reglas)}
                  vista={vistas.has(a.id)}
                  alVer={() => {
                    alAbrirAgente(a.agente);
                    setAbierto(false);
                  }}
                  alVisto={() => alMarcarVista(a.id)}
                />
              ))
            )}

            <div className="mt-1 flex flex-col gap-2 border-t border-borde px-1 pt-3 pb-1">
              <h3 className="m-0 text-rotulo font-semibold">{t("Reglas vigiladas")}</h3>
              {/* Dos columnas también en el teléfono (la campana mide ~344 px): cada regla es nombre y umbral en una línea. */}
              <ul className="m-0 grid list-none grid-cols-2 gap-1.5 p-0">
                {reglas.map((r) => (
                  <li key={r.id} className="flex min-w-0 flex-col rounded-control bg-elevado/60 px-2.5 py-1.5" title={tServidor(r.nota)}>
                    <span className="flex h-4 min-w-0 items-center justify-between gap-2">
                      <span className="min-w-0 truncate text-ayuda">{tServidor(r.nombre)}</span>
                      <span className="dato shrink-0 text-ayuda font-medium">{tServidor(r.umbral)}</span>
                    </span>
                    <span className="h-4 truncate text-[11px] leading-4 text-tenue">{r.omision ? t("por omisión, no está escrita") : r.fuente}</span>
                  </li>
                ))}
              </ul>
              {avisos && (
                <div className="flex flex-col gap-1">
                  <h3 className="m-0 text-rotulo font-semibold">{t("Notificaciones de Windows")}</h3>
                  {!avisos.activos ? (
                    <p className="m-0 text-ayuda text-tenue">{t("Apagadas: Atalaya arrancó con {opcion}.", { opcion: "-avisos=false" })}</p>
                  ) : (
                    <>
                      {([
                        { clave: "decisiones", nombre: "Decisiones que te esperan", ayuda: "Lo que una sesión te pregunta, aunque el panel esté cerrado." },
                        { clave: "alertas", nombre: "Alertas de error", ayuda: "Un agente que pasó su tope o dos sobre el mismo archivo." },
                      ] as const).map(({ clave, nombre, ayuda }) => (
                        <button
                          key={clave}
                          type="button"
                          aria-pressed={avisos[clave]}
                          disabled={guardando}
                          onClick={() => ejecutarAviso(() => alCambiarAvisos({ [clave]: !avisos[clave] }))}
                          className="flex min-h-10 items-center gap-2.5 rounded-control px-1 text-left text-rotulo disabled:opacity-50"
                        >
                          <span aria-hidden="true" className={`relative h-5 w-[34px] shrink-0 rounded-full transition-colors duration-150 ${avisos[clave] ? "bg-acento" : "bg-borde-fuerte"}`}>
                            <span className={`absolute top-0.5 size-4 rounded-full bg-superficie transition-[left] duration-150 ${avisos[clave] ? "left-4" : "left-0.5"}`} />
                          </span>
                          <span className="flex flex-col">
                            <span>{t(nombre)}</span>
                            <span className="text-ayuda text-tenue">{t(ayuda)}</span>
                          </span>
                        </button>
                      ))}
                      <div className="flex items-center gap-2">
                        <button type="button" disabled={guardando} onClick={() => ejecutarAviso(alProbarAviso, true)} className="h-7 rounded-control px-2 text-ayuda text-suave hover:bg-elevado hover:text-texto disabled:opacity-50">
                          {t("Probar")}
                        </button>
                        <span className="h-5 min-w-0 flex-1 truncate text-ayuda text-tenue" title={estado}>{estado}</span>
                      </div>
                    </>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* «Marcar todas» queda fija abajo, a la vista aunque la lista se desplace. */}
          {sinVer.length > 1 && (
            <div className="flex justify-end border-t border-borde px-3 py-2">
              <button type="button" onClick={alMarcarTodas} className="h-7 rounded-control px-2 text-ayuda text-suave hover:bg-elevado hover:text-texto">
                {t("Marcar todas como vistas")}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
