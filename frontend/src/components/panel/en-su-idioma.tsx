"use client";

import { useSyncExternalStore } from "react";
import { Panel } from "./panel";

const nada = () => () => {};

/**
 * El panel solo se pinta en el navegador: la página estática se arma en español y el idioma elegido (lib/i18n.ts) se
 * sabe al abrirla. Pintarlo antes haría que React encontrara otro texto al hidratar.
 */
export function PanelEnSuIdioma() {
  const enElNavegador = useSyncExternalStore(
    nada,
    () => true,
    () => false,
  );
  return enElNavegador ? <Panel /> : null;
}
