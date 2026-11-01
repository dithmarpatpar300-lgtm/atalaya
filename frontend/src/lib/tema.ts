// El tema: la preferencia se guarda, «sistema» es
// no haber elegido, y el cambio se aplica en un solo cuadro con un fundido de la página entera.
import { useSyncExternalStore } from "react";
import { CLAVE_TEMA, CONSULTA_OSCURO } from "./tema-arranque";

/** Lo que la persona eligió. "sistema" no se guarda: es no haber elegido. */
export type Preferencia = "sistema" | "claro" | "oscuro";
/** Lo que se ve. */
export type Tema = "claro" | "oscuro";

let preferencia: Preferencia | undefined;
const escuchas = new Set<() => void>();

// Una sola consulta para todo el módulo; se crea al primer uso porque al prerenderizar no hay `window`.
let consulta: MediaQueryList | undefined;
const sistema = () => (consulta ??= matchMedia(CONSULTA_OSCURO));

function leerGuardada(): Preferencia {
  try {
    const guardada = localStorage.getItem(CLAVE_TEMA);
    return guardada === "claro" || guardada === "oscuro" ? guardada : "sistema";
  } catch {
    return "sistema";
  }
}

function leerPreferencia(): Preferencia {
  preferencia ??= leerGuardada();
  return preferencia;
}

function leerTema(): Tema {
  const elegida = leerPreferencia();
  if (elegida !== "sistema") return elegida;
  return sistema().matches ? "oscuro" : "claro";
}

const menosMovimiento = () => matchMedia("(prefers-reduced-motion: reduce)").matches;

function aplicar(animado: boolean) {
  const raiz = document.documentElement;
  const tema = leerTema();
  if (raiz.dataset.tema === tema) return;

  const cambiar = () => {
    raiz.classList.add("tema-cambio");
    raiz.dataset.tema = tema;
    // Calcula los colores nuevos con las transiciones apagadas, antes de volver a encenderlas.
    void raiz.offsetHeight;
    raiz.classList.remove("tema-cambio");
  };

  if (animado && "startViewTransition" in document && !menosMovimiento()) {
    const fundido = document.startViewTransition(cambiar);
    fundido.ready.catch(() => {});
    fundido.finished.catch(() => {});
  } else {
    cambiar();
  }
}

function avisar() {
  for (const escucha of escuchas) escucha();
}

function alCambiarSistema() {
  aplicar(false);
  avisar();
}

function alCambiarAlmacenamiento(evento: StorageEvent) {
  if (evento.key !== null && evento.key !== CLAVE_TEMA) return;
  preferencia = leerGuardada();
  aplicar(false);
  avisar();
}

function suscribir(escucha: () => void): () => void {
  if (escuchas.size === 0) {
    sistema().addEventListener("change", alCambiarSistema);
    window.addEventListener("storage", alCambiarAlmacenamiento);
  }
  escuchas.add(escucha);
  return () => {
    escuchas.delete(escucha);
    if (escuchas.size === 0) {
      sistema().removeEventListener("change", alCambiarSistema);
      window.removeEventListener("storage", alCambiarAlmacenamiento);
    }
  };
}

export function fijarPreferencia(nueva: Preferencia): void {
  preferencia = nueva;
  try {
    if (nueva === "sistema") localStorage.removeItem(CLAVE_TEMA);
    else localStorage.setItem(CLAVE_TEMA, nueva);
  } catch {
    // Sin almacenamiento la elección vale hasta que se cierre la pestaña.
  }
  aplicar(true);
  avisar();
}

export function usePreferenciaTema(): Preferencia {
  return useSyncExternalStore(suscribir, leerPreferencia, () => "sistema");
}

export function useTema(): Tema {
  return useSyncExternalStore(suscribir, leerTema, () => "claro");
}
