"use client";

import { type KeyboardEvent, useEffect, useId, useMemo, useRef, useState } from "react";
import { api, type DetallePaso, type Paso } from "@/lib/api";
import { hora } from "@/lib/formato";
import { ANCHO_CANALETA, armarFilas, coloresDeHilo, type Fila, geometria, salidaDelPaso, tokensCortos, ventana } from "@/lib/hilos";
import { CuerpoPaso, type Modo } from "./detalle-paso";

// Colores por tipo, para la actividad: ejecutar y planificar en ciruela, editar y escribir en ámbar, responder en
// verde, delegar en celeste.
const TIPO: Record<string, string> = {
  Ejecuta: "bg-acento-suave text-acento-fuerte",
  Planifica: "bg-acento-suave text-acento-fuerte",
  Edita: "bg-aviso-suave text-aviso",
  Escribe: "bg-aviso-suave text-aviso",
  Responde: "bg-ok-suave text-ok",
  Entrega: "bg-ok-suave text-ok",
  Cierra: "bg-ok-suave text-ok",
  Delega: "bg-opcional-suave text-opcional",
  Pide: "bg-opcional-suave text-opcional",
  // De la traza en vivo de Codex: lo que razona y un error de su turno.
  Razona: "bg-elevado text-suave",
  Error: "bg-error-suave text-error",
  // Una consulta a Atalaya por MCP (backend/internal/servidor/mcp.go).
  Atalaya: "bg-acento text-sobre-acento",
};

const VISIBLES = 60;
// Los saltos ya hechos (por su `cuando`), fuera del componente: al volver a Agentes, la actividad se monta de nuevo y
// no debe repetir un salto viejo.
const atendidos = new Set<number>();
const CLAVE = "atalaya:actividad";
const TRAZO: Record<string, { color: string; ancho: number; raya?: string; opacidad?: number }> = {
  riel: { color: "var(--riel)", ancho: 1.5, opacidad: 0.7 },
  desvio: { color: "var(--riel)", ancho: 1.5 },
  vuelta: { color: "var(--riel)", ancho: 1.5 },
  hilo: { color: "var(--riel)", ancho: 1.4, raya: "2 4" },
  paralelo: { color: "var(--acento)", ancho: 1.5, opacidad: 0.9 },
  delega: { color: "var(--opcional)", ancho: 1.5, raya: "4 3" },
  etapa: { color: "var(--acento)", ancho: 2 },
};

// seguir: sin foco en la ventana, la vista baja sola al último paso; con foco no se mueve (lo estás leyendo).
type Ajustes = { modo: Modo; tokens: boolean; seguir: boolean };

function leerAjustes(): Ajustes {
  try {
    const a = JSON.parse(localStorage.getItem(CLAVE) ?? "{}") as { modo?: string; tokens?: boolean; seguir?: boolean };
    return { modo: a.modo === "crudo" ? "crudo" : "estilizado", tokens: a.tokens !== false, seguir: a.seguir !== false };
  } catch {
    return { modo: "estilizado", tokens: true, seguir: true };
  }
}

function guardarAjustes(a: Ajustes) {
  try {
    localStorage.setItem(CLAVE, JSON.stringify(a));
  } catch {}
}

const conSegundos = (iso: string) => {
  const f = new Date(iso);
  return `${hora(f)}:${String(f.getSeconds()).padStart(2, "0")}`;
};

/** Lo que domina en un paso: ↓ lo que su resultado mete al contexto, o ↑ lo que la IA generó. */
function tokDominante(p: Paso): { texto: string; valor: number; sube: boolean } {
  const sal = salidaDelPaso(p);
  const sube = sal > p.tok_entra;
  const valor = sube ? sal : p.tok_entra;
  return { texto: valor ? `${sube ? "↑" : "↓"} ${tokensCortos(valor)}` : "—", valor, sube };
}

