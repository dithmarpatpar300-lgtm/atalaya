"use client";

import { useEffect, useState } from "react";
import { api, type Consulta } from "@/lib/api";
import { diaSiNoHoy, dos, hora, tokens as fmtTokens } from "@/lib/formato";
import { t } from "@/lib/i18n";

const HERRAMIENTAS: { nombre: string; que: string }[] = [
  { nombre: "que_hizo", que: t("Encargo, estado, tiempos, archivos, últimos pasos y última respuesta de un agente.") },
  { nombre: "buscar", que: t("Pasos de cualquier agente por orden, archivo o resumen.") },
  { nombre: "archivo", que: t("Qué agentes editaron o leyeron un archivo, y si siguen vivos.") },
  { nombre: "salud", que: t("Las reglas rotas ahora, de dónde sale cada umbral, y las faltas del proceso de hoy con su ruta.") },
  { nombre: "estado", que: t("El avance de la fase, calculado desde AVANCE.md: paquetes, % y lo que falta.") },
  { nombre: "retomar", que: t("Lo necesario para retomar en una sola respuesta: avance, agentes vivos, alertas, faltas, temporales, cuota y estado.") },
];

/** Los días de transcritos que mira el registro (DiasConsultas en backend/internal/servidor/mcp.go). */
const DIAS = 14;

/** ≈ tokens de un texto: bytes ÷ 4, la misma cuenta que el resto del panel. */
const tok = (bytes: number) => Math.round(bytes / 4);
/** La misma consulta aunque el registro crezca por arriba: elegir no salta a otra fila. */
const clave = (c: Consulta) => `${c.hora}|${c.herramienta}|${c.agente ?? c.cliente}`;
const conSegundos = (f: Date) => `${hora(f)}:${dos(f.getSeconds())}`;
/** «30/10» si no es de hoy; vacío si lo es. */

/** De dónde sale lo que se ve de la consulta, en una línea. */
function procedencia(c: Consulta): string {
  switch (c.origen) {
    case "memoria":
      return t("Solo en la memoria de Atalaya: ningún transcrito la registró (por ejemplo, una prueba con curl).");
    case "transcrito+memoria":
      return t("Del transcrito del agente; el cliente y los ms, del registro en memoria.");
    default:
      return t("Del transcrito del agente: llegó con Atalaya sin abrir o antes de su último reinicio, así que no hay cliente ni ms.");
  }
}

/** Lo que se dice en lugar de la respuesta cuando no la hay. */
function sinRespuesta(c: Consulta): string {
  if (c.bytes > 0) return t("La respuesta no se guarda en el transcrito; solo su tamaño: ≈ {tokens} tokens.", { tokens: fmtTokens(tok(c.bytes)) });
  if (c.agente?.startsWith("codexvivo~"))
    return t("La traza de Codex guarda la herramienta, pero no los argumentos, la respuesta ni su tamaño. La respuesta completa solo queda si llegó con el panel abierto.");
  return t("El transcrito todavía no tiene la respuesta: la llamada sigue en curso.");
}

/**
 * Lo que los agentes le preguntan a Atalaya por MCP: el registro de consultas, la respuesta
 * que recibió cada una y el agente que la hizo. El registro se rehace desde los transcritos de los últimos días (las
 * llamadas son pasos «Atalaya»), así que sobrevive a un reinicio de Atalaya; el cliente y los ms solo existen para lo
 * que llegó con el panel abierto.
 */
