"use client";

import { type KeyboardEvent, useEffect, useId, useRef, useState } from "react";
import { IconoBuscar } from "@/components/ui/iconos";
import { type Agente, api, type DocumentoDatapedia, esActivo, type ProyectoResumen } from "@/lib/api";
import { FONDO_PROVEEDOR } from "@/lib/proveedor";
import { nombreAgente, titulos } from "@/lib/arbol";
import { fijarPreferencia } from "@/lib/tema";
import type { Seccion } from "./cabecera";

export interface AccionesPaleta {
  abrirAgente: (id: string) => void;
  elegirProyecto: (nombre: string) => void;
  abrirDocumento: (d: DocumentoDatapedia) => void;
  irA: (seccion: Seccion) => void;
}

interface Item {
  clave: string;
  grupo: string;
  etiqueta: string;
  detalle: string;
  color: string;
  hacer: () => void;
}

const normal = (s: string) => s.normalize("NFD").replace(/[̀-ͯ]/g, "").toLowerCase();

/**
 * La paleta de órdenes (Ctrl K, ⌘K en Mac): una sola puerta a todo. Busca entre los agentes del proyecto, los
 * proyectos, la datapedia (documentos y trazas, con su índice FTS) y las acciones (tema, secciones). Es un <dialog>
 * modal: atrapa el foco y Escape lo cierra. El buscador es un «combobox»: las flechas mueven la opción activa e Intro
 * la ejecuta.
 */
