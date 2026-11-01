// La cuota de Claude leída con /usage (servidor: GET /api/uso-claude y POST /api/uso-claude/leer; el contrato está en
// docs/trazabilidad/F6/temporales/2026-11-01-uso-claude-contrato.md). Aquí va la lógica pura del panel: la frescura,
// cómo se muestra cada ventana, las series del historial, la escala y la tabla. El panel nunca decide el estado: lo
// toma de `estado`, `edad_s` y `vencida`, y así la franja y el detalle dicen lo mismo.

import { cuando, cuandoReinicia, dos, duracion, hora, porcentaje } from "./formato.ts";
import { idioma, t, tServidor } from "./i18n.ts";

export type EstadoUsoClaude = "ok" | "viejo" | "limitado" | "leyendo" | "sin-claude" | "sin-suscripcion" | "error";

export interface VentanaUsoClaude {
  clave: string;
  etiqueta: string;
  porcentaje: number;
  reinicia_en: number;
  duracion_min: number;
  severidad: string;
  activa: boolean;
  vencida: boolean;
}

/** Un punto del historial: la hora (unix s) y el % de cada ventana. `r`, si el servidor lo manda, los reinicios. */
export interface PuntoUsoClaude {
  t: number;
  v: Record<string, number>;
  r?: Record<string, number>;
}

export interface UsoClaude {
  estado: EstadoUsoClaude;
  detalle: string;
  tomada_en: number;
  edad_s: number;
  proxima_en: number;
  limitado_hasta: number | null;
  ventanas: VentanaUsoClaude[];
  extra_activo: boolean;
  historial: PuntoUsoClaude[];
}

/** Los periodos del gráfico, en horas (`?horas=`). */
export const PERIODOS = [
  { horas: 24, rotulo: "24 h" },
  { horas: 168, rotulo: "7 días" },
  { horas: 720, rotulo: "30 días" },
] as const;

/** Si la franja de Claude sale de /usage: cuando el servidor la tiene y Claude Code está. */
export const usaUsoClaude = (u: UsoClaude | null): u is UsoClaude => !!u && u.estado !== "sin-claude";

/** El nombre del modelo de una ventana `weekly_scoped:<modelo>`; vacío si no es de un modelo. */
const modeloDe = (clave: string) => (clave.startsWith("weekly_scoped:") ? clave.slice("weekly_scoped:".length) : "");

/** El nombre de la ventana en el detalle y la leyenda: «5 horas», «Semana · todos los modelos», «Semana · Fable». */
export function etiquetaVentana(clave: string, etiquetaServidor = ""): string {
  if (clave === "session") return t("5 horas");
  if (clave === "weekly_all") return t("Semana · todos los modelos");
  const m = modeloDe(clave);
  if (m) return t("Semana · {modelo}", { modelo: m });
  return etiquetaServidor ? tServidor(etiquetaServidor) : clave;
}

/** El nombre corto, para la franja: «5 h», «Semana», «Fable». */
export function etiquetaCorta(clave: string, etiquetaServidor = ""): string {
  if (clave === "session") return "5 h";
  if (clave === "weekly_all") return t("Semana");
  return modeloDe(clave) || (etiquetaServidor ? tServidor(etiquetaServidor) : clave);
}

/** Los segundos desde la lectura buena: corre con el reloj si hay `tomada_en`; si no, el `edad_s` del servidor. */
export function edad(u: UsoClaude, ahora: number): number {
  if (ahora > 0 && u.tomada_en > 0) return Math.max(0, ahora - u.tomada_en);
  return Math.max(0, u.edad_s);
}

export type Tono = "normal" | "aviso" | "error" | "tenue";

export interface Frescura {
  texto: string;
  tono: Tono;
  /** El texto largo para el `title`: la hora de la lectura y el porqué del servidor. */
  titulo: string;
}

/**
 * La línea de frescura de la franja y del detalle, a partir de `estado`, `edad_s`, `limitado_hasta` y el `detalle` del
 * servidor. `espera` es el `proxima_en` de un 429: mientras no llega, se dice cuándo se puede volver a leer (no es un
 * error). `fallo`: la última consulta al servidor no respondió.
 */
