"use client";

import { type ReactNode, useEffect, useRef, useState } from "react";
import { Segmentado } from "@/components/ui/segmentado";
import { api, type DetallePaso, type Paso, type RespuestaPasos } from "@/lib/api";
import { FONDO_PROVEEDOR } from "@/lib/proveedor";
import { hora } from "@/lib/formato";
import { CuerpoPaso } from "./detalle-paso";

const TIPOS = ["Edita", "Escribe", "Ejecuta", "Lee", "Busca", "Delega", "Atalaya"];
const CUANDO: { valor: string; nombre: string; horas: number }[] = [
  { valor: "hoy", nombre: "24 h", horas: 24 },
  { valor: "semana", nombre: "7 días", horas: 168 },
  { valor: "todo", nombre: "Todo", horas: 0 },
];

/** Resalta cada palabra buscada dentro del texto (sin distinguir mayúsculas). */
function Resaltado({ texto, palabras }: { texto: string; palabras: string[] }) {
  if (palabras.length === 0) return <>{texto}</>;
  const re = new RegExp(`(${palabras.map((p) => p.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")).join("|")})`, "gi");
  const partes: ReactNode[] = texto.split(re).map((t, i) =>
    i % 2 ? (
      <mark key={i} className="rounded-[3px] bg-acento-suave px-0.5 text-acento-fuerte">
        {t}
      </mark>
    ) : (
      t
    ),
  );
  return <>{partes}</>;
}

function Opcion({ activo, children, alPulsar }: { activo: boolean; children: ReactNode; alPulsar: () => void }) {
  return (
    <button
      type="button"
      aria-pressed={activo}
      onClick={alPulsar}
      className={`flex h-7 items-center rounded-chip border px-2 text-ayuda transition-colors duration-150 ${
        activo ? "border-acento bg-acento-suave font-medium text-acento-fuerte" : "border-borde text-suave hover:border-borde-realce hover:text-texto"
      }`}
    >
      {children}
    </button>
  );
}

/**
 * Buscar en los pasos de todos los agentes: ¿quién tocó controller.rs hoy?, ¿qué orden
 * falló? Busca en el resumen, el archivo, la herramienta y la orden de cada paso; a la derecha, el paso elegido
 * con su diff o su salida, y el salto a la actividad del agente.
 */
