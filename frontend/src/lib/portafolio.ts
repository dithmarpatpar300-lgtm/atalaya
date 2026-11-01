// La portada «Tus proyectos»: tipos de /api/portafolio (backend/internal/servidor/portafolio.go)
// y lo que se calcula para dibujar cada tarjeta. Todo sale de los archivos del proyecto; donde falta uno, se dice.

/** Uno de los siete archivos del contrato de trazabilidad. */
export interface ArchivoContrato {
  id: "agents" | "mcp" | "estado" | "avance" | "operacion" | "subagentes" | "registro" | string;
  /** Como lo nombra el kit («docs/trazabilidad/F<n>/AVANCE.md»). */
  nombre: string;
  /** Relativa a la raíz: la que se encontró o, si falta, la que se espera. */
  ruta: string;
  esta: boolean;
}

/** El avance de la fase en curso, de su AVANCE.md (internal/avance). */
export interface AvancePortafolio {
  fase: string;
  ruta: string;
  ahora: string;
  calculado: number;
  horas: number;
  horas_hechas: number;
  /** Lo avanzado de los paquetes en curso (horas × %). */
  horas_en_curso: number;
  /** Σ horas × % de todos los paquetes: la nota «43,9 de 53 h». */
  horas_avanzadas: number;
}

export interface ProyectoPortafolio {
  nombre: string;
  sigla: string;
  ruta: string;
  /** Nivel del contrato que declara proyectos.json: 0 (sin contrato) a 3. */
  nivel: number;
  /** La carpeta está en el disco. */
  existe: boolean;
  fase?: string;
  contrato: ArchivoContrato[];
  avance: AvancePortafolio | null;
  /** Hoy: subagentes y llamadas a Codex sin faltas ni medias, de cuántos. */
  salud: { en_regla: number; total: number };
  vivos: number;
  /** Consultas a Atalaya por MCP desde la medianoche. */
  consultas: number;
  /** Las preguntas de sus sesiones que te esperan. */
  decisiones: number;
}

export type FiltroPortafolio = "todos" | "vivos" | "falta";

/** Coma decimal y siempre un decimal: «82,8». */
export const uno = (n: number) => (Number.isFinite(n) ? n : 0).toFixed(1).replace(".", ",");

/** Horas sin decimales si son enteras («53»), si no con uno («43,9»). */
export const horas = (n: number) => {
  const r = Math.round((Number.isFinite(n) ? n : 0) * 10) / 10;
  return Number.isInteger(r) ? String(r) : uno(r);
};

export const archivosQueTiene = (p: Pick<ProyectoPortafolio, "contrato">) => p.contrato.filter((c) => c.esta).length;

export const leFaltaAlgo = (p: Pick<ProyectoPortafolio, "contrato" | "existe">) => !p.existe || p.contrato.some((c) => !c.esta);

export function filtrar<T extends Pick<ProyectoPortafolio, "contrato" | "existe" | "vivos">>(ps: T[], f: FiltroPortafolio): T[] {
  if (f === "vivos") return ps.filter((p) => p.vivos > 0);
  if (f === "falta") return ps.filter(leFaltaAlgo);
  return ps;
}

export function cuentas(ps: Pick<ProyectoPortafolio, "contrato" | "existe" | "vivos">[]): Record<FiltroPortafolio, number> {
  return { todos: ps.length, vivos: filtrar(ps, "vivos").length, falta: filtrar(ps, "falta").length };
}

/** El chip de la tarjeta: «nivel 3» o «sin contrato». */
export const chipNivel = (nivel: number) => (nivel > 0 ? `nivel ${nivel}` : "sin contrato");

/** La segunda línea de la tarjeta: la fase y qué pasa ahora, o lo que falta para saberlo. */
export function lineaFase(p: Pick<ProyectoPortafolio, "fase" | "avance" | "existe">): string {
  if (!p.existe) return "La carpeta no está en el disco";
  if (p.avance) return p.avance.ahora ? `${p.avance.fase} · ${p.avance.ahora}` : p.avance.fase;
  if (p.fase) return `${p.fase} · sin AVANCE.md`;
  return "Sin fases en docs/trazabilidad";
}