export function frescura(u: UsoClaude, ahora: number, espera = 0, fallo = false): Frescura {
  const hay = u.tomada_en > 0;
  const partes = [hay ? t("Lectura de las {hora}", { hora: cuando(new Date(u.tomada_en * 1000)) }) : t("Sin lectura todavía")];
  if (u.detalle) partes.push(tServidor(u.detalle));
  const titulo = partes.join(". ");
  const hace = duracion(edad(u, ahora));
  if (fallo) return { texto: t("sin respuesta de Atalaya"), tono: "aviso", titulo };
  if (espera > 0 && ahora > 0 && ahora < espera) {
    return { texto: t("se puede volver a leer a las {hora}", { hora: hora(new Date(espera * 1000)) }), tono: "tenue", titulo };
  }
  switch (u.estado) {
    case "ok":
      return { texto: t("hace {d} · /usage", { d: hace }), tono: "normal", titulo };
    case "viejo":
      return { texto: hay ? t("vieja: hace {d}", { d: hace }) : t("sin lectura todavía"), tono: "aviso", titulo };
    case "limitado":
      return u.limitado_hasta
        ? { texto: t("limitada hasta las {hora}", { hora: hora(new Date(u.limitado_hasta * 1000)) }), tono: "aviso", titulo }
        : { texto: t("limitada por /usage"), tono: "aviso", titulo };
    case "leyendo":
      return { texto: t("leyendo /usage…"), tono: "tenue", titulo };
    case "sin-suscripcion":
      return { texto: t("sin suscripción: /usage no da la cuota"), tono: "aviso", titulo };
    case "error":
      return { texto: t("no se pudo leer /usage"), tono: "error", titulo };
    default:
      return { texto: "", tono: "tenue", titulo };
  }
}

export interface VistaVentana {
  /** El %, o null si la ventana ya se reinició y falta leerla (nunca el % viejo). */
  pct: number | null;
  /** La cuenta atrás («reinicia en 2 h 10 min») o «se reinició · falta leerla». */
  cuenta: string;
  /** Lo mismo, con la hora del reinicio, para el `title`. */
  titulo: string;
}

/**
 * Cómo se muestra una ventana. `vencida` la decide el servidor; mientras llega la lectura siguiente (un par de
 * segundos), un reinicio que ya pasó en el reloj también se trata como vencido, para no mostrar nunca el % viejo.
 */
export function vistaVentana(v: VentanaUsoClaude, ahora: number): VistaVentana {
  const vencida = v.vencida || (ahora > 0 && v.reinicia_en > 0 && v.reinicia_en <= ahora);
  if (vencida) return { pct: null, cuenta: t("se reinició · falta leerla"), titulo: t("Se reinició; falta leerla otra vez") };
  const pct = Math.min(100, Math.max(0, v.porcentaje));
  if (!(v.reinicia_en > 0) || ahora <= 0) return { pct, cuenta: "", titulo: "" };
  const d = duracion(v.reinicia_en - ahora);
  return {
    pct,
    cuenta: t("reinicia en {d}", { d }),
    titulo: t("Reinicia {cuando}, en {d}", { cuando: cuandoReinicia(v.reinicia_en, ahora), d }),
  };
}

/** Una serie del gráfico: la ventana, su nombre y su color (0 a 2; desde la cuarta, el 2 con trazo discontinuo). */
export interface Serie {
  clave: string;
  etiqueta: string;
  color: 0 | 1 | 2;
  discontinua: boolean;
}

const RANGO_CLAVE = (c: string) => (c === "session" ? 0 : c === "weekly_all" ? 1 : 2);

/**
 * Las series en orden fijo: 5 horas, la semana, y las de cada modelo por nombre. El color sigue a la ventana, nunca a
 * su posición en la lista: una ventana que aparece o se va no repinta a las demás.
 */
export function seriesDe(u: UsoClaude): Serie[] {
  const etiquetas = new Map(u.ventanas.map((v) => [v.clave, v.etiqueta]));
  const claves = new Set(u.ventanas.map((v) => v.clave));
  for (const p of u.historial) for (const c of Object.keys(p.v)) claves.add(c);
  const orden = [...claves].sort((a, b) => RANGO_CLAVE(a) - RANGO_CLAVE(b) || a.localeCompare(b));
  let modelos = 0;
  return orden.map((clave) => {
    const r = RANGO_CLAVE(clave);
    const extra = r === 2 && modelos++ > 0;
    return { clave, etiqueta: etiquetaVentana(clave, etiquetas.get(clave)), color: r as 0 | 1 | 2, discontinua: extra };
  });
}

/** Más de esto entre dos lecturas (se leen cada 10 min) y la línea se corta: no se inventa lo que no se leyó. */
export const HUECO_S = 35 * 60;

export interface Reinicio {
  t: number;
  claves: string[];
  /** Verdadero si la hora viene del servidor; falso si se ve solo por la bajada (pasó entre `desde` y `t`). */
  exacto: boolean;
  desde: number;
}

