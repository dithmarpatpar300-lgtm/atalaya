// El color de cada proveedor, el mismo en todo el panel: las series validadas de los gráficos (globals.css,
// --serie-claude y --serie-codex; ΔE 8,2 entre ellas para un protán). El acento y `opcional` quedan para la interfaz.

export type Proveedor = "Claude" | "Codex";

/** Para SVG (stroke, fill) y estilos en línea. */
export const COLOR_PROVEEDOR: Record<Proveedor, string> = { Claude: "var(--serie-claude)", Codex: "var(--serie-codex)" };

/** Para un punto o una barra con clase de Tailwind. */
export const FONDO_PROVEEDOR: Record<Proveedor, string> = { Claude: "bg-serie-claude", Codex: "bg-serie-codex" };
