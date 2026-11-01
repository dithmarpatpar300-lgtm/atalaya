"use client";

import { type ClipboardEvent, useEffect, useRef, useState } from "react";
import { IconoCerrar } from "@/components/ui/iconos";
import { api } from "@/lib/api";
import { decimal, t } from "@/lib/i18n";
import {
  type Borrador,
  borradorVacio,
  cuerpo,
  elegir,
  elegirOtra,
  motivoImagen,
  type PreguntaCentro,
  respondida,
  textoRespuesta,
  TIPOS_IMAGEN,
  visibles,
} from "@/lib/preguntas";

interface Imagen {
  nombre: string;
  bytes: number;
  base64: string;
  url: string;
}

const kb = (n: number) => (n < 1024 * 1024 ? `${Math.max(1, Math.round(n / 1024))} KB` : `${decimal(n / 1024 / 1024, 1)} MB`);

function leerBase64(f: File): Promise<string> {
  return new Promise((ok, mal) => {
    const r = new FileReader();
    r.onload = () => ok(String(r.result).replace(/^data:[^,]*,/, ""));
    r.onerror = () => mal(r.error);
    r.readAsDataURL(f);
  });
}

/**
 * Responder desde Atalaya: un paso por pregunta, con sus opciones y «Escribir otra respuesta»;
 * imágenes adjuntas o pegadas (Ctrl V), que van con toda la respuesta; volver a cualquier paso y cambiarlo (lo que
 * dependía de esa opción se vuelve a preguntar); y un resumen antes de enviar. Nada sale hasta «Enviar a la sesión».
 */