function Leyenda() {
  const muestra = (d: string, color: string, raya?: string, extra?: string) => (
    <svg width="26" height="16" aria-hidden="true" className="shrink-0">
      <path d={d} fill="none" stroke={color} strokeWidth="1.5" strokeDasharray={raya} />
      {extra && <path d={extra} fill="none" stroke={color} strokeWidth="1.5" />}
    </svg>
  );
  return (
    <ul className="m-0 flex list-none flex-wrap items-center gap-x-3.5 gap-y-1 p-0 text-ayuda text-suave">
      <li className="flex items-center gap-1.5">{muestra("M4 0V6Q4 12 10 12H22", "var(--riel)")}desvío</li>
      <li className="flex items-center gap-1.5">{muestra("M22 2V6Q22 12 16 12H5", "var(--riel)", undefined, "M5 9L2 12L5 15")}vuelta</li>
      <li className="flex items-center gap-1.5">{muestra("M6 2C16 2 16 14 6 14", "var(--hilo-1)", "2 3")}mismo archivo</li>
      <li className="flex items-center gap-1.5">{muestra("M4 2C13 2 13 14 4 14", "var(--acento)")}en paralelo</li>
      <li className="flex items-center gap-1.5">{muestra("M2 8H22", "var(--opcional)", "4 3")}delega</li>
    </ul>
  );
}

/**
 * La actividad como línea de tiempo con hilos: filas de alto fijo a la
 * izquierda, dibujadas por un solo SVG (lib/hilos.ts), y el inspector del paso elegido a la derecha. Las filas son
 * un grupo con foco itinerante: las flechas mueven la elección, Inicio y Fin van a los extremos. Cada fila dice en
 * texto lo que el dibujo muestra (se desvía, vuelve a, en paralelo), así el lector de pantalla no depende del SVG.
 */
