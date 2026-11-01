"use client";

/** Grupo de botones de igual ancho, el elegido en relieve. */
export function Segmentado<T extends string>({
  etiqueta,
  opciones,
  valor,
  alCambiar,
}: {
  etiqueta: string;
  opciones: { valor: T; nombre: string; cuenta?: number }[];
  valor: T;
  alCambiar: (v: T) => void;
}) {
  return (
    <div
      role="group"
      aria-label={etiqueta}
      className="grid h-(--alto-control-compacto) rounded-control bg-elevado p-0.5"
      style={{ gridTemplateColumns: `repeat(${opciones.length}, minmax(0, 1fr))` }}
    >
      {opciones.map((o) => {
        const on = o.valor === valor;
        return (
          <button
            key={o.valor}
            type="button"
            aria-pressed={on}
            onClick={() => alCambiar(o.valor)}
            className={`flex items-center justify-center gap-1.5 rounded-chip text-rotulo transition-colors duration-150 ${
              on ? "bg-superficie font-medium text-texto shadow-[0_1px_2px_rgb(0_0_0/0.08)]" : "text-suave hover:text-texto"
            }`}
          >
            {o.nombre}
            {o.cuenta !== undefined && <span className="dato text-ayuda text-tenue">{o.cuenta}</span>}
          </button>
        );
      })}
    </div>
  );
}
