// El idioma del panel: español (el de origen) o inglés. El texto en español es la clave: t("Tus proyectos") da
// «Your projects» en inglés y, si una frase no tiene traducción, se muestra en español. Los textos que arma el
// servidor (alertas, salud, notas de estadísticas) se traducen con patrones (tServidor). El idioma se guarda en este
// navegador; sin elegir, sigue al del navegador. Cambiarlo recarga la página: así ningún texto queda a medias.

import { EN_AGENTES } from "./i18n/en-agentes.ts";
import { EN_CABECERA } from "./i18n/en-cabecera.ts";
import { EN_CONFIGURAR } from "./i18n/en-configurar.ts";
import { EN_VISTAS } from "./i18n/en-vistas.ts";
import { PATRONES_SERVIDOR } from "./i18n/servidor.ts";

export type Idioma = "es" | "en";

export const CLAVE_IDIOMA = "atalaya:idioma";

/** El idioma guardado o, sin elegir, el del navegador (español si empieza por «es»; inglés si no). */
export function idiomaInicial(): Idioma {
  if (typeof window === "undefined") return "es";
  try {
    const g = localStorage.getItem(CLAVE_IDIOMA);
    if (g === "es" || g === "en") return g;
  } catch {}
  return (navigator.language || "es").toLowerCase().startsWith("es") ? "es" : "en";
}

let actual: Idioma = idiomaInicial();

export const idioma = (): Idioma => actual;

/** Solo para las pruebas: fija el idioma sin tocar el navegador. */
export function fijarIdioma(i: Idioma) {
  actual = i;
}

export function cambiarIdioma(i: Idioma) {
  try {
    localStorage.setItem(CLAVE_IDIOMA, i);
  } catch {}
  window.location.reload();
}

const EN: Record<string, string> = { ...EN_CABECERA, ...EN_AGENTES, ...EN_VISTAS, ...EN_CONFIGURAR };

/**
 * El texto en el idioma del panel. `es` es la frase en español, con huecos {nombre} que llena `vars`:
 * t("hace {d}", { d: "6 min" }).
 */
export function t(es: string, vars?: Record<string, string | number>): string {
  let s = actual === "en" ? (EN[es] ?? es) : es;
  if (vars) for (const [k, v] of Object.entries(vars)) s = s.split(`{${k}}`).join(String(v));
  return s;
}

/** Un texto armado por el servidor, en español: en inglés, el primer patrón que encaja; si ninguno, tal cual. */
export function tServidor(texto: string): string {
  if (actual !== "en" || !texto) return texto;
  for (const [patron, en] of PATRONES_SERVIDOR) {
    const m = patron.exec(texto);
    if (!m) continue;
    // Un reemplazo puede ser una función: recibe tServidor (para traducir una parte que es otro texto del servidor) y
    // los grupos del patrón.
    return typeof en === "string" ? texto.replace(patron, en) : en(tServidor, ...m.slice(1));
  }
  return texto;
}

/** Una cifra con `dec` decimales: coma en español («4,1»), punto en inglés («4.1»). */
export function decimal(n: number, dec = 1): string {
  const s = (Number.isFinite(n) ? n : 0).toFixed(dec);
  return actual === "en" ? s : s.replace(".", ",");
}
