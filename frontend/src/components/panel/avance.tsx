"use client";

import { useEffect, useState } from "react";
import { HiloFase } from "@/components/ui/hilo-fase";
import { Markdown } from "@/components/ui/markdown";
import { api } from "@/lib/api";
import { COLOR_HILO, horas, horasQueFaltan, paqueteInicial, type RespuestaAvance, tonoEstado, uno } from "@/lib/avance";

const tarjeta = "rounded-tarjeta border border-borde bg-superficie";

const PLANTILLA = `Avance: NN % — qué pasa ahora
Plan: docs/planes/fase-N.md

| Paquete | Qué | Horas | Estado | % | Evidencia |
|---|---|---|---|---|---|
| W1 | … | 3 | hecho | 100 | docs/… |
| W2 | … | 4 | en curso | 50 | … |
| W3 | … | 2 | sin empezar | 0 | |

- 12:25 · W1 cerrado`;

/**
 * El avance por fase: la bitácora del coordinador, docs/trazabilidad/F<n>/AVANCE.md, con el %
 * que calcula Atalaya pesando cada paquete por sus horas, el hilo de la fase y la tabla de paquetes. Solo lee: el
 * archivo lo escribe el coordinador al mover un paquete.
 */
export function Avance({ proyecto, version }: { proyecto: string; version: number }) {
  const [r, setR] = useState<RespuestaAvance | null>(null);
  const [fase, setFase] = useState<string | null>(null);
  const [paquete, setPaquete] = useState<string | null>(null);
  const [doc, setDoc] = useState<string | null>(null);
  const [error, setError] = useState(false);

  useEffect(() => {
    if (!proyecto) return;
    const c = new AbortController();
    api
      .avance(proyecto, c.signal)
      .then((x) => {
        setR(x);
        setError(false);
      })
      .catch((e: unknown) => {
        if (!(e instanceof DOMException && e.name === "AbortError")) setError(true);
      });
    return () => c.abort();
  }, [proyecto, version]);

  const f = r?.fases.find((x) => x.fase === fase) ?? r?.fases[0] ?? null;
  const pq = f ? paqueteInicial(f, paquete) : null;
  const coincide = f && f.declarado >= 0 && Math.abs(f.declarado - f.calculado) < 1;

  return (
    <main id="contenido" tabIndex={-1} aria-label="Avance" className="relative mx-auto flex w-full max-w-[1520px] flex-col gap-5 p-5">
      <div className="flex flex-wrap items-end gap-3">
        <div className="flex min-w-0 flex-[1_1_32rem] flex-col gap-1">
          <h1 className="m-0 text-titulo font-semibold">{f ? `${r?.proyecto}, fase ${f.fase.replace(/^F/, "")}` : `Avance de ${proyecto || "…"}`}</h1>
          <p className="m-0 max-w-3xl text-suave">
            Atalaya lee el archivo de avance de la fase y calcula el % con las horas de cada paquete: la misma cuenta que se hace a mano, pero repetible. Si la
            primera línea no coincide con la tabla, lo avisa.
          </p>
        </div>
        {r && r.fases.length > 1 && (
          <div role="group" aria-label="Fase" className="flex gap-0.5 rounded-control bg-elevado p-[3px]">
            {r.fases.map((x) => {
              const on = x.fase === f?.fase;
              return (
                <button
                  key={x.fase}
                  type="button"
                  aria-pressed={on}
                  onClick={() => {
                    setFase(x.fase);
                    setPaquete(null);
                    setDoc(null); // el AVANCE.md abierto es el de la otra fase
                  }}
                  className={`h-[30px] rounded-chip px-3 text-rotulo whitespace-nowrap ${on ? "bg-superficie font-medium text-texto shadow-sm" : "text-suave hover:text-texto"}`}
                >
                  Fase {x.fase.replace(/^F/, "")}, <span className="dato">{uno(x.calculado)} %</span>
                </button>
              );
            })}
          </div>
        )}
      </div>

      {r === null && !error && <div className="esqueleto h-40 rounded-tarjeta" aria-label="Cargando el avance" />}
      {r === null && error && <p className="m-0 rounded-panel bg-error-suave px-4 py-3 text-error">No se pudo leer el avance. Se reintenta con el próximo aviso.</p>}

      {r && !f && (
        <section className={`flex flex-col gap-3 p-5 ${tarjeta}`}>
          <h2 className="m-0 text-encabezado font-semibold">Este proyecto todavía no tiene su avance por fase</h2>
          <p className="m-0 max-w-3xl text-suave">
            {r.sin} El coordinador lo crea en <span className="dato">docs/trazabilidad/F&lt;n&gt;/AVANCE.md</span> y lo actualiza al mover un paquete, en el mismo
            paso en que actualiza el estado para retomar. Esta es la forma que lee Atalaya:
          </p>
          <pre className="dato m-0 max-w-3xl overflow-x-auto rounded-panel bg-hundido p-3.5 text-[12.5px] leading-relaxed">{PLANTILLA}</pre>
        </section>
      )}

      {f && (
        <>
          <section aria-label="El hilo de la fase" className={`grid grid-cols-1 items-center gap-6 p-5 md:grid-cols-[14rem_minmax(0,1fr)] ${tarjeta} border-borde-fuerte`}>
            <div className="flex flex-col gap-1">
              <span className="text-[56px] leading-none font-semibold tracking-[-0.02em]">{uno(f.calculado)} %</span>
              <span className="text-suave">
                {horas(Math.round(((f.horas * f.calculado) / 100) * 10) / 10)} de {horas(f.horas)} h
              </span>
              <span className={`text-ayuda ${coincide ? "text-ok" : "text-aviso"}`}>
                {f.declarado < 0 ? "la primera línea no dice su %" : coincide ? `coincide con la primera línea (${uno(f.declarado)} %)` : `la primera línea dice ${uno(f.declarado)} %`}
              </span>
            </div>
            <div className="min-w-0">
              <HiloFase fase={f} elegido={pq} alElegir={setPaquete} alto={24} etiquetas />
            </div>
          </section>

          <div className="grid grid-cols-1 items-start gap-5 xl:grid-cols-[minmax(0,1fr)_26rem]">
            <section aria-labelledby="paq-t" className={`flex min-w-0 flex-col gap-0.5 p-4 ${tarjeta}`}>
              <h2 id="paq-t" className="m-0 mb-1.5 text-encabezado font-semibold">
                Paquetes
              </h2>
              <div className="overflow-x-auto">
                <div className="flex min-w-[36rem] flex-col gap-0.5">
                  <span className="grid grid-cols-[3.5rem_minmax(0,1fr)_3.5rem_10rem_3.5rem] gap-3 px-3 pb-1.5 text-ayuda text-tenue">
                    <span>Paquete</span>
                    <span>Qué</span>
                    <span className="text-right">Horas</span>
                    <span>Avance</span>
                    <span className="text-right">%</span>
                  </span>
                  {f.paquetes.map((p) => {
                    const on = pq?.id === p.id;
                    const color = p.estado === "hecho" ? COLOR_HILO.hecho : p.estado === "en curso" ? COLOR_HILO.curso : p.estado === "bloqueado" ? COLOR_HILO.bloqueado : COLOR_HILO.falta;
                    return (
                      <button
                        key={p.id}
                        type="button"
                        aria-pressed={on}
                        onClick={() => setPaquete(p.id)}
                        onFocus={() => setPaquete(p.id)}
                        className={`grid h-[34px] grid-cols-[3.5rem_minmax(0,1fr)_3.5rem_10rem_3.5rem] items-center gap-3 rounded-control px-3 text-left ${on ? "bg-seleccion" : "hover:bg-elevado/60"}`}
                      >
                        <span className="dato font-semibold">{p.id}</span>
                        <span className="truncate text-rotulo" title={p.que}>
                          {p.que || "—"}
                        </span>
                        <span className="dato text-right text-ayuda text-suave">{horas(p.horas)} h</span>
                        <span aria-hidden="true" className="relative h-2 rounded-full bg-elevado">
                          <span className="absolute inset-y-0 left-0 rounded-r-full" style={{ width: `${p.pct}%`, background: color }} />
                        </span>
                        <span className="dato text-right text-ayuda">{p.pct} %</span>
                      </button>
                    );
                  })}
                </div>
              </div>
            </section>

            <div className="flex min-w-0 flex-col gap-5">
              <section aria-live="polite" aria-label="El paquete elegido" className={`flex h-[13.5rem] flex-col gap-2.5 p-5 ${tarjeta} border-borde-fuerte`}>
                {pq ? (
                  <>
                    <span className="flex items-baseline gap-2.5">
                      <span className="dato text-xl font-semibold">{pq.id}</span>
                      <span className="truncate font-semibold" title={pq.que}>
                        {pq.que || "—"}
                      </span>
                    </span>
                    <dl className="m-0 grid grid-cols-[6rem_minmax(0,1fr)] gap-x-3 gap-y-1.5 text-[13px]">
                      <dt className="text-tenue">Estado</dt>
                      <dd className={`m-0 ${tonoEstado(pq.estado).clase}`}>
                        {tonoEstado(pq.estado).palabra}, {pq.pct} %
                      </dd>
                      <dt className="text-tenue">Pesa</dt>
                      <dd className="m-0">
                        {horas(pq.horas)} h de {horas(f.horas)}, el {f.horas > 0 ? uno((pq.horas / f.horas) * 100) : "0,0"} % de la fase
                      </dd>
                      <dt className="text-tenue">Evidencia</dt>
                      <dd className="dato m-0 line-clamp-2 text-ayuda" title={pq.evidencia}>
                        {pq.evidencia ? <ConEnlaces texto={pq.evidencia} alAbrir={setDoc} /> : "—"}
                      </dd>
                    </dl>
                  </>
                ) : (
                  <span className="text-suave">Elige un paquete.</span>
                )}
              </section>

              <section className={`flex flex-col gap-2 p-5 ${tarjeta}`}>
                <h2 className="m-0 text-encabezado font-semibold">Lo que falta</h2>
                <p className="m-0 text-rotulo">
                  Unas <span className="dato">{horas(Math.round(horasQueFaltan(f) * 10) / 10)} h</span> de lo estimado, en{" "}
                  {f.paquetes.filter((p) => p.pct < 100).length} paquetes. Ahora: {f.ahora || "—"}
                </p>
                {f.plan && (
                  <p className="m-0 truncate text-ayuda text-suave" title={f.plan}>
                    Plan: <ConEnlaces texto={f.plan} alAbrir={setDoc} />
                  </p>
                )}
                {f.avisos.length > 0 && (
                  <ul className="m-0 flex list-none flex-col gap-1 p-0">
                    {f.avisos.map((a) => (
                      <li key={a} className="text-ayuda text-aviso">
                        ⚠ {a}
                      </li>
                    ))}
                  </ul>
                )}
                {f.hitos.length > 0 && (
                  <ol className="m-0 flex list-none flex-col gap-1 border-t border-borde p-0 pt-2">
                    {f.hitos.slice(0, 6).map((h, i) => (
                      <li key={i} className="grid grid-cols-[3rem_minmax(0,1fr)] gap-2 text-[12.5px]">
                        <span className="dato text-tenue">{h.hora}</span>
                        <span className="truncate text-suave" title={h.texto}>
                          {h.texto}
                        </span>
                      </li>
                    ))}
                  </ol>
                )}
                <button
                  type="button"
                  onClick={() => setDoc(f.ruta)}
                  className="mt-1 h-9 self-start rounded-control border border-borde-fuerte px-3.5 text-rotulo font-medium hover:bg-elevado"
                >
                  Abrir {f.ruta.split("/").slice(-2).join("/")}
                </button>
              </section>
            </div>
          </div>
        </>
      )}
      {doc && f && <DocumentoAvance proyecto={r?.proyecto ?? proyecto} ruta={doc} version={version} alCerrar={() => setDoc(null)} />}
    </main>
  );
}

