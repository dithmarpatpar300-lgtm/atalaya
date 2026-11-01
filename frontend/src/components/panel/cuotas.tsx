"use client";

import { useState } from "react";
import { IconoChevron, IconoRecargar } from "@/components/ui/iconos";
import type { EstadoCuotas, PanelCuota, PosturaProveedor, VentanaCuota } from "@/lib/api";
import { cuandoReinicia, duracion, haceCuanto, porcentaje, ritmo, tokens } from "@/lib/formato";
import { t, tServidor } from "@/lib/i18n";
import { usaUsoClaude } from "@/lib/uso-claude";
import { useAhora } from "@/lib/vivo";
import { DetalleClaude, FranjaClaude, useUsoClaude } from "./uso-claude";

const NOMBRE: Record<string, string> = { "Claude Code": "Claude", Codex: "Codex" };
const VENTANA: Record<string, string> = { "5 h": "5 horas", Semanal: "Semana" };

function etiquetaVentana(v: VentanaCuota): string {
  return VENTANA[v.label] ? t(VENTANA[v.label]) : v.label;
}

function Postura({ postura }: { postura?: PosturaProveedor }) {
  if (!postura) return null;
  const normal = postura.stance === "normal";
  const texto = postura.stance.charAt(0).toUpperCase() + postura.stance.slice(1);
  return (
    <span
      title={postura.reasons.length ? postura.reasons.map(tServidor).join("; ") : undefined}
      className={`rounded-chip px-2 py-0.5 text-ayuda font-medium ${normal ? "bg-ok-suave text-ok" : "bg-aviso-suave text-aviso"}`}
    >
      {texto}
    </span>
  );
}

function Ventana({ v, color, ahora, principal }: { v: VentanaCuota; color: string; ahora: number; principal: boolean }) {
  const usado = Math.min(100, Math.max(0, v.used_percent));
  const vencida = v.resets_at > 0 && v.resets_at <= ahora;
  const quedanTok = v.tokens_left_estimate ? ` ${t("(≈{n} tokens)", { n: tokens(v.tokens_left_estimate) })}` : "";
  return (
    <div className="grid grid-cols-[minmax(6rem,8.75rem)_minmax(0,1fr)_4rem] items-center gap-x-4 gap-y-1">
      <span className="text-rotulo">{etiquetaVentana(v)}</span>
      <div
        className="h-1.5 rounded-full bg-elevado"
        role="meter"
        aria-label={t("{ventana}: {pct} usado", { ventana: etiquetaVentana(v), pct: porcentaje(usado) })}
        aria-valuemin={0}
        aria-valuemax={100}
        aria-valuenow={usado}
      >
        <div className={`h-1.5 rounded-full ${color}`} style={{ width: `${usado}%`, opacity: principal ? 1 : 0.6 }} />
      </div>
      <span className="dato text-right">{porcentaje(usado)}</span>
      {/* La cuenta atrás cambia cada minuto: en su propia línea fija, sin empujar la cifra de al lado (lo que cambia en vivo no mueve el diseño). */}
      <span className="col-span-2 flex flex-col text-ayuda text-suave">
        <span>{t("Quedan {pct}{tok}.", { pct: porcentaje(100 - usado), tok: quedanTok })}</span>
        <span className="dato h-4 truncate">
          {vencida
            ? t("Ya se reinició; falta una lectura nueva.")
            : v.resets_at > 0 && t("Reinicia {cuando}, en {d}.", { cuando: cuandoReinicia(v.resets_at, ahora), d: duracion(v.resets_at - ahora) })}
        </span>
      </span>
      <span className="dato text-right text-ayuda text-suave" title={v.pace ? t("Ritmo de la ventana") : undefined}>
        {v.pace ? ritmo(v.pace.percent_per_hour) : t("sin dato")}
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
      {p.notice && <p className="m-0 text-ayuda text-aviso">{tServidor(p.notice)}.</p>}
      {postura && (
        <p className="m-0 text-ayuda text-tenue">
          {t("Margen {margen}; ritmo parejo hasta el reinicio {ritmo}.", { margen: porcentaje(postura.headroom_percent), ritmo: ritmo(postura.even_pace_percent_per_hour) })}
        </p>
      )}
    </div>
  );
}

