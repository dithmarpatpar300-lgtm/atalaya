// La lógica pura del encargo detallado: el linaje, los archivos que tocó y dónde cae
// cada hito de la bitácora sobre el tiempo. Sin React: se prueba con `node --test` (encargo.test.mts).
import type { Agente, Paso } from "./api.ts";
import { nombreAgente, titulos } from "./arbol.ts";
import { t } from "./i18n.ts";

export interface NodoLinaje {
  id: string;
  nombre: string;
  nota: string;
  nivel: number;
  /** Índice del padre en la lista, o -1. */
  padre: number;
  yo: boolean;
  proveedor: "Claude" | "Codex";
  activo: boolean;
}

const vivo = (a: Agente) => a.estado === "Trabajando" || a.estado === "En espera";
const notaHijo = (h: Agente) => (h.tarea ? recortar(h.tarea, 40) : h.proveedor === "Codex" ? t("llamada a Codex") : t("subagente"));

/**
 * El linaje de un agente: quién lo lanzó, sus hermanos (lo demás que nació de la misma sesión, subagentes y llamadas
 * a Codex, con lo que coincidió en el tiempo marcado «en paralelo») y lo que él lanzó. Una sesión muestra a sus
 * hijos. Una llamada a Codex sin sesión enlazada lo dice en su nota.
 */
export function linaje(a: Agente, todos: Agente[]): NodoLinaje[] {
  const padre = a.padre ? todos.find((x) => x.id === a.padre) : undefined;
  const coincide = (x: Agente) => Date.parse(x.inicio) < Date.parse(a.ultimo) && Date.parse(x.ultimo) > Date.parse(a.inicio);
  const hijos = todos.filter((x) => x.padre === a.id);
  const nodo = (x: Agente, nivel: number, padreIdx: number, nota: string): NodoLinaje => ({
    id: x.id, nombre: nombreAgente(x), nota, nivel, padre: padreIdx, yo: x.id === a.id, proveedor: x.proveedor, activo: vivo(x),
  });
  const out: NodoLinaje[] = [];
  if (a.padre) {
    // La sesión puede no estar en la lista (se cortó o es de otro filtro): se nombra por su id y se puede abrir igual.
    out.push(
      padre
        ? nodo(padre, 0, -1, recortar(titulos(padre).alterno, 40))
        : { id: a.padre, nombre: t("Sesión {id}", { id: a.padre.slice(0, 8) }), nota: t("fuera de la lista"), nivel: 0, padre: -1, yo: false, proveedor: "Claude", activo: false },
    );
    out.push(nodo(a, 1, 0, t("este")));
    const hermanos = todos
      .filter((x) => x.padre === a.padre && x.id !== a.id)
      .sort((x, y) => Number(coincide(y)) - Number(coincide(x)) || Date.parse(x.inicio) - Date.parse(y.inicio))
      .slice(0, 5);
    for (const h of hermanos) out.push(nodo(h, 1, 0, coincide(h) ? t("en paralelo") : Date.parse(h.inicio) < Date.parse(a.inicio) ? t("antes") : t("después")));
    for (const h of hijos.slice(0, 5)) out.push(nodo(h, 2, 1, notaHijo(h)));
  } else {
    out.push(nodo(a, 0, -1, a.tipo === "codex" ? t("sin sesión enlazada") : t("este")));
    for (const h of hijos.slice(0, 8)) out.push(nodo(h, 1, 0, notaHijo(h)));
  }
  return out;
}

function recortar(s: string, n: number): string {
  return s.length > n ? s.slice(0, n - 1) + "…" : s;
}

export interface ArchivoTocado {
  hilo: string;
  nombre: string;
  lee: number;
  busca: number;
  edita: number;
  /** null si el encargo no nombra archivos (no hay frontera con qué comparar). */
  nombrado: boolean | null;
}

/** Los archivos que tocó, con cuántas veces los leyó, buscó o editó, y si el encargo los nombraba. */
export function archivosTocados(pasos: Paso[], nombrados: string[] = []): ArchivoTocado[] {
  const por = new Map<string, ArchivoTocado>();
  const enEncargo = new Set(nombrados.map((n) => n.toLowerCase()));
  for (const p of pasos) {
    if (!p.hilo) continue;
    let f = por.get(p.hilo);
    if (!f) {
      const nombre = p.hilo.split("/").pop() ?? p.hilo;
      f = { hilo: p.hilo, nombre, lee: 0, busca: 0, edita: 0, nombrado: nombrados.length ? enEncargo.has(nombre.toLowerCase()) : null };
      por.set(p.hilo, f);
    }
    if (p.tipo === "Lee") f.lee++;
    else if (p.tipo === "Busca") f.busca++;
    else if (p.tipo === "Edita" || p.tipo === "Escribe") f.edita++;
  }
  // Primero lo que editó, luego lo más leído.
  return [...por.values()].sort((a, b) => b.edita - a.edita || b.lee + b.busca - (a.lee + a.busca));
}

/**
 * Dónde cae un hito («21:37», hora local) sobre la barra de tiempo, de 0 a 1, contando desde el inicio del agente y
 * sobre `minutos` (el presupuesto, o lo que duró). Un hito que pasa de medianoche se cuenta al día siguiente.
 */
export function posicionHito(hito: string, inicioIso: string, minutos: number): number | null {
  const m = /^(\d{1,2}):(\d{2})$/.exec(hito);
  if (!m || minutos <= 0) return null;
  const inicio = new Date(inicioIso);
  const desde = inicio.getHours() * 60 + inicio.getMinutes() + inicio.getSeconds() / 60;
  let delta = Number(m[1]) * 60 + Number(m[2]) - desde;
  if (delta < -60) delta += 24 * 60;
  return Math.max(0, Math.min(1, delta / minutos));
}
