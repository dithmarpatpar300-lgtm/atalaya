// Tipos y llamadas a la API de solo lectura del backend (backend/internal/servidor).

import type { RespuestaAvance } from "./avance.ts";
import { tServidor } from "./i18n.ts";
import type { InformeEstadisticas, PeriodoEstadisticas } from "./estadisticas.ts";
import type { RespuestaDecisiones } from "./decisiones.ts";
import type { PreguntaCentro } from "./preguntas.ts";
import type { RespuestaLinea } from "./linea.ts";
import type { ProyectoPortafolio } from "./portafolio.ts";
import type { ResumenSesion } from "./sesion.ts";
import type { InformeProceso, Periodo } from "./salud.ts";

export interface Ritmo {
  full_at: number;
  percent_per_hour: number;
}

export interface VentanaCuota {
  duration_mins: number;
  label: string;
  pace: Ritmo | null;
  resets_at: number;
  tokens: { cache_read: number; cache_write: number; fresh: number; input: number; messages: number; output: number } | null;
  tokens_left_estimate: number | null;
  used_percent: number;
}

export interface PanelCuota {
  captured_at: number;
  notice: string | null;
  plan: string | null;
  provider: string;
  source: string;
  windows: VentanaCuota[];
}

export interface PosturaProveedor {
  even_pace_percent_per_hour: number;
  headroom_percent: number;
  provider: string;
  reasons: string[];
  stance: string;
}

export interface EstadoCuotas {
  lectura: {
    advice: { first_choice: string; providers: PosturaProveedor[]; summary: string } | null;
    panels: PanelCuota[];
    taken_at: number;
  } | null;
  origen: "ejecutable" | "archivos" | "";
  leida_en: number;
  error?: string;
  /** Falso si la herramienta opcional Cuotas no está (ni su ejecutable ni sus archivos): la franja no se muestra. */
  instalada: boolean;
}

export interface Tokens {
  entrada: number;
  cache_escrita: number;
  cache_leida: number;
  salida: number;
}

export interface Avance {
  porcentaje: number;
  texto: string;
  hitos: { hora: string; texto: string }[];
  ruta: string;
}

export type EstadoAgente = "Trabajando" | "En espera" | "Entregado" | "Inactiva";

export interface Agente {
  id: string;
  proyecto: string;
  proveedor: "Claude" | "Codex";
  tipo: "sesion" | "subagente" | "codex";
  rol: string;
  tarea: string;
  /** El nombre del agente, uno solo para todo el panel (servidor: nombreVista). */
  nombre?: string;
  /** El modelo corto, igual en todas las secciones: «sonnet-5-5/high». */
  modelo_corto?: string;
  /** El nombre que Claude Code le dio a la sesión (custom-title, ai-title o summary), si tiene. */
  titulo?: string;
  /** Una sesión interactiva de Codex (la TUI), no una llamada de codex.ps1. */
  interactiva?: boolean;
  modelo: string;
  effort: string;
  clase: string;
  presupuesto: string;
  base: string;
  estado: EstadoAgente;
  inicio: string;
  ultimo: string;
  tokens: Tokens;
  avance: Avance | null;
  bitacora: string;
  informe: string;
  pasos: number;
  /** La sesión que lo lanzó: la de un subagente, o la que corrió codex.ps1 para una llamada a Codex. */
  padre?: string;
  asignacion?: string;
  encargo?: string;
  /** Solo en el detalle: la línea «Asignación» partida, los archivos que nombra el encargo y sus entregables. */
  nivel?: string;
  clase_detalle?: string;
  rol_asignado?: string;
  modelo_asignado?: string;
  cuota?: string;
  nombrados?: string[];
  informe_existe?: boolean;
  acta?: string;
  acta_estado?: "abierta" | "cerrada";
}

export interface Peticion {
  hora: string;
  turno: number;
  etapa: string;
  entrada: number;
  cache_escrita: number;
  cache_leida: number;
  salida: number;
  contexto: number;
  resumen: string;
}

export interface ContextoAgente {
  /** Inferida: 1 M si alguna petición pasó de 200 k. */
  ventana: number;
  peticiones: Peticion[];
  compactaciones: { hora: string; antes: number; despues: number; motivo: string; peticion: number }[];
  lleno: { categoria: string; tokens: number }[];
  pesados: { n: number; hora: string; tipo: string; resumen: string; tokens: number }[];
  totales: Tokens;
}

export interface Paso {
  n: number;
  hora: string;
  tipo: string;
  herramienta: string;
  resumen: string;
  meta: string;
  error: boolean;
  hecho: boolean;
  /** Continuidad (backend/internal/claude/continuidad.go). */
  turno: number;
  etapa: string;
  lote: number;
  paralelo: number;
  hilo: string;
  profundidad: number;
  desvio: boolean;
  /** El paso anterior del mismo hilo al que se vuelve tras un desvío; -1 si no. */
  continua: number;
  subagente?: string;
  /** Tokens: lo que el resultado mete al contexto (≈ bytes ÷ 4), la salida del mensaje (de todo su lote) y el contexto antes y después. */
  tok_entra: number;
  tok_salida: number;
  tok_contexto: number;
  tok_tras: number;
}

