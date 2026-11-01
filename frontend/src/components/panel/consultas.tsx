"use client";

import { useEffect, useState } from "react";
import { api, type Consulta, type FilaAhorro, type LecturaEvitable, type ResumenConsultas } from "@/lib/api";
import { ancho, cuentas, delPeriodo, diasSeguidos, type Filtro, PASA, topeRedondo } from "@/lib/consultas";
import { diaSiNoHoy, dos, hora, tokens as fmtTokens } from "@/lib/formato";
import { t } from "@/lib/i18n";
import { Fantasmas, FantasmasDeTabla, Paginador, usePaginaViva } from "../ui/paginador";
import { Segmentado } from "../ui/segmentado";

const HERRAMIENTAS: { nombre: string; que: string }[] = [
  { nombre: "que_hizo", que: t("Encargo, estado, tiempos, archivos, últimos pasos y última respuesta de un agente.") },
  { nombre: "buscar", que: t("Pasos de cualquier agente por orden, archivo o resumen.") },
  { nombre: "archivo", que: t("Qué agentes editaron o leyeron un archivo, y si siguen vivos.") },
  { nombre: "salud", que: t("Las reglas rotas ahora, de dónde sale cada umbral, y las faltas del proceso de hoy con su ruta.") },
  { nombre: "estado", que: t("El avance de la fase, calculado desde AVANCE.md: paquetes, % y lo que falta.") },
  { nombre: "retomar", que: t("Lo necesario para retomar en una sola respuesta: avance, agentes vivos, alertas, faltas, temporales, cuota y estado.") },
];

/** Qué lectura reemplaza cada herramienta (servidor/ahorro.go); las demás no tienen una lectura fija que medir. */
const REEMPLAZA: Record<string, string> = {
  estado: t("AVANCE.md y el plan"),
  retomar: t("ESTADO, AVANCE y AGENTS.md"),
  que_hizo: t("bitácora, acta e informe"),
  salud: t("AGENTS.md"),
};

const ROLES: Record<string, string> = {
  sesion: t("Sesión principal"),
  subagente: t("Subagente"),
  codex: "Codex",
  "sin agente": t("Sin agente"),
};

/** El máximo del registro (DiasConsultas en backend/internal/servidor/mcp.go). */
const DIAS = 14;
const PERIODOS = [
  { valor: "1", nombre: t("24 h") },
  { valor: "7", nombre: t("7 días") },
  { valor: "14", nombre: t("14 días") },
];
const FILTROS: { valor: Filtro; nombre: string }[] = [
  { valor: "todas", nombre: t("Todas") },
  { valor: "ahorro", nombre: t("Con ahorro") },
  { valor: "releyo", nombre: t("Releyó igual") },
  { valor: "repetidas", nombre: t("Repetidas") },
  { valor: "fallos", nombre: t("Fallos") },
];

// Los dos colores de los gráficos. Lo evitado lleva --serie-ahorro (un azul que no es ni Claude ni Codex, porque aquí
// el color no nombra a un proveedor; validado en claro y en oscuro); lo respondido, el gris del riel, como «el resto»,
// siempre con su cifra al lado.
const EVITADO = "var(--serie-ahorro)";
const RESPONDIDO = "var(--riel)";

/** ≈ tokens de un texto: bytes ÷ 4, la misma cuenta que el resto del panel. */
const tok = (bytes: number) => Math.round(bytes / 4);
/** La misma consulta aunque el registro crezca por arriba: elegir no salta a otra fila. */
const clave = (c: Consulta) => `${c.hora}|${c.herramienta}|${c.agente ?? c.cliente}`;
const claveLectura = (l: LecturaEvitable) => `${l.hora}|${l.agente}|${l.paso}|${l.ruta}`;
const conSegundos = (f: Date) => `${hora(f)}:${dos(f.getSeconds())}`;
const cuandoCorto = (iso: string) => {
  const f = new Date(iso);
  const d = diaSiNoHoy(f);
  return d ? `${d} ${hora(f)}` : hora(f);
};

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

/** El ahorro de una consulta en la fila del registro: la cifra, «hasta» si es cota, o por qué no hay. */
function ahorroDe(c: Consulta): { texto: string; tenue: boolean } {
  if (c.sin_medir) return { texto: t("sin medir"), tenue: true };
  const n = c.ahorro_tokens ?? 0;
  if (n <= 0) return { texto: "0", tenue: true };
  return { texto: c.ahorro_es_cota ? t("hasta {n}", { n: fmtTokens(n) }) : fmtTokens(n), tenue: false };
}

/** Una consulta para abrir al agente desde una lectura evitable (panel.tsx solo mira agente, proyecto y paso). */
const comoConsulta = (l: LecturaEvitable): Consulta => ({
  hora: l.hora,
  cliente: "",
  herramienta: "",
  argumentos: "",
  bytes: 0,
  ms: 0,
  error: false,
  respuesta: "",
  agente: l.agente,
  nombre: l.nombre,
  proyecto: l.proyecto,
  paso: l.paso,
});

