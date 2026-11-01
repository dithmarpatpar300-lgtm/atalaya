// La lista de agentes como árbol: cada sesión con lo que nació de ella colgando debajo, sus
// subagentes y sus llamadas a Codex (las dos traen `padre`). Sin React: se prueba con `node --test` (arbol.test.mts).
import type { Agente } from "./api.ts";
import { t } from "./i18n.ts";

export interface FilaArbol {
  agente: Agente;
  nivel: 0 | 1;
  /** Índice de la fila del padre, o -1. */
  padre: number;
  /** Tiene sesión, pero no está en la lista (otro filtro): va suelto, con su clase de agente nombrada aparte. */
  huerfano: boolean;
  /** Una llamada a Codex cuya sesión no está en los transcritos (se borró o se lanzó a mano): va como raíz. Una sesión
   * interactiva de Codex no tiene sesión que la lance: no cuenta. */
  sinSesion: boolean;
}

/**
 * Ordena para el árbol: conserva el orden de la lista (activos primero, luego lo más reciente), pero cada hijo
 * (subagente o llamada a Codex) va justo debajo de su sesión, con sus hermanos ordenados por su último paso, del más
 * reciente al más viejo.
 */
export function ordenarArbol(lista: Agente[]): FilaArbol[] {
  const ids = new Set(lista.map((a) => a.id));
  const hijos = new Map<string, Agente[]>();
  for (const a of lista) {
    if (a.padre && ids.has(a.padre)) {
      const h = hijos.get(a.padre) ?? [];
      h.push(a);
      hijos.set(a.padre, h);
    }
  }
  const filas: FilaArbol[] = [];
  for (const a of lista) {
    if (a.padre && ids.has(a.padre)) continue; // va con su padre
    const i = filas.length;
    filas.push({ agente: a, nivel: 0, padre: -1, huerfano: !!a.padre, sinSesion: a.tipo === "codex" && !a.interactiva && !a.padre });
    const suyos = (hijos.get(a.id) ?? []).sort((x, y) => Date.parse(y.ultimo) - Date.parse(x.ultimo));
    for (const h of suyos) filas.push({ agente: h, nivel: 1, padre: i, huerfano: false, sinSesion: false });
  }
  return filas;
}

export interface Titulos {
  /** Lo que nombra al agente: el proyecto en una sesión; el rol en un subagente o una llamada a Codex. */
  principal: string;
  /** La clase del encargo, que va aparte y con menos peso. */
  clase: string;
  /** Con menos peso: el nombre que se le dio, si tiene uno propio; si no, su tarea (en una sesión, el primer pedido). */
  alterno: string;
  /** El texto entero para `title`: el nombre y la tarea si los hay los dos. */
  completo: string;
}

const comparable = (s: string) =>
  s
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .replace(/[\s_-]+/g, " ")
    .trim();

/** Cómo se nombra un agente en la lista, la cabecera del detalle, el linaje y la paleta. */
export function titulos(a: Agente): Titulos {
  // El nombre lo resuelve el servidor, uno solo para todo el panel («medidor-w6b», «Auditor E2», «Codex · …»); la
  // clase va aparte, así que no se repite al final.
  const base = a.nombre || (a.tipo === "sesion" ? a.proyecto || a.rol : a.rol);
  const principal = a.clase && base.endsWith(` ${a.clase}`) ? base.slice(0, -(a.clase.length + 1)) : base;
  // Un nombre igual al proyecto («Tienda Demo» en Tienda Demo) no dice nada nuevo: va la tarea; y si el nombre ya es
  // la tarea (un general-purpose sin documentos), su rol.
  const nombre = a.titulo && comparable(a.titulo) !== comparable(principal) ? a.titulo : "";
  const tarea = a.tarea && a.tarea !== principal ? a.tarea : a.tipo === "sesion" ? "" : a.rol;
  const alterno = nombre || tarea || t("Sin encargo");
  return { principal, clase: a.clase, alterno, completo: nombre && a.tarea ? `${nombre} — ${a.tarea}` : alterno };
}

/** El principal con su clase, en una sola cadena («Codificador I2», «Tienda Demo»). */
export function nombreAgente(a: Agente): string {
  const ti = titulos(a);
  return `${ti.principal}${ti.clase ? ` ${ti.clase}` : ""}`;
}

export const ALTO_FILA = 62;
export const X_NIVEL = [18, 38] as const;

/** Los trazos del árbol: un codo curvo de cada hijo a su padre, como el hilo de respuestas. */
export function trazosArbol(filas: FilaArbol[]): { d: string; codex: boolean }[] {
  const cy = (i: number) => i * ALTO_FILA + ALTO_FILA / 2;
  return filas.flatMap((f, i) => {
    if (f.padre < 0) return [];
    const xa = X_NIVEL[0];
    return [{ d: `M ${xa} ${cy(f.padre) + 7} V ${cy(i) - 10} Q ${xa} ${cy(i)} ${xa + 10} ${cy(i)} H ${X_NIVEL[1] - 6}`, codex: f.agente.proveedor === "Codex" }];
  });
}
