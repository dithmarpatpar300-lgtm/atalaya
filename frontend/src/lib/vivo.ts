"use client";

import { useEffect, useRef, useState, useSyncExternalStore } from "react";

export interface EventoCambio {
  tipo: "cuotas" | "agentes" | "datapedia" | "alertas" | "consultas" | "proceso" | "decisiones" | "avisos";
  ids?: string[];
}

export type Conexion = "conectando" | "en vivo" | "reconectando";

/**
 * Una sola conexión SSE a /api/eventos para toda la página. El servidor avisa qué cambió y la página vuelve a pedir
 * solo eso. Si la conexión cae, EventSource reconecta solo (el servidor pide 3 s) y al volver se pide todo otra vez.
 */
export function useEventos(alCambio: (e: EventoCambio | { tipo: "reconectado" }) => void): Conexion {
  const [conexion, setConexion] = useState<Conexion>("conectando");
  const ref = useRef(alCambio);
  useEffect(() => {
    ref.current = alCambio;
  });
  useEffect(() => {
    const es = new EventSource("/api/eventos");
    // También en la primera conexión: lo que cambió entre la carga de la página y la conexión (por ejemplo, el panel
    // que terminaba de leer los transcritos al arrancar) no llegó por SSE y se vuelve a pedir.
    es.addEventListener("hola", () => {
      setConexion("en vivo");
      ref.current({ tipo: "reconectado" });
    });
    es.addEventListener("cambio", (m) => {
      try {
        ref.current(JSON.parse((m as MessageEvent<string>).data) as EventoCambio);
      } catch {}
    });
    es.onerror = () => {
      setConexion("reconectando");
    };
    return () => es.close();
  }, []);
  return conexion;
}

// Un reloj de un segundo, compartido, para las cuentas atrás.
let segundo = Math.floor(Date.now() / 1000);
const relojes = new Set<() => void>();
let intervalo: ReturnType<typeof setInterval> | undefined;

function suscribirReloj(f: () => void) {
  relojes.add(f);
  if (!intervalo) {
    intervalo = setInterval(() => {
      segundo = Math.floor(Date.now() / 1000);
      relojes.forEach((r) => r());
    }, 1000);
  }
  return () => {
    relojes.delete(f);
    if (relojes.size === 0 && intervalo) {
      clearInterval(intervalo);
      intervalo = undefined;
    }
  };
}

/** La hora actual en segundos unix, que avanza cada segundo. 0 al prerenderizar. */
export function useAhora(): number {
  return useSyncExternalStore(
    suscribirReloj,
    () => segundo,
    () => 0,
  );
}
