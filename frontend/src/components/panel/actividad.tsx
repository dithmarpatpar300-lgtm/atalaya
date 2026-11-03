"use client";

import { type KeyboardEvent, useCallback, useEffect, useId, useMemo, useRef, useState } from "react";
import { api, type DetallePaso, type MapaSesion as Mapa, type Paso } from "@/lib/api";
import { decimal, t, tServidor } from "@/lib/i18n";
import { hora } from "@/lib/formato";
import { armarFilas, type Fila, metaPaso, nombreEtapa, salidaDelPaso, tokensCortos, ventana } from "@/lib/hilos";
import {
  coloresDeArchivos,
  colocar,
  columnas,
  enlazar,
  franja,
  type Lote,
  nombreArchivo,
  plegar,
  type Renglon,
  renglones,
  resumenLote,
  trazar,
} from "@/lib/lotes";
import { paginaDe, ventanaAnclada } from "@/lib/paginas";
import { Fantasmas, Paginador } from "@/components/ui/paginador";
import { CuerpoPaso, type Modo } from "./detalle-paso";
import { MapaSesion } from "./mapa-sesion";

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

// Lo mismo, solo el texto: en las filas va la palabra con su color, sin la pastilla.
const TIPO_TEXTO: Record<string, string> = {
  Ejecuta: "text-acento-fuerte",
  Planifica: "text-acento-fuerte",
  Atalaya: "text-acento-fuerte",
  Edita: "text-aviso",
  Escribe: "text-aviso",
  Responde: "text-ok",
  Entrega: "text-ok",
  Cierra: "text-ok",
  Delega: "text-opcional",
  Pide: "text-opcional",
  Error: "text-error",
};

// Pasos por página; el tamaño se recuerda en este navegador. Plegados por lote, 100 pasos son unas 25 filas.
const TAMANOS = [50, 100] as const;
const CLAVE_TAMANO = "atalaya:por-pagina:actividad";

function leerTamano(): number {
  try {
    const n = Number(localStorage.getItem(CLAVE_TAMANO));
    return (TAMANOS as readonly number[]).includes(n) ? n : TAMANOS[1];
  } catch {
    return TAMANOS[1];
  }
}

// Los saltos ya hechos (por su `cuando`), fuera del componente: al volver a Agentes, la actividad se monta de nuevo y
// no debe repetir un salto viejo.
const atendidos = new Set<number>();
const CLAVE = "atalaya:actividad";

// seguir: si nadie usa la ventana, la vista baja sola al último paso; mientras se usa no se mueve (la estás leyendo).
type Ajustes = { modo: Modo; tokens: boolean; seguir: boolean };
/** Con el foco pero sin tocarla este tiempo, la ventana cuenta como que nadie la usa. */
const SIN_TOCAR_MS = 60_000;
/** Lo que se mide de la lista con el mapa al lado: por debajo, el mapa no se muestra. */
const ANCHO_CON_MAPA = 600;
const ANCHO_MAPA = 92;

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

/** Lo que domina: ↓ lo que los resultados meten al contexto, o ↑ lo que la IA generó. */
function tokDominante(filas: readonly Fila[]): { texto: string; valor: number; sube: boolean; entra: number; sale: number } {
  let entra = 0;
  let sale = 0;
  for (const f of filas) {
    entra += f.paso.tok_entra;
    sale += salidaDelPaso(f.paso);
  }
  const sube = sale > entra;
  const valor = sube ? sale : entra;
  return { texto: valor ? `${sube ? "↑" : "↓"} ${tokensCortos(valor)}` : "—", valor, sube, entra, sale };
}