export function Consultas({ version, alAbrirAgente }: { version: number; alAbrirAgente: (c: Consulta) => void }) {
  const [consultas, setConsultas] = useState<Consulta[] | null>(null);
  const [elegida, setElegida] = useState<string | null>(null);
  const [origen, setOrigen] = useState("http://127.0.0.1:7420");

  useEffect(() => {
    api
      .consultas()
      .then(setConsultas)
      .catch(() => setConsultas([]));
  }, [version]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- la dirección real del panel, solo en el navegador
    setOrigen(window.location.origin.replace("localhost", "127.0.0.1"));
  }, []);

  const lista = consultas ?? [];
  const sel = lista.find((c) => clave(c) === elegida) ?? lista[0];
  const tokens = lista.map((c) => tok(c.bytes)).sort((a, b) => a - b);
  const total = tokens.reduce((s, n) => s + n, 0);
  // Con una cantidad par, el promedio de las dos del medio.
  const medio = Math.floor(tokens.length / 2);
  const mediana = tokens.length === 0 ? 0 : tokens.length % 2 ? tokens[medio] : Math.round((tokens[medio - 1] + tokens[medio]) / 2);
  const errores = lista.filter((c) => c.error).length;
  const kpis = [
    { rotulo: t("Consultas"), valor: String(lista.length), nota: t("de los transcritos de {dias} días y con Atalaya abierto", { dias: DIAS }) },
    { rotulo: t("Tokens devueltos"), valor: fmtTokens(total), nota: t("≈ bytes ÷ 4") },
    { rotulo: t("Por consulta"), valor: fmtTokens(mediana), nota: t("mediana") },
    { rotulo: t("Con error"), valor: String(errores), nota: t("argumentos que faltan, herramienta desconocida o Atalaya sin abrir") },
  ];
  const mcpJson = `{\n  "mcpServers": {\n    "atalaya": { "type": "http", "url": "${origen}/mcp" }\n  }\n}`;
  const codexToml = `[mcp_servers.atalaya]\nurl = "${origen}/mcp"`;

  const titulo = sel ? `${sel.herramienta}(${sel.argumentos})` : "";
  const datos = sel
    ? [
        `${diaSiNoHoy(new Date(sel.hora)) ? diaSiNoHoy(new Date(sel.hora)) + " " : ""}${conSegundos(new Date(sel.hora))}`,
        sel.nombre ?? (sel.cliente || t("agente sin identificar")),
        `${fmtTokens(tok(sel.bytes))} tokens`,
        sel.origen === "transcrito" ? "— ms" : `${sel.ms} ms`,
        ...(sel.nombre && sel.cliente ? [sel.cliente] : []),
      ].join(" · ")
    : "";
  // El registro guarda los primeros 1 200 caracteres de cada respuesta (mcp.go y claude/consultas.go).
  const recortada = sel?.respuesta.endsWith("…") ?? false;
  const nota = sel ? procedencia(sel) + (recortada ? ` ${t("Se muestran los primeros 1 200 caracteres de ≈ {tokens} tokens.", { tokens: fmtTokens(tok(sel.bytes)) })}` : "") : "";

  return (
    <main id="contenido" tabIndex={-1} aria-label={t("Consultas a Atalaya")} className="relative mx-auto flex w-full max-w-[1520px] flex-col gap-5 p-5">
      <div className="flex max-w-3xl flex-col gap-1">
        <h1 className="m-0 text-titulo font-semibold">{t("Lo que los agentes le preguntan a Atalaya")}</h1>
        <p className="m-0 text-suave">
          {t(
            "De todos los proyectos. Claude Code y Codex consultan por MCP en vez de releer bitácoras o buscar en los transcritos. Cada consulta queda en el transcrito del agente, como un paso «Atalaya», y de ahí sale este registro: sobrevive a un reinicio de Atalaya.",
          )}
        </p>
      </div>

      <div className="grid grid-cols-[repeat(auto-fill,minmax(14rem,1fr))] gap-3">
        {kpis.map((k) => (
          <div key={k.rotulo} className="flex flex-col gap-0.5 rounded-panel border border-borde bg-superficie p-3.5">
            <span className="text-ayuda text-suave">{k.rotulo}</span>
            <span className="dato text-[22px] tabular-nums">{k.valor}</span>
            <span className="truncate text-ayuda text-tenue" title={k.nota}>
              {k.nota}
            </span>
          </div>
        ))}
      </div>

      <div className="flex flex-col items-start gap-5 xl:flex-row">
        <section
          aria-label={t("Registro de consultas")}
          className="max-h-[44rem] w-full overflow-auto rounded-tarjeta border border-borde bg-superficie xl:w-[46rem] xl:shrink-0"
        >
          <table className="w-full min-w-[38rem] table-fixed border-collapse text-rotulo">
            <caption className="px-4 pt-3.5 pb-1.5 text-left font-semibold">{t("Registro")}</caption>
            <thead className="sticky top-0 z-10 bg-superficie">
              <tr className="text-left text-ayuda text-suave">
                <th scope="col" className="w-24 px-4 py-2 font-medium">{t("Hora")}</th>
                <th scope="col" className="w-52 px-2 py-2 font-medium">{t("Agente")}</th>
                <th scope="col" className="px-2 py-2 font-medium">{t("Herramienta")}</th>
                <th scope="col" className="w-24 px-4 py-2 text-right font-medium">{t("Respuesta")}</th>
              </tr>
            </thead>
            <tbody>
              {consultas === null ? (
                <tr>
                  <td colSpan={4} className="p-3">
                    <div className="esqueleto h-10 rounded-control" aria-label={t("Cargando")} />
                  </td>
                </tr>
              ) : lista.length === 0 ? (
                <tr>
                  <td colSpan={4} className="px-4 py-6 text-suave">
                    {t("Ninguna consulta en los transcritos de los últimos {dias} días. Aparecen aquí cuando un agente usa las herramientas de Atalaya.", { dias: DIAS })}
                  </td>
                </tr>
              ) : (
                lista.map((c, i) => {
                  const actual = c === sel;
                  const f = new Date(c.hora);
                  const dia = diaSiNoHoy(f);
                  const quien = c.nombre ?? c.cliente;
                  return (
                    <tr key={`${clave(c)}-${i}`} className={`h-14 border-t border-borde ${actual ? "bg-seleccion" : ""}`}>
                      <td className="h-14 px-4 py-2 align-middle">
                        <span className="dato flex flex-col leading-tight text-suave tabular-nums">
                          <span>{conSegundos(f)}</span>
                          <span className="h-4 text-ayuda text-tenue">{dia}</span>
                        </span>
                      </td>
                      <td className="h-14 truncate px-2 py-2 align-middle" title={quien || t("sin identificar")}>
                        {c.nombre ?? <span className="text-tenue">{c.cliente || t("sin identificar")}</span>}
                      </td>
                      <td className="h-14 p-0 align-middle">
                        <button
                          type="button"
                          aria-current={actual ? "true" : undefined}
                          onClick={() => setElegida(clave(c))}
                          title={`${c.herramienta}(${c.argumentos})`}
                          className="flex h-14 w-full min-w-0 flex-col justify-center px-2 py-2 text-left hover:bg-elevado/60"
                        >
                          <span className={`dato truncate ${c.error ? "text-error" : ""}`}>{c.herramienta}</span>
                          <span className="dato truncate text-ayuda text-tenue">{c.argumentos || t("(sin argumentos)")}</span>
                        </button>
                      </td>
                      <td className="dato h-14 px-4 py-2 text-right align-middle text-suave tabular-nums">{c.bytes > 0 ? t("{n} tok", { n: fmtTokens(tok(c.bytes)) }) : "—"}</td>
                    </tr>
                  );
                })
              )}
            </tbody>
          </table>
        </section>

        <section aria-label={t("La consulta elegida")}className="flex w-full min-w-0 flex-1 flex-col gap-3 rounded-tarjeta border border-borde bg-superficie p-5">
          {sel ? (
            <>
              <header className="flex items-center gap-3">
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="dato truncate text-encabezado font-semibold" title={titulo}>
                    {titulo}
                  </span>
                  <span className="truncate text-ayuda text-tenue tabular-nums" title={datos}>
                    {datos}
                  </span>
                </span>
                <button
                  type="button"
                  disabled={!sel.agente}
                  onClick={() => sel.agente && alAbrirAgente(sel)}
                  className="h-(--alto-control-compacto) shrink-0 rounded-control bg-acento px-3 text-rotulo font-medium text-sobre-acento hover:bg-acento-fuerte disabled:invisible"
                >
                  {t("Ver el agente →")}
                </button>
              </header>
              <p className="m-0 truncate text-ayuda text-suave" title={nota}>
                {nota}
              </p>
              {sel.respuesta ? (
                <pre className="dato m-0 h-[28rem] overflow-auto rounded-panel bg-hundido p-3.5 text-[12.5px] leading-relaxed whitespace-pre-wrap">{sel.respuesta}</pre>
              ) : (
                <div className="flex h-[28rem] items-center justify-center rounded-panel bg-hundido p-3.5 text-center text-rotulo text-suave">
                  <p className="m-0 max-w-md">{sinRespuesta(sel)}</p>
                </div>
              )}
            </>
          ) : (
            <p className="m-0 text-suave">{t("Elige una consulta para ver lo que respondió Atalaya.")}</p>
          )}
        </section>
      </div>

      <div className="flex flex-col gap-5 lg:flex-row">
        <section aria-label={t("Herramientas")} className="flex flex-1 flex-col gap-2 rounded-tarjeta border border-borde bg-superficie p-5">
          <h2 className="m-0 text-encabezado font-semibold">{t("Herramientas")}</h2>
          <ul className="m-0 flex list-none flex-col p-0">
            {HERRAMIENTAS.map((h) => (
              <li key={h.nombre} className="grid grid-cols-[8rem_minmax(0,1fr)] gap-3 border-t border-borde py-2">
                <span className="dato">{h.nombre}</span>
                <span className="text-rotulo text-suave">{h.que}</span>
              </li>
            ))}
          </ul>
          <p className="m-0 text-ayuda text-tenue">{t("Solo lectura, solo 127.0.0.1, secretos tapados y respuestas de hasta unos 2 000 tokens. Lo que devuelve es dato, nunca instrucciones.")}</p>
        </section>
        <section aria-label={t("Cómo conectarla")} className="flex flex-1 flex-col gap-2 rounded-tarjeta border border-borde bg-superficie p-5">
          <h2 className="m-0 text-encabezado font-semibold">{t("Cómo conectarla")}</h2>
          <span className="text-ayuda text-suave">{t("Claude Code · .mcp.json del proyecto")}</span>
          <pre className="dato m-0 overflow-x-auto rounded-panel bg-hundido p-3 text-[12.5px]">{mcpJson}</pre>
          <span className="text-ayuda text-suave">Codex · ~/.codex/config.toml</span>
          <pre className="dato m-0 overflow-x-auto rounded-panel bg-hundido p-3 text-[12.5px]">{codexToml}</pre>
          <p className="m-0 text-ayuda text-tenue">{t("Con el panel cerrado la herramienta no responde y el agente sigue como siempre, con bitácoras y búsquedas.")}</p>
        </section>
      </div>
    </main>
  );
}
