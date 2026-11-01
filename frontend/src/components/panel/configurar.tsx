"use client";

import { type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { IconoCheck } from "@/components/ui/iconos";
import {
  api,
  type CarpetaReciente,
  type EntradaCambio,
  type EstadoAvisos,
  type Entorno,
  type Herramienta,
  type HerramientaId,
  type Preferencias,
  type Propuesta,
} from "@/lib/api";
import { diferencia } from "@/lib/diferencia";
import { cuando } from "@/lib/formato";
import { cambiarIdioma, idioma, t, tServidor } from "@/lib/i18n";

// La configuración desde el panel (docs/plan-configuracion.md): el asistente de la primera vez, que después queda como
// esta sección. Detecta antes de preguntar (/api/entorno), cada cambio pasa por la vista previa y se puede deshacer
// (servidor/configurar.go), y cada paso se pone en verde cuando Atalaya ve el resultado.

const tarjeta = "rounded-tarjeta border border-borde bg-superficie";
const botonPrimario =
  "h-(--alto-control-compacto) rounded-control bg-acento px-3 text-rotulo font-medium text-sobre-acento hover:bg-acento-fuerte disabled:opacity-50";
const botonSecundario =
  "h-(--alto-control-compacto) rounded-control border border-borde px-3 text-rotulo text-suave hover:border-borde-realce hover:text-texto disabled:opacity-50";
const CLAVE_PASO = "atalaya:asistente-paso";
/** El asistente se saltó en este navegador: no se abre solo otra vez (panel.tsx). */
export const CLAVE_SALTADO = "atalaya:asistente-saltado";

type Paso = "idioma" | "herramientas" | "coordina" | "proyectos" | "conectar" | "kit" | "avisos" | "privacidad" | "listo";
const PASOS: Paso[] = ["idioma", "herramientas", "coordina", "proyectos", "conectar", "kit", "avisos", "privacidad", "listo"];

const NOMBRE_PASO: Record<Paso, string> = {
  idioma: "Idioma",
  herramientas: "Tus herramientas",
  coordina: "Quién coordina",
  proyectos: "Tus proyectos",
  conectar: "Conectar por MCP",
  kit: "El kit en cada proyecto",
  avisos: "Avisos y arranque",
  privacidad: "Privacidad y datos",
  listo: "Listo",
};

/** Cómo trabaja la persona: quién coordina y quién ayuda. */
type Modo = "claude+codex" | "codex+claude" | "claude" | "codex";
const MODOS: { modo: Modo; principal: HerramientaId; secundario?: HerramientaId; titulo: string; nota: string }[] = [
  { modo: "claude+codex", principal: "claude", secundario: "codex", titulo: "Claude Code coordina, Codex ayuda", nota: "La sesión de Claude Code reparte el trabajo y le pasa a Codex lo que tiene la especificación cerrada." },
  { modo: "codex+claude", principal: "codex", secundario: "claude", titulo: "Codex coordina, Claude Code ayuda", nota: "La sesión de Codex reparte el trabajo y le pasa a Claude Code el diseño y la depuración difícil." },
  { modo: "claude", principal: "claude", titulo: "Solo Claude Code", nota: "Sin Codex: Atalaya sigue a tus sesiones y subagentes de Claude Code." },
  { modo: "codex", principal: "codex", titulo: "Solo Codex", nota: "Sin Claude Code: Atalaya sigue a tus sesiones de Codex, de la terminal y de la app." },
];

const modoDe = (p: Preferencias): Modo | null => MODOS.find((m) => m.principal === p.principal && (m.secundario ?? "") === (p.secundario ?? ""))?.modo ?? null;

/** Lo que propone el asistente si todavía no se eligió: la herramienta más usada, y la otra de ayuda si está. */
function modoSugerido(e: Entorno): Modo {
  const hay = (id: HerramientaId) => e.herramientas.some((h) => h.id === id && (h.instalada || h.datos_existen));
  const principal: HerramientaId = e.principal || (hay("claude") ? "claude" : hay("codex") ? "codex" : "claude");
  const otra: HerramientaId = principal === "claude" ? "codex" : "claude";
  return (hay(otra) ? `${principal}+${otra}` : principal) as Modo;
}

const NIVELES = [
  { nivel: 1, nombre: "1 · Mirar", nota: "Solo lo registra: Atalaya sigue a sus agentes." },
  { nivel: 2, nombre: "2 · Con agentes", nota: "Además, el kit de operación: roles, reglas y avance por fase." },
  { nivel: 3, nombre: "3 · Completo", nota: "Además, entregas por versión y un rol de ensayo." },
];

/** Un cambio esperando confirmación: qué archivo u orden toca, y las líneas que cambian. */
function VistaPrevia({ propuesta, aplicando, alAplicar, alCancelar }: { propuesta: Propuesta; aplicando: boolean; alAplicar: () => void; alCancelar: () => void }) {
  return (
    <div className="flex flex-col gap-3 rounded-panel border border-acento/40 bg-acento-suave/30 p-4">
      <p className="m-0 font-medium">{tServidor(propuesta.resumen)}</p>
      {propuesta.cambios.map((c, i) =>
        c.tipo === "archivo" ? (
          <div key={i} className="flex flex-col gap-1.5">
            <span className="dato truncate text-ayuda text-suave" title={c.ruta}>
              {c.existia ? t("Cambia {ruta}", { ruta: c.ruta ?? "" }) : t("Crea {ruta}", { ruta: c.ruta ?? "" })}
            </span>
            <pre className="dato m-0 max-h-64 overflow-auto rounded-control bg-hundido p-3 text-[12px] leading-relaxed">
              {diferencia(c.antes ?? "", c.despues ?? "").map((l, j) => (
                <div key={j} className={l.signo === "+" ? "text-ok" : l.signo === "−" ? "text-error" : "text-tenue"}>
                  {l.signo} {l.texto}
                </div>
              ))}
            </pre>
          </div>
        ) : (
          <pre key={i} className="dato m-0 overflow-x-auto rounded-control bg-hundido p-3 text-[12px]">
            {c.programa} {(c.argumentos ?? []).join(" ")}
          </pre>
        ),
      )}
      <p className="m-0 text-ayuda text-suave">
        {propuesta.cambios.every((c) => c.tipo === "orden")
          ? t("Atalaya corre esta orden sin intérprete de órdenes; desde el historial de abajo se puede deshacer (corre la orden inversa).")
          : t("Antes de escribir, Atalaya guarda una copia; desde el historial de abajo se puede deshacer.")}
      </p>
      <div className="flex gap-2">
        <button type="button" className={botonPrimario} disabled={aplicando} onClick={alAplicar}>
          {aplicando ? t("Aplicando…") : t("Aplicar")}
        </button>
        <button type="button" className={botonSecundario} disabled={aplicando} onClick={alCancelar}>
          {t("Cancelar")}
        </button>
      </div>
    </div>
  );
}

/** Una orden para copiar, con su botón. */
function Orden({ texto }: { texto: string }) {
  const [copiada, setCopiada] = useState(false);
  return (
    <div className="flex items-center gap-2 rounded-control bg-hundido py-1.5 pr-1.5 pl-3">
      <code className="dato min-w-0 flex-1 truncate text-[12.5px]" title={texto}>
        {texto}
      </code>
      <button
        type="button"
        className="h-7 shrink-0 rounded-chip px-2 text-ayuda text-suave hover:bg-elevado hover:text-texto"
        onClick={() => {
          void navigator.clipboard?.writeText(texto).then(() => {
            setCopiada(true);
            setTimeout(() => setCopiada(false), 1500);
          });
        }}
      >
        {copiada ? t("Copiada") : t("Copiar")}
      </button>
    </div>
  );
}

function Estado({ ok, si, no }: { ok: boolean; si: string; no: string }) {
  return (
    <span className={`flex items-center gap-1.5 text-rotulo ${ok ? "text-ok" : "text-suave"}`}>
      <span aria-hidden="true" className={`size-[7px] rounded-full ${ok ? "bg-ok" : "bg-borde-campo"}`} />
      {ok ? si : no}
    </span>
  );
}

const ENLACE_INSTALAR: Record<HerramientaId, string> = {
  claude: "https://docs.claude.com/en/docs/claude-code/setup",
  codex: "https://developers.openai.com/codex/cli",
};

function TarjetaHerramienta({ h }: { h: Herramienta }) {
  const color = h.id === "claude" ? "var(--claude, #8b5cf6)" : "var(--codex, #38bdf8)";
  return (
    <div className={`flex flex-col gap-2.5 p-4 ${tarjeta}`}>
      <span className="flex items-center gap-2 font-semibold">
        <span aria-hidden="true" className="size-2.5 rounded-full" style={{ background: color }} />
        {h.nombre}
        {h.version && <span className="dato text-ayuda font-normal text-suave">{h.version}</span>}
      </span>
      <Estado ok={h.instalada} si={t("Instalada")} no={t("No está en el PATH")} />
      <Estado ok={h.datos_existen} si={t("Tiene datos en {ruta}", { ruta: h.datos })} no={t("Sin datos todavía en {ruta}", { ruta: h.datos })} />
      <Estado
        ok={h.mcp}
        si={t("Conectada a Atalaya (configuración global)")}
        no={h.id === "claude" ? t("Sin conexión global (puede ir por proyecto, en .mcp.json)") : t("Sin conexión a Atalaya")}
      />
      {!h.instalada && (
        <a className="text-rotulo text-acento hover:underline" href={ENLACE_INSTALAR[h.id]} target="_blank" rel="noreferrer">
          {t("Cómo instalar {nombre}", { nombre: h.nombre })}
        </a>
      )}
    </div>
  );
}

/** Lo que necesita el asistente; fresco pasa por alto la caché de las herramientas («Comprobar de nuevo»). */
function traer(fresco: boolean): Promise<[Entorno, Preferencias, EntradaCambio[]]> {
  const entorno = fresco ? fetch("/api/entorno?fresco=1", { cache: "no-store" }).then((r) => r.json() as Promise<Entorno>) : api.entorno();
  return Promise.all([entorno, api.preferencias(), api.historialCambios()]);
}

export function Configurar({ alTerminar }: { alTerminar: () => void }) {
  const [entorno, setEntorno] = useState<Entorno | null>(null);
  const [prefs, setPrefs] = useState<Preferencias | null>(null);
  const [historial, setHistorial] = useState<EntradaCambio[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [paso, setPaso] = useState<Paso>(() => {
    try {
      const g = localStorage.getItem(CLAVE_PASO) as Paso | null;
      if (g && PASOS.includes(g)) return g;
    } catch {}
    return "idioma";
  });
  const [comprobando, setComprobando] = useState(false);
  const pedido = useRef(0);

  const guardar = useCallback((r: Promise<[Entorno, Preferencias, EntradaCambio[]]>) => {
    const numero = ++pedido.current;
    return r.then(
      ([e, p, h]) => {
        if (numero !== pedido.current) return;
        setEntorno(e);
        setPrefs(p);
        setHistorial(h);
        setError(null);
      },
      (e: unknown) => { if (numero === pedido.current) setError(e instanceof Error ? e.message : String(e)); },
    );
  }, []);
  const cargar = useCallback((fresco = false) => guardar(traer(fresco)), [guardar]);

  // Al abrir y cada 10 s mientras está abierto: lo que se instala, se conecta o aparece (un servidor recién encendido
  // tarda unos segundos en leer los transcritos) se ve sin recargar.
  useEffect(() => {
    const actualizar = () => { if (!document.hidden) void guardar(traer(false)); };
    actualizar();
    const cada = setInterval(actualizar, 10_000);
    document.addEventListener("visibilitychange", actualizar);
    return () => { clearInterval(cada); document.removeEventListener("visibilitychange", actualizar); };
  }, [guardar]);

  useEffect(() => {
    try {
      localStorage.setItem(CLAVE_PASO, paso);
    } catch {}
  }, [paso]);

  const comprobar = async () => {
    setComprobando(true);
    await cargar(true);
    setComprobando(false);
  };

  const hecho = useMemo<Record<Paso, boolean>>(() => {
    const e = entorno;
    return {
      idioma: true,
      herramientas: !!e && e.herramientas.some((h) => h.instalada),
      coordina: !!prefs?.principal,
      proyectos: !!e && e.registrados.length > 0,
      // En verde cuando Atalaya vio llegar una consulta por MCP, no solo cuando la configuración dice que debería.
      conectar: !!e && Object.values(e.clientes ?? {}).some((c) => !!c?.consulta),
      kit: !!e && !!prefs?.principal && e.skills?.[prefs.principal] === "al día",
      avisos: !!e && e.arranque?.activo,
      privacidad: false,
      listo: !!prefs?.asistente,
    };
  }, [entorno, prefs]);

  const i = PASOS.indexOf(paso);
  const ir = (p: Paso) => setPaso(p);

  return (
    <main id="contenido" tabIndex={-1} aria-label={t("Configurar Atalaya")} className="mx-auto flex w-full max-w-[1180px] flex-col gap-5 p-5">
      <div className="flex flex-wrap items-end justify-between gap-3">
        <div className="flex flex-col gap-1.5">
          <h1 className="m-0 text-[22px] font-semibold tracking-[-0.01em]">{t("Configurar Atalaya")}</h1>
          <p className="m-0 max-w-3xl text-suave">
            {t("Atalaya mira primero qué hay en este equipo y solo te pregunta lo que no puede saber. Cada cambio se muestra antes de hacerse y se puede deshacer.")}
          </p>
        </div>
        {!prefs?.asistente && (
          <button
            type="button"
            className={botonSecundario}
            onClick={() => {
              // Saltarlo no es una preferencia: se recuerda en este navegador, para no abrirlo solo otra vez.
              try {
                localStorage.setItem(CLAVE_SALTADO, "1");
              } catch {}
              alTerminar();
            }}
          >
            {t("Saltar por ahora")}
          </button>
        )}
      </div>

      {error && <p className="m-0 rounded-panel bg-error-suave px-4 py-3 text-error">{error}</p>}

      <div className="grid grid-cols-1 items-start gap-5 md:grid-cols-[15rem_minmax(0,1fr)]">
        <ol aria-label={t("Pasos")} className={`m-0 flex list-none flex-col gap-0.5 p-2 ${tarjeta}`}>
          {PASOS.map((p, n) => {
            const on = p === paso;
            return (
              <li key={p}>
                <button
                  type="button"
                  aria-current={on ? "step" : undefined}
                  onClick={() => ir(p)}
                  className={`flex min-h-10 w-full items-center gap-2.5 rounded-control px-2.5 text-left ${on ? "bg-seleccion font-medium" : "text-suave hover:bg-elevado/60 hover:text-texto"}`}
                >
                  <span
                    aria-hidden="true"
                    className={`grid size-5 shrink-0 place-items-center rounded-full text-[11px] ${hecho[p] && p !== "listo" ? "bg-ok text-fondo" : on ? "bg-acento text-sobre-acento" : "border border-borde-campo"}`}
                  >
                    {hecho[p] && p !== "listo" ? <IconoCheck className="size-3" /> : n + 1}
                  </span>
                  {t(NOMBRE_PASO[p])}
                </button>
              </li>
            );
          })}
        </ol>

        <section aria-label={t(NOMBRE_PASO[paso])} className={`flex min-h-[22rem] flex-col gap-4 p-5 ${tarjeta}`}>
          <h2 className="m-0 text-encabezado font-semibold">{t(NOMBRE_PASO[paso])}</h2>
          {!entorno || !prefs ? (
            <div className="esqueleto h-40 rounded-panel" aria-label={t("Cargando")} />
          ) : paso === "idioma" ? (
            <PasoIdioma />
          ) : paso === "herramientas" ? (
            <PasoHerramientas entorno={entorno} comprobando={comprobando} alComprobar={comprobar} />
          ) : paso === "coordina" ? (
            <PasoCoordina entorno={entorno} prefs={prefs} alGuardado={() => cargar()} />
          ) : paso === "proyectos" ? (
            <PasoProyectos entorno={entorno} alCambio={() => cargar()} />
          ) : paso === "conectar" ? (
            <PasoConectar entorno={entorno} prefs={prefs} comprobando={comprobando} alComprobar={comprobar} alCambio={() => cargar(true)} />
          ) : paso === "kit" ? (
            <PasoKit entorno={entorno} prefs={prefs} alCambio={() => cargar()} />
          ) : paso === "avisos" ? (
            <PasoAvisos entorno={entorno} alCambio={() => cargar()} />
          ) : paso === "privacidad" ? (
            <PasoPrivacidad entorno={entorno} prefs={prefs} alCambio={() => cargar()} />
          ) : (
            <PasoListo entorno={entorno} prefs={prefs} alTerminar={alTerminar} alGuardado={() => cargar()} />
          )}
          <div className="mt-auto flex justify-between gap-2 border-t border-borde pt-4">
            <button type="button" className={botonSecundario} disabled={i === 0} onClick={() => ir(PASOS[i - 1])}>
              {t("Anterior")}
            </button>
            {i < PASOS.length - 1 && (
              <button type="button" className={botonPrimario} onClick={() => ir(PASOS[i + 1])}>
                {t("Siguiente: {paso}", { paso: t(NOMBRE_PASO[PASOS[i + 1]]) })}
              </button>
            )}
          </div>
        </section>
      </div>

      <Historial historial={historial} alCambio={() => cargar()} />
    </main>
  );
}

function PasoIdioma() {
  const actual = idioma();
  return (
    <div className="flex flex-col gap-3">
      <p className="m-0 text-suave">{t("El idioma del panel. Se guarda en este navegador y se puede cambiar cuando quieras con el botón ES/EN de la cabecera.")}</p>
      <div className="flex gap-2">
        {(["es", "en"] as const).map((i) => (
          <button
            key={i}
            type="button"
            aria-pressed={actual === i}
            onClick={() => actual !== i && cambiarIdioma(i)}
            className={`h-10 min-w-36 rounded-control border px-4 ${actual === i ? "border-acento bg-acento-suave font-medium text-acento-fuerte" : "border-borde text-suave hover:text-texto"}`}
          >
            {i === "es" ? "Español" : "English"}
          </button>
        ))}
      </div>
    </div>
  );
}

function PasoHerramientas({ entorno, comprobando, alComprobar }: { entorno: Entorno; comprobando: boolean; alComprobar: () => void }) {
  const sistema = entorno.sistema === "windows" ? "Windows" : entorno.sistema === "darwin" ? "macOS" : entorno.sistema === "linux" ? "Linux" : entorno.sistema;
  return (
    <div className="flex flex-col gap-4">
      <p className="m-0 text-suave">
        {t("Atalaya {version} en {sistema}. Esto es lo que encontró de Claude Code y de Codex; con una de las dos alcanza.", { version: entorno.version, sistema })}
      </p>
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        {entorno.herramientas.map((h) => (
          <TarjetaHerramienta key={h.id} h={h} />
        ))}
      </div>
      <p className="m-0 text-ayuda text-suave">
        {t("Para comprobarlo, abre una sesión nueva y pídele: «pregúntale a Atalaya cómo vamos». El paso se pone en verde cuando llega esa consulta.")}
      </p>
      <div>
        <button type="button" className={botonSecundario} disabled={comprobando} onClick={alComprobar}>
          {comprobando ? t("Comprobando…") : t("Comprobar de nuevo")}
        </button>
      </div>
    </div>
  );
}

/** Aplica una propuesta de una vez (para lo que es solo de Atalaya, como las preferencias). */
async function proponerYAplicar(p: Promise<Propuesta>): Promise<EntradaCambio> {
  const prop = await p;
  return api.aplicarCambio(prop.id);
}

function PasoCoordina({ entorno, prefs, alGuardado }: { entorno: Entorno; prefs: Preferencias; alGuardado: () => void }) {
  const guardado = modoDe(prefs);
  const [modo, setModo] = useState<Modo>(guardado ?? modoSugerido(entorno));
  const [estado, setEstado] = useState<{ ok?: string; error?: string; guardando?: boolean }>({});
  const elegido = MODOS.find((m) => m.modo === modo)!;
  const guardar = async () => {
    setEstado({ guardando: true });
    try {
      const e = await proponerYAplicar(api.proponerPreferencias({ ...prefs, principal: elegido.principal, secundario: elegido.secundario }));
      setEstado({ ok: tServidor(e.resumen) });
      alGuardado();
    } catch (err) {
      setEstado({ error: err instanceof Error ? err.message : String(err) });
    }
  };
  return (
    <div className="flex flex-col gap-4">
      <p className="m-0 text-suave">
        {t("Quién reparte el trabajo y quién ayuda. Atalaya pone en la raíz del árbol a la sesión que coordina y, en cada proyecto con agentes, el kit lleva las reglas para ese reparto.")}
      </p>
      <div role="radiogroup" aria-label={t("Quién coordina")} className="grid grid-cols-1 gap-2.5 lg:grid-cols-2">
        {MODOS.map((m) => {
          const on = m.modo === modo;
          return (
            <button
              key={m.modo}
              type="button"
              role="radio"
              aria-checked={on}
              onClick={() => setModo(m.modo)}
              className={`flex flex-col gap-1 rounded-panel border p-3.5 text-left ${on ? "border-acento bg-acento-suave/40" : "border-borde hover:border-borde-realce"}`}
            >
              <span className="font-medium">
                {t(m.titulo)}
                {guardado === m.modo && <span className="ml-2 text-ayuda font-normal text-ok">{t("guardado")}</span>}
              </span>
              <span className="text-ayuda text-suave">{t(m.nota)}</span>
            </button>
          );
        })}
      </div>
      {elegido.principal === "codex" && elegido.secundario && (
        <p className="m-0 text-ayuda text-suave">
          {t("Con Codex de principal, Codex le pasa a Claude Code el diseño y la depuración con scripts/claude.ps1, que corre sin pantalla, solo lee salvo que se le pida escribir y nunca aplica su parche solo. Atalaya lo muestra colgado de la sesión de Codex que lo lanzó.")}
        </p>
      )}
      <div className="flex items-center gap-3">
        <button type="button" className={botonPrimario} disabled={estado.guardando || guardado === modo} onClick={guardar}>
          {estado.guardando ? t("Guardando…") : t("Guardar")}
        </button>
        {estado.ok && <span className="text-rotulo text-ok">{estado.ok}</span>}
        {estado.error && <span className="text-rotulo text-error">{estado.error}</span>}
      </div>
    </div>
  );
}

function PasoProyectos({ entorno, alCambio }: { entorno: Entorno; alCambio: () => void }) {
  const libres = entorno.carpetas.filter((c) => !c.registrada);
  const [elegidas, setElegidas] = useState<Record<string, number>>({});
  const [otra, setOtra] = useState("");
  const [extra, setExtra] = useState<string[]>([]);
  const [quitar, setQuitar] = useState<string[]>([]);
  const [propuesta, setPropuesta] = useState<Propuesta | null>(null);
  const [aplicando, setAplicando] = useState(false);
  const [mensaje, setMensaje] = useState<{ ok?: string; error?: string }>({});

  const alternar = (ruta: string, nivelDefecto: number) =>
    setElegidas((e) => {
      const n = { ...e };
      if (ruta in n) delete n[ruta];
      else n[ruta] = nivelDefecto;
      return n;
    });

  const pedidos = Object.entries(elegidas).map(([ruta, nivel]) => ({ ruta, nivel }));
  const verCambio = async () => {
    setMensaje({});
    try {
      setPropuesta(await api.proponerProyectos({ agregar: pedidos, quitar }));
    } catch (e) {
      setMensaje({ error: e instanceof Error ? e.message : String(e) });
    }
  };
  const aplicar = async () => {
    if (!propuesta) return;
    setAplicando(true);
    try {
      const e = await api.aplicarCambio(propuesta.id);
      setMensaje(e.error ? { error: tServidor(e.error) } : { ok: tServidor(e.resumen) });
      setPropuesta(null);
      setElegidas({});
      setExtra([]);
      setQuitar([]);
      alCambio();
    } catch (e) {
      setMensaje({ error: e instanceof Error ? e.message : String(e) });
    }
    setAplicando(false);
  };

  const fila = (c: Pick<CarpetaReciente, "ruta" | "nombre"> & Partial<CarpetaReciente>) => {
    const on = c.ruta in elegidas;
    const defecto = c.marcas?.agents_md ? 2 : 1;
    return (
      <li key={c.ruta} className={`grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 rounded-control px-2.5 py-2 ${on ? "bg-seleccion" : ""}`}>
        <input type="checkbox" className="size-4 accent-(--acento)" checked={on} onChange={() => alternar(c.ruta, defecto)} aria-label={t("Registrar {nombre}", { nombre: c.nombre })} />
        <span className="flex min-w-0 flex-col">
          <span className="truncate font-medium" title={c.nombre}>
            {c.nombre}
          </span>
          <span className="dato truncate text-ayuda text-tenue" title={c.ruta}>
            {c.ruta}
          </span>
          <span className="flex flex-wrap gap-x-3 text-ayuda text-suave">
            {!!c.claude && <span>{c.claude === 1 ? t("Claude: 1 sesión") : t("Claude: {n} sesiones", { n: c.claude })}</span>}
            {!!c.codex && <span>{c.codex === 1 ? t("Codex: 1 sesión") : t("Codex: {n} sesiones", { n: c.codex })}</span>}
            {c.ultimo && <span>{t("última {cuando}", { cuando: cuando(new Date(c.ultimo)) })}</span>}
            {c.marcas?.agents_md && <span>AGENTS.md</span>}
            {c.marcas?.mcp_atalaya && <span>{t(".mcp.json con Atalaya")}</span>}
          </span>
        </span>
        <select
          aria-label={t("Nivel de {nombre}", { nombre: c.nombre })}
          disabled={!on}
          value={elegidas[c.ruta] ?? defecto}
          onChange={(e) => setElegidas((x) => ({ ...x, [c.ruta]: Number(e.target.value) }))}
          className="h-(--alto-control-compacto) rounded-control border border-borde bg-superficie px-2 text-rotulo disabled:opacity-40"
        >
          {NIVELES.map((n) => (
            <option key={n.nivel} value={n.nivel}>
              {t(n.nombre)}
            </option>
          ))}
        </select>
      </li>
    );
  };

  return (
    <div className="flex flex-col gap-4">
      <p className="m-0 text-suave">
        {t("Atalaya agrupa a los agentes por la carpeta del proyecto. Estas son las carpetas donde trabajaron tus agentes en los últimos 30 días; marca las que quieres seguir.")}
      </p>
      <ul className="m-0 flex list-none flex-col gap-0.5 p-0">
        {libres.length === 0 && extra.length === 0 && <li className="text-suave">{t("No hay carpetas nuevas con agentes recientes.")}</li>}
        {libres.map((c) => fila(c))}
        {extra.map((ruta) => fila({ ruta, nombre: ruta.split(/[\\/]/).filter(Boolean).pop() ?? ruta }))}
      </ul>
      <form
        className="flex gap-2"
        onSubmit={(e) => {
          e.preventDefault();
          const r = otra.trim();
          if (!r || extra.includes(r)) return;
          setExtra((x) => [...x, r]);
          setElegidas((x) => ({ ...x, [r]: 1 }));
          setOtra("");
        }}
      >
        <input
          value={otra}
          onChange={(e) => setOtra(e.target.value)}
          placeholder={t("Otra carpeta (ruta completa, por ejemplo {ejemplo})", { ejemplo: entorno.sistema === "windows" ? "D:\\Proyectos\\Mi-App" : "/home/ana/mi-app" })}
          aria-label={t("Otra carpeta")}
          className="dato h-(--alto-control-compacto) min-w-0 flex-1 rounded-control border border-borde bg-superficie px-3 text-rotulo"
          spellCheck={false}
        />
        <button type="submit" className={botonSecundario}>
          {t("Agregar")}
        </button>
      </form>
      <p className="m-0 text-ayuda text-suave">{NIVELES.map((n) => `${t(n.nombre)}: ${t(n.nota)}`).join(" ")}</p>

      {entorno.registrados.length > 0 && (
        <div className="flex flex-col gap-1.5">
          <h3 className="m-0 text-rotulo font-semibold">{t("Ya registrados")}</h3>
          <ul className="m-0 flex list-none flex-col gap-0.5 p-0">
            {entorno.registrados.map((r) => {
              const sale = quitar.includes(r.nombre);
              return (
                <li key={r.nombre} className="grid grid-cols-[minmax(0,1fr)_auto] items-center gap-3 px-2.5 py-1.5">
                  <span className={`flex min-w-0 flex-col ${sale ? "line-through opacity-60" : ""}`}>
                    <span className="truncate">
                      <span className="dato mr-2 text-ayuda text-suave">{r.sigla}</span>
                      {r.nombre} <span className="text-ayuda text-suave">{t("nivel {n}", { n: r.nivel })}</span>
                      {!r.existe && <span className="ml-2 text-ayuda text-aviso">{t("la carpeta ya no existe")}</span>}
                    </span>
                    <span className="dato truncate text-ayuda text-tenue" title={r.ruta}>
                      {r.ruta}
                    </span>
                  </span>
                  <button type="button" className={botonSecundario} onClick={() => setQuitar((q) => (sale ? q.filter((x) => x !== r.nombre) : [...q, r.nombre]))}>
                    {sale ? t("No quitar") : t("Quitar")}
                  </button>
                </li>
              );
            })}
          </ul>
        </div>
      )}

      {propuesta ? (
        <VistaPrevia propuesta={propuesta} aplicando={aplicando} alAplicar={aplicar} alCancelar={() => setPropuesta(null)} />
      ) : (
        <div className="flex items-center gap-3">
          <button type="button" className={botonPrimario} disabled={pedidos.length === 0 && quitar.length === 0} onClick={verCambio}>
            {t("Ver el cambio")}
          </button>
          {mensaje.ok && <span className="text-rotulo text-ok">{mensaje.ok}</span>}
          {mensaje.error && <span className="text-rotulo text-error">{mensaje.error}</span>}
        </div>
      )}
    </div>
  );
}

/**
 * Una herramienta en el paso de conexión: su estado, el botón para que Atalaya corra la orden oficial (con su vista
 * previa) y la misma orden para copiar, para quien prefiere hacerlo a mano.
 */
function ConexionHerramienta({
  h,
  estado,
  nota,
  orden,
  alCambio,
}: {
  h: Herramienta;
  estado: ReactNode;
  nota: string;
  orden: string;
  alCambio: () => void;
}) {
  const [propuesta, setPropuesta] = useState<Propuesta | null>(null);
  const [aplicando, setAplicando] = useState(false);
  const [mensaje, setMensaje] = useState<{ ok?: string; error?: string }>({});
  const pedir = async (accion: "conectar" | "desconectar") => {
    setMensaje({});
    try {
      setPropuesta(await api.proponerMCP(h.id, accion));
    } catch (e) {
      setMensaje({ error: e instanceof Error ? e.message : String(e) });
    }
  };
  const aplicar = async () => {
    if (!propuesta) return;
    setAplicando(true);
    try {
      const e = await api.aplicarCambio(propuesta.id);
      setMensaje(e.error ? { error: tServidor(e.error) } : { ok: tServidor(e.resumen) });
      setPropuesta(null);
      alCambio();
    } catch (e) {
      setMensaje({ error: e instanceof Error ? e.message : String(e) });
    }
    setAplicando(false);
  };
  return (
    <div className="flex flex-col gap-2">
      <h3 className="m-0 flex items-center gap-3 text-rotulo font-semibold">
        {h.nombre}
        {estado}
      </h3>
      {!h.instalada ? (
        <p className="m-0 text-ayuda text-suave">{t("No está instalada en este equipo.")}</p>
      ) : propuesta ? (
        <VistaPrevia propuesta={propuesta} aplicando={aplicando} alAplicar={aplicar} alCancelar={() => setPropuesta(null)} />
      ) : (
        <div className="flex flex-wrap items-center gap-3">
          {h.mcp ? (
            <button type="button" className={botonSecundario} onClick={() => pedir("desconectar")}>
              {t("Desconectar")}
            </button>
          ) : (
            <button type="button" className={botonPrimario} onClick={() => pedir("conectar")}>
              {t("Conectar por mí")}
            </button>
          )}
          {mensaje.ok && <span className="text-rotulo text-ok">{mensaje.ok}</span>}
          {mensaje.error && <span className="text-rotulo text-error">{mensaje.error}</span>}
        </div>
      )}
      <details className="text-ayuda text-suave">
        <summary className="cursor-pointer select-none hover:text-texto">{t("Prefiero hacerlo yo")}</summary>
        <div className="mt-2 flex flex-col gap-1.5">
          <span>{nota}</span>
          <Orden texto={orden} />
        </div>
      </details>
    </div>
  );
}

function PasoConectar({
  entorno,
  prefs,
  comprobando,
  alComprobar,
  alCambio,
}: {
  entorno: Entorno;
  prefs: Preferencias;
  comprobando: boolean;
  alComprobar: () => void;
  alCambio: () => void;
}) {
  const url = typeof window === "undefined" ? "http://127.0.0.1:7420/mcp" : `${window.location.origin}/mcp`;
  const usa = (id: HerramientaId) => !prefs.principal || prefs.principal === id || prefs.secundario === id;
  const claude = entorno.herramientas.find((h) => h.id === "claude");
  const codex = entorno.herramientas.find((h) => h.id === "codex");
  return (
    <div className="flex flex-col gap-4">
      <p className="m-0 text-suave">
        {t("Por MCP tus agentes le preguntan a Atalaya («¿cómo vamos?», «¿qué hizo el codificador?») en vez de releer documentos. Cada herramienta se conecta una vez; Atalaya lo ve en cuanto llega la conexión.")}
      </p>
      {usa("claude") && claude && (
        <ConexionHerramienta
          h={claude}
          estado={
            <EstadoConexion
              configurada={claude.mcp || entorno.registrados.some((r) => r.marcas.mcp_atalaya)}
              cliente={entorno.clientes?.claude}
              si={claude.mcp ? t("conectada para todos los proyectos") : t("conectada en algunos proyectos (.mcp.json)")}
            />
          }
          nota={t("Para todos tus proyectos, una vez (en un proyecto con el kit, su .mcp.json ya lo hace y además le dice a Atalaya de qué proyecto se trata):")}
          orden={`claude mcp add --transport http --scope user atalaya ${url}`}
          alCambio={alCambio}
        />
      )}
      {usa("codex") && codex && (
        <ConexionHerramienta
          h={codex}
          estado={<EstadoConexion configurada={codex.mcp} cliente={entorno.clientes?.codex} si={t("conectada")} />}
          nota={t("Una vez, para todos tus proyectos:")}
          orden={`codex mcp add atalaya --url ${url}`}
          alCambio={alCambio}
        />
      )}
      <div>
        <button type="button" className={botonSecundario} disabled={comprobando} onClick={alComprobar}>
          {comprobando ? t("Comprobando…") : t("Comprobar de nuevo")}
        </button>
      </div>
    </div>
  );
}

function PasoListo({ entorno, prefs, alTerminar, alGuardado }: { entorno: Entorno; prefs: Preferencias; alTerminar: () => void; alGuardado: () => void }) {
  const [guardando, setGuardando] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const modo = modoDe(prefs);
  const conectadas = entorno.herramientas.filter((h) => h.mcp).map((h) => h.nombre);
  const terminar = async () => {
    setGuardando(true);
    try {
      if (!prefs.asistente && prefs.principal) {
        await proponerYAplicar(api.proponerPreferencias({ ...prefs, asistente: true }));
        alGuardado();
      }
      alTerminar();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
    setGuardando(false);
  };
  const linea = (ok: boolean, texto: string) => (
    <li className="flex items-start gap-2.5">
      <span aria-hidden="true" className={`mt-1.5 size-[7px] shrink-0 rounded-full ${ok ? "bg-ok" : "bg-aviso"}`} />
      {texto}
    </li>
  );
  return (
    <div className="flex flex-col gap-4">
      <ul className="m-0 flex list-none flex-col gap-2 p-0">
        {linea(
          entorno.herramientas.some((h) => h.instalada),
          entorno.herramientas
            .filter((h) => h.instalada)
            .map((h) => `${h.nombre} ${h.version ?? ""}`.trim())
            .join(" · ") || t("Ni Claude Code ni Codex están instalados todavía"),
        )}
        {linea(!!modo, modo ? t(MODOS.find((m) => m.modo === modo)!.titulo) : t("Falta elegir quién coordina"))}
        {linea(
          entorno.registrados.length > 0,
          entorno.registrados.length === 1
            ? t("1 proyecto registrado")
            : entorno.registrados.length > 1
              ? t("{n} proyectos registrados", { n: entorno.registrados.length })
              : t("Ningún proyecto registrado todavía"),
        )}
        {linea(conectadas.length > 0, conectadas.length ? t("Conectadas a Atalaya: {lista}", { lista: conectadas.join(", ") }) : t("Ninguna herramienta conectada por MCP"))}
      </ul>
      {!prefs.principal && <p className="m-0 text-ayuda text-aviso">{t("Elige quién coordina para terminar; lo demás se puede completar después.")}</p>}
      <p className="m-0 text-ayuda text-suave">{t("Todo esto queda en Configuración (el engranaje de la cabecera) para cambiarlo cuando quieras.")}</p>
      <div className="flex items-center gap-3">
        <button type="button" className={botonPrimario} disabled={guardando || !prefs.principal} onClick={terminar}>
          {t("Ir a la portada")}
        </button>
        {error && <span className="text-rotulo text-error">{error}</span>}
      </div>
    </div>
  );
}

/** La conexión de una herramienta: verde cuando Atalaya vio su consulta; si solo está configurada, qué falta. */
function EstadoConexion({ configurada, cliente, si }: { configurada: boolean; cliente?: { conexion?: string; consulta?: string }; si: string }) {
  if (cliente?.consulta) return <Estado ok si={t("consultó a Atalaya {cuando}", { cuando: cuando(new Date(cliente.consulta)) })} no="" />;
  if (cliente?.conexion) return <Estado ok={false} si="" no={t("se conectó {cuando}; falta su primera consulta", { cuando: cuando(new Date(cliente.conexion)) })} />;
  return <Estado ok={false} si="" no={configurada ? t("{estado}; falta su primera consulta", { estado: si }) : t("sin conectar")} />;
}

/**
 * Un cambio por la puerta con su botón: pide la vista previa, la muestra y la aplica. El mismo recorrido que «Conectar
 * por mí», para la skill, el arranque, la datapedia o el diario.
 */
function ConPropuesta({ texto, primario = true, proponer, alCambio, deshabilitado }: { texto: string; primario?: boolean; proponer: () => Promise<Propuesta>; alCambio: () => void; deshabilitado?: boolean }) {
  const [propuesta, setPropuesta] = useState<Propuesta | null>(null);
  const [aplicando, setAplicando] = useState(false);
  const [mensaje, setMensaje] = useState<{ ok?: string; error?: string }>({});
  // Armar la vista previa puede tardar (la skill compara cada archivo del kit): el botón lo dice mientras tanto.
  const [pidiendo, setPidiendo] = useState(false);
  const pedir = async () => {
    setMensaje({});
    setPidiendo(true);
    try {
      setPropuesta(await proponer());
    } catch (e) {
      setMensaje({ error: e instanceof Error ? e.message : String(e) });
    }
    setPidiendo(false);
  };
  const aplicar = async () => {
    if (!propuesta) return;
    setAplicando(true);
    try {
      const e = await api.aplicarCambio(propuesta.id);
      setMensaje(e.error ? { error: tServidor(e.error) } : { ok: tServidor(e.resumen) });
      setPropuesta(null);
      alCambio();
    } catch (e) {
      setMensaje({ error: e instanceof Error ? e.message : String(e) });
    }
    setAplicando(false);
  };
  if (propuesta) return <VistaPrevia propuesta={propuesta} aplicando={aplicando} alAplicar={aplicar} alCancelar={() => setPropuesta(null)} />;
  return (
    <div className="flex flex-wrap items-center gap-3">
      <button type="button" className={primario ? botonPrimario : botonSecundario} disabled={deshabilitado || pidiendo} onClick={pedir}>
        {pidiendo ? t("Preparando la vista previa…") : texto}
      </button>
      {mensaje.ok && <span className="text-rotulo text-ok">{mensaje.ok}</span>}
      {mensaje.error && <span className="text-rotulo text-error">{mensaje.error}</span>}
    </div>
  );
}

const NOMBRE_HERRAMIENTA: Record<HerramientaId, string> = { claude: "Claude Code", codex: "Codex" };

/** El paso 5 del diseño: la skill en cada herramienta y, por proyecto, el pedido para el agente que coordina. */
function PasoKit({ entorno, prefs, alCambio }: { entorno: Entorno; prefs: Preferencias; alCambio: () => void }) {
  const usadas = ([prefs.principal, prefs.secundario].filter(Boolean) as HerramientaId[]).filter((id) => entorno.herramientas.some((h) => h.id === id && h.instalada));
  const principal = prefs.principal ? NOMBRE_HERRAMIENTA[prefs.principal] : "";
  const origen = typeof window === "undefined" ? "http://127.0.0.1:7420" : window.location.origin;
  return (
    <div className="flex flex-col gap-4">
      <p className="m-0 text-suave">
        {t("Atalaya no toca tus proyectos: el kit (reglas, roles, avance por fase) lo instala tu agente con la skill atalaya-proyecto, que funde los textos con lo que el proyecto ya tiene. Aquí Atalaya copia la skill a tu herramienta y te da el pedido para cada proyecto.")}
      </p>
      {!prefs.principal && <p className="m-0 text-ayuda text-aviso">{t("Elige primero quién coordina: la skill va a esa herramienta.")}</p>}
      {usadas.map((id) => {
        const estado = entorno.skills?.[id] ?? "";
        return (
          <div key={id} className="flex flex-col gap-2">
            <h3 className="m-0 flex items-center gap-3 text-rotulo font-semibold">
              {t("La skill en {nombre}", { nombre: NOMBRE_HERRAMIENTA[id] })}
              <Estado ok={estado === "al día"} si={t("al día")} no={estado === "distinta" ? t("otra versión") : estado === "falta" ? t("no está") : t("sin el kit junto al ejecutable")} />
            </h3>
            {(estado === "falta" || estado === "distinta") && (
              <ConPropuesta texto={estado === "falta" ? t("Copiar la skill") : t("Poner la skill al día")} proponer={() => api.proponerSkill(id)} alCambio={alCambio} />
            )}
          </div>
        );
      })}
      <h3 className="m-0 text-rotulo font-semibold">{t("Tus proyectos")}</h3>
      {entorno.registrados.length === 0 ? (
        <p className="m-0 text-ayuda text-suave">{t("Todavía no hay proyectos registrados.")}</p>
      ) : (
        <ul className="m-0 flex list-none flex-col gap-3 p-0">
          {entorno.registrados.map((r) => {
            const listo = r.nivel <= 1 ? r.marcas.agents_md : r.marcas.agents_md && r.marcas.trazabilidad;
            const pedido = t("Usa la skill atalaya-proyecto para preparar el proyecto {ruta} en el nivel {nivel}. Atalaya está en {origen}.", { ruta: r.ruta, nivel: r.nivel, origen });
            return (
              <li key={r.nombre} className="flex flex-col gap-1.5">
                <span className="flex flex-wrap items-center gap-3">
                  <span className="font-medium">{r.nombre}</span>
                  <span className="text-ayuda text-suave">{t("nivel {n}", { n: r.nivel })}</span>
                  <Estado ok={listo} si={t("kit instalado")} no={r.nivel <= 1 ? t("sin AGENTS.md") : t("falta el kit")} />
                </span>
                {!listo && r.existe && (
                  <details className="text-ayuda text-suave">
                    <summary className="cursor-pointer select-none hover:text-texto">{t("El pedido para {agente}", { agente: principal || t("tu agente") })}</summary>
                    <div className="mt-2 flex flex-col gap-1.5">
                      <span>{t("Ábrelo en una sesión nueva en esa carpeta. Atalaya lo pone en verde cuando ve AGENTS.md y el avance de la fase 0.")}</span>
                      <Orden texto={pedido} />
                    </div>
                  </details>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

/** El paso 6: los avisos del sistema y arrancar Atalaya con él. */
function PasoAvisos({ entorno, alCambio }: { entorno: Entorno; alCambio: () => void }) {
  const [avisos, setAvisos] = useState<EstadoAvisos | null>(null);
  const [mensaje, setMensaje] = useState<string | null>(null);
  useEffect(() => {
    api.avisos().then(setAvisos, () => setAvisos(null));
  }, []);
  const cambiar = async (c: { decisiones?: boolean; alertas?: boolean }) => {
    try {
      setAvisos(await api.configurarAvisos(c));
    } catch (e) {
      setMensaje(e instanceof Error ? e.message : String(e));
    }
  };
  const probar = async () => {
    try {
      setAvisos(await api.probarAviso());
      setMensaje(t("Aviso de prueba enviado: debería verse en la esquina de la pantalla."));
    } catch (e) {
      setMensaje(e instanceof Error ? e.message : String(e));
    }
  };
  const a = entorno.arranque;
  return (
    <div className="flex flex-col gap-4">
      <h3 className="m-0 text-rotulo font-semibold">{t("Avisos")}</h3>
      {!avisos?.activos ? (
        <p className="m-0 text-ayuda text-suave">{t("Los avisos del sistema están apagados en este equipo (por ahora solo hay en Windows); lo que te espera se ve igual en la campana del panel.")}</p>
      ) : (
        <div className="flex flex-col gap-2">
          <label className="flex items-center gap-2.5">
            <input type="checkbox" checked={avisos.decisiones} onChange={(e) => void cambiar({ decisiones: e.target.checked })} />
            {t("Cuando un agente te espera con una decisión")}
          </label>
          <label className="flex items-center gap-2.5">
            <input type="checkbox" checked={avisos.alertas} onChange={(e) => void cambiar({ alertas: e.target.checked })} />
            {t("Cuando salta una alerta de error")}
          </label>
          <div>
            <button type="button" className={botonSecundario} onClick={probar}>
              {t("Enviar un aviso de prueba")}
            </button>
          </div>
        </div>
      )}
      {mensaje && <p className="m-0 text-ayuda text-suave">{mensaje}</p>}
      <h3 className="m-0 flex items-center gap-3 text-rotulo font-semibold">
        {t("Arrancar con el sistema")}
        {a?.disponible && <Estado ok={a.activo} si={t("arranca al iniciar sesión")} no={t("no arranca solo")} />}
      </h3>
      {!a?.disponible ? (
        <p className="m-0 text-ayuda text-suave">{t("En este sistema Atalaya no sabe arrancar solo; la guía de instalación dice cómo.")}</p>
      ) : (
        <>
          <p className="m-0 text-ayuda text-suave">
            {entorno.sistema === "windows"
              ? t("Un acceso directo a {programa} en tu carpeta de inicio de Windows.", { programa: a.programa ?? "atalaya.exe" })
              : t("Una unidad de systemd de tu usuario que corre {programa} al iniciar sesión.", { programa: a.programa ?? "atalaya" })}
          </p>
          <ConPropuesta
            texto={a.activo ? t("Que no arranque solo") : t("Arrancar con el sistema")}
            primario={!a.activo}
            proponer={() => api.proponerArranque(!a.activo)}
            alCambio={alCambio}
          />
        </>
      )}
    </div>
  );
}

const DIAS_DIARIO = [30, 90, 365, 0];

/** El paso 7: qué proyectos entran en la datapedia y cuántos días guarda el diario. */
function PasoPrivacidad({ entorno, prefs, alCambio }: { entorno: Entorno; prefs: Preferencias; alCambio: () => void }) {
  const dias = prefs.diario_dias ?? 0;
  // Una sola vista previa a la vez, siempre en el mismo lugar (al pie del paso): pedir otra reemplaza la anterior, y
  // los botones y el selector de días no se mueven (lo que cambia no mueve el diseño).
  const [abierta, setAbierta] = useState<{ quien: string; propuesta: Propuesta } | null>(null);
  const [pidiendo, setPidiendo] = useState<string | null>(null);
  const [aplicando, setAplicando] = useState(false);
  const [mensaje, setMensaje] = useState<{ ok?: string; error?: string }>({});
  const pedir = async (quien: string, proponer: () => Promise<Propuesta>) => {
    setMensaje({});
    setPidiendo(quien);
    try {
      setAbierta({ quien, propuesta: await proponer() });
    } catch (e) {
      setAbierta(null);
      setMensaje({ error: e instanceof Error ? e.message : String(e) });
    }
    setPidiendo(null);
  };
  const aplicar = async () => {
    if (!abierta) return;
    setAplicando(true);
    try {
      const e = await api.aplicarCambio(abierta.propuesta.id);
      setMensaje(e.error ? { error: tServidor(e.error) } : { ok: tServidor(e.resumen) });
      setAbierta(null);
      alCambio();
    } catch (e) {
      setMensaje({ error: e instanceof Error ? e.message : String(e) });
    }
    setAplicando(false);
  };
  const boton = (quien: string, texto: string, proponer: () => Promise<Propuesta>, on: boolean, guardado = false) => (
    <button
      type="button"
      aria-pressed={guardado || on}
      disabled={guardado || pidiendo !== null}
      onClick={() => void pedir(quien, proponer)}
      className={`h-(--alto-control-compacto) min-w-24 rounded-control border px-3 text-rotulo ${
        guardado
          ? "border-transparent bg-seleccion font-medium text-texto"
          : on
            ? "border-acento bg-acento-suave text-acento-fuerte"
            : "border-borde text-suave hover:border-borde-realce hover:text-texto disabled:opacity-60"
      }`}
    >
      {pidiendo === quien ? t("Preparando…") : texto}
    </button>
  );
  return (
    <div className="flex flex-col gap-4">
      <p className="m-0 text-suave">
        {t("Todo queda en este equipo: Atalaya no manda nada afuera. Aquí eliges qué proyectos se indexan para buscar y cuánto recuerda el diario.")}
      </p>
      <h3 className="m-0 text-rotulo font-semibold">{t("La datapedia")}</h3>
      <p className="m-0 text-ayuda text-suave">
        {t("Un proyecto fuera de la datapedia se sigue igual en el panel, pero sus documentos, su memoria y las trazas de sus agentes no se indexan ni salen en las búsquedas.")}
      </p>
      {entorno.registrados.length === 0 ? (
        <p className="m-0 text-ayuda text-suave">{t("Todavía no hay proyectos registrados.")}</p>
      ) : (
        <ul className="m-0 grid list-none grid-cols-[minmax(0,14rem)_minmax(0,12rem)_auto] items-center gap-x-4 gap-y-2 p-0 max-sm:grid-cols-[minmax(0,1fr)_auto]">
          {entorno.registrados.map((r) => (
            <li key={r.nombre} className="contents">
              <span className="truncate font-medium" title={r.nombre}>{r.nombre}</span>
              <span className="max-sm:hidden"><Estado ok={r.datapedia} si={t("en la datapedia")} no={t("fuera de la datapedia")} /></span>
              <span>
                {boton(
                  "dp:" + r.nombre,
                  r.datapedia ? t("Sacar") : t("Volver a indexar"),
                  () => api.proponerProyectos({ datapedia: [{ nombre: r.nombre, dentro: !r.datapedia }] }),
                  abierta?.quien === "dp:" + r.nombre,
                )}
              </span>
            </li>
          ))}
        </ul>
      )}
      <h3 className="m-0 text-rotulo font-semibold">{t("El diario")}</h3>
      <p className="m-0 text-ayuda text-suave">{t("Un archivo por día con lo que hicieron tus agentes, para las estadísticas y la línea de lo que los transcritos ya no tienen.")}</p>
      {!prefs.principal ? (
        <p className="m-0 text-ayuda text-aviso">{t("Elige primero quién coordina: esta preferencia se guarda junto a esa.")}</p>
      ) : (
        <div className="flex flex-wrap items-center gap-2" role="group" aria-label={t("Días que guarda el diario")}>
          {DIAS_DIARIO.map((d) => (
            <span key={d}>
              {boton(
                "dias:" + d,
                d ? t("{n} días", { n: d }) : t("Siempre"),
                () => api.proponerPreferencias({ ...prefs, diario_dias: d }),
                abierta?.quien === "dias:" + d,
                d === dias,
              )}
            </span>
          ))}
        </div>
      )}
      <div aria-live="polite" className="flex flex-col gap-2 border-t border-borde pt-4">
        {abierta ? (
          <VistaPrevia propuesta={abierta.propuesta} aplicando={aplicando} alAplicar={aplicar} alCancelar={() => setAbierta(null)} />
        ) : (
          <p className={`m-0 text-ayuda ${mensaje.error ? "text-error" : mensaje.ok ? "text-ok" : "text-tenue"}`}>
            {mensaje.error ?? mensaje.ok ?? t("Elige un cambio arriba: su vista previa aparece aquí, antes de aplicarlo.")}
          </p>
        )}
      </div>
    </div>
  );
}

function Historial({ historial, alCambio }: { historial: EntradaCambio[]; alCambio: () => void }) {
  const [error, setError] = useState<string | null>(null);
  const [haciendo, setHaciendo] = useState<string | null>(null);
  if (historial.length === 0) return null;
  const deshacer = async (id: string) => {
    setHaciendo(id);
    setError(null);
    try {
      const e = await api.deshacerCambio(id);
      if (e.error) setError(tServidor(e.error));
      alCambio();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    }
    setHaciendo(null);
  };
  return (
    <section aria-labelledby="hist-t" className={`flex flex-col gap-2 p-4 ${tarjeta}`}>
      <h2 id="hist-t" className="m-0 text-encabezado font-semibold">
        {t("Historial de cambios")}
      </h2>
      {error && <p className="m-0 text-rotulo text-error">{error}</p>}
      <ul className="m-0 flex list-none flex-col gap-0.5 p-0">
        {historial.slice(0, 12).map((e) => (
          <li key={e.id} className="grid min-h-10 grid-cols-[5.5rem_minmax(0,1fr)_auto] items-center gap-3 px-1">
            <span className="dato text-ayuda text-suave">{cuando(new Date(e.hora))}</span>
            <span className={`truncate text-rotulo ${e.deshecha ? "line-through opacity-60" : ""}`} title={e.error ? tServidor(e.error) : undefined}>
              {tServidor(e.resumen)}
              {e.error && <span className="ml-2 text-error">{t("falló")}</span>}
            </span>
            {!e.deshace && !e.deshecha ? (
              <button type="button" className={botonSecundario} disabled={haciendo === e.id} onClick={() => deshacer(e.id)}>
                {t("Deshacer")}
              </button>
            ) : (
              <span className="text-ayuda text-tenue">{e.deshecha ? t("deshecho") : ""}</span>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}
