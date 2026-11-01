"use client";

import { type KeyboardEvent, useRef } from "react";
import { Segmentado } from "@/components/ui/segmentado";
import { COLOR } from "@/lib/alertas";
import { type Agente, type Alerta, esActivo } from "@/lib/api";
import { COLOR_PROVEEDOR, FONDO_PROVEEDOR } from "@/lib/proveedor";
import { ALTO_FILA, nombreAgente, ordenarArbol, titulos, trazosArbol, X_NIVEL } from "@/lib/arbol";
import { haceCuanto } from "@/lib/formato";
import { useAhora } from "@/lib/vivo";

export type Filtro = "activos" | "hechos" | "todos";

export function enFiltro(a: Agente, f: Filtro): boolean {
  return f === "todos" || (f === "activos" ? esActivo(a.estado) : !esActivo(a.estado));
}

function derecha(a: Agente, ahora: number): string {
  if (a.estado === "Trabajando" && a.avance && a.avance.porcentaje >= 0) return `${a.avance.porcentaje} %`;
  if (a.estado === "Entregado") return "listo";
  if (ahora > 0) return haceCuanto(a.ultimo, ahora);
  return "";
}

/**
 * La lista de agentes como árbol: cada sesión, nombrada por su proyecto y con su nombre o su primer
 * pedido debajo, con sus subagentes y sus llamadas a Codex colgando de ella por un codo curvo, el mismo hilo de la
 * línea de tiempo (el de Codex, celeste y discontinuo). Una llamada a Codex sin sesión enlazada (ninguna orden de
 * los transcritos la lanzó con su tema) va como raíz con «sin sesión» delante, en tono neutro. El punto lleno es un
 * agente activo; el hueco, uno que ya entregó. Las flechas mueven el foco entre las filas.
 */
