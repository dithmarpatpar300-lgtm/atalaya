"use client";

import { type KeyboardEvent, useCallback, useRef, useState } from "react";
import { IconoCheck, IconoLuna, IconoPantalla, IconoSol } from "@/components/ui/iconos";
import { fijarPreferencia, type Preferencia, usePreferenciaTema, useTema } from "@/lib/tema";
import { useCerrarFuera } from "@/lib/use-fuera";
import { t } from "@/lib/i18n";

const OPCIONES: { valor: Preferencia; nombre: string; Icono: typeof IconoSol }[] = [
  { valor: "sistema", nombre: "Sistema", Icono: IconoPantalla },
  { valor: "claro", nombre: "Claro", Icono: IconoSol },
  { valor: "oscuro", nombre: "Oscuro", Icono: IconoLuna },
];

/**
 * El tema en un solo botón redondo, que abre un menú con las
 * tres opciones de su selector (Sistema, Claro, Oscuro) en el mismo orden. El botón muestra el icono de lo elegido;
 * «Sistema» dice cuál se está viendo. Es el patrón «menu button» de la WAI-ARIA: las flechas, Inicio y Fin mueven el
 * foco entre las opciones y Escape cierra devolviendo el foco al botón.
 */
export function MenuTema() {
  const preferencia = usePreferenciaTema();
  const tema = useTema();
  const [abierto, setAbierto] = useState(false);
  const [animar, setAnimar] = useState(false);
  const boton = useRef<HTMLButtonElement>(null);
  const menu = useRef<HTMLDivElement>(null);
  const opciones = useRef<(HTMLButtonElement | null)[]>([]);
  const cerrar = useCallback(() => setAbierto(false), []);
  useCerrarFuera(abierto, menu, boton, cerrar);

  const actual = OPCIONES.find((o) => o.valor === preferencia) ?? OPCIONES[0];
  const nombre = preferencia === "sistema" ? t("Sistema (ahora {tema})", { tema: t(tema) }) : t(actual.nombre);

  function abrir() {
    setAbierto(true);
    // El foco entra en la opción elegida cuando el menú ya está pintado.
    requestAnimationFrame(() => opciones.current[OPCIONES.indexOf(actual)]?.focus());
  }

  function elegir(valor: Preferencia) {
    setAnimar(true);
    fijarPreferencia(valor);
    setAbierto(false);
    boton.current?.focus();
  }

  function alTecla(e: KeyboardEvent<HTMLDivElement>) {
    const i = opciones.current.findIndex((b) => b === document.activeElement);
    let j: number;
    if (e.key === "ArrowDown") j = (i + 1) % OPCIONES.length;
    else if (e.key === "ArrowUp") j = (i - 1 + OPCIONES.length) % OPCIONES.length;
    else if (e.key === "Home") j = 0;
    else if (e.key === "End") j = OPCIONES.length - 1;
    else if (e.key === "Tab") {
      setAbierto(false);
      return;
    } else return;
    e.preventDefault();
    opciones.current[j]?.focus();
  }

  return (
    <div className="relative">
      <button
        ref={boton}
        type="button"
        onClick={() => (abierto ? setAbierto(false) : abrir())}
        onKeyDown={(e) => {
          if (e.key === "ArrowDown" && !abierto) {
            e.preventDefault();
            abrir();
          }
        }}
        aria-haspopup="menu"
        aria-expanded={abierto}
        aria-label={t("Tema: {nombre}", { nombre })}
        title={t("Tema: {nombre}", { nombre })}
        className={`grid size-[34px] place-items-center rounded-full border transition-colors duration-150 ${
          abierto ? "border-acento bg-elevado text-texto" : "border-borde text-suave hover:border-borde-fuerte hover:bg-elevado hover:text-texto"
        }`}
      >
        <actual.Icono className={`size-[15px] ${animar ? "animate-icono-tema" : ""}`} />
      </button>

      {abierto && (
        <div ref={menu} role="menu" aria-label={t("Tema")} onKeyDown={alTecla} className="capa aparecer-menu absolute top-[42px] right-0 z-40 w-56 rounded-panel p-1.5">
          {OPCIONES.map(({ valor, nombre: n, Icono }, i) => {
            const elegida = valor === preferencia;
            return (
              <button
                key={valor}
                ref={(b) => {
                  opciones.current[i] = b;
                }}
                type="button"
                role="menuitemradio"
                aria-checked={elegida}
                onClick={() => elegir(valor)}
                className={`flex min-h-10 w-full items-center gap-2.5 rounded-control px-2.5 text-left hover:bg-elevado focus-visible:bg-elevado focus-visible:outline-none ${
                  elegida ? "bg-elevado" : ""
                }`}
              >
                <Icono className="size-4 shrink-0 text-suave" />
                <span className="flex flex-1 flex-col">
                  <span className={elegida ? "font-semibold" : ""}>{t(n)}</span>
                  {valor === "sistema" && <span className="text-ayuda text-tenue">{t("Ahora: {tema}", { tema: t(tema) })}</span>}
                </span>
                <span aria-hidden="true" className="grid size-4 place-items-center text-acento">
                  {elegida && <IconoCheck className="size-4" />}
                </span>
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}