/**
 * Lo que los agentes le preguntan a Atalaya por MCP y lo que eso les ahorró: los tokens que no entraron al contexto
 * (la lectura que la consulta reemplaza menos su respuesta, servidor/ahorro.go), por herramienta, proyecto, rol y día,
 * las lecturas que una consulta habría dado y el registro de cada consulta con su detalle.
 */
export function Consultas({ version, alAbrirAgente }: { version: number; alAbrirAgente: (c: Consulta) => void }) {
  const [consultas, setConsultas] = useState<Consulta[] | null>(null);
  const [resumen, setResumen] = useState<ResumenConsultas | null>(null);
  const [ahora, setAhora] = useState(0);
  const [dias, setDias] = useState(DIAS);
  const [filtro, setFiltro] = useState<Filtro>("todas");
  const [vistaDias, setVistaDias] = useState<"grafico" | "tabla">("grafico");
  const [elegida, setElegida] = useState<string | null>(null);
  const [origen, setOrigen] = useState("http://127.0.0.1:7420");

  useEffect(() => {
    const c = new AbortController();
    Promise.all([api.consultas(c.signal), api.resumenConsultas(dias, c.signal)])
      .then(([lista, r]) => {
        setConsultas(lista ?? []);
        setResumen(r);
        // El «ahora» del resumen: la lista se corta en la misma ventana que el servidor.
        setAhora(Date.parse(r.desde) + r.dias * 86_400_000);
      })
      .catch((e) => {
        if (c.signal.aborted) return;
        console.warn(e);
        setConsultas([]);
      });
    return () => c.abort();
  }, [version, dias]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- la dirección real del panel, solo en el navegador
    setOrigen(window.location.origin.replace("localhost", "127.0.0.1"));
  }, []);

  const periodo = consultas && ahora ? delPeriodo(consultas, dias, ahora) : [];
  const cuenta = cuentas(periodo);
  const lista = periodo.filter(PASA[filtro]);
  const pag = usePaginaViva(lista, clave, { almacen: "consultas", opciones: [10, 25, 50], defecto: 10, reinicio: `${dias}|${filtro}` });
  const sel = lista.find((c) => clave(c) === elegida) ?? pag.visibles[0];
  // Un día más que el periodo: la ventana empieza a esta hora de hace `dias` días, y ese primer día cuenta (corto).
  const porDia = resumen ? diasSeguidos(resumen.por_dia, dias + 1, ahora) : null;
  const cota = periodo.some((c) => c.ahorro_es_cota && (c.ahorro_tokens ?? 0) > 0);

  const mcpJson = `{\n  "mcpServers": {\n    "atalaya": { "type": "http", "url": "${origen}/mcp" }\n  }\n}`;
  const codexToml = `[mcp_servers.atalaya]\nurl = "${origen}/mcp"`;

  return (
    <main id="contenido" tabIndex={-1} aria-label={t("Consultas a Atalaya")} className="relative mx-auto flex w-full max-w-[1520px] min-w-0 flex-col gap-5 p-4 sm:p-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex max-w-3xl min-w-0 flex-col gap-1">
          <h1 className="m-0 text-titulo font-semibold">{t("Lo que los agentes le preguntan a Atalaya")}</h1>
          <p className="m-0 text-suave">
            {t(
              "De todos los proyectos. Claude Code y Codex consultan por MCP en vez de releer bitácoras o buscar en los transcritos. Cada consulta queda en el transcrito del agente, como un paso «Atalaya», y de ahí sale este registro: sobrevive a un reinicio de Atalaya.",
            )}
          </p>
        </div>
        <div className="w-full sm:w-64">
          <Segmentado etiqueta={t("Periodo")} opciones={PERIODOS} valor={String(dias)} alCambiar={(v) => setDias(Number(v))} />
        </div>
      </div>

      <Ahorro total={resumen?.total ?? null} cota={cota} />

      <div className="grid min-w-0 gap-5 xl:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
        <PorHerramienta filas={resumen?.por_herramienta ?? null} cargando={!resumen} />
        <div className="flex min-w-0 flex-col gap-5">
          <Barras titulo={t("Por proyecto")} filas={resumen?.por_proyecto ?? null} cargando={!resumen} nombre={(k) => k} />
          <Barras titulo={t("Por rol")} filas={resumen?.por_rol ?? null} cargando={!resumen} nombre={(k) => ROLES[k] ?? k} />
        </div>
      </div>

      <section aria-label={t("Día a día")} className="flex min-w-0 flex-col gap-3 rounded-tarjeta border border-borde bg-superficie p-4 sm:p-5">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="m-0 text-encabezado font-semibold">{t("Día a día")}</h2>
          <div className="w-40">
            <Segmentado
              etiqueta={t("Ver como")}
              opciones={[
                { valor: "grafico", nombre: t("Gráfico") },
                { valor: "tabla", nombre: t("Tabla") },
              ]}
              valor={vistaDias}
              alCambiar={setVistaDias}
            />
          </div>
        </div>
        {vistaDias === "tabla" ? (
          <TablaDias filas={porDia} />
        ) : (
          <div className="grid min-w-0 gap-5 md:grid-cols-2">
            {/* Una serie por gráfico: consultas y tokens tienen escalas distintas y no comparten eje. */}
            <Columnas
              titulo={t("Consultas por día")}
              filas={porDia}
              valor={(f) => f.n}
              formato={String}
              detalle={(f) => t("{tokens} tokens evitados · {e} errores", { tokens: fmtTokens(f.ahorro_tokens), e: f.errores })}
            />
            <Columnas
              titulo={t("Tokens evitados por día")}
              filas={porDia}
              valor={(f) => f.ahorro_tokens}
              formato={fmtTokens}
              detalle={(f) => t("{n} consultas · {e} errores", { n: f.n, e: f.errores })}
            />
          </div>
        )}
        <p className="m-0 text-ayuda text-tenue">{t("La primera columna es el día en que empieza el periodo: queda corta, desde la misma hora de hoy.")}</p>
      </section>

      <PodriaHaber resumen={resumen} dias={dias} alAbrir={(l) => alAbrirAgente(comoConsulta(l))} />

      <div className="flex min-w-0 flex-col items-stretch gap-5 xl:flex-row xl:items-start">
        <section aria-label={t("Registro de consultas")} className="flex w-full min-w-0 flex-col rounded-tarjeta border border-borde bg-superficie xl:w-[46rem] xl:shrink-0">
          <div className="flex flex-col gap-2.5 px-4 pt-3.5 pb-2">
            <h2 className="m-0 text-encabezado font-semibold">{t("Registro")}</h2>
            <div role="group" aria-label={t("Filtrar el registro")} className="flex flex-wrap gap-1.5">
              {FILTROS.map((f) => {
                const on = f.valor === filtro;
                return (
                  <button
                    key={f.valor}
                    type="button"
                    aria-pressed={on}
                    onClick={() => setFiltro(f.valor)}
                    className={`flex h-7 items-center gap-1.5 rounded-chip border px-2.5 text-rotulo transition-colors duration-150 ${
                      on ? "border-acento bg-acento-suave font-medium text-acento-fuerte" : "border-borde text-suave hover:bg-elevado hover:text-texto"
                    }`}
                  >
                    {f.nombre}
                    <span className="dato text-ayuda tabular-nums">{cuenta[f.valor]}</span>
                  </button>
                );
              })}
            </div>
          </div>
          <div className="min-w-0 overflow-x-auto">
            {/* A 375 px sale la columna del agente (sigue en el detalle): el ahorro queda a la vista sin desplazar. */}
            <table className="w-full table-fixed border-collapse text-rotulo sm:min-w-[34rem]">
              <thead>
                <tr className="text-left text-ayuda text-suave">
                  <th scope="col" className="w-24 px-4 py-2 font-medium">{t("Hora")}</th>
                  <th scope="col" className="hidden w-44 px-2 py-2 font-medium sm:table-cell">{t("Agente")}</th>
                  <th scope="col" className="px-2 py-2 font-medium">{t("Herramienta")}</th>
                  <th scope="col" className="w-32 px-4 py-2 text-right font-medium" title={t("Tokens que no entraron al contexto")}>
                    {t("Ahorro")}
                  </th>
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
                    <td colSpan={4} className="h-14 px-4 py-6 text-suave">
                      {filtro === "todas"
                        ? t("Ninguna consulta en los transcritos de los últimos {dias} días. Aparecen aquí cuando un agente usa las herramientas de Atalaya.", { dias })
                        : t("Ninguna consulta con este filtro en el periodo.")}
                    </td>
                  </tr>
                ) : (
                  <>
                    {pag.visibles.map((c, i) => {
                      const actual = c === sel;
                      const f = new Date(c.hora);
                      const quien = c.nombre ?? c.cliente;
                      const a = ahorroDe(c);
                      const marca = c.releyo ? t("releyó igual") : c.repetida ? t("repetida") : "";
                      return (
                        <tr key={`${clave(c)}-${i}`} className={`h-14 border-t border-borde ${actual ? "bg-seleccion" : ""}`}>
                          <td className="h-14 px-4 py-2 align-middle">
                            <span className="dato flex flex-col leading-tight text-suave tabular-nums">
                              <span>{conSegundos(f)}</span>
                              <span className="h-4 text-ayuda text-tenue">{diaSiNoHoy(f)}</span>
                            </span>
                          </td>
                          <td className="hidden h-14 truncate px-2 py-2 align-middle sm:table-cell" title={quien || t("sin identificar")}>
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
                          <td className="h-14 px-4 py-2 text-right align-middle">
                            <span className="dato flex min-w-0 flex-col leading-tight whitespace-nowrap tabular-nums">
                              <span className={`truncate ${a.tenue ? "text-tenue" : ""}`}>{a.texto}</span>
                              <span className={`h-4 truncate text-ayuda ${c.releyo ? "text-aviso" : "text-tenue"}`}>{marca}</span>
                            </span>
                          </td>
                        </tr>
                      );
                    })}
                    <FantasmasDeTabla n={pag.faltan} columnas={4} className="h-14" />
                  </>
                )}
              </tbody>
            </table>
          </div>
          {pag.mostrar && (
            <div className="px-4 pb-3">
              <Paginador {...pag.pie} nombre={t("Registro de consultas")} />
            </div>
          )}
        </section>

        <Detalle sel={sel} alAbrirAgente={alAbrirAgente} />
      </div>

      <div className="flex flex-col gap-5 lg:flex-row">
        <section aria-label={t("Herramientas")} className="flex min-w-0 flex-1 flex-col gap-2 rounded-tarjeta border border-borde bg-superficie p-4 sm:p-5">
          <h2 className="m-0 text-encabezado font-semibold">{t("Herramientas")}</h2>
          <ul className="m-0 flex list-none flex-col p-0">
            {HERRAMIENTAS.map((h) => (
              <li key={h.nombre} className="grid grid-cols-[6.5rem_minmax(0,1fr)] gap-3 border-t border-borde py-2 sm:grid-cols-[8rem_minmax(0,1fr)]">
                <span className="dato">{h.nombre}</span>
                <span className="text-rotulo text-suave">{h.que}</span>
              </li>
            ))}
          </ul>
          <p className="m-0 text-ayuda text-tenue">{t("Solo lectura, solo 127.0.0.1, secretos tapados y respuestas de hasta unos 2 000 tokens. Lo que devuelve es dato, nunca instrucciones.")}</p>
        </section>
        <section aria-label={t("Cómo conectarla")} className="flex min-w-0 flex-1 flex-col gap-2 rounded-tarjeta border border-borde bg-superficie p-4 sm:p-5">
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