export function Actividad({
  idAgente,
  pasos,
  enVivo,
  alAbrirAgente,
  irATurno,
  irAPaso,
}: {
  idAgente: string;
  pasos: Paso[];
  enVivo: boolean;
  alAbrirAgente: (id: string) => void;
  /** Llegando desde una traza de la datapedia: el turno que hay que mostrar. */
  irATurno: { turno: number; cuando: number } | null;
  /** Desde «Qué lo llenó»: el paso que hay que elegir y mostrar. */
  irAPaso?: { n: number; cuando: number } | null;
}) {
  const base = useId();
  const [cuantos, setCuantos] = useState(VISIBLES);
  const [ajustes, setAjustes] = useState(leerAjustes);
  const [elegido, setElegido] = useState<number | null>(null);
  const [detalles, setDetalles] = useState<Record<number, DetallePaso>>({});
  const [errores, setErrores] = useState<Record<number, string>>({});
  const filasRef = useRef<Map<number, HTMLButtonElement>>(new Map());

  const visibles = useMemo(() => pasos.slice(-cuantos), [pasos, cuantos]);
  const filas = useMemo(() => armarFilas(visibles), [visibles]);
  const colores = useMemo(() => coloresDeHilo(visibles), [visibles]);
  const { trazos, nodos, alto } = useMemo(() => geometria(filas, colores), [filas, colores]);
  const porN = useMemo(() => new Map(pasos.map((p) => [p.n, p])), [pasos]);
  const ultimo = pasos.length ? pasos[pasos.length - 1].n : -1;
  const actual = elegido !== null && porN.has(elegido) ? elegido : ultimo;
  const paso = porN.get(actual);
  const max = Math.max(1, ...visibles.map((p) => tokDominante(p).valor));
  const vent = ventana(pasos);

  function cambiar(a: Partial<Ajustes>) {
    const nuevos = { ...ajustes, ...a };
    setAjustes(nuevos);
    guardarAjustes(nuevos);
  }

  // El detalle del paso elegido; uno que seguía en curso se vuelve a pedir cuando termina.
  const det = detalles[actual];
  const hechoAhora = paso?.hecho;
  useEffect(() => {
    if (actual < 0 || (det && (det.hecho || !hechoAhora))) return;
    api
      .paso(idAgente, actual)
      .then((d) => setDetalles((m) => ({ ...m, [actual]: d })))
      .catch((e: Error) => setErrores((m) => ({ ...m, [actual]: e.message })));
  }, [idAgente, actual, det, hechoAhora]);

  // Llegada desde una traza: elige el pedido de ese turno, muestra lo necesario y lleva la vista a él.
  const turnoPedido = irATurno?.turno;
  const cuandoPedido = irATurno?.cuando;
  useEffect(() => {
    if (turnoPedido === undefined || cuandoPedido === undefined || atendidos.has(cuandoPedido)) return;
    const i = pasos.findIndex((p) => p.turno === turnoPedido);
    if (i < 0) return;
    const n = pasos[i].n;
    const t = setTimeout(() => {
      atendidos.add(cuandoPedido);
      setCuantos((c) => Math.max(c, pasos.length - i));
      setElegido(n);
      requestAnimationFrame(() => filasRef.current.get(n)?.scrollIntoView({ behavior: "smooth", block: "start" }));
    }, 0);
    return () => clearTimeout(t);
    // Solo cuando llega un pedido nuevo (cuándo), no cada vez que crece la lista de pasos.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [turnoPedido, cuandoPedido, idAgente]);

  // Desde «Qué lo llenó»: elige ese paso y lleva la vista a él (fuera del efecto, para no encadenar pintados).
  const pasoPedido = irAPaso?.n;
  const cuandoPaso = irAPaso?.cuando;
  useEffect(() => {
    if (pasoPedido === undefined || cuandoPaso === undefined || atendidos.has(cuandoPaso)) return;
    const t = setTimeout(() => {
      if (irA(pasoPedido)) atendidos.add(cuandoPaso);
    }, 0);
    return () => clearTimeout(t);
    // Solo cuando llega un pedido nuevo (cuándo).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pasoPedido, cuandoPaso]);

  // Seguir: llega un paso nuevo y la ventana no tiene el foco → se elige el último y la vista baja hasta él. Con el
  // foco no se toca nada: quien lee no debe ver saltar la página. El salto es directo, en el mismo efecto, sin
  // animación, temporizador ni requestAnimationFrame: con la ventana oculta o minimizada el navegador los frena hasta
  // un minuto, y nadie está mirando.
  const seguir = ajustes.seguir && enVivo;
  useEffect(() => {
    if (!seguir || ultimo < 0 || document.hasFocus()) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- seguir es volver al último paso, como al abrir
    setElegido(null);
    filasRef.current.get(ultimo)?.scrollIntoView({ block: "center" });
  }, [seguir, ultimo]);

  function elegir(n: number, enfocar = false) {
    setElegido(n);
    if (enfocar) requestAnimationFrame(() => filasRef.current.get(n)?.focus());
  }

  // «Retoma»: va al paso donde se dejó ese archivo, aunque haya que mostrar pasos anteriores.
  function irA(n: number): boolean {
    const i = pasos.findIndex((p) => p.n === n);
    if (i < 0) return false;
    setCuantos((c) => Math.max(c, pasos.length - i));
    setElegido(n);
    requestAnimationFrame(() => {
      const fila = filasRef.current.get(n);
      fila?.scrollIntoView({ behavior: "smooth", block: "center" });
      fila?.focus({ preventScroll: true });
    });
    return true;
  }

  function alTecla(e: KeyboardEvent<HTMLDivElement>) {
    const ns = filas.filter((f) => f.tipo !== "etapa").map((f) => f.paso.n);
    const i = ns.indexOf(actual);
    let j: number;
    if (e.key === "ArrowDown") j = Math.min(ns.length - 1, i + 1);
    else if (e.key === "ArrowUp") j = Math.max(0, i - 1);
    else if (e.key === "Home") j = 0;
    else if (e.key === "End") j = ns.length - 1;
    else return;
    e.preventDefault();
    elegir(ns[j], true);
  }

  // Totales por turno, para su rótulo.
  const totalTurno = new Map<number, number>();
  for (const p of visibles) totalTurno.set(p.turno, (totalTurno.get(p.turno) ?? 0) + p.tok_entra + salidaDelPaso(p));

  const texto = (f: Fila): string => {
    const p = f.paso;
    const marcas: string[] = [];
    if (p.desvio) marcas.push("se desvía de su hilo");
    const q = p.continua >= 0 ? porN.get(p.continua) : undefined;
    if (q) marcas.push(`vuelve a ${p.hilo} de las ${hora(new Date(q.hora))}`);
    if (p.paralelo > 1 && p.herramienta !== "Respuesta") marcas.push(`en paralelo con ${p.paralelo - 1} más`);
    if (p.subagente) marcas.push("delega en un subagente");
    return `${hora(new Date(p.hora))}, ${p.tipo}, ${p.resumen || p.herramienta}${marcas.length ? ". " + marcas.join(". ") : ""}`;
  };

  return (
    <section aria-labelledby={`${base}-t`} className="flex flex-col gap-4 rounded-tarjeta border border-borde bg-superficie px-6 py-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id={`${base}-t`} className="m-0 text-encabezado font-semibold">
          Actividad <span className="dato text-ayuda font-normal text-tenue">{pasos.length} pasos</span>
        </h2>
        <div className="flex flex-wrap items-center gap-3">
          <Leyenda />
          <button
            type="button"
            onClick={() => cambiar({ tokens: !ajustes.tokens })}
            aria-pressed={ajustes.tokens}
            title="Tokens de cada paso: ↓ lo que su resultado mete al contexto, ↑ lo que la IA generó"
            className={`flex h-7 items-center gap-1.5 rounded-control border px-2.5 text-ayuda transition-colors duration-150 ${
              ajustes.tokens ? "border-acento bg-acento-suave text-acento-fuerte" : "border-borde text-suave hover:text-texto"
            }`}
          >
            <span className="dato">↓↑</span> Tokens
          </button>
          <button
            type="button"
            onClick={() => cambiar({ seguir: !ajustes.seguir })}
            aria-pressed={ajustes.seguir}
            title="Seguir: cuando la ventana no tiene el foco, la vista baja sola al último paso. Con el foco no se mueve."
            className={`flex h-7 items-center gap-1.5 rounded-control border px-2.5 text-ayuda transition-colors duration-150 ${
              ajustes.seguir ? "border-acento bg-acento-suave text-acento-fuerte" : "border-borde text-suave hover:text-texto"
            }`}
          >
            <span className="dato">↧</span> Seguir
          </button>
          {enVivo && <span className="rounded-chip bg-ok-suave px-2 py-0.5 text-ayuda font-medium text-ok">En vivo</span>}
        </div>
      </div>

      {pasos.length === 0 ? (
        <p className="m-0 text-rotulo text-suave">Todavía no hay pasos.</p>
      ) : (
        <div className="flex flex-wrap items-start gap-5">
          {/* Línea de tiempo */}
          <div className="min-w-0 flex-[3_1_40rem]">
            {pasos.length > cuantos && (
              <button
                type="button"
                onClick={() => setCuantos((c) => c + VISIBLES)}
                className="mb-2 text-rotulo text-acento underline underline-offset-2 hover:text-acento-fuerte"
              >
                Mostrar {Math.min(VISIBLES, pasos.length - cuantos)} pasos anteriores
              </button>
            )}
            <div role="group" aria-label="Pasos; las flechas mueven la elección" onKeyDown={alTecla} className="relative">
              {filas.map((f) => {
                const p = f.paso;
                if (f.tipo === "etapa") {
                  return (
                    <div key={`e-${p.n}`} aria-hidden="true" className="grid h-[30px] grid-cols-[3.5rem_7.5rem_minmax(0,1fr)] items-center">
                      <span />
                      <span />
                      <span className="flex min-w-0 items-center gap-2 text-ayuda text-acento-fuerte">
                        <span className="shrink-0 text-[10.5px] font-semibold tracking-[0.06em] uppercase">Etapa {f.nEtapa}</span>
                        <span className="truncate">{f.etapa}</span>
                        <span className="h-px flex-1 bg-borde" />
                      </span>
                    </div>
                  );
                }
                const esElegido = p.n === actual;
                const vivo = p.n === ultimo && enVivo;
                const tok = tokDominante(p);
                const q = p.continua >= 0 ? porN.get(p.continua) : undefined;
                const primeroDelLote =
                  p.paralelo > 1 && p.herramienta !== "Respuesta" && !visibles.some((o) => o.lote === p.lote && o.n < p.n && o.herramienta !== "Respuesta");
                return (
                  <button
                    key={p.n}
                    ref={(b) => {
                      if (b) filasRef.current.set(p.n, b);
                      else filasRef.current.delete(p.n);
                    }}
                    type="button"
                    tabIndex={esElegido ? 0 : -1}
                    aria-pressed={esElegido}
                    aria-label={texto(f)}
                    onClick={() => elegir(p.n)}
                    style={{ height: f.alto }}
                    className={`grid w-full scroll-mt-28 grid-cols-[3.5rem_7.5rem_minmax(0,1fr)] items-center rounded-control text-left transition-colors duration-150 ${
                      esElegido ? "bg-seleccion" : "hover:bg-elevado/60"
                    }`}
                  >
                    {/* La hora entra 10 px: el fondo de la fila elegida la rodea con aire y no la toca. */}
                    <span className="dato pl-2.5 text-ayuda whitespace-nowrap text-tenue" title={conSegundos(p.hora)}>
                      {hora(new Date(p.hora))}
                    </span>
                    <span />
                    {f.tipo === "turno" ? (
                      <span className="flex min-w-0 items-center gap-2.5 pr-2.5">
                        <span className="dato shrink-0 rounded-chip bg-opcional-suave px-[7px] text-[11px] leading-[18px] font-semibold whitespace-nowrap text-opcional">
                          Turno {p.turno}
                        </span>
                        <span className="min-w-0 flex-1 truncate font-medium">{p.resumen}</span>
                        {/* En un teléfono la fila solo tiene sitio para el tipo y el resumen: lo demás está en el inspector y en la etiqueta de la fila. */}
                        {ajustes.tokens && (
                          <span className="dato shrink-0 text-ayuda whitespace-nowrap text-tenue max-sm:hidden">
                            {tokensCortos(totalTurno.get(p.turno) ?? 0)} en el turno
                          </span>
                        )}
                      </span>
                    ) : (
                      <span className="flex min-w-0 items-center gap-2 pr-2.5">
                        <span className={`shrink-0 rounded-chip px-2 py-px text-ayuda font-medium ${TIPO[p.tipo] ?? "bg-elevado text-suave"}`}>{p.tipo}</span>
                        <span className={`min-w-0 flex-1 truncate ${esElegido || vivo ? "text-texto" : "text-suave"}`}>{p.resumen || p.herramienta}</span>
                        {q && (
                          <span
                            className="dato shrink-0 rounded-chip border px-1.5 text-[11px] max-sm:hidden"
                            style={{ borderColor: colores.get(p.hilo), color: colores.get(p.hilo) }}
                          >
                            ↩ {p.hilo.split("/").pop()} · {hora(new Date(q.hora))}
                          </span>
                        )}
                        {primeroDelLote && (
                          <span className="dato shrink-0 rounded-chip bg-acento-suave px-1.5 text-[11px] text-acento-fuerte max-sm:hidden">∥ {p.paralelo}</span>
                        )}
                        {p.subagente && (
                          <span className="shrink-0 rounded-full border border-dashed border-opcional px-2 text-[11.5px] text-opcional max-sm:hidden">→ subagente</span>
                        )}
                        <span className={`dato hidden shrink-0 text-ayuda sm:inline ${p.error ? "text-error" : "text-tenue"}`}>
                          {p.hecho ? p.meta : "en curso"}
                        </span>
                        {ajustes.tokens && (
                          <span
                            title={`Entró al contexto ≈ ${tokensCortos(p.tok_entra)} · generó ${tokensCortos(salidaDelPaso(p))} · contexto tras el paso ≈ ${tokensCortos(p.tok_tras)}`}
                            className="flex w-[84px] shrink-0 flex-col items-end gap-[3px] border-l border-borde pl-2 max-sm:hidden"
                          >
                            {/* Una sola línea siempre (lo que cambia en vivo no mueve el diseño): «↓ 999,9 k» cabe entera; lo que no quepa se corta con puntos. */}
                            <span className={`dato max-w-full truncate text-ayuda ${tok.sube ? "text-opcional" : "text-suave"}`}>{tok.texto}</span>
                            <span aria-hidden="true" className="flex h-[2px] w-full justify-end rounded-full bg-elevado">
                              <span
                                className={`h-[2px] rounded-full opacity-70 ${tok.sube ? "bg-opcional" : "bg-suave"}`}
                                style={{ width: `${Math.max(4, (tok.valor / max) * 100)}%` }}
                              />
                            </span>
                          </span>
                        )}
                      </span>
                    )}
                  </button>
                );
              })}
              <svg
                aria-hidden="true"
                width={ANCHO_CANALETA}
                height={alto}
                className="pointer-events-none absolute top-0 left-14 overflow-visible"
              >
                <defs>
                  <marker id={`${base}-flecha`} viewBox="0 0 8 8" refX="6" refY="4" markerWidth="7" markerHeight="7" orient="auto-start-reverse">
                    <path d="M0 0L8 4L0 8z" fill="var(--riel)" />
                  </marker>
                </defs>
                {trazos.map((t, i) => {
                  const e = TRAZO[t.clase];
                  return (
                    <path
                      key={i}
                      d={t.d}
                      fill="none"
                      stroke={t.color ?? e.color}
                      strokeWidth={e.ancho}
                      strokeDasharray={e.raya}
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      opacity={e.opacidad}
                      markerEnd={t.clase === "vuelta" ? `url(#${base}-flecha)` : undefined}
                    />
                  );
                })}
                {nodos.map((d) => {
                  const p = porN.get(d.n)!;
                  const esElegido = d.n === actual;
                  const vivo = d.n === ultimo && enVivo;
                  const relleno = p.error
                    ? "var(--error)"
                    : p.tipo === "Pide"
                      ? "var(--opcional-suave)"
                      : (colores.get(p.hilo) ?? (p.herramienta === "Respuesta" ? "var(--ok)" : "var(--borde-campo)"));
                  return (
                    <g key={d.n}>
                      {esElegido && <circle cx={d.cx} cy={d.cy} r={9} fill="none" stroke="var(--acento)" strokeWidth={2} />}
                      <circle
                        cx={d.cx}
                        cy={d.cy}
                        r={p.tipo === "Pide" ? 6 : vivo ? 5.5 : 4.5}
                        fill={relleno}
                        stroke={p.tipo === "Pide" ? "var(--opcional)" : "var(--superficie)"}
                        strokeWidth={p.tipo === "Pide" ? 1.5 : 2}
                        className={vivo ? "pulso" : undefined}
                      />
                    </g>
                  );
                })}
              </svg>
            </div>
          </div>

          {/* Inspector */}
          {paso && (
            <Inspector
              paso={paso}
              detalle={detalles[actual]}
              error={errores[actual]}
              modo={ajustes.modo}
              alModo={(modo) => cambiar({ modo })}
              ventana={vent}
              retoma={paso.continua >= 0 ? porN.get(paso.continua) : undefined}
              alRetomar={irA}
              alAbrirAgente={alAbrirAgente}
            />
          )}
        </div>
      )}
    </section>
  );
}

function Inspector({
  paso,
  detalle,
  error,
  modo,
  alModo,
  ventana,
  retoma,
  alRetomar,
  alAbrirAgente,
}: {
  paso: Paso;
  detalle?: DetallePaso;
  error?: string;
  modo: Modo;
  alModo: (m: Modo) => void;
  ventana: number;
  retoma?: Paso;
  alRetomar: (n: number) => void;
  alAbrirAgente: (id: string) => void;
}) {
  const enLote = paso.paralelo > 1 && paso.herramienta !== "Respuesta";
  const lugar = paso.profundidad ? `dentro de un desvío (nivel ${paso.profundidad})` : paso.continua >= 0 ? "retoma su hilo" : "camino principal";
  const celdas = [
    {
      k: "↑ Generó la IA",
      v: tokensCortos(salidaDelPaso(paso)),
      nota: enLote ? `${tokensCortos(paso.tok_salida)} del mensaje, entre ${paso.paralelo} llamadas` : "salida del mensaje",
      color: "text-opcional",
    },
    {
      k: "↓ Entró al contexto",
      v: paso.tok_entra ? `≈ ${tokensCortos(paso.tok_entra)}` : "—",
      nota: paso.tok_entra ? "resultado ≈ bytes ÷ 4" : "sin resultado propio",
      color: "text-texto",
    },
    {
      k: "Contexto tras el paso",
      v: paso.tok_tras ? `≈ ${tokensCortos(paso.tok_tras)}` : "—",
      nota: paso.tok_tras ? `${((paso.tok_tras / ventana) * 100).toFixed(1).replace(".", ",")} % de ${ventana === 1_000_000 ? "1 M" : "200 k"}` : "sin petición todavía",
      color: "text-acento-fuerte",
    },
  ];
  return (
    <aside aria-label="Paso elegido" className="sticky top-[68px] min-w-0 flex-[2_1_27rem] overflow-hidden rounded-panel border border-borde bg-hundido">
      <div className="flex flex-wrap items-center gap-x-2.5 gap-y-2 border-b border-borde bg-superficie px-3.5 py-3">
        <span className={`rounded-chip px-2 py-px text-ayuda font-medium ${TIPO[paso.tipo] ?? "bg-elevado text-suave"}`}>{paso.tipo}</span>
        <span className="dato min-w-0 flex-[1_1_9rem] truncate text-rotulo">{paso.resumen || paso.herramienta}</span>
        <span className="dato shrink-0 text-ayuda text-tenue">
          {conSegundos(paso.hora)} · {paso.hecho ? paso.meta : "en curso"}
        </span>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-borde px-3.5 py-2.5">
        <div role="group" aria-label="Cómo se muestra el paso" className="flex gap-0.5 rounded-control bg-elevado p-[3px]">
          {(
            [
              ["estilizado", "Estilizado"],
              ["crudo", "RAW"],
            ] as const
          ).map(([id, nombre]) => (
            <button
              key={id}
              type="button"
              aria-pressed={modo === id}
              onClick={() => alModo(id)}
              className={`h-[26px] rounded-chip px-2.5 text-[12.5px] transition-colors duration-150 ${
                modo === id ? "bg-superficie text-texto shadow-[0_1px_2px_rgb(0_0_0/0.18)]" : "text-suave hover:text-texto"
              }`}
            >
              {nombre}
            </button>
          ))}
        </div>
        <span className="text-ayuda text-tenue">{lugar}</span>
      </div>
      <div className="grid grid-cols-3 border-b border-borde">
        {celdas.map((c) => (
          <div key={c.k} className="min-w-0 border-r border-borde px-3.5 py-2 last:border-r-0">
            <div className="truncate text-[11px] text-tenue">{c.k}</div>
            <div className={`dato text-cuerpo ${c.color}`}>{c.v}</div>
            <div className="truncate text-[11px] text-tenue" title={c.nota}>
              {c.nota}
            </div>
          </div>
        ))}
      </div>
      {(retoma || paso.subagente) && (
        <div className="flex flex-wrap gap-2 border-b border-borde px-3.5 py-2.5">
          {retoma && (
            <button
              type="button"
              onClick={() => alRetomar(retoma.n)}
              className="dato rounded-chip border border-acento/40 px-2 py-0.5 text-ayuda text-acento-fuerte hover:bg-acento-suave"
            >
              ↩ Retoma {paso.hilo.split("/").pop()} de las {hora(new Date(retoma.hora))}
            </button>
          )}
          {paso.subagente && (
            <button
              type="button"
              onClick={() => alAbrirAgente(paso.subagente!)}
              className="rounded-chip border border-opcional/50 px-2 py-0.5 text-ayuda text-opcional hover:bg-opcional-suave"
            >
              Abrir el subagente →
            </button>
          )}
        </div>
      )}
      {error ? (
        <p className="m-0 p-3.5 text-rotulo text-error">No se pudo leer el paso: {error}</p>
      ) : detalle ? (
        <CuerpoPaso d={detalle} modo={modo} />
      ) : (
        <div className="esqueleto m-3.5 h-16 rounded-chip" aria-label="Cargando el paso" />
      )}
    </aside>
  );
}