export interface Trozo {
  viejo_desde: number;
  viejo_lineas: number;
  nuevo_desde: number;
  nuevo_lineas: number;
  /** Cada línea empieza con " ", "+" o "-". */
  lineas: string[];
}

export interface DetallePaso extends Paso {
  orden: string;
  salida: string;
  archivo?: string;
  diff?: Trozo[];
  stdout?: string;
  stderr?: string;
  interrumpido?: boolean;
  entrada_cruda?: string;
  salida_cruda?: string;
}

/** Lo que lleva el Markdown de un agente exportado (backend/internal/servidor/exportar.go). */
export interface OpcionesExportar {
  encargo: boolean;
  resumen: boolean;
  archivos: boolean;
  pasos: boolean;
  diffs: boolean;
  tokens: boolean;
  salidas: "recortadas" | "completas" | "ninguna";
}

export interface Detalle {
  agente: Agente;
  pasos: Paso[];
  desde: number;
}

export interface ProyectoResumen {
  nombre: string;
  agentes: number;
  activos: number;
  ultimo: number;
}

export interface DocumentoDatapedia {
  id: number;
  proyecto: string;
  tipo: string;
  titulo: string;
  resumen: string;
  ruta: string;
  mtime: number;
  tam: number;
  /** En una búsqueda: el trozo que coincide, con las coincidencias entre \u0002 y \u0003. */
  fragmento?: string;
  cuerpo?: string;
}

export interface Busqueda {
  resultados: DocumentoDatapedia[];
  cuentas: { tipo: string; cantidad: number }[];
}

export type NivelAlerta = "error" | "aviso" | "info";

/** Una regla vigilada y de dónde salió su umbral (backend/internal/alertas). */
export interface Regla {
  id: string;
  nombre: string;
  umbral: string;
  nota: string;
  fuente?: string;
  omision: boolean;
}

export interface Alerta {
  id: string;
  nivel: NivelAlerta;
  regla: string;
  fuente?: string;
  proyecto: string;
  agente: string;
  nombre: string;
  titulo: string;
  texto: string;
  hora: string;
  valor: number;
  max: number;
  archivo?: string;
  otro?: string;
  chip: string;
}

export interface RespuestaAlertas {
  reglas: Regla[];
  alertas: Alerta[];
}

export interface EnviadoAviso {
  hora: string;
  titulo: string;
  texto: string;
  seccion: string;
  claves: string[];
  error?: string;
}

export interface EstadoAvisos {
  activos: boolean;
  decisiones: boolean;
  alertas: boolean;
  enviados: EnviadoAviso[];
}

/** Una búsqueda en los pasos de todos los agentes, agrupada por agente (backend/internal/servidor/pasos.go). */
export interface RespuestaPasos {
  total: number;
  grupos: { agente: Agente; pasos: Paso[] }[];
}

/** Una consulta de un agente a Atalaya por MCP (backend/internal/servidor/mcp.go). */
export interface Consulta {
  hora: string;
  cliente: string;
  herramienta: string;
  argumentos: string;
  bytes: number;
  ms: number;
  error: boolean;
  respuesta: string;
  agente?: string;
  nombre?: string;
  /** Para abrir al agente en su proyecto y en el paso de la consulta; paso es -1 si no se sabe. */
  proyecto?: string;
  paso: number;
  /** De dónde salió: el transcrito del agente (sobrevive a un reinicio), el registro en memoria de Atalaya o los dos. */
  origen?: "transcrito" | "memoria" | "transcrito+memoria";
}

/** Lanza el error que manda el servidor (`{ error }`) o, si no trae uno, el código. */
async function siFallo(r: Response) {
  if (r.ok) return;
  let msg = `${r.status}`;
  try {
    msg = ((await r.json()) as { error?: string }).error ?? msg;
  } catch {}
  throw new Error(tServidor(msg));
}

async function pedir<T>(ruta: string, senal?: AbortSignal): Promise<T> {
  const r = await fetch(ruta, { signal: senal, cache: "no-store" });
  await siFallo(r);
  return (await r.json()) as T;
}

/**
 * Las escrituras del panel (responder una pregunta, los avisos), con la cabecera que Atalaya exige
 * (servidor/preguntas_mcp.go y avisos.go).
 */
async function escribirPanel<T>(ruta: string, cuerpo?: unknown): Promise<T> {
  const r = await fetch(ruta, {
    method: "POST",
    headers: { "Content-Type": "application/json", "X-Atalaya-Panel": "1" },
    body: cuerpo === undefined ? undefined : JSON.stringify(cuerpo),
  });
  await siFallo(r);
  return (await r.json()) as T;
}

