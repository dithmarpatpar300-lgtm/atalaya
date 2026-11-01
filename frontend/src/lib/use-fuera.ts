"use client";

import { type RefObject, useEffect } from "react";

/**
 * Cierra lo que flota (un menú, un selector) con un clic fuera de `caja` o con Escape. Solo escucha mientras está
 * abierto. Escape devuelve el foco a `boton`, como un menú nativo.
 */
export function useCerrarFuera(
  abierto: boolean,
  caja: RefObject<HTMLElement | null>,
  boton: RefObject<HTMLElement | null>,
  cerrar: () => void,
) {
  useEffect(() => {
    if (!abierto) return;
    function alPulsar(e: PointerEvent) {
      const t = e.target as Node;
      if (caja.current?.contains(t) || boton.current?.contains(t)) return;
      cerrar();
    }
    function alTecla(e: KeyboardEvent) {
      if (e.key !== "Escape") return;
      e.preventDefault();
      cerrar();
      boton.current?.focus();
    }
    document.addEventListener("pointerdown", alPulsar);
    document.addEventListener("keydown", alTecla);
    return () => {
      document.removeEventListener("pointerdown", alPulsar);
      document.removeEventListener("keydown", alTecla);
    };
  }, [abierto, caja, boton, cerrar]);
}
