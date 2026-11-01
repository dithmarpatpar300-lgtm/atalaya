"use client";

import { type ReactNode, useEffect, useRef, useState } from "react";
import { Markdown } from "@/components/ui/markdown";
import { Segmentado } from "@/components/ui/segmentado";
import { api, type Busqueda, type DocumentoDatapedia } from "@/lib/api";
import { haceCuanto } from "@/lib/formato";
import { idioma, t, tServidor } from "@/lib/i18n";
import { useAhora } from "@/lib/vivo";
import { BuscarPasos } from "./buscar-pasos";
import { MensajeNivel } from "@/components/ui/tarjeta-alerta";

export const TIPOS: Record<string, string> = {
  memoria: t("Memoria"),
  traza: t("Trazas"),
  regla: t("Reglas"),
  adr: t("ADR"),
  plan: t("Planes"),
  informe: t("Informes"),
  bitacora: t("Bitácoras"),
  acta: t("Actas"),
  codex: t("Codex"),
  estudio: t("Estudios"),
  documento: t("Otros"),
  // «Archivo» ya significa «File» en el diccionario: aquí es la carpeta docs/archivo/ (lo archivado).
  archivo: idioma() === "en" ? "Archive" : "Archivo",
};
const ORDEN = Object.keys(TIPOS);

/** El fragmento de la búsqueda con las coincidencias resaltadas (vienen entre \u0002 y \u0003). */
function Fragmento({ texto }: { texto: string }) {
  const partes: ReactNode[] = [];
  texto.split("\u0002").forEach((trozo, i) => {
    if (i === 0) {
      partes.push(trozo);
      return;
    }
    const [marcado, resto] = trozo.split("\u0003");
    partes.push(
      <mark key={i} className="rounded-[3px] bg-acento-suave px-0.5 text-acento-fuerte">
        {marcado}
      </mark>,
      resto,
    );
  });
  return <>{partes}</>;
}

// Los documentos ya abiertos desde la paleta (por su `cuando`): al volver a la datapedia no se reabre uno viejo.
const abiertos = new Set<number>();

/** Quita el primer «# Título» del cuerpo cuando es el mismo que ya muestra la cabecera del documento. */
function sinTitulo(cuerpo: string, titulo: string): string {
  return cuerpo.replace(/^(\s*---\r?\n[\s\S]*?\r?\n---\r?\n)?(\s*)#\s+(.+?)\s*(\r?\n|$)/, (todo, fm = "", blanco, h) =>
    h.trim() === titulo.trim() ? fm + blanco : todo,
  );
}

function Chip({ activo, children, alPulsar }: { activo: boolean; children: ReactNode; alPulsar: () => void }) {
  return (
    <button
      type="button"
      aria-pressed={activo}
      onClick={alPulsar}
      className={`flex h-7 items-center gap-1.5 rounded-chip border px-2 text-ayuda transition-colors duration-150 ${
        activo ? "border-acento bg-acento-suave font-medium text-acento-fuerte" : "border-borde text-suave hover:border-borde-realce hover:text-texto"
      }`}
    >
      {children}
    </button>
  );
}

/** «trazas/<agente>/turno-<n>»: la traza de un turno de un agente (backend/internal/claude/trazas.go). */
function deTraza(ruta: string): { agente: string; turno: number } | null {
  const m = /^trazas\/(.+)\/turno-(\d+)$/.exec(ruta);
  return m ? { agente: m[1], turno: Number(m[2]) } : null;
}

