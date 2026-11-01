"use client";

import { useState } from "react";
import { IconoChevron, IconoRecargar } from "@/components/ui/iconos";
import type { EstadoCuotas, PanelCuota, PosturaProveedor, VentanaCuota } from "@/lib/api";
import { cuandoReinicia, duracion, haceCuanto, porcentaje, ritmo, tokens } from "@/lib/formato";
import { useAhora } from "@/lib/vivo";

const NOMBRE: Record<string, string> = { "Claude Code": "Claude", Codex: "Codex" };
const VENTANA: Record<string, string> = { "5 h": "5 horas", Semanal: "Semana" };

function etiquetaVentana(v: VentanaCuota): string {
  return VENTANA[v.label] ?? v.label;
}

function Postura({ postura }: { postura?: PosturaProveedor }) {
  if (!postura) return null;
  const normal = postura.stance === "normal";
  const texto = postura.stance.charAt(0).toUpperCase() + postura.stance.slice(1);
  return (
    <span
      title={postura.reasons.length ? postura.reasons.join("; ") : undefined}
      className={`rounded-chip px-2 py-0.5 text-ayuda font-medium ${normal ? "bg-ok-suave text-ok" : "bg-aviso-suave text-aviso"}`}
    >
      {texto}
    </span>
  );
}

function Ventana({ v, color, ahora, principal }: { v: VentanaCuota; color: string; ahora: number; principal: boolean }) {
  const usado = Math.min(100, Math.max(0, v.used_percent));
  const vencida = v.resets_at > 0 && v.resets_at <= ahora;
  const quedanTok = v.tokens_left_estimate ? ` (≈${tokens(v.tokens_left_estimate)} tokens)` : "";
  return (
    <div className="grid grid-cols-[minmax(6rem,8.75rem)_minmax(0,1fr)_4rem] items-center gap-x-4 gap-y-1">
      <span className="text-rotulo">{etiquetaVentana(v)}</span>
      <div
        className="h-1.5 rounded-full bg-elevado"
        role="meter"
        aria-label={`${etiquetaVentana(v)}: ${porcentaje(usado)} usado`}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={usado}
      >
        <div className={`h-1.5 rounded-full ${color}`} style={{ width: `${usado}%`, opacity: principal ? 1 : 0.6 }} />
      </div>
      <span className="dato text-right">{porcentaje(usado)}</span>
      {/* La cuenta atrás cambia cada minuto: en su propia línea fija, sin empujar la cifra de al lado (lo que cambia en vivo no mueve el diseño). */}
      <span className="col-span-2 flex flex-col text-ayuda text-suave">
        <span>
          Quedan {porcentaje(100 - usado)}
          {quedanTok}.
        </span>
        <span className="dato h-4 truncate">
          {vencida ? "Ya se reinició; falta una lectura nueva." : v.resets_at > 0 && `Reinicia ${cuandoReinicia(v.resets_at, ahora)}, en ${duracion(v.resets_at - ahora)}.`}
        </span>
      </span>
      <span className="dato text-right text-ayuda text-suave" title={v.pace ? "Ritmo de la ventana" : undefined}>
        {v.pace ? ritmo(v.pace.percent_per_hour) : "sin dato"}
      </span>
    </div>
  );
}

function Tarjeta({ p, postura, ahora }: { p: PanelCuota; postura?: PosturaProveedor; ahora: number }) {
  const codex = p.provider === "Codex";
  const color = codex ? "bg-opcional" : "bg-acento";
  return (
    <div className="flex min-w-0 flex-col gap-3.5 rounded-panel border border-borde p-4">
      <div className="flex flex-wrap items-center gap-2.5">
        <span aria-hidden="true" className={`size-2.5 rounded-[2px] ${color}`} />
        <span className="font-semibold">{NOMBRE[p.provider] ?? p.provider}</span>
        <span className="text-ayuda text-suave">{p.plan ?? ""}</span>
        <span className="flex-1" />
        <Postura postura={postura} />
      </div>
      {p.windows.map((v, i) => (
        <Ventana key={v.label} v={v} color={color} ahora={ahora} principal={i === 0} />
      ))}
      {p.notice && <p className="m-0 text-ayuda text-aviso">{p.notice}.</p>}
      {postura && (
        <p className="m-0 text-ayuda text-tenue">
          Margen {porcentaje(postura.headroom_percent)}; ritmo parejo hasta el reinicio {ritmo(postura.even_pace_percent_per_hour)}.
        </p>
      )}
    </div>
  );
}

function fuente(p: PanelCuota): string {
  const n = NOMBRE[p.provider] ?? p.provider;
  if (p.source.startsWith("statusLine")) return `${n} por la statusLine`;
  if (p.source.includes("app-server")) return `${n} por su app-server`;
  return `${n} por el ${p.source}`;
}