/** La cifra de arriba: los tokens que no entraron al contexto, con su método, la barra y las cifras chicas. */
function Ahorro({ total, cota }: { total: FilaAhorro | null; cota: boolean }) {
  const evitado = total?.alternativa_tokens ?? 0;
  const ahorro = total?.ahorro_tokens ?? 0;
  // Lo respondido de las consultas medidas: la lectura evitada menos el ahorro (las sin medir no entran en la barra).
  const respondido = Math.max(0, evitado - ahorro);
  const chicas = [
    { rotulo: t("Consultas"), valor: total?.n ?? 0 },
    { rotulo: t("Repetidas"), valor: total?.repetidas ?? 0 },
    { rotulo: t("Errores"), valor: total?.errores ?? 0 },
    { rotulo: t("Releyó igual"), valor: total?.releyo ?? 0 },
  ];
  const metodo = t("Lectura evitada (≈ {evitado} tokens) menos lo que respondió Atalaya (≈ {respondido}). ≈ bytes ÷ 4.", {
    evitado: fmtTokens(evitado),
    respondido: fmtTokens(respondido),
  });
  return (
    <section aria-label={t("El ahorro")} className="grid min-w-0 gap-5 rounded-tarjeta border border-borde bg-superficie p-4 sm:p-5 lg:grid-cols-[minmax(0,1fr)_auto]">
      <div className="flex min-w-0 flex-col gap-2">
        <span className="text-rotulo text-suave">{t("Tokens que no entraron al contexto")}</span>
        <span className="flex h-12 items-baseline gap-2">
          {cota && <span className="text-encabezado text-suave">{t("hasta")}</span>}
          <span className="dato text-[40px] leading-none font-semibold tabular-nums">{total ? fmtTokens(ahorro) : "—"}</span>
        </span>
        <p className="m-0 truncate text-ayuda text-suave" title={metodo + (cota ? ` ${t("Hay consultas sin tamaño de respuesta (Codex): su ahorro se cuenta entero y el total es un máximo.")}` : "")}>
          {metodo}
        </p>
        <div className="flex h-3 w-full min-w-0 gap-0.5" aria-hidden="true">
          <span className="h-3 rounded-l-[4px]" style={{ width: `${evitado ? (respondido / evitado) * 100 : 0}%`, background: RESPONDIDO }} />
          <span className="h-3 flex-1 rounded-r-[4px]" style={{ background: evitado ? EVITADO : "var(--elevado)" }} />
        </div>
        <div className="flex flex-wrap gap-x-4 gap-y-1 text-ayuda text-suave">
          <span className="flex items-center gap-1.5">
            <span aria-hidden="true" className="size-2.5 rounded-[2px]" style={{ background: RESPONDIDO }} />
            {t("Respondido")} <span className="dato tabular-nums text-texto">{fmtTokens(respondido)}</span>
          </span>
          <span className="flex items-center gap-1.5">
            <span aria-hidden="true" className="size-2.5 rounded-[2px]" style={{ background: EVITADO }} />
            {t("No entró")} <span className="dato tabular-nums text-texto">{fmtTokens(ahorro)}</span>
          </span>
        </div>
      </div>
      <dl className="m-0 grid grid-cols-2 gap-x-6 gap-y-3 self-center sm:grid-cols-4 lg:grid-cols-2">
        {chicas.map((c) => (
          <div key={c.rotulo} className="flex min-w-[6rem] flex-col">
            <dt className="text-ayuda text-suave">{c.rotulo}</dt>
            <dd className="dato m-0 text-[22px] tabular-nums">{total ? c.valor : "—"}</dd>
          </div>
        ))}
      </dl>
    </section>
  );
}

