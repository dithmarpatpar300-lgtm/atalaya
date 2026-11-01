"use client";

import { cambiarIdioma, idioma } from "@/lib/i18n";

/**
 * El idioma del panel en un botón redondo, junto al tema: muestra el de ahora («ES», «EN») y cambia al otro. La
 * etiqueta va en los dos idiomas, para que la encuentre quien no lee el que está puesto.
 */
export function BotonIdioma() {
  const actual = idioma();
  const etiqueta = actual === "es" ? "Idioma: español. Switch to English" : "Language: English. Cambiar a español";
  return (
    <button
      type="button"
      onClick={() => cambiarIdioma(actual === "es" ? "en" : "es")}
      aria-label={etiqueta}
      title={etiqueta}
      className="dato grid size-[34px] shrink-0 place-items-center rounded-full border border-borde text-[11px] font-semibold text-suave transition-colors duration-150 hover:border-borde-fuerte hover:bg-elevado hover:text-texto"
    >
      {actual.toUpperCase()}
    </button>
  );
}