export function ListaAgentes({
  agentes,
  cargando,
  filtro,
  alFiltrar,
  elegido,
  alElegir,
  alertas,
}: {
  agentes: Agente[];
  cargando: boolean;
  filtro: Filtro;
  alFiltrar: (f: Filtro) => void;
  elegido: string | null;
  alElegir: (id: string) => void;
  // La alerta más grave de cada agente: su chip ocupa el lugar de la barra de avance.
  alertas: Map<string, Alerta>;
}) {
  const ahora = useAhora();
  const lista = agentes.filter((a) => enFiltro(a, filtro));
  const filas = ordenarArbol(lista);
  const trazos = trazosArbol(filas);
  const cuenta = (f: Filtro) => agentes.filter((a) => enFiltro(a, f)).length;
  const caja = useRef<HTMLUListElement>(null);

  function alTecla(e: KeyboardEvent<HTMLUListElement>) {
    if (e.key !== "ArrowDown" && e.key !== "ArrowUp" && e.key !== "Home" && e.key !== "End") return;
    const botones = [...(caja.current?.querySelectorAll<HTMLButtonElement>("button") ?? [])];
    const i = botones.indexOf(document.activeElement as HTMLButtonElement);
    const j = e.key === "Home" ? 0 : e.key === "End" ? botones.length - 1 : Math.max(0, Math.min(botones.length - 1, i + (e.key === "ArrowDown" ? 1 : -1)));
    e.preventDefault();
    botones[j]?.focus();
  }

  return (
    <aside
      aria-label="Agentes"
      className="w-full overflow-hidden rounded-tarjeta border border-borde bg-superficie lg:sticky lg:top-[4.25rem] lg:w-[21.25rem] lg:shrink-0"
    >
      <div className="flex flex-col gap-3 border-b border-borde p-4">
        <h2 className="m-0 text-encabezado font-semibold">Agentes</h2>
        <Segmentado
          etiqueta="Filtro"
          valor={filtro}
          alCambiar={alFiltrar}
          opciones={[
            { valor: "activos", nombre: "Activos", cuenta: cuenta("activos") },
            { valor: "hechos", nombre: "Hechos", cuenta: cuenta("hechos") },
            { valor: "todos", nombre: "Todos", cuenta: cuenta("todos") },
          ]}
        />
      </div>
      {cargando && agentes.length === 0 ? (
        <div className="flex flex-col gap-2 p-3" aria-hidden="true">
          {[0, 1, 2, 3].map((i) => (
            <div key={i} className="esqueleto h-12 rounded-control" />
          ))}
        </div>
      ) : filas.length === 0 ? (
        <p className="m-0 px-4 py-6 text-rotulo text-suave">No hay agentes en este filtro. Los encargos nuevos aparecen aquí al lanzarse.</p>
      ) : (
        <div className="scroll-interno max-h-[calc(100vh-14rem)] overflow-y-auto py-1.5">
          <div className="relative">
            <ul ref={caja} onKeyDown={alTecla} className="m-0 list-none p-0">
              {filas.map((f) => {
                const a = f.agente;
                const actual = a.id === elegido;
                const codex = a.proveedor === "Codex";
                const pct = a.estado === "Trabajando" && a.avance && a.avance.porcentaje >= 0 ? a.avance.porcentaje : null;
                const alerta = alertas.get(a.id);
                const t = titulos(a);
                const parentesco = codex
                  ? f.nivel
                    ? ", llamada a Codex de la sesión"
                    : f.sinSesion
                      ? ", llamada a Codex sin sesión enlazada: ninguna orden de los transcritos la lanzó con su tema"
                      : f.huerfano
                        ? ", llamada a Codex de una sesión fuera de este filtro"
                        : ""
                  : f.nivel || f.huerfano
                    ? ", subagente"
                    : a.tipo === "sesion"
                      ? ", sesión"
                      : "";
                // Lo suelto dice de dónde viene, con un tono neutro: no es un error.
                const prefijo = f.sinSesion ? "sin sesión" : f.huerfano ? (codex ? "llamada" : "subagente") : "";
                return (
                  <li key={a.id} style={{ height: ALTO_FILA }}>
                    <button
                      type="button"
                      onClick={() => alElegir(a.id)}
                      aria-current={actual ? "true" : undefined}
                      aria-label={`${nombreAgente(a)}, ${a.estado.toLowerCase()}${parentesco}${alerta ? `, alerta: ${alerta.titulo}, ${alerta.chip}` : ""}. ${t.completo}`}
                      className={`grid h-full w-full grid-cols-[minmax(0,1fr)_auto] items-center gap-x-2.5 pr-3.5 text-left transition-colors duration-150 ${
                        actual ? "bg-seleccion" : "hover:bg-elevado/60"
                      }`}
                      style={{ paddingLeft: f.nivel ? 46 : 28 }}
                    >
                      <span className="flex min-w-0 flex-col">
                        <span className="truncate font-medium whitespace-nowrap" title={nombreAgente(a)}>
                          {t.principal}
                          {t.clase && <span className="ml-1.5 text-ayuda font-normal text-tenue">{t.clase}</span>}
                        </span>
                        <span
                          className="truncate text-[12.5px] whitespace-nowrap text-suave"
                          title={f.sinSesion ? `${t.completo}\nSin sesión enlazada: ninguna orden de los transcritos corrió codex.ps1 con su tema (la sesión ya no está, o la lanzó un script con el tema en una variable).` : t.completo}
                        >
                          {prefijo && <span className="text-tenue">{prefijo} · </span>}
                          {t.alterno}
                        </span>
                      </span>
                      <span className="flex flex-col items-end gap-1">
                        <span className={`dato text-ayuda ${a.estado === "Trabajando" ? "text-ok" : "text-suave"}`}>{derecha(a, ahora)}</span>
                        {alerta ? (
                          <span
                            aria-hidden="true"
                            className={`dato rounded-chip px-1.5 text-[11px] leading-[17px] whitespace-nowrap ${COLOR[alerta.nivel].texto} ${COLOR[alerta.nivel].fondo}`}
                          >
                            {alerta.chip}
                          </span>
                        ) : pct !== null && (
                          <span aria-hidden="true" className="block h-[3px] w-11 rounded-full bg-elevado">
                            <span className={`block h-[3px] rounded-full ${FONDO_PROVEEDOR[codex ? "Codex" : "Claude"]}`} style={{ width: `${pct}%` }} />
                          </span>
                        )}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
            <svg aria-hidden="true" width="56" height={filas.length * ALTO_FILA} className="pointer-events-none absolute top-0 left-0 overflow-visible">
              {trazos.map((t, i) => (
                <path
                  key={i}
                  d={t.d}
                  fill="none"
                  stroke={t.codex ? COLOR_PROVEEDOR.Codex : "var(--riel)"}
                  strokeWidth="1.5"
                  strokeDasharray={t.codex ? "4 3" : undefined}
                  strokeLinecap="round"
                />
              ))}
              {filas.map((f, i) => {
                const a = f.agente;
                const color = COLOR_PROVEEDOR[a.proveedor];
                const vivo = esActivo(a.estado);
                return (
                  <circle
                    key={a.id}
                    cx={X_NIVEL[f.nivel]}
                    cy={i * ALTO_FILA + ALTO_FILA / 2}
                    r={f.nivel ? 4 : 5}
                    fill={vivo ? color : "var(--superficie)"}
                    stroke={color}
                    strokeWidth="1.5"
                    className={a.estado === "Trabajando" ? "pulso" : undefined}
                  />
                );
              })}
            </svg>
          </div>
        </div>
      )}
    </aside>
  );
}
