"use client";

import { type KeyboardEvent, useEffect, useId, useMemo, useRef, useState } from "react";
import { api, type DetallePaso, type Paso } from "@/lib/api";
import { decimal, t, tServidor } from "@/lib/i18n";
import { hora } from "@/lib/formato";
import { ANCHO_CANALETA, armarFilas, coloresDeHilo, type Fila, geometria, metaPaso, salidaDelPaso, tokensCortos, ventana } from "@/lib/hilos";
import { paginaDe, ventanaAnclada } from "@/lib/paginas";
import { Paginador, RegionEstable } from "@/components/ui/paginador";
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

// Pasos por página; el tamaño se recuerda en este navegador.
const TAMANOS = [50, 100] as const;
const CLAVE_TAMANO = "atalaya:por-pagina:actividad";

function leerTamano(): number {
  try {
    const n = Number(localStorage.getItem(CLAVE_TAMANO));
    return (TAMANOS as readonly number[]).includes(n) ? n : TAMANOS[0];
  } catch {
    return TAMANOS[0];
  }
}

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
      <li className="flex items-center gap-1.5">{muestra("M4 0V6Q4 12 10 12H22", "var(--riel)")}{t("desvío")}</li>
      <li className="flex items-center gap-1.5">{muestra("M22 2V6Q22 12 16 12H5", "var(--riel)", undefined, "M5 9L2 12L5 15")}{t("vuelta")}</li>
      <li className="flex items-center gap-1.5">{muestra("M6 2C16 2 16 14 6 14", "var(--hilo-1)", "2 3")}{t("mismo archivo")}</li>
      <li className="flex items-center gap-1.5">{muestra("M4 2C13 2 13 14 4 14", "var(--acento)")}{t("en paralelo")}</li>
      <li className="flex items-center gap-1.5">{muestra("M2 8H22", "var(--opcional)", "4 3")}{t("delega")}</li>
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
  desde = 0,
  enVivo,
  alAbrirAgente,
  irATurno,
  irAPaso,
}: {
  idAgente: string;
  /** Los pasos que mandó el servidor: los últimos (hasta 3000). */
  pasos: Paso[];
  /** El número del primero de `pasos`; los anteriores se piden por página (api.pasosAntes). */
  desde?: number;
  enVivo: boolean;
  alAbrirAgente: (id: string) => void;
  /** Llegando desde una traza de la datapedia: el turno que hay que mostrar. */
  irATurno: { turno: number; cuando: number } | null;
  /** Desde «Qué lo llenó»: el paso que hay que elegir y mostrar. */
  irAPaso?: { n: number; cuando: number } | null;
}) {
  const base = useId();
  const [ajustes, setAjustes] = useState(leerAjustes);
  const [elegido, setElegido] = useState<number | null>(null);
  const [detalles, setDetalles] = useState<Record<number, DetallePaso>>({});
  const [errores, setErrores] = useState<Record<number, string>>({});
  const filasRef = useRef<Map<number, HTMLButtonElement>>(new Map());

  // Por páginas ancladas (lib/paginas.ts, ventanaAnclada): la última es la de ahora, llena al abrir y crece por abajo
  // con lo que llega; las anteriores no cambian. «Ancla» es cuántos pasos tenía el agente (todos, no solo los que
  // llegaron) al abrir o al volver a la última página; 0 = todavía no se fijó (se toma lo que haya). Al cambiar de
  // agente se empieza de nuevo.
  const [tamano, setTamano] = useState(leerTamano);
  const [hoja, setHoja] = useState<{ agente: string; ancla: number; pagina: number }>({ agente: idAgente, ancla: 0, pagina: Number.MAX_SAFE_INTEGER });
  const h = hoja.agente === idAgente && hoja.ancla > 0 ? hoja : { agente: idAgente, ancla: desde + pasos.length, pagina: Number.MAX_SAFE_INTEGER };
  // Con los primeros pasos (o con otro agente), el ancla se fija: desde ahí lo que llega crece por abajo y no corre nada.
  if (h !== hoja && pasos.length > 0) setHoja(h);
  // Las páginas se cuentan sobre todos los pasos del agente, no solo los que llegaron: el paso N está en la posición N.
  // Las de antes de `desde` se piden al servidor al abrirlas, de a una; así la lista nunca crece de más.
  const total = desde + pasos.length;
  const vista = ventanaAnclada(total, h.ancla, tamano, h.pagina);
  const finVisible = Math.min(vista.fin, vista.inicio + 2 * tamano);
  const nuevos = vista.fin - finVisible;
  /** Va a la página `p`; la última se vuelve a anclar en lo de ahora (otra vez llena, sin lo que creció de más). */
  function irAPagina(p: number) {
    const ultima = p >= vista.paginas;
    setHoja({ agente: idAgente, ancla: ultima ? total : h.ancla, pagina: ultima ? Number.MAX_SAFE_INTEGER : p });
  }
  /** La página donde está el paso de la posición `i` de `pasos`, sin volver a anclar. */
  function mostrarPosicion(i: number) {
    setHoja({ agente: idAgente, ancla: h.ancla, pagina: paginaDe(desde + i, total, h.ancla, tamano) });
  }

  // Una página vieja, de antes de lo que llegó: la pide y la guarda por su tramo («inicio-fin»).
  const vieja = vista.inicio < desde;
  const tramo = `${idAgente}:${vista.inicio}-${vista.fin}`;
  const [viejas, setViejas] = useState<Record<string, Paso[]>>({});
  const [erroresViejas, setErroresViejas] = useState<Record<string, string>>({});
  const errorVieja = erroresViejas[tramo];
  useEffect(() => {
    if (!vieja || viejas[tramo]) return;
    const ctl = new AbortController();
    api
      .pasosAntes(idAgente, vista.fin, vista.fin - vista.inicio, ctl.signal)
      .then((d) => {
        setErroresViejas((m) => { const copia = { ...m }; delete copia[tramo]; return copia; });
        setViejas((m) => ({ ...m, [tramo]: d.pasos }));
      })
      .catch((e: Error) => e.name !== "AbortError" && setErroresViejas((m) => ({ ...m, [tramo]: e.message })));
    return () => ctl.abort();
  }, [vieja, tramo, viejas, idAgente, vista.inicio, vista.fin]);

  const visibles = useMemo(
    () => (vieja ? (viejas[tramo] ?? []).slice(0, finVisible - vista.inicio) : pasos.slice(vista.inicio - desde, finVisible - desde)),
    [vieja, viejas, tramo, pasos, vista.inicio, finVisible, desde],
  );
  const filas = useMemo(() => armarFilas(visibles), [visibles]);
  const colores = useMemo(() => coloresDeHilo(visibles), [visibles]);
  const { trazos, nodos, alto } = useMemo(() => geometria(filas, colores), [filas, colores]);
  const porN = useMemo(() => new Map([...pasos, ...(vieja ? visibles : [])].map((p) => [p.n, p])), [pasos, vieja, visibles]);
  const ultimo = pasos.length ? pasos[pasos.length - 1].n : -1;
  const actual = elegido !== null && porN.has(elegido) ? elegido : ultimo;
  const foco = visibles.some((p) => p.n === actual) ? actual : filas.find((f) => f.tipo !== "etapa")?.paso.n;
  const paso = porN.get(actual);
  const max = Math.max(1, ...visibles.map((p) => tokDominante(p).valor));
  const vent = useMemo(() => ventana(pasos), [pasos]);

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
    const espera = setTimeout(() => {
      atendidos.add(cuandoPedido);
      mostrarPosicion(i);
      setElegido(n);
      requestAnimationFrame(() => filasRef.current.get(n)?.scrollIntoView({ behavior: "smooth", block: "start" }));
    }, 0);
    return () => clearTimeout(espera);
    // Solo cuando llega un pedido nuevo (cuándo), no cada vez que crece la lista de pasos.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [turnoPedido, cuandoPedido, idAgente]);

  // Desde «Qué lo llenó»: elige ese paso y lleva la vista a él (fuera del efecto, para no encadenar pintados).
  const pasoPedido = irAPaso?.n;
  const cuandoPaso = irAPaso?.cuando;
  useEffect(() => {
    if (pasoPedido === undefined || cuandoPaso === undefined || atendidos.has(cuandoPaso)) return;
    const espera = setTimeout(() => {
      if (irA(pasoPedido)) atendidos.add(cuandoPaso);
    }, 0);
    return () => clearTimeout(espera);
    // Solo cuando llega un pedido nuevo (cuándo).
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [pasoPedido, cuandoPaso]);

  // Seguir: llega un paso nuevo y la ventana no tiene el foco → se elige el último y la vista baja hasta él. Con el
  // foco no se toca nada: quien lee no debe ver saltar la página. El salto es directo, en el mismo efecto, sin
  // animación, temporizador ni requestAnimationFrame: con la ventana oculta o minimizada el navegador los frena hasta
  // un minuto, y nadie está mirando.
  const seguir = ajustes.seguir && enVivo;
  const bajar = useRef(false);
  useEffect(() => {
    if (!seguir || ultimo < 0 || document.hasFocus()) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- seguir es volver al último paso, como al abrir
    setElegido(null);
    // A la última página, anclada en lo de ahora: llena y con el último paso (nadie mira, así que puede correrse).
    setHoja({ agente: idAgente, ancla: total, pagina: Number.MAX_SAFE_INTEGER });
    bajar.current = true;
    // eslint-disable-next-line react-hooks/exhaustive-deps -- solo con un paso nuevo
  }, [seguir, ultimo]);
  // La bajada, ya pintada la página con el último paso.
  useEffect(() => {
    if (!bajar.current) return;
    bajar.current = false;
    filasRef.current.get(ultimo)?.scrollIntoView({ block: "center" });
  });

  function elegir(n: number, enfocar = false) {
    setElegido(n);
    if (enfocar) requestAnimationFrame(() => filasRef.current.get(n)?.focus());
  }

  // «Retoma»: va al paso donde se dejó ese archivo, aunque haya que mostrar pasos anteriores.
  function irA(n: number): boolean {
    const i = pasos.findIndex((p) => p.n === n);
    if (i < 0) return false;
    mostrarPosicion(i);
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
    const i = ns.indexOf(foco ?? actual);
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
    if (p.desvio) marcas.push(t("se desvía de su hilo"));
    const q = p.continua >= 0 ? porN.get(p.continua) : undefined;
    if (q) marcas.push(t("vuelve a {hilo} de las {hora}", { hilo: p.hilo, hora: hora(new Date(q.hora)) }));
    if (p.paralelo > 1 && p.herramienta !== "Respuesta") marcas.push(t("en paralelo con {n} más", { n: p.paralelo - 1 }));
    if (p.subagente) marcas.push(t("delega en un subagente"));
    return `${hora(new Date(p.hora))}, ${t(p.tipo)}, ${p.resumen || p.herramienta}${marcas.length ? ". " + marcas.join(". ") : ""}`;
  };

  return (
    <section aria-labelledby={`${base}-t`} className="flex flex-col gap-4 rounded-tarjeta border border-borde bg-superficie px-6 py-5">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 id={`${base}-t`} className="m-0 text-encabezado font-semibold">
          {t("Actividad")} <span className="dato text-ayuda font-normal text-tenue">{total === 1 ? t("1 paso") : t("{n} pasos", { n: total })}</span>
        </h2>
        <div className="flex flex-wrap items-center gap-3">
          <Leyenda />
          <button
            type="button"
            onClick={() => cambiar({ tokens: !ajustes.tokens })}
            aria-pressed={ajustes.tokens}
            title={t("Tokens de cada paso: ↓ lo que su resultado mete al contexto, ↑ lo que la IA generó")}
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
            title={t("Seguir: cuando la ventana no tiene el foco, la vista baja sola al último paso. Con el foco no se mueve.")}
            className={`flex h-7 items-center gap-1.5 rounded-control border px-2.5 text-ayuda transition-colors duration-150 ${
              ajustes.seguir ? "border-acento bg-acento-suave text-acento-fuerte" : "border-borde text-suave hover:text-texto"
            }`}
          >
            <span className="dato">↧</span> {t("Seguir")}
          </button>
          {enVivo && <span className="rounded-chip bg-ok-suave px-2 py-0.5 text-ayuda font-medium text-ok">{t("En vivo")}</span>}
        </div>
      </div>

      {pasos.length === 0 ? (
        <p className="m-0 text-rotulo text-suave">{t("Todavía no hay pasos.")}</p>
      ) : (
        <div className="flex flex-wrap items-start gap-5">
          {/* Línea de tiempo */}
          <div className="min-w-0 flex-[3_1_40rem]">
            {/* Las páginas, arriba y no al pie: la última crece por abajo con cada paso, y un pie bajaría con ella. */}
            {total > TAMANOS[0] && (
                <Paginador
                  arriba
                  nombre={t("Actividad")}
                  pagina={vista.pagina}
                  paginas={vista.paginas}
                  desde={vista.inicio + 1}
                  hasta={finVisible}
                  total={total}
                  alIr={irAPagina}
                  nuevos={nuevos}
                  alVerNuevos={() => irAPagina(vista.paginas)}
                  porPagina={{
                    valor: tamano,
                    opciones: TAMANOS,
                    alCambiar: (n) => {
                      setTamano(n);
                      try {
                        localStorage.setItem(CLAVE_TAMANO, String(n));
                      } catch {}
                      setHoja({ agente: idAgente, ancla: total, pagina: Number.MAX_SAFE_INTEGER });
                    },
                  }}
                />
            )}
            <RegionEstable activa={vista.paginas > 1 && vista.pagina < vista.paginas} clave={`${idAgente}:${tamano}:${h.ancla}`}>
            {vieja && !viejas[tramo] && (
              <p className={`m-0 py-3 text-rotulo ${errorVieja ? "text-error" : "text-suave"}`} aria-live="polite">
                {errorVieja ? t("No se pudieron traer esos pasos: {error}", { error: errorVieja }) : t("Trayendo los pasos {desde}–{hasta}…", { desde: vista.inicio + 1, hasta: vista.fin })}
              </p>
            )}
            <div role="group" aria-label={t("Pasos; las flechas mueven la elección")} onKeyDown={alTecla} className="relative">
              {filas.map((f) => {
                const p = f.paso;
                if (f.tipo === "etapa") {
                  return (
                    <div key={`e-${p.n}`} aria-hidden="true" className="grid h-[30px] grid-cols-[3.5rem_7.5rem_minmax(0,1fr)] items-center">
                      <span />
                      <span />
                      <span className="flex min-w-0 items-center gap-2 text-ayuda text-acento-fuerte">
                        <span className="shrink-0 text-[10.5px] font-semibold tracking-[0.06em] uppercase">{t("Etapa {n}", { n: f.nEtapa })}</span>
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
                    tabIndex={p.n === foco ? 0 : -1}
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
                          {t("Turno {n}", { n: p.turno })}
                        </span>
                        <span className="min-w-0 flex-1 truncate font-medium">{p.resumen}</span>
                        {/* En un teléfono la fila solo tiene sitio para el tipo y el resumen: lo demás está en el inspector y en la etiqueta de la fila. */}
                        {ajustes.tokens && (
                          <span className="dato shrink-0 text-ayuda whitespace-nowrap text-tenue max-sm:hidden">
                            {t("{tokens} en el turno", { tokens: tokensCortos(totalTurno.get(p.turno) ?? 0) })}
                          </span>
                        )}
                      </span>
                    ) : (
                      <span className="flex min-w-0 items-center gap-2 pr-2.5">
                        <span className={`shrink-0 rounded-chip px-2 py-px text-ayuda font-medium ${TIPO[p.tipo] ?? "bg-elevado text-suave"}`}>{t(p.tipo)}</span>
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
                          <span className="shrink-0 rounded-full border border-dashed border-opcional px-2 text-[11.5px] text-opcional max-sm:hidden">→ {t("subagente")}</span>
                        )}
                        <span className={`dato hidden shrink-0 text-ayuda sm:inline ${p.error ? "text-error" : "text-tenue"}`}>
                          {p.hecho ? metaPaso(p.meta) : t("en curso")}
                        </span>
                        {ajustes.tokens && (
                          <span
                            title={t("Entró al contexto ≈ {entra} · generó {sale} · contexto tras el paso ≈ {tras}", {
                              entra: tokensCortos(p.tok_entra),
                              sale: tokensCortos(salidaDelPaso(p)),
                              tras: tokensCortos(p.tok_tras),
                            })}
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
                {trazos.map((tz, i) => {
                  const e = TRAZO[tz.clase];
                  return (
                    <path
                      key={i}
                      d={tz.d}
                      fill="none"
                      stroke={tz.color ?? e.color}
                      strokeWidth={e.ancho}
                      strokeDasharray={e.raya}
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      opacity={e.opacidad}
                      markerEnd={tz.clase === "vuelta" ? `url(#${base}-flecha)` : undefined}
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
            </RegionEstable>
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
  const lugar = paso.profundidad ? t("dentro de un desvío (nivel {n})", { n: paso.profundidad }) : paso.continua >= 0 ? t("retoma su hilo") : t("camino principal");
  const celdas = [
    {
      k: t("↑ Generó la IA"),
      v: tokensCortos(salidaDelPaso(paso)),
      nota: enLote ? t("{tokens} del mensaje, entre {n} llamadas", { tokens: tokensCortos(paso.tok_salida), n: paso.paralelo }) : t("salida del mensaje"),
      color: "text-opcional",
    },
    {
      k: t("↓ Entró al contexto"),
      v: paso.tok_entra ? `≈ ${tokensCortos(paso.tok_entra)}` : "—",
      nota: paso.tok_entra ? t("resultado ≈ bytes ÷ 4") : t("sin resultado propio"),
      color: "text-texto",
    },
    {
      k: t("Contexto tras el paso"),
      v: paso.tok_tras ? `≈ ${tokensCortos(paso.tok_tras)}` : "—",
      nota: paso.tok_tras ? t("{pct} % de {ventana}", { pct: decimal((paso.tok_tras / ventana) * 100), ventana: ventana === 1_000_000 ? "1 M" : "200 k" }) : t("sin petición todavía"),
      color: "text-acento-fuerte",
    },
  ];
  return (
    <aside aria-label={t("Paso elegido")} className="sticky top-[68px] min-w-0 flex-[2_1_27rem] overflow-hidden rounded-panel border border-borde bg-hundido">
      <div className="flex flex-wrap items-center gap-x-2.5 gap-y-2 border-b border-borde bg-superficie px-3.5 py-3">
        <span className={`rounded-chip px-2 py-px text-ayuda font-medium ${TIPO[paso.tipo] ?? "bg-elevado text-suave"}`}>{t(paso.tipo)}</span>
        <span className="dato min-w-0 flex-[1_1_9rem] truncate text-rotulo">{paso.resumen || paso.herramienta}</span>
        <span className="dato shrink-0 text-ayuda text-tenue">
          {conSegundos(paso.hora)} · {paso.hecho ? metaPaso(paso.meta) : t("en curso")}
        </span>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-2 border-b border-borde px-3.5 py-2.5">
        <div role="group" aria-label={t("Cómo se muestra el paso")} className="flex gap-0.5 rounded-control bg-elevado p-[3px]">
          {(
            [
              ["estilizado", t("Estilizado")],
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
              {t("↩ Retoma {archivo} de las {hora}", { archivo: paso.hilo.split("/").pop() ?? "", hora: hora(new Date(retoma.hora)) })}
            </button>
          )}
          {paso.subagente && (
            <button
              type="button"
              onClick={() => alAbrirAgente(paso.subagente!)}
              className="rounded-chip border border-opcional/50 px-2 py-0.5 text-ayuda text-opcional hover:bg-opcional-suave"
            >
              {t("Abrir el subagente →")}
            </button>
          )}
        </div>
      )}
      {error ? (
        <p className="m-0 p-3.5 text-rotulo text-error">{t("No se pudo leer el paso: {error}", { error: tServidor(error) })}</p>
      ) : detalle ? (
        <CuerpoPaso d={detalle} modo={modo} />
      ) : (
        <div className="esqueleto m-3.5 h-16 rounded-chip" aria-label={t("Cargando el paso")} />
      )}
    </aside>
  );
}