/** Una línea por proveedor: sus ventanas con barra fina y cifra, y su postura. */
function Franja({ p, postura, ahora }: { p: PanelCuota; postura?: PosturaProveedor; ahora: number }) {
  const codex = p.provider === "Codex";
  const color = codex ? "bg-opcional" : "bg-acento";
  // En un teléfono: el nombre y la postura en una línea, las ventanas en la siguiente.
  return (
    <div className="flex min-w-0 flex-[1_1_24rem] flex-wrap items-center gap-x-3 gap-y-1.5">
      <span aria-hidden="true" className={`size-2 shrink-0 rounded-[2px] ${color}`} />
      <span className="w-14 shrink-0 font-semibold">{NOMBRE[p.provider] ?? p.provider}</span>
      <span className="flex min-w-0 basis-full gap-3 max-sm:order-last sm:flex-1 sm:basis-0">
      {p.windows.map((v, i) => {
        const usado = Math.min(100, Math.max(0, v.used_percent));
        const vencida = v.resets_at > 0 && v.resets_at <= ahora;
        return (
          <span
            key={v.label}
            title={vencida ? "Ya se reinició; falta una lectura nueva" : v.resets_at > 0 && ahora ? `Reinicia ${cuandoReinicia(v.resets_at, ahora)}, en ${duracion(v.resets_at - ahora)}` : undefined}
            className="flex min-w-0 flex-[1_1_7rem] items-center gap-2"
          >
            <span className="shrink-0 text-ayuda whitespace-nowrap text-suave">{v.label}</span>
            <span
              role="meter"
              aria-label={`${NOMBRE[p.provider] ?? p.provider}, ${etiquetaVentana(v)}: ${porcentaje(usado)} usado`}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={usado}
              className="h-[5px] min-w-10 flex-1 rounded-full bg-elevado"
            >
              <span className={`block h-[5px] rounded-full ${color}`} style={{ width: `${usado}%`, opacity: i === 0 ? 1 : 0.6 }} />
            </span>
            <span className={`dato w-11 shrink-0 text-right text-ayuda ${vencida ? "text-tenue" : ""}`}>{porcentaje(usado)}</span>
          </span>
        );
      })}
      </span>
      <span className="max-sm:ml-auto">
        <Postura postura={postura} />
      </span>
    </div>
  );
}

/**
 * Las cuotas en una franja compacta: una línea por proveedor y el consejo de reparto. «Detalle»
 * despliega las tarjetas con cada ventana, su reinicio y su ritmo. Lo que cambia cada segundo («leídas hace…») va en
 * un hueco de ancho fijo, para no mover nada (lo que cambia en vivo no mueve el diseño).
 */
export function Cuotas({ estado, leyendo, alLeer }: { estado: EstadoCuotas | null; leyendo: boolean; alLeer: () => void }) {
  const ahora = useAhora();
  const [abierto, setAbierto] = useState(false);
  const lec = estado?.lectura;
  const posturas = new Map((lec?.advice?.providers ?? []).map((p) => [p.provider, p]));
  // Cuotas es opcional: sin ella (ya leído, no instalada), la franja no aparece en vez de mostrar un error.
  if (estado && estado.leida_en > 0 && !estado.instalada && !lec) return null;
  return (
    <section aria-labelledby="cuotas" className="flex flex-col gap-3 rounded-tarjeta border border-borde bg-superficie px-4 py-3 sm:px-5">
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2.5">
        <h2 id="cuotas" className="sr-only">
          Cuotas
        </h2>
        {lec ? (
          lec.panels.map((p) => <Franja key={p.provider} p={p} postura={posturas.get(p.provider)} ahora={ahora} />)
        ) : (
          <div className="esqueleto h-5 flex-1 rounded-chip" aria-label="Leyendo las cuotas con la herramienta opcional Cuotas" />
        )}
      </div>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2 text-ayuda text-suave">
        <span className="min-w-0 flex-[1_1_20rem]">
          {lec?.advice ? (
            <>
              <span className="text-acento-fuerte">{lec.advice.summary}</span> <span className="font-medium text-texto">Primero: {lec.advice.first_choice}.</span>
            </>
          ) : estado?.error ? (
            <span className="text-aviso">
              {estado.origen === "archivos" ? "cuotas.exe no respondió; se muestra lo que dicen sus archivos, sin consejo. " : ""}
              {estado.error} Si no está instalada, Cuotas es opcional.
            </span>
          ) : null}
        </span>
        <span className="w-[7.5rem] shrink-0 truncate text-right whitespace-nowrap text-tenue" title={lec ? lec.panels.map(fuente).join(", ") : undefined}>
          {lec && ahora > 0 ? `Leídas ${haceCuanto(estado!.leida_en, ahora)}` : estado?.error ? "Sin leer" : "Leyendo…"}
        </span>
        <button
          type="button"
          onClick={alLeer}
          disabled={leyendo}
          aria-label="Leer las cuotas ahora"
          title="Leer ahora"
          className="grid size-7 place-items-center rounded-control border border-borde text-suave hover:border-borde-fuerte hover:bg-elevado hover:text-texto disabled:cursor-wait disabled:text-tenue"
        >
          <IconoRecargar className={`size-3.5 ${leyendo ? "animate-spin" : ""}`} />
        </button>
        <button
          type="button"
          onClick={() => setAbierto((a) => !a)}
          aria-expanded={abierto}
          aria-controls="cuotas-detalle"
          disabled={!lec}
          className="flex h-7 items-center gap-1.5 rounded-control border border-borde px-2.5 text-suave hover:border-borde-fuerte hover:bg-elevado hover:text-texto disabled:text-tenue"
        >
          Detalle
          <IconoChevron className={`size-3 transition-transform duration-150 ${abierto ? "-rotate-90" : "rotate-90"}`} />
        </button>
      </div>
      {abierto && lec && (
        <div id="cuotas-detalle" className="grid gap-4 border-t border-borde pt-3.5 lg:grid-cols-2">
          {lec.panels.map((p) => (
            <Tarjeta key={p.provider} p={p} postura={posturas.get(p.provider)} ahora={ahora} />
          ))}
        </div>
      )}
    </section>
  );
}