export interface Trazos {
  /** Por ventana, sus tramos: cada uno, puntos [t, %] seguidos, sin huecos ni reinicios por medio. */
  tramos: Map<string, [number, number][][]>;
  reinicios: Reinicio[];
}

/** Cuánto tiene que caer una ventana sin hora de reinicio para contarlo como reinicio: menos es redondeo. */
export const CAIDA_REINICIO = 5;

/**
 * Arma los tramos de cada serie dentro de [ini, fin]. Dentro de una ventana el uso solo crece: la línea sigue el máximo
 * acumulado, así las dos fuentes (`/usage` y el mod, que redondean distinto) no la hacen saltar 57-58-57. Un reinicio
 * es real si pasó la hora que da el servidor (`r`) o si la ventana cae `CAIDA_REINICIO` puntos o más; entonces la
 * línea baja en vertical a 0 a esa hora y sigue, sin hueco, y se marca. Solo un hueco de más de `hueco` segundos corta
 * el tramo.
 */
export function trazos(historial: PuntoUsoClaude[], claves: string[], ini: number, fin: number, hueco = HUECO_S): Trazos {
  const pts = historial.filter((p) => p.t >= ini && p.t <= fin).sort((a, b) => a.t - b.t);
  const tramos = new Map<string, [number, number][][]>();
  const marcas = new Map<number, Reinicio>();
  const marcar = (tt: number, clave: string, exacto: boolean, desde: number) => {
    const k = Math.round(tt / 60) * 60;
    const m = marcas.get(k);
    if (m) {
      if (!m.claves.includes(clave)) m.claves.push(clave);
      m.exacto = m.exacto && exacto;
    } else marcas.set(k, { t: k, claves: [clave], exacto, desde });
  };
  for (const clave of claves) {
    const lista: [number, number][][] = [];
    let actual: [number, number][] = [];
    let previo: PuntoUsoClaude | null = null;
    let maximo = 0; // el máximo de la ventana en curso: lo que se dibuja
    for (const p of pts) {
      const crudo = p.v[clave];
      if (typeof crudo !== "number" || !Number.isFinite(crudo)) continue;
      const v = Math.min(100, Math.max(0, crudo));
      if (previo) {
        const rp = previo.r?.[clave];
        const reinicioServidor = typeof rp === "number" && rp > previo.t && rp <= p.t;
        const caida = v <= maximo - CAIDA_REINICIO;
        if (p.t - previo.t > hueco) {
          if (actual.length) lista.push(actual);
          actual = [];
          if (reinicioServidor || caida) {
            marcar(reinicioServidor ? rp : p.t, clave, reinicioServidor, previo.t);
            maximo = 0;
          }
        } else if (reinicioServidor || caida) {
          // La bajada a 0 a la hora del reinicio (o, sin ella, en esta lectura), en el mismo tramo: sin hueco.
          const tr = reinicioServidor ? rp : p.t;
          marcar(tr, clave, reinicioServidor, previo.t);
          actual.push([tr, maximo], [tr, 0]);
          maximo = 0;
        }
      }
      maximo = Math.max(maximo, v);
      actual.push([p.t, maximo]);
      previo = p;
    }
    if (actual.length) lista.push(actual);
    tramos.set(clave, lista);
  }
  return { tramos, reinicios: [...marcas.values()].sort((a, b) => a.t - b.t) };
}

/** La escala: de unix s y % a píxeles del área de trazado. El eje y va siempre de 0 a 100 %. */
export function escala(ini: number, fin: number, ancho: number, alto: number) {
  const span = Math.max(1, fin - ini);
  return {
    x: (tt: number) => ((tt - ini) / span) * ancho,
    y: (pct: number) => alto - (Math.min(100, Math.max(0, pct)) / 100) * alto,
    t: (x: number) => ini + (Math.min(ancho, Math.max(0, x)) / Math.max(1, ancho)) * span,
  };
}

/** El `d` de un trazo SVG con sus tramos. */
export function camino(tramos: [number, number][][], x: (t: number) => number, y: (p: number) => number): string {
  return tramos
    .map((tr) => tr.map(([tt, v], i) => `${i ? "L" : "M"}${x(tt).toFixed(1)},${y(v).toFixed(1)}`).join(""))
    .join("");
}

const DIA_CORTO = ["dom", "lun", "mar", "mié", "jue", "vie", "sáb"];
const DAY_SHORT = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