export const api = {
  cuotas: (leer = false) => pedir<EstadoCuotas>(`/api/cuotas${leer ? "?leer=1" : ""}`),
  proyectos: () => pedir<ProyectoResumen[]>("/api/proyectos"),
  agentes: (proyecto: string) => pedir<Agente[]>(`/api/agentes?filtro=todos&proyecto=${encodeURIComponent(proyecto)}`),
  agente: (id: string, senal?: AbortSignal) => pedir<Detalle>(`/api/agentes/${encodeURIComponent(id)}?max=3000`, senal),
  contexto: (id: string, senal?: AbortSignal) => pedir<ContextoAgente>(`/api/agentes/${encodeURIComponent(id)}/contexto`, senal),
  paso: (id: string, n: number) => pedir<DetallePaso>(`/api/agentes/${encodeURIComponent(id)}/pasos/${n}`),
  doc: (proyecto: string, ruta: string) =>
    pedir<{ ruta: string; texto: string }>(`/api/doc?proyecto=${encodeURIComponent(proyecto)}&ruta=${encodeURIComponent(ruta)}`),
  buscar: (f: { q: string; tipo: string; proyecto: string }, senal?: AbortSignal) =>
    pedir<Busqueda>(`/api/datapedia?${new URLSearchParams({ q: f.q, tipo: f.tipo, proyecto: f.proyecto, limite: "80" })}`, senal),
  documento: (id: number) => pedir<DocumentoDatapedia>(`/api/datapedia/${id}`),
  documentoPorRuta: (proyecto: string, ruta: string) =>
    pedir<DocumentoDatapedia>(`/api/datapedia/ruta?${new URLSearchParams({ proyecto, ruta })}`),
  alertas: (proyecto: string) => pedir<RespuestaAlertas>(`/api/alertas?proyecto=${encodeURIComponent(proyecto)}`),
  pasos: (f: { q: string; tipo: string; proyecto: string; horas: number }, senal?: AbortSignal) =>
    pedir<RespuestaPasos>(`/api/pasos?${new URLSearchParams({ q: f.q, tipo: f.tipo, proyecto: f.proyecto, horas: f.horas ? String(f.horas) : "" })}`, senal),
  consultas: () => pedir<Consulta[]>("/api/consultas"),
  proceso: (proyecto: string, periodo: Periodo, senal?: AbortSignal) =>
    pedir<InformeProceso>(`/api/proceso?${new URLSearchParams({ proyecto, periodo })}`, senal),
  estadisticas: (proyecto: string, periodo: PeriodoEstadisticas, senal?: AbortSignal) =>
    pedir<InformeEstadisticas>(`/api/estadisticas?${new URLSearchParams({ proyecto, periodo })}`, senal),
  avance: (proyecto: string, senal?: AbortSignal) => pedir<RespuestaAvance>(`/api/avance?${new URLSearchParams({ proyecto })}`, senal),
  sesion: (id: string, senal?: AbortSignal) => pedir<ResumenSesion>(`/api/agentes/${encodeURIComponent(id)}/sesion`, senal),
  portafolio: (senal?: AbortSignal) => pedir<ProyectoPortafolio[]>("/api/portafolio", senal),
  exportar: (id: string, o: OpcionesExportar, senal?: AbortSignal) =>
    pedir<{ nombre: string; archivo: string; markdown: string }>(
      `/api/agentes/${encodeURIComponent(id)}/exportar?${new URLSearchParams({
        encargo: o.encargo ? "1" : "0",
        resumen: o.resumen ? "1" : "0",
        archivos: o.archivos ? "1" : "0",
        pasos: o.pasos ? "1" : "0",
        diffs: o.diffs ? "1" : "0",
        tokens: o.tokens ? "1" : "0",
        salidas: o.salidas,
      })}`,
      senal,
    ),
  decisiones: (senal?: AbortSignal) => pedir<RespuestaDecisiones>("/api/decisiones", senal),
  avisos: (senal?: AbortSignal) => pedir<EstadoAvisos>("/api/avisos", senal),
  /** Enciende o apaga un tipo de aviso; lo que no va, no cambia. */
  configurarAvisos: (c: { decisiones?: boolean; alertas?: boolean }) => escribirPanel<EstadoAvisos>("/api/avisos", c),
  /** Manda una notificación de prueba y espera a que salga. */
  probarAviso: () => escribirPanel<EstadoAvisos>("/api/avisos/probar"),
  pregunta: (id: string, senal?: AbortSignal) => pedir<PreguntaCentro>(`/api/preguntas/${encodeURIComponent(id)}`, senal),
  responder: async (id: string, cuerpo: unknown): Promise<void> => {
    await escribirPanel(`/api/preguntas/${encodeURIComponent(id)}/responder`, cuerpo);
  },
  linea: (proyecto: string, dia: string, senal?: AbortSignal) => pedir<RespuestaLinea>(`/api/linea?${new URLSearchParams({ proyecto, dia })}`, senal),
};

export const esActivo = (e: EstadoAgente) => e === "Trabajando" || e === "En espera";