export function Datapedia({
  proyecto,
  version,
  alAbrirTraza,
  alAbrirPaso,
  abrir,
}: {
  proyecto: string;
  version: number;
  alAbrirTraza: (proyecto: string, agente: string, turno: number) => void;
  alAbrirPaso: (proyecto: string, agente: string, n: number) => void;
  /** Un documento que hay que abrir (desde la paleta de órdenes); `cuando` distingue dos pedidos del mismo. */
  abrir?: { id: number; cuando: number } | null;
}) {
  const ahora = useAhora();
  // Qué se busca: los documentos (memoria, reglas, informes) o los pasos de los agentes (mejora 5).
  const [modo, setModo] = useState<"documentos" | "pasos">("documentos");
  const [q, setQ] = useState("");
  const [tipo, setTipo] = useState("");
  const [alcance, setAlcance] = useState<"proyecto" | "todos">("todos");
  const [busqueda, setBusqueda] = useState<Busqueda | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [elegido, setElegido] = useState<number | null>(null);
  const [doc, setDoc] = useState<DocumentoDatapedia | null>(null);
  const [aviso, setAviso] = useState<string | null>(null);
  const campo = useRef<HTMLInputElement>(null);
  const elegidoRef = useRef<number | null>(null);

  useEffect(() => {
    elegidoRef.current = elegido;
  }, [elegido]);

  // Desde la paleta: abre ese documento (fuera del efecto, para no encadenar pintados).
  const abrirId = abrir?.id;
  const abrirCuando = abrir?.cuando;
  useEffect(() => {
    if (abrirId === undefined || abrirCuando === undefined || abiertos.has(abrirCuando)) return;
    const espera = setTimeout(() => {
      abiertos.add(abrirCuando);
      setElegido(abrirId);
    }, 0);
    return () => clearTimeout(espera);
  }, [abrirId, abrirCuando]);

  // «/» lleva a la búsqueda, como en la mayoría de los buscadores.
  useEffect(() => {
    function alTecla(e: KeyboardEvent) {
      const el = e.target as HTMLElement;
      if (e.key === "/" && !/^(INPUT|TEXTAREA|SELECT)$/.test(el.tagName) && !el.isContentEditable) {
        e.preventDefault();
        campo.current?.focus();
      }
    }
    document.addEventListener("keydown", alTecla);
    return () => document.removeEventListener("keydown", alTecla);
  }, []);

  // La búsqueda, 200 ms después de la última tecla; la anterior se cancela.
  useEffect(() => {
    const control = new AbortController();
    const espera = setTimeout(() => {
      api
        .buscar({ q, tipo, proyecto: alcance === "proyecto" ? proyecto : "" }, control.signal)
        .then((b) => {
          setBusqueda(b);
          setError(null);
          if (elegidoRef.current === null && b.resultados.length) setElegido(b.resultados[0].id);
        })
        .catch((e: Error) => {
          if (e.name !== "AbortError") setError(tServidor(e.message));
        });
    }, 200);
    return () => {
      clearTimeout(espera);
      control.abort();
    };
  }, [q, tipo, alcance, proyecto, version]);

  useEffect(() => {
    if (elegido === null) return;
    api
      .documento(elegido)
      .then((d) => {
        setDoc(d);
        setAviso(null);
      })
      .catch((e: Error) => setAviso(tServidor(e.message)));
  }, [elegido, version]);

  function abrirRuta(ruta: string) {
    if (!doc) return;
    api
      .documentoPorRuta(doc.proyecto, ruta)
      .then((d) => {
        setDoc(d);
        setElegido(d.id);
        setAviso(null);
      })
      .catch(() => setAviso(t("{ruta} no está en la datapedia (solo se indexan los .md de docs/, la memoria y las reglas).", { ruta })));
  }

  const cuentas = new Map((busqueda?.cuentas ?? []).map((c) => [c.tipo, c.cantidad]));
  const total = [...cuentas.values()].reduce((s, n) => s + n, 0);
  const resultados = busqueda?.resultados ?? [];
  const visible = doc && doc.id === elegido ? doc : null;

  const selector = (
    <Segmentado
      etiqueta={t("Qué buscar")}
      valor={modo}
      alCambiar={setModo}
      opciones={[
        { valor: "documentos", nombre: t("Documentos") },
        { valor: "pasos", nombre: t("Pasos de los agentes") },
      ]}
    />
  );
  if (modo === "pasos") return <BuscarPasos proyecto={proyecto} version={version} selector={selector} alAbrirPaso={alAbrirPaso} />;

  return (
    <main id="contenido" tabIndex={-1} aria-label={t("Datapedia")} className="relative mx-auto flex w-full max-w-[1520px] flex-col items-start gap-5 p-5 lg:flex-row">
      <aside
        aria-label={t("Búsqueda en la datapedia")}
        className="w-full overflow-hidden rounded-tarjeta border border-borde bg-superficie lg:sticky lg:top-[4.25rem] lg:w-[26rem] lg:shrink-0"
      >
        <div className="flex flex-col gap-3 border-b border-borde p-4">
          <h2 className="m-0 text-encabezado font-semibold">{t("Datapedia")}</h2>
          {selector}
          <label className="sr-only" htmlFor="buscar">
            {t("Buscar en la memoria y los documentos")}
          </label>
          <div className="relative">
            <input
              ref={campo}
              id="buscar"
              type="search"
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder={t("Buscar: bitácora, Retry-After, cuota…")}
              autoComplete="off"
              spellCheck={false}
              className="h-(--alto-control) w-full rounded-control border border-borde-campo bg-superficie pr-10 pl-3 text-cuerpo text-texto placeholder:text-tenue hover:border-suave focus-visible:border-acento"
            />
            {q === "" && (
              <kbd aria-hidden="true" className="dato absolute top-1/2 right-3 -translate-y-1/2 rounded-chip border border-borde px-1.5 text-ayuda text-tenue">
                /
              </kbd>
            )}
          </div>
          <Segmentado
            etiqueta={t("Proyectos")}
            valor={alcance}
            alCambiar={setAlcance}
            opciones={[
              { valor: "todos", nombre: t("Todos los proyectos") },
              { valor: "proyecto", nombre: proyecto || t("Este proyecto") },
            ]}
          />
          <div role="group" aria-label={t("Tipo de documento")} className="flex flex-wrap gap-1.5">
            <Chip activo={tipo === ""} alPulsar={() => setTipo("")}>
              {t("Todo")} <span className="dato text-tenue">{total}</span>
            </Chip>
            {ORDEN.filter((k) => cuentas.has(k) || k === tipo).map((k) => (
              <Chip key={k} activo={tipo === k} alPulsar={() => setTipo(tipo === k ? "" : k)}>
                {TIPOS[k]} <span className="dato text-tenue">{cuentas.get(k) ?? 0}</span>
              </Chip>
            ))}
          </div>
        </div>
        {error ? (
          <p className="m-0 px-4 py-6 text-rotulo text-error">{t("No se pudo buscar: {error}", { error })}</p>
        ) : !busqueda ? (
          <div className="flex flex-col gap-2 p-3" aria-hidden="true">
            {[0, 1, 2, 3].map((i) => (
              <div key={i} className="esqueleto h-14 rounded-control" />
            ))}
          </div>
        ) : resultados.length === 0 ? (
          <p className="m-0 px-4 py-6 text-rotulo text-suave">{t("Nada coincide. Prueba con menos palabras o quita el filtro de tipo.")}</p>
        ) : (
          <ul aria-live="polite" className="scroll-interno m-0 flex max-h-[calc(100vh-19rem)] list-none flex-col gap-0.5 overflow-y-auto p-1.5">
            {resultados.map((d) => {
              const actual = d.id === elegido;
              return (
                <li key={d.id}>
                  <button
                    type="button"
                    onClick={() => setElegido(d.id)}
                    aria-current={actual ? "true" : undefined}
                    className={`flex w-full flex-col gap-0.5 rounded-control px-3 py-2.5 text-left ${actual ? "bg-acento-suave" : "hover:bg-elevado"}`}
                  >
                    <span className="flex items-center gap-2">
                      <span className="min-w-0 flex-1 truncate font-medium">{d.titulo}</span>
                      <span className="shrink-0 text-ayuda text-tenue">{TIPOS[d.tipo] ?? d.tipo}</span>
                    </span>
                    <span className="dato truncate text-ayuda text-tenue">
                      {alcance === "todos" ? `${d.proyecto} · ` : ""}
                      {d.ruta}
                    </span>
                    <span className="line-clamp-2 text-rotulo text-suave">{d.fragmento ? <Fragmento texto={d.fragmento} /> : d.resumen}</span>
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </aside>

      <div className="min-w-0 flex-1">
        <article aria-labelledby="doc-titulo" className="flex flex-col gap-4 rounded-tarjeta border border-borde bg-superficie px-6 py-5">
          {aviso && <MensajeNivel nivel="aviso">{aviso}</MensajeNivel>}
          {visible ? (
            <>
              <header className="flex flex-col gap-1 border-b border-borde pb-4">
                <div className="flex flex-wrap items-center gap-2.5">
                  <h1 id="doc-titulo" className="m-0 text-titulo font-semibold">
                    {visible.titulo}
                  </h1>
                  <span className="rounded-chip bg-elevado px-2 py-0.5 text-ayuda font-medium text-suave">{TIPOS[visible.tipo] ?? visible.tipo}</span>
                  {(() => {
                    const tr = deTraza(visible.ruta);
                    return tr ? (
                      <button
                        type="button"
                        onClick={() => alAbrirTraza(visible.proyecto, tr.agente, tr.turno)}
                        className="ml-auto h-(--alto-control-compacto) rounded-control bg-acento px-3 text-rotulo font-medium text-sobre-acento hover:bg-acento-fuerte"
                      >
                        {t("Ver el turno en Agentes →")}
                      </button>
                    ) : null;
                  })()}
                </div>
                <span className="dato text-ayuda text-tenue">
                  {visible.proyecto} · {visible.ruta}
                  {ahora > 0 && ` · ${t("cambiado {cuando}", { cuando: haceCuanto(visible.mtime, ahora) })}`}
                </span>
              </header>
              <div className="min-w-0 text-cuerpo">
                <Markdown texto={sinTitulo(visible.cuerpo ?? "", visible.titulo)} ruta={visible.ruta} alAbrirRuta={abrirRuta} />
              </div>
            </>
          ) : elegido !== null ? (
            <div className="esqueleto h-40 rounded-panel" aria-label={t("Cargando")} />
          ) : (
            <p className="m-0 text-suave">{t("Busca en la memoria, las reglas, los ADR, los planes, las bitácoras y los informes del centro de operaciones.")}</p>
          )}
        </article>
      </div>
    </main>
  );
}