/** Por herramienta: veces, la respuesta frente a la lectura que evita (dos barras en la misma escala) y el ahorro. */
function PorHerramienta({ filas, cargando }: { filas: FilaAhorro[] | null; cargando: boolean }) {
  const lista = filas ?? [];
  const tope = topeRedondo(Math.max(0, ...lista.map((f) => Math.max(f.alternativa_tokens, tok(f.bytes)))));
  return (
    <section aria-label={t("Por herramienta")} className="flex min-w-0 flex-col gap-2 rounded-tarjeta border border-borde bg-superficie p-4 sm:p-5">
      <h2 className="m-0 text-encabezado font-semibold">{t("Por herramienta")}</h2>
      <div className="flex flex-wrap gap-x-4 text-ayuda text-suave">
        <span className="flex items-center gap-1.5">
          <span aria-hidden="true" className="size-2.5 rounded-[2px]" style={{ background: RESPONDIDO }} />
          {t("Respuesta")}
        </span>
        <span className="flex items-center gap-1.5">
          <span aria-hidden="true" className="size-2.5 rounded-[2px]" style={{ background: EVITADO }} />
          {t("Lectura que evita")}
        </span>
        <span className="dato ml-auto text-tenue tabular-nums">{t("escala: {n} tokens", { n: fmtTokens(tope) })}</span>
      </div>
      {cargando ? (
        <div className="esqueleto h-40 rounded-control" aria-label={t("Cargando")} />
      ) : lista.length === 0 ? (
        <p className="m-0 text-suave">{t("Ninguna consulta en el periodo.")}</p>
      ) : (
        <ul className="m-0 flex list-none flex-col p-0">
          {lista.map((f) => {
            const respuesta = tok(f.bytes);
            const medible = f.alternativa_tokens > 0;
            const reemplaza = REEMPLAZA[f.clave] ?? t("sin lectura medible");
            return (
              <li key={f.clave} className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-x-3 gap-y-1 border-t border-borde py-2 sm:grid-cols-[7rem_3rem_minmax(0,1fr)_5rem]">
                <span className="flex min-w-0 items-baseline gap-2 sm:contents">
                  <span className="dato truncate">{f.clave}</span>
                  <span className="dato text-ayuda text-suave tabular-nums sm:text-right sm:text-rotulo">{t("{n}×", { n: f.n })}</span>
                </span>
                <span className="dato text-right tabular-nums sm:order-last">
                  {medible ? fmtTokens(f.ahorro_tokens) : <span className="text-tenue">—</span>}
                </span>
                <span className="col-span-2 flex min-w-0 flex-col gap-1 sm:col-span-1">
                  <span className="flex h-2 items-center" title={t("Respuesta: ≈ {n} tokens", { n: fmtTokens(respuesta) })}>
                    <span className="h-2 rounded-r-[4px]" style={{ width: `${ancho(respuesta, tope)}%`, background: RESPONDIDO }} />
                  </span>
                  <span className="flex h-2 items-center" title={t("Lectura que evita: ≈ {n} tokens", { n: fmtTokens(f.alternativa_tokens) })}>
                    <span className="h-2 rounded-r-[4px]" style={{ width: `${ancho(f.alternativa_tokens, tope)}%`, background: EVITADO }} />
                  </span>
                  <span className={`truncate text-ayuda ${medible ? "text-suave" : "text-tenue"}`} title={reemplaza}>
                    {medible ? t("en vez de {que}", { que: reemplaza }) : reemplaza}
                  </span>
                </span>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}

/** Barras horizontales simples del ahorro: por proyecto o por rol. */
function Barras({ titulo, filas, cargando, nombre }: { titulo: string; filas: FilaAhorro[] | null; cargando: boolean; nombre: (clave: string) => string }) {
  const lista = [...(filas ?? [])].sort((a, b) => b.ahorro_tokens - a.ahorro_tokens || b.n - a.n);
  const tope = Math.max(1, ...lista.map((f) => f.ahorro_tokens));
  return (
    <section aria-label={titulo} className="flex min-w-0 flex-col gap-2 rounded-tarjeta border border-borde bg-superficie p-4 sm:p-5">
      <h2 className="m-0 text-encabezado font-semibold">{titulo}</h2>
      {cargando ? (
        <div className="esqueleto h-20 rounded-control" aria-label={t("Cargando")} />
      ) : lista.length === 0 ? (
        <p className="m-0 text-suave">{t("Ninguna consulta en el periodo.")}</p>
      ) : (
        <ul className="m-0 flex list-none flex-col gap-1.5 p-0">
          {lista.map((f) => {
            const n = nombre(f.clave);
            return (
              <li key={f.clave} className="grid grid-cols-[minmax(0,8rem)_minmax(0,1fr)_auto] items-center gap-3">
                <span className="truncate text-rotulo" title={n}>
                  {n}
                </span>
                <span className="flex h-3 items-center" title={t("{n} consultas", { n: f.n })}>
                  <span className="h-3 rounded-r-[4px]" style={{ width: `${ancho(f.ahorro_tokens, tope)}%`, background: EVITADO }} />
                </span>
                <span className="dato w-24 text-right text-rotulo tabular-nums">
                  {fmtTokens(f.ahorro_tokens)} <span className="text-ayuda text-tenue">· {f.n}</span>
                </span>
              </li>
            );
          })}
        </ul>
      )}
      <p className="m-0 text-ayuda text-tenue">{t("Tokens evitados · consultas")}</p>
    </section>
  );
}

/** Los mismos días que los gráficos, como tabla: las cifras sin pasar el ratón. Del alto de los gráficos, con su
 * propio desplazamiento, para que cambiar de vista no mueva lo de abajo. */
function TablaDias({ filas }: { filas: FilaAhorro[] | null }) {
  const dia = (k: string) => `${k.slice(8, 10)}/${k.slice(5, 7)}`;
  return (
    <div className="scroll-interno h-48 overflow-auto rounded-panel border border-borde">
      <table className="w-full border-collapse text-rotulo">
        <caption className="sr-only">{t("Consultas y tokens evitados por día")}</caption>
        <thead className="sticky top-0 bg-elevado">
          <tr className="text-left text-suave">
            <th className="px-3 py-1.5 font-medium">{t("Día")}</th>
            <th className="px-2 py-1.5 text-right font-medium">{t("Consultas")}</th>
            <th className="px-2 py-1.5 text-right font-medium">{t("Tokens evitados")}</th>
            <th className="px-3 py-1.5 text-right font-medium">{t("Errores")}</th>
          </tr>
        </thead>
        <tbody>
          {[...(filas ?? [])].reverse().map((f) => (
            <tr key={f.clave} className="border-t border-borde">
              <td className="dato px-3 py-1.5 tabular-nums">{dia(f.clave)}</td>
              <td className="dato px-2 py-1.5 text-right tabular-nums">{f.n}</td>
              <td className="dato px-2 py-1.5 text-right tabular-nums">{fmtTokens(f.ahorro_tokens)}</td>
              <td className="dato px-3 py-1.5 text-right tabular-nums">{f.errores}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/** Un gráfico de columnas de una sola serie por día, con tooltip al pasar el ratón o al llegar con el teclado. */
function Columnas({
  titulo,
  filas,
  valor,
  formato,
  detalle,
}: {
  titulo: string;
  filas: FilaAhorro[] | null;
  valor: (f: FilaAhorro) => number;
  formato: (n: number) => string;
  /** La línea chica del tooltip: lo que el otro gráfico mide ese día, como texto (sin segundo eje). */
  detalle: (f: FilaAhorro) => string;
}) {
  const [activa, setActiva] = useState(-1);
  const lista = filas ?? [];
  const tope = topeRedondo(Math.max(0, ...lista.map(valor)));
  const dia = (k: string) => `${k.slice(8, 10)}/${k.slice(5, 7)}`;
  const f = lista[activa];
  return (
    <figure className="m-0 flex min-w-0 flex-col gap-1.5">
      <figcaption className="flex items-baseline justify-between gap-2 text-rotulo font-medium">
        {titulo}
        <span className="dato text-ayuda font-normal text-tenue tabular-nums">{formato(tope)}</span>
      </figcaption>
      {filas === null ? (
        <div className="esqueleto h-36 rounded-control" aria-label={t("Cargando")} />
      ) : (
        <div className="relative h-36 border-t border-b border-borde" onPointerLeave={() => setActiva(-1)}>
          {/* La raya de la mitad, recesiva. */}
          <span aria-hidden="true" className="absolute inset-x-0 top-1/2 border-t border-dashed border-borde" />
          <ul className="absolute inset-0 m-0 flex list-none items-end gap-0.5 p-0">
            {lista.map((d, i) => {
              const v = valor(d);
              return (
                <li key={d.clave} className="flex h-full min-w-0 flex-1">
                  <button
                    type="button"
                    aria-label={`${dia(d.clave)}: ${formato(v)}`}
                    onPointerEnter={() => setActiva(i)}
                    onFocus={() => setActiva(i)}
                    onBlur={() => setActiva(-1)}
                    className={`flex h-full w-full items-end justify-center rounded-t-[4px] outline-offset-0 ${i === activa ? "bg-elevado/70" : ""}`}
                  >
                    <span className="block w-full max-w-8 rounded-t-[4px]" style={{ height: `${v > 0 ? Math.max(2, (v / tope) * 100) : 0}%`, background: EVITADO }} />
                  </button>
                </li>
              );
            })}
          </ul>
          {f && (
            <div
              role="status"
              className="pointer-events-none absolute top-1 z-10 rounded-control border border-borde bg-superficie px-2.5 py-1.5 text-ayuda whitespace-nowrap shadow-[0_2px_8px_rgb(0_0_0/0.12)]"
              style={{
                left: `${((activa + 0.5) / lista.length) * 100}%`,
                transform: `translateX(${activa < lista.length / 3 ? "0" : activa > (lista.length * 2) / 3 ? "-100%" : "-50%"})`,
              }}
            >
              <span className="block text-suave">{dia(f.clave)}</span>
              <span className="dato block text-rotulo tabular-nums">{formato(valor(f))}</span>
              <span className="dato block text-tenue tabular-nums">{detalle(f)}</span>
            </div>
          )}
        </div>
      )}
      <div className="dato flex justify-between text-ayuda text-tenue tabular-nums" aria-hidden="true">
        <span>{lista[0] ? dia(lista[0].clave) : ""}</span>
        <span>{lista.length ? dia(lista[lista.length - 1].clave) : ""}</span>
      </div>
    </figure>
  );
}

/** Las lecturas de documentos que una consulta habría dado, por páginas; su total es una cota. */
function PodriaHaber({ resumen, dias, alAbrir }: { resumen: ResumenConsultas | null; dias: number; alAbrir: (l: LecturaEvitable) => void }) {
  const lecturas = resumen?.podria_haber_consultado.lecturas ?? [];
  const pag = usePaginaViva(lecturas, claveLectura, { almacen: "consultas-evitables", opciones: [8, 20, 50], defecto: 8, reinicio: String(dias) });
  const tokensCota = resumen?.podria_haber_consultado.tokens ?? 0;
  return (
    <section aria-label={t("Podría haber consultado")} className="flex min-w-0 flex-col gap-2 rounded-tarjeta border border-borde bg-superficie p-4 sm:p-5">
      <div className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1">
        <h2 className="m-0 text-encabezado font-semibold">{t("Podría haber consultado")}</h2>
        <span className="dato text-rotulo text-suave tabular-nums">{resumen ? t("hasta ≈ {n} tokens en {l} lecturas", { n: fmtTokens(tokensCota), l: lecturas.length }) : ""}</span>
      </div>
      <p className="m-0 text-ayuda text-suave">
        {t("Documentos que los agentes leyeron enteros y que una consulta les habría dado. Es una cota: la consulta también responde algo.")}
      </p>
      {!resumen ? (
        <div className="esqueleto h-24 rounded-control" aria-label={t("Cargando")} />
      ) : lecturas.length === 0 ? (
        <p className="m-0 text-suave">{t("Ninguna lectura evitable en el periodo.")}</p>
      ) : (
        <>
          <ul className="m-0 flex list-none flex-col p-0">
            {pag.visibles.map((l) => (
              <li key={claveLectura(l)} className="grid h-12 grid-cols-[4.5rem_minmax(0,1fr)_4rem] items-center gap-3 border-t border-borde sm:grid-cols-[6.5rem_12rem_minmax(0,1fr)_5rem]">
                <span className="dato truncate text-ayuda text-suave tabular-nums">{cuandoCorto(l.hora)}</span>
                <button
                  type="button"
                  onClick={() => alAbrir(l)}
                  title={t("Ver el agente en ese paso")}
                  className="hidden truncate text-left text-rotulo hover:text-acento-fuerte hover:underline sm:block"
                >
                  {l.nombre ?? l.agente}
                </button>
                <span className="flex min-w-0 flex-col">
                  <span className="dato truncate text-rotulo" title={l.ruta}>
                    {l.ruta}
                  </span>
                  <button type="button" onClick={() => alAbrir(l)} className="truncate text-left text-ayuda text-tenue hover:text-acento-fuerte sm:hidden">
                    {l.nombre ?? l.agente}
                  </button>
                </span>
                <span className="dato text-right text-rotulo tabular-nums">{t("{n} tok", { n: fmtTokens(tok(l.bytes)) })}</span>
              </li>
            ))}
            <Fantasmas n={pag.faltan} como="li" className="h-12 border-t border-borde" />
          </ul>
          {pag.mostrar && <Paginador {...pag.pie} nombre={t("Podría haber consultado")} />}
        </>
      )}
    </section>
  );
}

/** La consulta elegida: argumentos, respuesta, tiempo, tamaño y lo que reemplazó, y si el agente releyó igual. */
function Detalle({ sel, alAbrirAgente }: { sel: Consulta | undefined; alAbrirAgente: (c: Consulta) => void }) {
  if (!sel)
    return (
      <section aria-label={t("La consulta elegida")} className="flex w-full min-w-0 flex-1 flex-col gap-3 rounded-tarjeta border border-borde bg-superficie p-4 sm:p-5">
        <p className="m-0 text-suave">{t("Elige una consulta para ver lo que respondió Atalaya.")}</p>
      </section>
    );
  const titulo = `${sel.herramienta}(${sel.argumentos})`;
  const f = new Date(sel.hora);
  const datos = [
    `${diaSiNoHoy(f) ? diaSiNoHoy(f) + " " : ""}${conSegundos(f)}`,
    sel.nombre ?? (sel.cliente || t("agente sin identificar")),
    `${fmtTokens(tok(sel.bytes))} tokens`,
    sel.origen === "transcrito" ? "— ms" : `${sel.ms} ms`,
    ...(sel.nombre && sel.cliente ? [sel.cliente] : []),
  ].join(" · ");
  // El registro guarda los primeros 1 200 caracteres de cada respuesta (mcp.go y claude/consultas.go).
  const recortada = sel.respuesta.endsWith("…");
  const nota = procedencia(sel) + (recortada ? ` ${t("Se muestran los primeros 1 200 caracteres de ≈ {tokens} tokens.", { tokens: fmtTokens(tok(sel.bytes)) })}` : "");
  const evita = sel.evita ?? [];
  const a = ahorroDe(sel);
  return (
    <section aria-label={t("La consulta elegida")} className="flex w-full min-w-0 flex-1 flex-col gap-3 rounded-tarjeta border border-borde bg-superficie p-4 sm:p-5">
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

      {/* Lo que reemplazó, de alto fijo: cambiar de consulta no corre la respuesta de abajo. */}
      <div className="flex h-36 flex-col gap-1.5 rounded-panel border border-borde p-3">
        <div className="flex items-baseline justify-between gap-3">
          <span className="text-rotulo font-medium">{t("Lo que reemplazó")}</span>
          <span className={`dato text-rotulo tabular-nums ${a.tenue ? "text-tenue" : ""}`}>{sel.sin_medir ? a.texto : t("ahorro {n} tok", { n: a.texto })}</span>
        </div>
        {sel.sin_medir || evita.length === 0 ? (
          <p className="m-0 text-ayuda text-suave">{t("Sin lectura medible: esta herramienta no reemplaza un documento fijo.")}</p>
        ) : (
          <ul className="m-0 flex min-h-0 flex-1 list-none flex-col overflow-auto p-0">
            {evita.map((d) => (
              <li key={d.ruta} className="flex h-6 shrink-0 items-center justify-between gap-3 text-ayuda">
                <span className="dato truncate" title={d.ruta}>
                  {d.ruta}
                </span>
                <span className="dato shrink-0 text-suave tabular-nums">{t("{n} tok", { n: fmtTokens(tok(d.bytes)) })}</span>
              </li>
            ))}
          </ul>
        )}
        <p className={`m-0 truncate text-ayuda ${sel.releyo ? "text-aviso" : "text-tenue"}`}>
          {sel.releyo
            ? t("El agente releyó igual alguno de estos documentos: el ahorro no se cuenta.")
            : sel.repetida
              ? t("Repetida: la misma consulta del mismo agente en menos de 5 minutos.")
              : sel.ahorro_es_cota
                ? t("Hasta: la respuesta no tiene tamaño (Codex) y se cuenta entera la lectura evitada.")
                : t("El agente no releyó estos documentos después.")}
        </p>
      </div>

      {sel.respuesta ? (
        <pre className="dato m-0 h-[22rem] overflow-auto rounded-panel bg-hundido p-3.5 text-[12.5px] leading-relaxed whitespace-pre-wrap">{sel.respuesta}</pre>
      ) : (
        <div className="flex h-[22rem] items-center justify-center rounded-panel bg-hundido p-3.5 text-center text-rotulo text-suave">
          <p className="m-0 max-w-md">{sinRespuesta(sel)}</p>
        </div>
      )}
    </section>
  );
}