export function Paleta({
  abierta,
  alCerrar,
  agentes,
  proyectos,
  proyecto,
  acciones,
}: {
  abierta: boolean;
  alCerrar: () => void;
  agentes: Agente[];
  proyectos: ProyectoResumen[];
  proyecto: string;
  acciones: AccionesPaleta;
}) {
  const base = useId();
  const dialogo = useRef<HTMLDialogElement>(null);
  const [texto, setTexto] = useState("");
  const [activa, setActiva] = useState(0);
  const [docs, setDocs] = useState<DocumentoDatapedia[]>([]);

  useEffect(() => {
    const d = dialogo.current;
    if (!d) return;
    if (abierta && !d.open) d.showModal();
    if (!abierta && d.open) d.close();
  }, [abierta]);

  // La datapedia se consulta 150 ms después de la última tecla; la consulta anterior se cancela.
  const q = texto.trim();
  useEffect(() => {
    if (!abierta || q.length < 2) return;
    const control = new AbortController();
    const espera = setTimeout(() => {
      api
        .buscar({ q, tipo: "", proyecto: "" }, control.signal)
        .then((b) => setDocs(b.resultados.slice(0, 6)))
        .catch(() => {});
    }, 150);
    return () => {
      clearTimeout(espera);
      control.abort();
    };
  }, [q, abierta]);

  const nq = normal(q);
  const cuadra = (...partes: string[]) => !nq || normal(partes.join(" ")).includes(nq);
  const cerrarY = (f: () => void) => () => {
    alCerrar();
    f();
  };

  const items: Item[] = [
    ...agentes
      .filter((a) => cuadra(a.rol, a.clase, a.tarea, a.titulo ?? "", a.tipo === "sesion" ? `${a.proyecto} sesion` : "", a.estado))
      .slice(0, nq ? 6 : 4)
      .map((a) => ({
        clave: `a-${a.id}`,
        grupo: `Agentes de ${proyecto}`,
        etiqueta: `${nombreAgente(a)} · ${titulos(a).alterno}`,
        detalle: a.estado.toLowerCase(),
        color: a.proveedor === "Codex" || esActivo(a.estado) ? FONDO_PROVEEDOR[a.proveedor] : "bg-borde-fuerte",
        hacer: cerrarY(() => acciones.abrirAgente(a.id)),
      })),
    ...proyectos
      .filter((p) => p.nombre !== proyecto && cuadra(p.nombre, "proyecto"))
      .slice(0, 5)
      .map((p) => ({
        clave: `p-${p.nombre}`,
        grupo: "Proyectos",
        etiqueta: p.nombre,
        detalle: p.activos ? `${p.activos} activos` : `${p.agentes} agentes`,
        color: p.activos ? "bg-ok" : "bg-borde-fuerte",
        hacer: cerrarY(() => acciones.elegirProyecto(p.nombre)),
      })),
    ...(nq.length >= 2 ? docs : []).map((d) => ({
      clave: `d-${d.id}`,
      grupo: "Datapedia",
      etiqueta: d.titulo,
      // Una sesión bifurcada copia los turnos de la original: el id corto de la sesión las distingue.
      detalle: d.tipo === "traza" ? `${d.proyecto} · traza · ${d.ruta.split("/")[1]?.slice(0, 4) ?? ""}` : `${d.proyecto} · ${d.tipo}`,
      color: d.tipo === "traza" ? "bg-hilo-1" : "bg-borde-campo",
      hacer: cerrarY(() => acciones.abrirDocumento(d)),
    })),
    ...[
      { e: "Ir a Tus proyectos", d: "sección", f: () => acciones.irA("proyectos") },
      { e: "Ir a las Decisiones", d: "sección", f: () => acciones.irA("decisiones") },
      { e: "Ir a Agentes", d: "sección", f: () => acciones.irA("agentes") },
      { e: "Ir a la Línea de tiempo", d: "sección", f: () => acciones.irA("linea") },
      { e: "Ir al Avance de la fase", d: "sección", f: () => acciones.irA("avance") },
      { e: "Ir a la Salud del proceso", d: "sección", f: () => acciones.irA("salud") },
      { e: "Ir a las Estadísticas", d: "sección", f: () => acciones.irA("estadisticas") },
      { e: "Ir a la Datapedia", d: "sección", f: () => acciones.irA("datapedia") },
      { e: "Ir a las Consultas", d: "sección", f: () => acciones.irA("consultas") },
      { e: "Tema oscuro", d: "tema", f: () => fijarPreferencia("oscuro") },
      { e: "Tema claro", d: "tema", f: () => fijarPreferencia("claro") },
      { e: "Tema del sistema", d: "tema", f: () => fijarPreferencia("sistema") },
    ]
      .filter((x) => cuadra(x.e, x.d))
      .map((x) => ({ clave: `x-${x.e}`, grupo: "Acciones", etiqueta: x.e, detalle: x.d, color: "bg-texto", hacer: cerrarY(x.f) })),
  ];
  const indice = Math.min(activa, Math.max(0, items.length - 1));

  function alTecla(e: KeyboardEvent<HTMLInputElement>) {
    if (e.key === "ArrowDown") setActiva(Math.min(items.length - 1, indice + 1));
    else if (e.key === "ArrowUp") setActiva(Math.max(0, indice - 1));
    else if (e.key === "Enter" && items[indice]) items[indice].hacer();
    else return;
    e.preventDefault();
  }

  // La opción activa siempre a la vista.
  useEffect(() => {
    document.getElementById(`${base}-o${indice}`)?.scrollIntoView({ block: "nearest" });
  }, [base, indice]);

  return (
    <dialog
      ref={dialogo}
      onClose={() => {
        setTexto("");
        setActiva(0);
        alCerrar();
      }}
      onClick={(e) => {
        // Un clic en el velo (fuera de la caja) cierra.
        if (e.target === dialogo.current) alCerrar();
      }}
      aria-label="Paleta de órdenes"
      className="capa mx-auto mt-[12vh] w-[min(40rem,calc(100vw-2rem))] overflow-hidden rounded-tarjeta p-0 text-texto backdrop:bg-velo"
    >
      {abierta && (
        <>
          <div className="flex items-center gap-2.5 border-b border-borde px-3.5">
            <IconoBuscar className="size-4 shrink-0 text-suave" />
            <label htmlFor={`${base}-q`} className="sr-only">
              Buscar agentes, proyectos, documentos o acciones
            </label>
            <input
              id={`${base}-q`}
              autoFocus
              role="combobox"
              aria-expanded="true"
              aria-controls={`${base}-lista`}
              aria-activedescendant={items[indice] ? `${base}-o${indice}` : undefined}
              aria-autocomplete="list"
              value={texto}
              onChange={(e) => {
                setTexto(e.target.value);
                setActiva(0);
              }}
              onKeyDown={alTecla}
              placeholder="Buscar agentes, proyectos, documentos o acciones…"
              autoComplete="off"
              spellCheck={false}
              className="h-13 flex-1 bg-transparent text-[15px] placeholder:text-tenue focus-visible:outline-none"
            />
            <kbd className="dato rounded-chip border border-borde px-1.5 text-ayuda text-tenue">Esc</kbd>
          </div>
          <ul id={`${base}-lista`} role="listbox" aria-label="Resultados" className="scroll-interno m-0 max-h-[26rem] list-none overflow-y-auto p-1.5">
            {items.map((it, i) => (
              <li key={it.clave} role="none">
                {(i === 0 || items[i - 1].grupo !== it.grupo) && (
                  <div role="presentation" className="px-2.5 pt-2.5 pb-1 text-[11px] font-semibold tracking-[0.06em] text-tenue uppercase">
                    {it.grupo}
                  </div>
                )}
                <div
                  id={`${base}-o${i}`}
                  role="option"
                  aria-selected={i === indice}
                  onPointerMove={() => setActiva(i)}
                  onClick={it.hacer}
                  className={`flex min-h-[42px] cursor-pointer items-center gap-2.5 rounded-control px-2.5 ${i === indice ? "bg-elevado" : ""}`}
                >
                  <span aria-hidden="true" className={`size-2 shrink-0 rounded-[2px] ${it.color}`} />
                  <span className="min-w-0 flex-1 truncate whitespace-nowrap" title={it.etiqueta}>
                    {it.etiqueta}
                  </span>
                  <span className="shrink-0 truncate text-ayuda text-tenue">{it.detalle}</span>
                </div>
              </li>
            ))}
          </ul>
          {items.length === 0 && <p className="m-0 px-4 pb-4 text-rotulo text-suave">Nada coincide.</p>}
        </>
      )}
    </dialog>
  );
}