export function Responder({ id, alCerrar }: { id: string; alCerrar: () => void }) {
  const dialogo = useRef<HTMLDialogElement>(null);
  const archivo = useRef<HTMLInputElement>(null);
  const [p, setP] = useState<PreguntaCentro | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [paso, setPaso] = useState(0);
  const [b, setB] = useState<Borrador>(borradorVacio);
  const [nota, setNota] = useState("");
  const [imagenes, setImagenes] = useState<Imagen[]>([]);
  const [aviso, setAviso] = useState("");
  const [enviando, setEnviando] = useState(false);
  const [enviado, setEnviado] = useState(false);

  useEffect(() => {
    dialogo.current?.showModal();
    const c = new AbortController();
    api
      .pregunta(id, c.signal)
      .then(setP)
      .catch((e: unknown) => {
        if (!(e instanceof DOMException && e.name === "AbortError")) setError(e instanceof Error ? e.message : t("no se pudo leer"));
      });
    return () => c.abort();
  }, [id]);

  // Las vistas previas se sueltan al quitarlas o al cerrar el diálogo (no en cada cambio: las demás siguen en uso).
  const urls = useRef<string[]>([]);
  useEffect(() => {
    urls.current = imagenes.map((im) => im.url);
  }, [imagenes]);
  useEffect(() => () => urls.current.forEach((u) => URL.revokeObjectURL(u)), []);
  function quitar(im: Imagen) {
    URL.revokeObjectURL(im.url);
    setImagenes((xs) => xs.filter((x) => x !== im));
  }

  const vis = p ? visibles(p.preguntas, b.elecciones) : [];
  const enResumen = paso >= vis.length;
  const q = enResumen ? null : vis[paso];
  const listo = vis.every((x) => respondida(x, b));

  async function sumarArchivos(fs: File[]) {
    let ya = imagenes.length;
    const nuevas: Imagen[] = [];
    for (const f of fs) {
      const motivo = motivoImagen(f.type, f.size, ya);
      if (motivo) {
        setAviso(motivo);
        continue;
      }
      nuevas.push({ nombre: f.name || `captura-${ya + 1}.png`, bytes: f.size, base64: await leerBase64(f), url: URL.createObjectURL(f) });
      ya++;
    }
    if (nuevas.length) {
      setAviso("");
      setImagenes((xs) => [...xs, ...nuevas]);
    }
  }

  function alPegar(e: ClipboardEvent) {
    const fs = Array.from(e.clipboardData.files).filter((f) => TIPOS_IMAGEN.includes(f.type));
    if (fs.length) {
      e.preventDefault();
      void sumarArchivos(fs);
    }
  }

  async function enviar() {
    if (!p) return;
    setEnviando(true);
    setError(null);
    try {
      await api.responder(
        p.id,
        cuerpo(
          p.preguntas,
          b,
          nota,
          imagenes.map((im) => ({ nombre: im.nombre, base64: im.base64 })),
        ),
      );
      setEnviado(true);
    } catch (e) {
      setError(e instanceof Error ? e.message : t("no se pudo enviar"));
    } finally {
      setEnviando(false);
    }
  }

  const pasos = [...vis.map((x) => ({ id: x.id, texto: respondida(x, b) ? textoRespuesta(x, b) : x.encabezado || x.texto })), { id: "fin", texto: t("Revisar y enviar") }];

  return (
    <dialog
      ref={dialogo}
      onClose={alCerrar}
      onPaste={alPegar}
      aria-labelledby="responder-titulo"
      className="capa m-auto w-[min(60rem,calc(100vw-2rem))] rounded-tarjeta p-0 text-texto backdrop:bg-velo"
    >
      <div className="flex items-center gap-3 border-b border-borde px-5 py-3.5">
        <div className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-ayuda text-suave">
            {p ? t("{proyecto} · pregunta por Atalaya · la sesión espera", { proyecto: p.proyecto }) : t("Cargando…")}
          </span>
          <h2 id="responder-titulo" className="m-0 truncate text-encabezado font-semibold" title={p?.titulo}>
            {p?.titulo ?? t("Responder")}
          </h2>
        </div>
        <button type="button" onClick={() => dialogo.current?.close()} aria-label={t("Cerrar")} className="grid size-9 place-items-center rounded-control hover:bg-elevado">
          <IconoCerrar className="size-4" />
        </button>
      </div>

      {p && !enviado && (
        <ol aria-label={t("Pasos")} className="m-0 flex list-none gap-1.5 overflow-x-auto px-5 pt-4">
          {pasos.map((x, i) => {
            const actual = i === paso;
            // Se puede ir a un paso ya visto, o a uno cuyas preguntas anteriores están todas respondidas.
            const abierto = i <= paso || vis.slice(0, i).every((y) => respondida(y, b));
            return (
              <li key={x.id} className="min-w-[8rem] flex-1">
                <button
                  type="button"
                  disabled={!abierto}
                  aria-current={actual ? "step" : undefined}
                  onClick={() => setPaso(i)}
                  className={`flex h-[54px] w-full flex-col rounded-control border px-2.5 py-1.5 text-left disabled:opacity-50 ${actual ? "border-acento bg-seleccion" : "border-borde bg-hundido hover:border-borde-fuerte"}`}
                >
                  <span className="dato text-[11px] text-tenue">{t("Paso {n}", { n: i + 1 })}</span>
                  <span className="truncate text-rotulo" title={x.texto}>
                    {x.texto}
                  </span>
                </button>
              </li>
            );
          })}
        </ol>
      )}

      <div className="scroll-interno flex max-h-[62vh] min-h-[22rem] flex-col gap-3 overflow-y-auto px-5 py-4">
        {error && <p className="m-0 rounded-panel bg-error-suave px-3 py-2 text-error">{error}</p>}
        {!p && !error && <div className="esqueleto h-40 rounded-panel" aria-hidden="true" />}
        {p?.contexto && paso === 0 && !enviado && <p className="m-0 rounded-control bg-hundido px-3 py-2.5 text-rotulo text-suave">{p.contexto}</p>}

        {enviado && (
          <div role="status" className="flex flex-col items-start gap-2 py-10">
            <span className="rounded-chip bg-ok-suave px-2.5 py-0.5 font-semibold text-ok">{t("Enviado")}</span>
            <p className="m-0 text-[17px] font-semibold">{t("La sesión lo recibe ahora, con el texto y las imágenes.")}</p>
            <p className="m-0 text-suave">{t("Si la sesión no estaba esperando, lo recibe la próxima vez que pregunte por este id.")}</p>
          </div>
        )}

        {p && !enviado && q && (
          <fieldset className="m-0 flex flex-col gap-2 border-0 p-0">
            {q.encabezado && <span className="self-start rounded-chip bg-acento-suave px-2 text-ayuda font-semibold text-acento-fuerte">{q.encabezado}</span>}
            <legend className="sr-only">{q.texto}</legend>
            <p className="m-0 text-[17px] leading-snug font-semibold">{q.texto}</p>
            {q.multiple && <span className="text-ayuda text-suave">{t("Puedes elegir varias.")}</span>}
            {q.opciones.map((o, j) => {
              const on = (b.elecciones[q.id] ?? []).includes(o.etiqueta);
              return (
                <button
                  key={`${j}:${o.etiqueta}`}
                  type="button"
                  role={q.multiple ? "checkbox" : "radio"}
                  aria-checked={on}
                  onClick={() => setB(elegir(b, q, o.etiqueta))}
                  className={`grid min-h-[3.25rem] grid-cols-[1.25rem_minmax(0,1fr)] items-start gap-3 rounded-control border px-3.5 py-2.5 text-left ${on ? "border-acento bg-seleccion" : "border-borde bg-hundido hover:border-borde-fuerte"}`}
                >
                  <span aria-hidden="true" className={`mt-0.5 size-4 border-2 ${q.multiple ? "rounded-[4px]" : "rounded-full"} ${on ? "border-acento bg-acento" : "border-borde-fuerte"}`} />
                  <span className="flex flex-col gap-0.5">
                    <span className="flex flex-wrap items-center gap-2 font-semibold">
                      {o.etiqueta}
                      {o.recomendada && <span className="rounded-chip bg-acento-suave px-1.5 text-ayuda font-normal text-acento-fuerte">{t("recomendada")}</span>}
                    </span>
                    {o.nota && <span className="text-rotulo text-suave">{o.nota}</span>}
                    {(o.abre?.length ?? 0) > 0 && <span className="text-ayuda text-opcional">{t("Abre otra pregunta")}</span>}
                  </span>
                </button>
              );
            })}
            <button
              type="button"
              role={q.multiple ? "checkbox" : "radio"}
              aria-checked={!!b.otra[q.id]}
              onClick={() => setB(b.otra[q.id] && q.multiple ? { ...b, otra: { ...b.otra, [q.id]: false } } : elegirOtra(b, q))}
              className={`grid min-h-[3.25rem] grid-cols-[1.25rem_minmax(0,1fr)] items-start gap-3 rounded-control border border-dashed px-3.5 py-2.5 text-left ${b.otra[q.id] ? "border-acento bg-seleccion" : "border-borde-fuerte hover:bg-elevado/50"}`}
            >
              <span aria-hidden="true" className={`mt-0.5 size-4 border-2 ${q.multiple ? "rounded-[4px]" : "rounded-full"} ${b.otra[q.id] ? "border-acento bg-acento" : "border-borde-fuerte"}`} />
              <span className="flex flex-col gap-0.5">
                <span className="font-semibold">{t("Escribir otra respuesta")}</span>
                <span className="text-rotulo text-suave">{t("Con tus palabras; las imágenes van abajo.")}</span>
              </span>
            </button>
            {b.otra[q.id] && (
              <label className="flex flex-col gap-1.5">
                <span className="text-ayuda text-suave">{t("Tu respuesta para «{pregunta}»", { pregunta: q.encabezado || q.texto })}</span>
                <textarea
                  autoFocus
                  rows={3}
                  value={b.textos[q.id] ?? ""}
                  onChange={(e) => setB({ ...b, textos: { ...b.textos, [q.id]: e.target.value } })}
                  className="resize-y rounded-control border border-borde-fuerte bg-hundido px-3 py-2"
                />
              </label>
            )}
          </fieldset>
        )}

        {p && !enviado && enResumen && (
          <div className="flex flex-col gap-2">
            <p className="m-0 text-[17px] font-semibold">{t("Esto es lo que va a recibir la sesión")}</p>
            {vis.map((x, i) => (
              <div key={x.id} className="grid min-h-12 grid-cols-[9rem_minmax(0,1fr)_auto] items-center gap-3 rounded-control bg-hundido px-3 py-2">
                <span className="truncate text-ayuda text-suave" title={x.texto}>
                  {x.encabezado || x.texto}
                </span>
                <span className={`min-w-0 ${respondida(x, b) ? "font-semibold" : "text-error"}`}>{respondida(x, b) ? textoRespuesta(x, b) : t("falta responder")}</span>
                <button type="button" onClick={() => setPaso(i)} className="h-8 rounded-control border border-borde-fuerte px-2.5 text-rotulo hover:bg-elevado">
                  {t("Cambiar")}
                </button>
              </div>
            ))}
            <label className="mt-1 flex flex-col gap-1.5">
              <span className="text-rotulo text-suave">{t("Algo más para la sesión (opcional)")}</span>
              <textarea rows={2} value={nota} onChange={(e) => setNota(e.target.value)} className="resize-y rounded-control border border-borde-fuerte bg-hundido px-3 py-2" />
            </label>
          </div>
        )}

        {p && !enviado && (
          <div className="flex flex-col gap-2 border-t border-borde pt-3">
            <span className="text-ayuda text-suave">{t("Imágenes para la sesión, con toda la respuesta (adjunta o pega con Ctrl V · PNG, JPEG, GIF o WebP, hasta 5 MB y 4 imágenes)")}</span>
            <div className="flex flex-wrap items-center gap-2">
              {imagenes.map((im) => (
                <span key={im.url} className="flex h-12 items-center gap-2 rounded-control bg-elevado py-1 pr-1 pl-1">
                  {/* eslint-disable-next-line @next/next/no-img-element -- vista previa local de un blob */}
                  <img src={im.url} alt="" className="size-10 rounded-[6px] object-cover" />
                  <span className="flex max-w-[10rem] flex-col">
                    <span className="truncate text-ayuda" title={im.nombre}>
                      {im.nombre}
                    </span>
                    <span className="dato text-[11px] text-tenue">{kb(im.bytes)}</span>
                  </span>
                  <button type="button" aria-label={t("Quitar {nombre}", { nombre: im.nombre })} onClick={() => quitar(im)} className="grid size-8 place-items-center rounded-control hover:bg-borde">
                    <IconoCerrar className="size-3.5" />
                  </button>
                </span>
              ))}
              <button type="button" onClick={() => archivo.current?.click()} className="h-12 rounded-control border border-dashed border-borde-fuerte px-3 text-rotulo hover:bg-elevado">
                {t("Adjuntar imagen")}
              </button>
              <input
                ref={archivo}
                type="file"
                accept={TIPOS_IMAGEN.join(",")}
                multiple
                hidden
                onChange={(e) => {
                  void sumarArchivos(Array.from(e.target.files ?? []));
                  e.target.value = "";
                }}
              />
              <span role="status" className="text-ayuda text-aviso">
                {aviso}
              </span>
            </div>
          </div>
        )}
      </div>

      <div className="flex items-center gap-2.5 border-t border-borde px-5 py-3.5">
        {enviado ? (
          <>
            <span className="flex-1" />
            <button type="button" onClick={() => dialogo.current?.close()} className="h-9 rounded-control bg-acento px-4 font-medium text-sobre-acento hover:bg-acento-fuerte">
              {t("Listo")}
            </button>
          </>
        ) : (
          <>
            <button type="button" disabled={paso === 0} onClick={() => setPaso((x) => Math.max(0, x - 1))} className="h-9 rounded-control border border-borde-fuerte px-3.5 disabled:opacity-40">
              {t("Volver")}
            </button>
            <span className="dato flex-1 text-ayuda text-tenue">{p ? (enResumen ? (listo ? t("todo respondido") : t("falta algo")) : t("pregunta {n} de {total}", { n: paso + 1, total: vis.length })) : ""}</span>
            {enResumen ? (
              <button
                type="button"
                disabled={!listo || enviando}
                onClick={() => void enviar()}
                className="h-9 min-w-[10rem] rounded-control bg-acento px-4 font-medium text-sobre-acento hover:bg-acento-fuerte disabled:bg-elevado disabled:text-tenue"
              >
                {enviando ? t("Enviando…") : t("Enviar a la sesión")}
              </button>
            ) : (
              <button
                type="button"
                disabled={!q || !respondida(q, b)}
                onClick={() => setPaso((x) => x + 1)}
                className="h-9 min-w-[10rem] rounded-control bg-acento px-4 font-medium text-sobre-acento hover:bg-acento-fuerte disabled:bg-elevado disabled:text-tenue"
              >
                {paso === vis.length - 1 ? t("Revisar") : t("Siguiente")}
              </button>
            )}
          </>
        )}
      </div>
    </dialog>
  );
}