/** La cifra grande y su nota: «82,8 %» y «43,9 de 53 h»; sin AVANCE.md, una raya y por qué. */
export function cifraAvance(a: AvancePortafolio | null): { cifra: string; nota: string } {
  if (!a) return { cifra: "—", nota: "sin AVANCE.md" };
  if (a.horas <= 0) return { cifra: "—", nota: "la tabla no dice horas" };
  return { cifra: `${uno(a.calculado)} %`, nota: `${horas(a.horas_avanzadas)} de ${horas(a.horas)} h` };
}

export interface TramoHilo {
  tipo: "hecho" | "curso" | "falta" | "vacio";
  /** Peso en el ancho (horas); los tramos de peso 0 no se dibujan. */
  peso: number;
}

/** Los colores del hilo. Lo que falta y el hilo vacío usan los grises del tema. */
export const COLOR_TRAMO: Record<TramoHilo["tipo"], string> = {
  hecho: "#3f9f6a",
  curso: "#c9a227",
  falta: "var(--borde-fuerte)",
  vacio: "var(--borde)",
};

/**
 * Los tramos del hilo de la tarjeta: lo hecho, lo avanzado de lo que está en curso y lo que falta, cada uno tan ancho
 * como sus horas. Sin avance (o sin horas), un solo tramo gris tenue.
 */
export function tramosHilo(a: AvancePortafolio | null): TramoHilo[] {
  if (!a || a.horas <= 0) return [{ tipo: "vacio", peso: 1 }];
  const hecho = Math.max(0, Math.min(a.horas, a.horas_hechas));
  const curso = Math.max(0, Math.min(a.horas - hecho, a.horas_en_curso));
  const falta = Math.max(0, a.horas - hecho - curso);
  const ts: TramoHilo[] = [
    { tipo: "hecho", peso: hecho },
    { tipo: "curso", peso: curso },
    { tipo: "falta", peso: falta },
  ];
  return ts.filter((t) => t.peso > 0.0001);
}

/** El hilo dicho en texto, para el lector de pantalla. */
export function hiloEnTexto(a: AvancePortafolio | null): string {
  if (!a || a.horas <= 0) return "Sin avance que dibujar";
  const ts = tramosHilo(a);
  const peso = (t: TramoHilo["tipo"]) => ts.find((x) => x.tipo === t)?.peso ?? 0;
  return `${horas(peso("hecho"))} h hechas, ${horas(peso("curso"))} h avanzadas en curso y ${horas(peso("falta"))} h por hacer, de ${horas(a.horas)} h`;
}

/** La salud de hoy: «2 de 3» en regla; una raya si hoy no trabajó ningún subagente ni Codex. */
export function saludHoy(s: ProyectoPortafolio["salud"]): { cifra: string; titulo: string } {
  if (s.total === 0) return { cifra: "—", titulo: "Hoy no trabajó ningún subagente ni llamada a Codex" };
  return { cifra: `${s.en_regla} de ${s.total}`, titulo: `${s.en_regla} de ${s.total} agentes de hoy van en regla` };
}

/** El nombre corto de cada archivo para la frase de lo que falta. */
const CORTO: Record<string, string> = {
  agents: "AGENTS.md",
  mcp: ".mcp.json",
  estado: "el estado para retomar",
  avance: "AVANCE.md",
  operacion: "docs/operacion/AGENTES.md",
  subagentes: ".claude/agents/",
  registro: "el registro en proyectos.json",
};

function enumerar(xs: string[]): string {
  if (xs.length <= 1) return xs.join("");
  return `${xs.slice(0, -1).join(", ")} y ${xs[xs.length - 1]}`;
}

/** La frase del panel: qué le falta al proyecto, dicho de los archivos. */
export function fraseFalta(p: Pick<ProyectoPortafolio, "contrato" | "existe">): string {
  if (!p.existe) return "La carpeta del proyecto no está en el disco: revisa su ruta en proyectos.json.";
  const faltan = p.contrato.filter((c) => !c.esta);
  if (faltan.length === 0) return "Tiene todo el contrato de trazabilidad.";
  if (faltan.length === p.contrato.length) return `No tiene contrato: le faltan los ${faltan.length} archivos.`;
  const nombres = faltan.map((c) => CORTO[c.id] ?? c.nombre);
  return `${faltan.length === 1 ? "Le falta" : "Le faltan"} ${enumerar(nombres)}.`;
}

/** El proyecto que muestra el panel: el elegido si sigue a la vista, si no el primero. */
export function proyectoElegido<T extends { nombre: string }>(visibles: T[], elegido: string | null): T | null {
  return visibles.find((p) => p.nombre === elegido) ?? visibles[0] ?? null;
}