// Un texto con rutas de Markdown del proyecto («0e564e0 · docs/idea.md §7e»): cada ruta abre su documento aquí.
const reRutaMd = /((?:[\w.-]+\/)+[\w.-]+\.md)/g;
function ConEnlaces({ texto, alAbrir }: { texto: string; alAbrir: (ruta: string) => void }) {
  return (
    <>
      {texto.split(reRutaMd).map((parte, i) =>
        i % 2 === 1 ? (
          <button key={i} type="button" onClick={() => alAbrir(parte)} className="text-acento underline-offset-2 hover:underline">
            {parte}
          </button>
        ) : (
          parte
        ),
      )}
    </>
  );
}

// El archivo abierto se vuelve a pedir con cada aviso (version): se edita en vivo y no debe quedarse viejo.
function DocumentoAvance({ proyecto, ruta, version, alCerrar }: { proyecto: string; ruta: string; version: number; alCerrar: () => void }) {
  const [texto, setTexto] = useState<string | null>(null);
  useEffect(() => {
    let vigente = true;
    api
      .doc(proyecto, ruta)
      .then((d) => vigente && setTexto(d.texto))
      .catch(() => vigente && setTexto("No se pudo leer el archivo."));
    return () => {
      vigente = false;
    };
  }, [proyecto, ruta, version]);
  return (
    <section aria-label={ruta} className={`flex flex-col gap-3 p-5 ${tarjeta}`}>
      <div className="flex items-center justify-between gap-3">
        <span className="dato truncate text-ayuda text-suave">{ruta}</span>
        <button type="button" onClick={alCerrar} className="h-8 rounded-control border border-borde px-3 text-rotulo hover:bg-elevado">
          Cerrar
        </button>
      </div>
      {texto === null ? <div className="esqueleto h-24 rounded-control" aria-label="Cargando" /> : <Markdown texto={texto} ruta={ruta} />}
    </section>
  );
}
