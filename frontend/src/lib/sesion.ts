// La vista de la sesión principal: tipos de /api/agentes/{id}/sesion y lo que se calcula
// para mostrarla. Una sesión no tiene encargo ni bitácora: tiene pedidos de la persona y lo que siguió a cada uno.

import type { FaseAvance } from "./avance.ts";
import type { Decision } from "./decisiones.ts";
import { t } from "./i18n.ts";

export interface PedidoSesion {
  n: number;
  hora: string;
  tipo: "pedido" | "permiso";
  texto: string;
  caracteres: string;
  lanzados: string[];
  consultas: number;
  cuota: number;
  commits: number;
  pasos: number;
  etapas: string[];
  respuesta?: string;
}

export interface MiembroSesion {
  id: string;
  nombre: string;
  tarea: string;
  proveedor: "Claude" | "Codex";
  clase?: string;
  modelo: string;
  effort?: string;
  estado: string;
  inicio: string;
  /** Otra sesión principal que esta lanzó (start_session): con su propio equipo. */
  sesion?: boolean;
}

export interface ResumenSesion {
  pedidos: PedidoSesion[];
  /** `id`: la ruta completa normalizada, única (la clave de la lista); `ruta`: la corta, solo para mostrar. */
  documentos: { id: string; ruta: string; ediciones: number; ultima: string }[];
  commits: { hora: string; paso: number; orden: string }[];
  consultas: number;
  lecturas_cuota: number;
  ultima_cuota?: string;
  mensajes_sistema: number;
  equipo: MiembroSesion[];
  avance: FaseAvance | null;
  /** Lo que la sesión te preguntó (botones, preguntas de opciones o por Atalaya), de la más nueva a la más vieja. */
  decisiones?: Decision[];
}

/** Lo que siguió a un pedido, en una línea: «lanzó 1 agente, 6 consultas a Atalaya, leyó la cuota». */
export function loQueSiguio(p: PedidoSesion): string {
  const partes: string[] = [];
  const n = p.lanzados.length;
  if (n) partes.push(n === 1 ? t("lanzó 1 agente") : t("lanzó {n} agentes", { n }));
  if (p.consultas) partes.push(p.consultas === 1 ? t("1 consulta a Atalaya") : t("{n} consultas a Atalaya", { n: p.consultas }));
  if (p.cuota) partes.push(t("leyó la cuota"));
  if (p.commits) partes.push(p.commits === 1 ? t("comiteó") : t("comiteó {n} veces", { n: p.commits }));
  if (!partes.length) partes.push(p.pasos === 1 ? t("1 paso") : t("{n} pasos", { n: p.pasos }));
  return partes.join(", ");
}

/** El efecto corto de la columna derecha de la fila. */
export function efecto(p: PedidoSesion): string {
  const n = p.lanzados.length;
  if (n) return n === 1 ? t("1 agente lanzado") : t("{n} agentes lanzados", { n });
  if (p.commits) return t("commit");
  if (p.consultas) return p.consultas === 1 ? t("1 consulta") : t("{n} consultas", { n: p.consultas });
  return p.pasos === 1 ? t("1 paso") : t("{n} pasos", { n: p.pasos });
}

/** El pedido que se muestra al llegar: el elegido si existe, si no el último. */
export function pedidoInicial(ps: PedidoSesion[], elegido: number | null): PedidoSesion | null {
  return ps.find((p) => p.n === elegido) ?? ps[ps.length - 1] ?? null;
}