/** Las marcas del eje del tiempo: cada 3 h en 24 h, cada día en 7 días y cada 5 días en 30, en hora local. */
export function marcasTiempo(ini: number, fin: number, horas: number): { t: number; rotulo: string }[] {
  const fuera: { t: number; rotulo: string }[] = [];
  const f = new Date(ini * 1000);
  if (horas <= 48) {
    f.setMinutes(0, 0, 0);
    const paso = horas <= 24 ? 3 : 6;
    f.setHours(Math.ceil(f.getHours() / paso) * paso);
    for (; f.getTime() / 1000 <= fin; f.setHours(f.getHours() + paso)) {
      const tt = f.getTime() / 1000;
      if (tt >= ini) fuera.push({ t: tt, rotulo: hora(f) });
    }
    return fuera;
  }
  f.setHours(0, 0, 0, 0);
  f.setDate(f.getDate() + 1);
  const paso = horas <= 24 * 8 ? 1 : 5;
  for (; f.getTime() / 1000 <= fin; f.setDate(f.getDate() + paso)) {
    const tt = f.getTime() / 1000;
    const rotulo =
      paso === 1
        ? `${(idioma() === "en" ? DAY_SHORT : DIA_CORTO)[f.getDay()]} ${f.getDate()}`
        : idioma() === "en"
          ? `${dos(f.getMonth() + 1)}/${dos(f.getDate())}`
          : `${dos(f.getDate())}/${dos(f.getMonth() + 1)}`;
    fuera.push({ t: tt, rotulo });
  }
  return fuera;
}

/** El índice del punto más cercano a `tt` en una lista ordenada por hora; -1 si está vacía. */
export function masCercano(pts: { t: number }[], tt: number): number {
  if (!pts.length) return -1;
  let a = 0;
  let b = pts.length - 1;
  while (a < b) {
    const m = (a + b) >> 1;
    if (pts[m].t < tt) a = m + 1;
    else b = m;
  }
  if (a > 0 && Math.abs(pts[a - 1].t - tt) <= Math.abs(pts[a].t - tt)) return a - 1;
  return a;
}

/**
 * Las etiquetas al final de cada línea, sin pisarse: se ordenan por altura y se separan `sep` píxeles como mínimo,
 * dentro de [0, alto]. Devuelve la y de cada una, en el orden en que llegaron.
 */
export function separarEtiquetas(ys: number[], sep: number, alto: number): number[] {
  const orden = ys.map((y, i) => ({ y, i })).sort((a, b) => a.y - b.y);
  for (let k = 1; k < orden.length; k++) orden[k].y = Math.max(orden[k].y, orden[k - 1].y + sep);
  const exceso = orden.length ? orden[orden.length - 1].y - alto : 0;
  if (exceso > 0) for (const o of orden) o.y -= exceso;
  for (let k = orden.length - 2; k >= 0; k--) orden[k].y = Math.min(orden[k].y, orden[k + 1].y - sep);
  const fuera = new Array<number>(ys.length);
  for (const o of orden) fuera[o.i] = o.y;
  return fuera;
}

/**
 * La tabla alternativa al gráfico: la última lectura de cada tramo de tiempo (una hora en 24 h, seis en 7 días, un
 * día en 30), de la más nueva a la más vieja.
 */
export function filasTabla(historial: PuntoUsoClaude[], ini: number, fin: number, horas: number): PuntoUsoClaude[] {
  const tramo = (horas <= 24 ? 1 : horas <= 24 * 8 ? 6 : 24) * 3600;
  const ultimo = new Map<number, PuntoUsoClaude>();
  for (const p of historial) {
    if (p.t < ini || p.t > fin) continue;
    const k = Math.floor(p.t / tramo);
    const previo = ultimo.get(k);
    if (!previo || p.t > previo.t) ultimo.set(k, p);
  }
  return [...ultimo.values()].sort((a, b) => b.t - a.t);
}

/** Un % del historial o una raya si esa ventana no se leyó en ese punto. */
export const pctOGuion = (v: number | undefined) => (typeof v === "number" ? porcentaje(v) : "—");

/**
 * Cuándo volver a pedir el estado sin esperar al aviso del servidor: al minuto, o justo después del reinicio más
 * próximo o del fin de la limitación, para que una ventana vencida o una lectura vieja se vean a tiempo. En segundos.
 */
export function proximaConsulta(u: UsoClaude, ahora: number): number {
  let s = 60;
  const futuros = [...u.ventanas.map((v) => v.reinicia_en), u.limitado_hasta ?? 0].filter((x) => x > ahora);
  for (const f of futuros) s = Math.min(s, f - ahora + 2);
  return Math.max(2, s);
}