export function BuscarPasos({
  proyecto,
  version,
  selector,
  alAbrirPaso,
}: {
  proyecto: string;
  version: number;
  // El selector Documentos / Pasos de la datapedia, que va arriba de la búsqueda.
  selector: ReactNode;
  alAbrirPaso: (proyecto: string, agente: string, n: number) => void;
}) {
  const [q, setQ] = useState("");
  const [tipo, setTipo] = useState("");
  const [cuando, setCuando] = useState("hoy");
  const [alcance, setAlcance] = useState<"proyecto" | "todos">("proyecto");
  const [res, setRes] = useState<RespuestaPasos | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [elegido, setElegido] = useState<{ agente: string; proyecto: string; paso: Paso; rol: string } | null>(null);
  // El detalle con la clave del paso al que pertenece («agente#n»), para no mostrar el de otro mientras carga.
  const [detalle, setDetalle] = useState<{ clave: string; d: DetallePaso } | null>(null);
  const campo = useRef<HTMLInputElement>(null);

  useEffect(() => {
    campo.current?.focus();
  }, []);

  useEffect(() => {
    const control = new AbortController();
    const horas = CUANDO.find((c) => c.valor === cuando)?.horas ?? 0;
    const espera = setTimeout(() => {
      api
        .pasos({ q, tipo, proyecto: alcance === "proyecto" ? proyecto : "", horas }, control.signal)
        .then((r) => {
          setRes(r);
          setError(null);
        })
        .catch((e: Error) => {
          if (e.name !== "AbortError") setError(e.message);
        });
    }, 200);
    return () => {
      clearTimeout(espera);
      control.abort();
    };
  }, [q, tipo, cuando, alcance, proyecto, version]);

  const agenteElegido = elegido?.agente;
  const nElegido = elegido?.paso.n;
  useEffect(() => {
    if (agenteElegido === undefined || nElegido === undefined) return;
    const control = { vivo: true };
    api
      .paso(agenteElegido, nElegido)
      .then((d) => {
        if (control.vivo) setDetalle({ clave: `${agenteElegido}#${nElegido}`, d });
      })
      .catch(() => {});
    return () => {
      control.vivo = false;
    };
  }, [agenteElegido, nElegido]);

  const palabras = q.split(/\s+/).filter(Boolean);
  const grupos = res?.grupos ?? [];
  const mostrados = grupos.reduce((s, g) => s + g.pasos.length, 0);
  const visible = detalle && elegido && detalle.clave === `${elegido.agente}#${elegido.paso.n}` ? detalle.d : null;

  return (
    <main id="contenido" tabIndex={-1} aria-label="Pasos de los agentes" className="relative mx-auto flex w-full max-w-[1520px] flex-col items-start gap-5 p-5 lg:flex-row">
      <aside
        aria-label="Búsqueda en los pasos"
        className="w-full overflow-hidden rounded-tarjeta border border-borde bg-superficie lg:sticky lg:top-[4.25rem] lg:w-[30rem] lg:shrink-0"
      >
        <div className="flex flex-col gap-3 border-b border-borde p-4">
          <h2 className="m-0 text-encabezado font-semibold">Datapedia</h2>
          {selector}
          <label className="sr-only" htmlFor="buscar-pasos">
            Buscar en los pasos de los agentes
          </label>
          <div className="relative">
            <input
              ref={campo}
              id="buscar-pasos"
              type="search"
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder="Orden, archivo o resumen: controller.rs, check.ps1…"
              autoComplete="off"
              spellCheck={false}
              className="h-(--alto-control) w-full rounded-control border border-borde-campo bg-superficie pr-24 pl-3 text-cuerpo text-texto placeholder:text-tenue hover:border-suave focus-visible:border-acento"
            />
            <span className="dato pointer-events-none absolute top-1/2 right-3 -translate-y-1/2 text-ayuda text-tenue">{res ? `${res.total} pasos` : ""}</span>
          </div>
          <Segmentado
            etiqueta="Proyectos"
            valor={alcance}
            alCambiar={setAlcance}
            opciones={[
              { valor: "proyecto", nombre: proyecto || "Este proyecto" },
              { valor: "todos", nombre: "Todos los proyectos" },
            ]}
          />
          <div role="group" aria-label="Tipo de paso" className="flex flex-wrap gap-1.5">
            <Opcion activo={tipo === ""} alPulsar={() => setTipo("")}>
              Todos
            </Opcion>
            {TIPOS.map((t) => (
              <Opcion key={t} activo={tipo === t} alPulsar={() => setTipo(tipo === t ? "" : t)}>
                {t === "Atalaya" ? "Consultas a Atalaya" : t}
              </Opcion>
            ))}
          </div>
          <div role="group" aria-label="Cuándo" className="flex flex-wrap items-center gap-1.5">
            <span className="text-ayuda text-tenue">Cuándo</span>
            {CUANDO.map((c) => (
              <Opcion key={c.valor} activo={cuando === c.valor} alPulsar={() => setCuando(c.valor)}>
                {c.nombre}
              </Opcion>
            ))}
          </div>
        </div>
        {error ? (
          <p className="m-0 px-4 py-6 text-rotulo text-error">No se pudo buscar: {error}</p>
        ) : !res ? (
          <div className="flex flex-col gap-2 p-3" aria-hidden="true">
            {[0, 1, 2, 3].map((i) => (
              <div key={i} className="esqueleto h-12 rounded-control" />
            ))}
          </div>
        ) : grupos.length === 0 ? (
          <p className="m-0 px-4 py-6 text-rotulo text-suave">Ningún paso coincide. Prueba con parte del nombre del archivo, otro periodo o quita el filtro de tipo.</p>
        ) : (
          <div aria-live="polite" className="scroll-interno flex max-h-[calc(100vh-23rem)] flex-col gap-3 overflow-y-auto p-2">
            {res.total > mostrados && (
              <p className="m-0 px-2 text-ayuda text-tenue">
                Los {mostrados} más recientes de {res.total}. Afina con más palabras, el tipo o el periodo.
              </p>
            )}
            {grupos.map((g) => (
              <section key={g.agente.id} aria-label={`${g.agente.rol} ${g.agente.clase}`} className="flex flex-col gap-0.5">
                <h3 className="m-0 flex items-center gap-2 px-2 py-1 text-rotulo font-normal text-suave">
                  <span aria-hidden="true" className={`size-2 shrink-0 rounded-full ${FONDO_PROVEEDOR[g.agente.proveedor]}`} />
                  <span className="font-semibold text-texto">
                    {g.agente.rol}
                    {g.agente.clase && <span className="ml-1 font-normal text-tenue">{g.agente.clase}</span>}
                  </span>
                  <span className="min-w-0 flex-1 truncate">{g.agente.tarea}</span>
                  <span className="dato text-ayuda text-tenue">{g.pasos.length}</span>
                </h3>
                {g.pasos.map((p) => {
                  const actual = elegido?.agente === g.agente.id && elegido.paso.n === p.n;
                  return (
                    <button
                      key={p.n}
                      type="button"
                      aria-current={actual ? "true" : undefined}
                      onClick={() => setElegido({ agente: g.agente.id, proyecto: g.agente.proyecto, paso: p, rol: `${g.agente.rol} ${g.agente.clase}`.trim() })}
                      className={`grid grid-cols-[3rem_5.5rem_minmax(0,1fr)] items-start gap-2 rounded-control px-2 py-2 text-left ${actual ? "bg-seleccion" : "hover:bg-elevado"}`}
                    >
                      <span className="dato pt-px text-ayuda text-suave">{hora(new Date(p.hora))}</span>
                      <span className={`justify-self-start rounded-chip px-1.5 text-ayuda ${p.tipo === "Atalaya" ? "bg-acento text-sobre-acento" : "bg-elevado text-suave"}`}>
                        {p.tipo}
                      </span>
                      <span className="flex min-w-0 flex-col">
                        <span className={`truncate text-rotulo ${p.error ? "text-error" : ""}`}>
                          <Resaltado texto={p.resumen} palabras={palabras} />
                        </span>
                        {(p.hilo || p.meta) && (
                          <span className="dato truncate text-ayuda text-tenue">
                            {p.hilo && !p.resumen.includes(p.hilo) ? <Resaltado texto={p.hilo} palabras={palabras} /> : null}
                            {p.hilo && !p.resumen.includes(p.hilo) && p.meta ? " · " : ""}
                            {p.meta}
                          </span>
                        )}
                      </span>
                    </button>
                  );
                })}
              </section>
            ))}
          </div>
        )}
      </aside>

      <div className="min-w-0 flex-1">
        <article aria-label="El paso elegido" className="flex flex-col gap-4 rounded-tarjeta border border-borde bg-superficie px-6 py-5">
          {elegido ? (
            <>
              <header className="flex flex-wrap items-center gap-x-3 gap-y-1 border-b border-borde pb-4">
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="text-encabezado font-semibold">{elegido.rol}</span>
                  <span className="dato truncate text-ayuda text-tenue">
                    {elegido.proyecto} · paso {elegido.paso.n} · {hora(new Date(elegido.paso.hora))} · {elegido.paso.tipo}
                  </span>
                </span>
                <button
                  type="button"
                  onClick={() => alAbrirPaso(elegido.proyecto, elegido.agente, elegido.paso.n)}
                  className="h-(--alto-control-compacto) rounded-control bg-acento px-3 text-rotulo font-medium text-sobre-acento hover:bg-acento-fuerte"
                >
                  Abrir en la actividad →
                </button>
              </header>
              {visible ? (
                <div className="min-w-0 text-cuerpo">
                  <CuerpoPaso d={visible} modo="estilizado" />
                </div>
              ) : (
                <div className="esqueleto h-40 rounded-panel" aria-label="Cargando" />
              )}
            </>
          ) : (
            <p className="m-0 text-suave">
              Busca en lo que hicieron todos los agentes: una orden, un archivo, un resumen. Elige un paso para ver su diff o su salida.
            </p>
          )}
        </article>
      </div>
    </main>
  );
}