function fuente(p: PanelCuota): string {
  const n = NOMBRE[p.provider] ?? p.provider;
  if (p.source.startsWith("statusLine")) return t("{n} por la statusLine", { n });
  if (p.source.includes("app-server")) return t("{n} por su app-server", { n });
  return t("{n} por el {fuente}", { n, fuente: p.source });
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
            title={
              vencida
                ? t("Ya se reinició; falta una lectura nueva")
                : v.resets_at > 0 && ahora
                  ? t("Reinicia {cuando}, en {d}", { cuando: cuandoReinicia(v.resets_at, ahora), d: duracion(v.resets_at - ahora) })
                  : undefined
            }
            className="flex min-w-0 flex-[1_1_7rem] items-center gap-2"
          >
            <span className="shrink-0 text-ayuda whitespace-nowrap text-suave">{t(v.label)}</span>
            <span
              role="meter"
              aria-label={t("{nombre}, {ventana}: {pct} usado", { nombre: NOMBRE[p.provider] ?? p.provider, ventana: etiquetaVentana(v), pct: porcentaje(usado) })}
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
  // Claude sale de /usage cuando el servidor lo tiene (lib/uso-claude.ts); entonces su línea de Cuotas no se muestra,
  // para que la franja y el detalle no digan dos cifras distintas. Codex sigue con Cuotas.
  const vivo = useUsoClaude(estado);
  const claude = usaUsoClaude(vivo.uso) ? vivo.uso : null;
  const paneles = (lec?.panels ?? []).filter((p) => !(claude && p.provider === "Claude Code"));
  // Cuotas es opcional: sin ella (ya leído, no instalada) y sin /usage, la franja no aparece en vez de mostrar un error.
  if (!claude && estado && estado.leida_en > 0 && !estado.instalada && !lec) return null;
  return (
    <section aria-labelledby="cuotas" className="flex flex-col gap-3 rounded-tarjeta border border-borde bg-superficie px-4 py-3 sm:px-5">
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2.5">
        <h2 id="cuotas" className="sr-only">
          {t("Cuotas")}
        </h2>
        {claude && <FranjaClaude u={claude} ahora={ahora} vivo={vivo} />}
        {lec ? (
          paneles.map((p) => <Franja key={p.provider} p={p} postura={posturas.get(p.provider)} ahora={ahora} />)
        ) : claude && (estado?.error || (estado && !estado.instalada)) ? null : (
          <div className="esqueleto h-5 flex-1 rounded-chip" aria-label={t("Leyendo las cuotas con la herramienta opcional Cuotas")} />
        )}
      </div>
      <div className="flex flex-wrap items-center gap-x-3 gap-y-2 text-ayuda text-suave">
        <span className="min-w-0 flex-[1_1_20rem]">
          {lec?.advice ? (
            <>
              <span className="text-acento-fuerte">{tServidor(lec.advice.summary)}</span>{" "}
              <span className="font-medium text-texto">{t("Primero: {opcion}.", { opcion: tServidor(lec.advice.first_choice) })}</span>
            </>
          ) : estado?.error ? (
            <span className="text-aviso">
              {estado.origen === "archivos" ? `${t("cuotas.exe no respondió; se muestra lo que dicen sus archivos, sin consejo.")} ` : ""}
              {tServidor(estado.error)} {t("Si no está instalada, Cuotas es opcional.")}
            </span>
          ) : null}
        </span>
        <span className="w-[7.5rem] shrink-0 truncate text-right whitespace-nowrap text-tenue" title={lec ? lec.panels.map(fuente).join(", ") : undefined}>
          {lec && ahora > 0 ? t("Leídas {hace}", { hace: haceCuanto(estado!.leida_en, ahora) }) : estado?.error ? t("Sin leer") : t("Leyendo…")}
        </span>
        <button
          type="button"
          onClick={alLeer}
          disabled={leyendo}
          aria-label={t("Leer las cuotas ahora")}
          title={t("Leer las cuotas ahora")}
          className="grid size-7 place-items-center rounded-control border border-borde text-suave hover:border-borde-fuerte hover:bg-elevado hover:text-texto disabled:cursor-wait disabled:text-tenue"
        >
          <IconoRecargar className={`size-3.5 ${leyendo ? "animate-spin" : ""}`} />
        </button>
        <button
          type="button"
          onClick={() => setAbierto((a) => !a)}
          aria-expanded={abierto}
          aria-controls="cuotas-detalle"
          disabled={!lec && !claude}
          className="flex h-7 items-center gap-1.5 rounded-control border border-borde px-2.5 text-suave hover:border-borde-fuerte hover:bg-elevado hover:text-texto disabled:text-tenue"
        >
          {t("Detalle")}
          <IconoChevron className={`size-3 transition-transform duration-150 ${abierto ? "-rotate-90" : "rotate-90"}`} />
        </button>
      </div>
      {abierto && (lec || claude) && (
        <div id="cuotas-detalle" className="grid gap-4 border-t border-borde pt-3.5 lg:grid-cols-2">
          {claude && <DetalleClaude u={claude} ahora={ahora} vivo={vivo} />}
          {paneles.map((p) => (
            <Tarjeta key={p.provider} p={p} postura={posturas.get(p.provider)} ahora={ahora} />
          ))}
        </div>
      )}
    </section>
  );
}
