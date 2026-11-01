// La vista de la sesión principal: tipos de /api/agentes/{id}/sesion y lo que se calcula
// para mostrarla. Una sesión no tiene encargo ni bitácora: tiene pedidos de la persona y lo que siguió a cada uno.

import type { FaseAvance } from "./avance.ts";
import type { Decision } from "./decisiones.ts";

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
}

export interface ResumenSesion {
  pedidos: PedidoSesion[];
  documentos: { ruta: string; ediciones: number; ultima: string }[];
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

const plural = (n: number, uno: string, varios: string) => `${n} ${n === 1 ? uno : varios}`;

/** Lo que siguió a un pedido, en una línea: «lanzó 1 agente, 6 consultas a Atalaya, leyó la cuota». */
export function loQueSiguio(p: PedidoSesion): string {
  const partes: string[] = [];
  if (p.lanzados.length) partes.push(`lanzó ${plural(p.lanzados.length, "agente", "agentes")}`);
  if (p.consultas) partes.push(plural(p.consultas, "consulta a Atalaya", "consultas a Atalaya"));
  if (p.cuota) partes.push("leyó la cuota");
  if (p.commits) partes.push(p.commits === 1 ? "comiteó" : `comiteó ${p.commits} veces`);
  if (!partes.length) partes.push(plural(p.pasos, "paso", "pasos"));
  return partes.join(", ");
}

/** El efecto corto de la columna derecha de la fila. */
export function efecto(p: PedidoSesion): string {
  if (p.lanzados.length) return plural(p.lanzados.length, "agente lanzado", "agentes lanzados");
  if (p.commits) return "commit";
  if (p.consultas) return plural(p.consultas, "consulta", "consultas");
  return plural(p.pasos, "paso", "pasos");
}

/** El pedido que se muestra al llegar: el elegido si existe, si no el último. */
export function pedidoInicial(ps: PedidoSesion[], elegido: number | null): PedidoSesion | null {
  return ps.find((p) => p.n === elegido) ?? ps[ps.length - 1] ?? null;
}