function Leyenda() {
  return (
    <ul className="m-0 flex list-none flex-wrap items-center gap-x-3.5 gap-y-1 p-0 text-ayuda text-suave">
      <li className="flex items-center gap-1.5">
        <span aria-hidden="true" className="box-border size-2.5 rounded-full border-2 border-acento" />
        {t("lee")}
      </li>
      <li className="flex items-center gap-1.5">
        <span aria-hidden="true" className="size-2.5 rounded-full bg-acento" />
        {t("edita")}
      </li>
      <li className="flex items-center gap-1.5">
        <svg width="22" height="12" aria-hidden="true" className="shrink-0">
          <path d="M1 10H12A6 6 0 0 0 18 4V0" fill="none" stroke="var(--serie-5)" strokeWidth="1.75" />
        </svg>
        {t("un archivo sigue")}
      </li>
      <li className="flex items-center gap-1.5">
        <svg width="14" height="14" aria-hidden="true" className="shrink-0">
          <path d="M3 13V7A4 4 0 0 1 7 3H10" fill="none" stroke="var(--serie-1)" strokeWidth="1.75" strokeDasharray="2 3" strokeLinecap="round" />
        </svg>
        {t("viene de antes")}
      </li>
    </ul>
  );
}

/** La flecha de un lote que se abre; en uno de un solo paso guarda el lugar sin verse. */
function Flecha({ abierto, oculta }: { abierto: boolean; oculta: boolean }) {
  return (
    <svg width="12" height="12" aria-hidden="true" className={`transition-transform duration-150 ${abierto ? "rotate-90" : ""} ${oculta ? "opacity-0" : ""}`}>
      <path d="M4 2L8 6L4 10" fill="none" stroke="var(--texto-tenue)" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

/**
 * La actividad como línea de tiempo: una fila por lote (un mensaje de la IA con sus llamadas), que se abre para ver sus
 * pasos; un solo SVG dibuja el riel y los hilos que unen lotes por archivo (lib/lotes.ts); el mapa de toda la sesión al
 * costado y el inspector del paso elegido a la derecha. Las filas son un grupo con foco itinerante: arriba y abajo mueven
 * la elección, derecha e izquierda abren y cierran, Inicio y Fin van a los extremos. Cada fila dice en texto lo que el
 * dibujo muestra, así el lector de pantalla no depende del SVG.
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
  /** Los pasos que mandó el servidor: los últimos. */
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
  const filasRef = useRef<Map<string, HTMLButtonElement>>(new Map());
  // Los lotes abiertos, por el número de su primer paso. `forzado` es el paso al que se saltó: su lote se muestra abierto.
  const [abiertos, setAbiertos] = useState<ReadonlySet<number>>(() => new Set());
  const [forzado, setForzado] = useState<number | null>(null);
  const [focoClave, setFocoClave] = useState<string | null>(null);

  // Lo que mide la lista con su mapa: de ahí salen las columnas y si el mapa cabe.
  const [ancho, setAncho] = useState(0);
  const medir = useCallback((el: HTMLDivElement | null) => {
    if (!el) return;
    const ro = new ResizeObserver(([e]) => setAncho(e.contentRect.width));
    ro.observe(el);
    return () => ro.disconnect();
  }, []);

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
  // Cada página muestra a lo más `tamano` pasos: lo que llega después en la última se cuenta en «nuevos» (un clic, o
  // «Seguir» sin foco, vuelve a anclar). Así la página no crece en vivo y lo de abajo no se corre.
  const finVisible = Math.min(vista.fin, vista.inicio + tamano);
  const nuevos = vista.fin - finVisible;
  /** Va a la página `p`; la última se vuelve a anclar en lo de ahora (otra vez llena, sin lo que creció de más). */
  function irAPagina(p: number) {
    const ultima = p >= vista.paginas;
    setHoja({ agente: idAgente, ancla: ultima ? total : h.ancla, pagina: ultima ? Number.MAX_SAFE_INTEGER : p });
    // Lo elegido se queda en su página: aquí manda el último paso de la nueva.
    setElegido(null);
    setFocoClave(null);
  }
  /** La página donde está el paso de la posición `i` de todos los del agente, sin volver a anclar. */
  function mostrarPosicion(i: number) {
    setHoja({ agente: idAgente, ancla: h.ancla, pagina: paginaDe(i, total, h.ancla, tamano) });
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

  // El mapa de toda la sesión, solo si hay más de lo que cabe en una página. Se refresca cada 10 pasos: es un panorama.
  // Lo que no tiene transcrito de Claude (Codex) responde 404 y se queda sin mapa.
  const [mapa, setMapa] = useState<Mapa | null>(null);
  const conMapa = total > tamano;
  const corteMapa = Math.floor(total / 10);
  useEffect(() => {
    if (!conMapa) return;
    const ctl = new AbortController();
    api
      .mapa(idAgente, ctl.signal)
      .then(setMapa)
      .catch(() => {});
    return () => ctl.abort();
  }, [conMapa, idAgente, corteMapa]);

  const visibles = useMemo(
    () => (vieja ? (viejas[tramo] ?? []).slice(0, finVisible - vista.inicio) : pasos.slice(vista.inicio - desde, finVisible - desde)),
    [vieja, viejas, tramo, pasos, vista.inicio, finVisible, desde],
  );
  // Los pasos de antes de la página numeran sus etapas y dicen qué archivos vienen de antes; de una página vieja no se tienen (null).
  const previos = useMemo(() => (vieja ? null : pasos.slice(0, vista.inicio - desde)), [vieja, pasos, vista.inicio, desde]);
  const lotes = useMemo(() => plegar(armarFilas(visibles, previos)), [visibles, previos]);
  const { enlaces, llegadas } = useMemo(() => enlazar(lotes, previos), [lotes, previos]);
  const colores = useMemo(() => coloresDeArchivos(lotes), [lotes]);
  const enlazados = useMemo(() => new Set([...enlaces.flatMap((e) => e.archivos), ...llegadas.values()]), [enlaces, llegadas]);
  const loteDe = useMemo(() => {
    const m = new Map<number, number>();
    lotes.forEach((l, i) => l.filas.forEach((f) => m.set(f.paso.n, i)));
    return m;
  }, [lotes]);

  const porN = useMemo(() => new Map([...pasos, ...(vieja ? visibles : [])].map((p) => [p.n, p])), [pasos, vieja, visibles]);
  const ultimo = pasos.length ? pasos[pasos.length - 1].n : -1;
  // Sin nada elegido, el último de lo que se ve: en la última página, con pasos nuevos sin mostrar, no es el último de todos.
  const actual = elegido !== null && porN.has(elegido) ? elegido : (visibles.at(-1)?.n ?? ultimo);
  const paso = porN.get(actual);
  const eleccion = loteDe.get(actual) ?? -1;
  const vent = useMemo(() => ventana(pasos), [pasos]);

  const abierto = (l: Lote) => abiertos.has(l.n) || (forzado !== null && loteDe.get(forzado) === lotes.indexOf(l));
  const { filas, cy, alto } = renglones(lotes, abierto);
  const mapaVisible = Boolean(conMapa && mapa?.celdas.length && (ancho === 0 || ancho >= ANCHO_CON_MAPA));
  const cols = columnas(ancho === 0 ? 0 : ancho - (mapaVisible ? ANCHO_MAPA : 0), ajustes.tokens);
  // En el teléfono solo corren los hilos del lote elegido.
  const pista = colocar(enlaces, cols.canaleta, cols.angosto ? eleccion : undefined);
  const trazos = trazar(pista, cy, alto, eleccion, colores, llegadas);
  const max = Math.max(1, ...lotes.map((l) => tokDominante(l.filas).valor));
  const colorHilo = (archivo: string) => (enlazados.has(archivo) ? (colores.get(archivo) ?? "var(--riel)") : "var(--riel)");

  /** La fila donde se ve un paso: la suya si su lote está abierto, si no la del lote. */
  function claveDe(n: number): string | undefined {
    const i = loteDe.get(n);
    if (i === undefined) return undefined;
    const l = lotes[i];
    return l.filas.length > 1 && abierto(l) ? `p${n}` : `l${l.n}`;
  }
  const foco = focoClave !== null && filas.some((r) => r.clave === focoClave) ? focoClave : (claveDe(actual) ?? filas[0]?.clave);

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

  // Un salto pendiente: se hace cuando la fila de ese paso ya está pintada (la página puede estar llegando del servidor).
  const salto = useRef<{ n: number; bloque: ScrollLogicalPosition; suave: boolean; enfocar: boolean } | null>(null);
  function saltarSiSePuede() {
    const s = salto.current;
    const clave = s ? claveDe(s.n) : undefined;
    const fila = clave ? filasRef.current.get(clave) : undefined;
    if (!s || !fila) return;
    salto.current = null;
    fila.scrollIntoView({ behavior: s.suave ? "smooth" : "auto", block: s.bloque });
    if (s.enfocar) fila.focus({ preventScroll: true });
  }
  useEffect(saltarSiSePuede);

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
      mostrarPosicion(desde + i);
      setElegido(n);
      setFocoClave(null);
      salto.current = { n, bloque: "start", suave: true, enfocar: false };
      requestAnimationFrame(saltarSiSePuede);
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

  // Seguir: llega un paso nuevo y nadie está usando la ventana (sin foco, o con foco pero sin tocarla hace un minuto:
  // mirar otra pantalla no le quita el foco) → se elige el último y la vista baja hasta él. Mientras se usa no se toca
  // nada: quien lee no debe ver saltar la página. El salto es directo, en el mismo efecto, sin animación, temporizador
  // ni requestAnimationFrame: con la ventana oculta o minimizada el navegador los frena hasta un minuto.
  const seguir = ajustes.seguir && enVivo;
  const bajar = useRef(false);
  const ultimoGesto = useRef(0);
  useEffect(() => {
    const anotar = () => {
      ultimoGesto.current = Date.now();
    };
    anotar();
    const gestos = ["pointermove", "pointerdown", "keydown", "wheel"] as const;
    for (const g of gestos) window.addEventListener(g, anotar, { passive: true });
    return () => {
      for (const g of gestos) window.removeEventListener(g, anotar);
    };
  }, []);
  useEffect(() => {
    if (!seguir || ultimo < 0 || (document.hasFocus() && Date.now() - ultimoGesto.current < SIN_TOCAR_MS)) return;
    setElegido(null);
    setFocoClave(null);
    // A la última página, anclada en lo de ahora: llena y con el último paso (nadie mira, así que puede correrse).
    setHoja({ agente: idAgente, ancla: total, pagina: Number.MAX_SAFE_INTEGER });
    bajar.current = true;
    // eslint-disable-next-line react-hooks/exhaustive-deps -- solo con un paso nuevo
  }, [seguir, ultimo]);
  // La bajada, cuando ya está pintada la fila del último paso: este efecto corre también en la misma pasada que el de
  // arriba, antes de que la página nueva exista, y sin la fila la pendiente no se gasta.
  useEffect(() => {
    if (!bajar.current) return;
    const clave = claveDe(ultimo);
    const fila = clave ? filasRef.current.get(clave) : undefined;
    if (!fila) return;
    bajar.current = false;
    fila.scrollIntoView({ block: "center" });
  });

  function alternar(l: Lote) {
    const copia = new Set(abiertos);
    if (abierto(l)) {
      copia.delete(l.n);
      // Si estaba abierto por un salto, cerrarlo también lo suelta.
      if (forzado !== null && loteDe.get(forzado) === lotes.indexOf(l)) setForzado(null);
    } else copia.add(l.n);
    setAbiertos(copia);
  }

  /** La elección y el foco itinerante van a esa fila; el foco real, cuando ya está pintada (puede ser una que se acaba de abrir). */
  const enfocarFila = useRef<string | null>(null);
  useEffect(() => {
    if (enfocarFila.current === null) return;
    filasRef.current.get(enfocarFila.current)?.focus();
    enfocarFila.current = null;
  });
  function moverA(r: Renglon, enfocar = false) {
    setElegido(r.hijo ? r.hijo.paso.n : r.lote.n);
    setFocoClave(r.clave);
    if (enfocar) enfocarFila.current = r.clave;
  }

  // «Retoma» y el mapa: va a ese paso, en la página que le toque. Abrir su lote es para ver justo ese paso; desde el mapa
  // basta con llegar al tramo.
  function irA(n: number, abrir = true): boolean {
    if (n < 0 || n >= total) return false;
    mostrarPosicion(n);
    setElegido(n);
    setFocoClave(null);
    if (abrir) setForzado(n);
    salto.current = { n, bloque: "center", suave: true, enfocar: abrir };
    requestAnimationFrame(saltarSiSePuede);
    return true;
  }

  function alTecla(e: KeyboardEvent<HTMLDivElement>) {
    // La fila que tiene el foco de verdad; el foco itinerante puede ir detrás si llegó de otro modo que con el Tab.
    const enFoco = [...filasRef.current].find(([, b]) => b === e.target)?.[0] ?? foco;
    const i = filas.findIndex((r) => r.clave === enFoco);
    const r = filas[i];
    if (!r) return;
    let j = i;
    if (e.key === "ArrowDown") j = Math.min(filas.length - 1, i + 1);
    else if (e.key === "ArrowUp") j = Math.max(0, i - 1);
    else if (e.key === "Home") j = 0;
    else if (e.key === "End") j = filas.length - 1;
    else if (e.key === "ArrowRight") {
      if (!r.hijo && r.lote.filas.length > 1) {
        if (abierto(r.lote)) j = i + 1;
        else alternar(r.lote);
      }
    } else if (e.key === "ArrowLeft") {
      if (r.hijo) j = filas.findIndex((o) => o.clave === `l${r.lote.n}`);
      else if (r.lote.filas.length > 1 && abierto(r.lote)) alternar(r.lote);
    } else return;
    e.preventDefault();
    if (j !== i) moverA(filas[j], true);
  }

  // Totales por turno, para su rótulo.
  const totalTurno = new Map<number, number>();
  for (const p of visibles) totalTurno.set(p.turno, (totalTurno.get(p.turno) ?? 0) + p.tok_entra + salidaDelPaso(p));

  const textoPaso = (f: Fila): string => {
    const p = f.paso;
    const marcas: string[] = [];
    if (p.desvio) marcas.push(t("se desvía de su hilo"));
    const q = p.continua >= 0 ? porN.get(p.continua) : undefined;
    if (q) marcas.push(t("vuelve a {hilo} de las {hora}", { hilo: p.hilo, hora: hora(new Date(q.hora)) }));
    if (p.paralelo > 1 && p.herramienta !== "Respuesta") marcas.push(t("en paralelo con {n} más", { n: p.paralelo - 1 }));
    if (p.subagente) marcas.push(t("delega en un subagente"));
    // La etapa va primero: el lector de pantalla la oye donde empieza, como se ve su marca.
    const etapa = f.marca ? `${nombreEtapa(f.marca)}. ` : "";
    return `${etapa}${hora(new Date(p.hora))}, ${t(p.tipo)}, ${p.resumen || p.herramienta}${marcas.length ? ". " + marcas.join(". ") : ""}`;
  };
  const textoLote = (l: Lote, i: number): string => {
    if (l.filas.length === 1) return textoPaso(l.filas[0]);
    const etapa = l.marca ? `${nombreEtapa(l.marca)}. ` : "";
    return `${etapa}${franja(l, i, enlaces, hora(new Date(l.filas[0].paso.hora)))}. ${resumenLote(l)}`;
  };
  const textoFranja = eleccion >= 0 ? franja(lotes[eleccion], eleccion, enlaces, hora(new Date(lotes[eleccion].filas[0].paso.hora))) : "";

  /** La etapa, dentro de su fila: una marca de ancho tope que se recorta, con el nombre entero en el title. En el teléfono no cabe: queda la raya sobre el riel y la etiqueta de la fila. */
  const marcaEtapa = (m: Fila["marca"]) =>
    m &&
    !cols.angosto && (
      <span
        aria-hidden="true"
        title={nombreEtapa(m)}
        className="max-w-[60%] flex-none truncate rounded-chip border border-acento px-1.5 text-[10.5px] leading-[18px] font-semibold text-acento-fuerte"
      >
        {m.sigue ? "↳ " : ""}
        {nombreEtapa(m)}
      </span>
    );

  /** Los tokens de una fila: una sola línea siempre (lo que cambia en vivo no mueve el diseño). */
  const celdaTokens = (lista: readonly Fila[], barra: boolean) => {
    const tok = tokDominante(lista);
    return (
      <span
        title={t("Entró al contexto ≈ {entra} · generó {sale} · contexto tras el paso ≈ {tras}", {
          entra: tokensCortos(tok.entra),
          sale: tokensCortos(tok.sale),
          tras: tokensCortos(lista[lista.length - 1].paso.tok_tras),
        })}
        className="flex min-w-0 flex-col items-end gap-[3px] border-l border-borde pl-2"
      >
        <span className={`dato max-w-full truncate text-ayuda ${tok.sube ? "text-opcional" : "text-suave"}`}>{tok.texto}</span>
        {barra && (
          <span aria-hidden="true" className="flex h-[2px] w-full justify-end rounded-full bg-elevado">
            <span className={`h-[2px] rounded-full opacity-70 ${tok.sube ? "bg-opcional" : "bg-suave"}`} style={{ width: `${Math.max(4, (tok.valor / max) * 100)}%` }} />
          </span>
        )}
      </span>
    );
  };

  const claseFila = "grid w-full scroll-mt-28 items-center gap-x-2 pr-3 text-left transition-colors duration-150 focus-visible:[outline-offset:-2px]";

  return (
    <section aria-labelledby={`${base}-t`} className="flex flex-col gap-4 rounded-tarjeta border border-borde bg-superficie px-6 py-5 max-sm:px-3">
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
            title={t("Seguir: la vista baja sola al último paso cuando la ventana no tiene el foco o lleva un minuto sin que la toques. Mientras la usas no se mueve.")}
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
          <div className="flex min-w-0 flex-[3_1_40rem] flex-col gap-2">
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
            {/* La franja de la elección: un hueco de alto fijo con una línea, el resto en su title. */}
            <div className="box-border flex h-[30px] items-center overflow-hidden rounded-control bg-hundido px-2.5 text-ayuda whitespace-nowrap text-suave">
              <span className="truncate" title={textoFranja}>
                {textoFranja}
              </span>
            </div>
            <div ref={medir} className="flex overflow-clip rounded-panel border border-borde">
              <div
                role="group"
                aria-label={t("Pasos por lote; arriba y abajo mueven la elección, derecha e izquierda abren y cierran un lote")}
                onKeyDown={alTecla}
                className="relative min-w-0 flex-1"
              >
                {vieja && !viejas[tramo] && (
                  <p className={`absolute inset-x-0 top-0 z-10 m-0 px-3 py-3 text-rotulo ${errorVieja ? "text-error" : "text-suave"}`} aria-live="polite">
                    {errorVieja ? t("No se pudieron traer esos pasos: {error}", { error: errorVieja }) : t("Trayendo los pasos {desde}–{hasta}…", { desde: vista.inicio + 1, hasta: vista.fin })}
                  </p>
                )}
                {filas.map((r) => {
                  const l = r.lote;
                  const refFila = (b: HTMLButtonElement | null) => {
                    if (b) filasRef.current.set(r.clave, b);
                    else filasRef.current.delete(r.clave);
                  };
                  const tabIndex = r.clave === foco ? 0 : -1;
                  // Un paso de un lote abierto: 30 px sobre el fondo hundido.
                  if (r.hijo) {
                    const f = r.hijo;
                    const p = f.paso;
                    const esElegido = p.n === actual;
                    return (
                      <button
                        key={r.clave}
                        ref={refFila}
                        type="button"
                        tabIndex={tabIndex}
                        aria-pressed={esElegido}
                        aria-label={textoPaso(f)}
                        onClick={() => moverA(r)}
                        style={{ height: r.alto, gridTemplateColumns: cols.plantilla }}
                        className={`${claseFila} text-ayuda ${esElegido ? "bg-seleccion" : "bg-hundido hover:bg-elevado/60"}`}
                      >
                        <span className="dato text-right text-[11px] whitespace-nowrap text-tenue" title={conSegundos(p.hora)}>
                          {hora(new Date(p.hora))}
                        </span>
                        <span />
                        <span />
                        <span className="flex min-w-0 items-center gap-2 pl-1.5">
                          {f.marca && f.marca !== l.marca && marcaEtapa(f.marca)}
                          <span className={`shrink-0 text-[11.5px] ${TIPO_TEXTO[p.tipo] ?? "text-suave"}`}>{t(p.tipo)}</span>
                          <span className={`min-w-0 flex-1 truncate ${esElegido ? "text-texto" : "text-suave"}`} title={p.resumen || p.herramienta}>
                            {p.resumen || p.herramienta}
                          </span>
                          {p.subagente && <span className="shrink-0 rounded-full border border-dashed border-opcional px-2 text-[11px] text-opcional max-sm:hidden">→ {t("subagente")}</span>}
                        </span>
                        {cols.archivos === "nombre" ? (
                          <span className={`dato truncate text-[11.5px] ${p.error ? "text-error" : "text-tenue"}`}>{p.hecho ? metaPaso(p.meta) : t("en curso")}</span>
                        ) : cols.archivos ? (
                          <span />
                        ) : null}
                        {cols.tokens && celdaTokens([f], false)}
                        <span />
                      </button>
                    );
                  }
                  const esElegido = r.i === eleccion;
                  const uno = l.filas.length === 1;
                  const p = l.filas[0].paso;
                  const abre = !uno && abierto(l);
                  const delegados = l.filas.filter((f) => f.paso.subagente).length;
                  const estilo = { height: r.alto, gridTemplateColumns: cols.plantilla };
                  const raya = r.i > 0 ? "border-t border-borde/40" : "";
                  const fondo = esElegido ? "bg-seleccion" : "hover:bg-elevado/60";
                  const celdaHora = (
                    <span className="dato text-right text-ayuda whitespace-nowrap text-tenue" title={conSegundos(p.hora)}>
                      {hora(new Date(p.hora))}
                    </span>
                  );
                  // El pedido es la fila de su turno.
                  if (l.turno) {
                    return (
                      <button
                        key={r.clave}
                        ref={refFila}
                        type="button"
                        tabIndex={tabIndex}
                        aria-pressed={esElegido}
                        aria-label={textoPaso(l.filas[0])}
                        onClick={() => moverA(r)}
                        style={estilo}
                        className={`${claseFila} box-border ${raya} ${fondo}`}
                      >
                        {celdaHora}
                        <span />
                        <span className="flex min-w-0 items-center gap-2.5" style={{ gridColumn: "3 / -1" }}>
                          <span className="dato shrink-0 rounded-chip bg-opcional-suave px-[7px] text-[11px] leading-[18px] font-semibold whitespace-nowrap text-opcional">
                            {t("Turno {n}", { n: p.turno })}
                          </span>
                          <span className="min-w-0 flex-1 truncate font-medium" title={p.resumen}>
                            {p.resumen}
                          </span>
                          {cols.tokens && (
                            <span className="dato shrink-0 text-ayuda whitespace-nowrap text-tenue">
                              {t("{tokens} en el turno", { tokens: tokensCortos(totalTurno.get(p.turno) ?? 0) })}
                            </span>
                          )}
                        </span>
                      </button>
                    );
                  }
                  const resumen = uno ? `${t(p.tipo)}: ${p.resumen || p.herramienta}` : resumenLote(l);
                  return (
                    <button
                      key={r.clave}
                      ref={refFila}
                      type="button"
                      tabIndex={tabIndex}
                      aria-pressed={esElegido}
                      aria-expanded={uno ? undefined : abre}
                      aria-label={textoLote(l, r.i)}
                      onClick={() => {
                        moverA(r);
                        if (!uno) alternar(l);
                      }}
                      style={estilo}
                      className={`${claseFila} box-border ${raya} ${fondo}`}
                    >
                      {celdaHora}
                      <span />
                      <span className="dato text-right text-ayuda text-acento-fuerte tabular-nums" title={uno ? undefined : t("{n} pasos del mismo mensaje, en paralelo", { n: l.filas.length })}>
                        {uno ? "" : `∥ ${l.filas.length}`}
                      </span>
                      <span className="flex min-w-0 items-center gap-2 pl-1.5">
                        {marcaEtapa(l.marca)}
                        <span className="min-w-0 flex-1 truncate" title={resumen}>
                          {uno ? (
                            <>
                              <span className={TIPO_TEXTO[p.tipo] ?? "text-suave"}>{t(p.tipo)}:</span> {p.resumen || p.herramienta}
                            </>
                          ) : (
                            resumen
                          )}
                        </span>
                        {!uno && l.errores > 0 && <span className="shrink-0 text-ayuda text-error">{t("{n} con error", { n: l.errores })}</span>}
                        {delegados > 0 && (
                          <span className="shrink-0 rounded-full border border-dashed border-opcional px-2 text-[11.5px] text-opcional max-sm:hidden">
                            → {delegados > 1 ? delegados : t("subagente")}
                          </span>
                        )}
                      </span>
                      {cols.archivos && (
                        <span className="flex min-w-0 items-center gap-1 text-ayuda text-suave">
                          {l.archivos.slice(0, 6).map((f) => (
                            <span key={f} aria-hidden="true" className="size-[7px] shrink-0 rounded-full" style={{ background: colorHilo(f) }} />
                          ))}
                          {cols.archivos === "nombre" &&
                            (l.archivos.length > 0 ? (
                              <span className="ml-1 min-w-0 truncate" title={l.archivos.map(nombreArchivo).join(", ")}>
                                {nombreArchivo(l.archivos[0])}
                                {l.archivos.length > 1 ? ` ${t("y {n} más", { n: l.archivos.length - 1 })}` : ""}
                              </span>
                            ) : (
                              uno && <span className={`dato truncate ${p.error ? "text-error" : "text-tenue"}`}>{p.hecho ? metaPaso(p.meta) : t("en curso")}</span>
                            ))}
                        </span>
                      )}
                      {cols.tokens && celdaTokens(l.filas, true)}
                      <Flecha abierto={abre} oculta={uno} />
                    </button>
                  );
                })}
                <svg aria-hidden="true" width={cols.canaleta} height={alto} className="pointer-events-none absolute top-0 overflow-visible" style={{ left: cols.hora + 8 }}>
                  {trazos.map((tz, i) => (
                    <path key={i} d={tz.d} fill="none" stroke={tz.color} strokeWidth={tz.ancho} opacity={tz.opacidad} strokeDasharray={tz.raya} strokeLinecap="round" strokeLinejoin="round" />
                  ))}
                  {/* Donde empieza una etapa, una marca corta cruza el riel en el borde de arriba de su fila (no tapa el nodo). */}
                  {lotes.map((l, i) =>
                    l.marca && !l.marca.sigue ? <path key={l.n} d={`M ${pista.rail - 5} ${cy[i] - 19} H ${pista.rail + 5}`} stroke="var(--acento)" strokeWidth={2} /> : null,
                  )}
                  {filas.map((r) => {
                    // Un paso abierto: un punto chico con el color de su archivo, relleno si no solo lee.
                    if (r.hijo) {
                      const p = r.hijo.paso;
                      const color = p.error ? "var(--error)" : p.hilo ? colorHilo(p.hilo) : "var(--riel)";
                      const cyHijo = r.y + r.alto / 2;
                      return (
                        <g key={r.clave}>
                          <circle cx={pista.rail} cy={cyHijo} r={4.5} fill="var(--fondo-hundido)" />
                          <circle cx={pista.rail} cy={cyHijo} r={2.5} fill={p.tipo === "Lee" ? "var(--fondo-hundido)" : color} stroke={color} strokeWidth={1.5} />
                        </g>
                      );
                    }
                    const l = r.lote;
                    const hay = l.archivos.length > 0;
                    const radio = hay || l.turno || l.errores ? 5 : 3.5;
                    const color = l.errores ? "var(--error)" : l.turno ? "var(--opcional)" : hay ? "var(--acento)" : "var(--riel)";
                    const relleno = l.turno ? "var(--opcional-suave)" : l.errores || l.edita || !hay ? color : "var(--superficie)";
                    const esElegido = r.i === eleccion;
                    const vivo = enVivo && l.filas.some((f) => f.paso.n === ultimo);
                    return (
                      <g key={r.clave}>
                        {esElegido ? (
                          <circle cx={pista.rail} cy={cy[r.i]} r={radio + 3} fill="var(--acento)" opacity={0.27} />
                        ) : (
                          <circle cx={pista.rail} cy={cy[r.i]} r={radio + 2} fill="var(--superficie)" />
                        )}
                        <circle cx={pista.rail} cy={cy[r.i]} r={radio - 1} fill={relleno} stroke={color} strokeWidth={2} className={vivo ? "pulso" : undefined} />
                      </g>
                    );
                  })}
                </svg>
                {vieja && !viejas[tramo] && <Fantasmas n={12} className="h-10 bg-hundido" />}
              </div>
              {mapaVisible && mapa && <MapaSesion mapa={mapa} inicio={vista.inicio} fin={finVisible} alto={alto} alIr={(n) => irA(n, false)} />}
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
