"use client";

import { useCallback, useEffect, useRef, useState, useSyncExternalStore } from "react";
import { Cabecera, type Seccion } from "@/components/cabecera/cabecera";
import { Campana } from "@/components/cabecera/campana";
import { type AccionesPaleta, Paleta } from "@/components/cabecera/paleta";
import { guardarVistas, leerVistas, porAgente } from "@/lib/alertas";
import {
  type Agente,
  api,
  type Detalle,
  type DocumentoDatapedia,
  type EstadoCuotas,
  type EstadoAvisos,
  type ProyectoResumen,
  type RespuestaAlertas,
} from "@/lib/api";
import { t } from "@/lib/i18n";
import { useEventos } from "@/lib/vivo";
import { Actividad } from "./actividad";
import { BandaAlerta } from "./banda-alerta";
import { CLAVE_SALTADO, Configurar } from "./configurar";
import { Consultas } from "./consultas";
import { Avance } from "./avance";
import { Decisiones } from "./decisiones";
import { Estadisticas } from "./estadisticas";
import { Linea } from "./linea";
import { Proyectos } from "./proyectos";
import { Salud } from "./salud";
import { Cuotas } from "./cuotas";
import { Datapedia } from "./datapedia";
import { DetalleAgente } from "./detalle-agente";
import { enFiltro, type Filtro, ListaAgentes } from "./lista-agentes";

const CLAVE = "atalaya:vista";

function leerVista(): { proyecto: string; filtro: Filtro } {
  try {
    const v = JSON.parse(localStorage.getItem(CLAVE) ?? "{}") as { proyecto?: string; filtro?: Filtro };
    return { proyecto: v.proyecto ?? "", filtro: v.filtro === "hechos" || v.filtro === "todos" ? v.filtro : "activos" };
  } catch {
    return { proyecto: "", filtro: "activos" };
  }
}

function guardarVista(v: { proyecto: string; filtro: Filtro }) {
  try {
    localStorage.setItem(CLAVE, JSON.stringify(v));
  } catch {}
}

// La sección va en el ancla de la dirección (#datapedia), así se puede enlazar y el botón Atrás funciona.
function suscribirAncla(f: () => void) {
  window.addEventListener("hashchange", f);
  return () => window.removeEventListener("hashchange", f);
}
const SECCIONES: Seccion[] = ["proyectos", "decisiones", "agentes", "linea", "avance", "salud", "estadisticas", "datapedia", "consultas", "configurar"];
// Sin ancla, la portada: todos los proyectos.
const leerSeccion = (): Seccion => SECCIONES.find((s) => window.location.hash === `#${s}`) ?? "proyectos";

function useSeccion(): Seccion {
  return useSyncExternalStore(suscribirAncla, leerSeccion, () => "proyectos");
}

export function Panel() {
  const [vista, setVista] = useState<{ proyecto: string; filtro: Filtro } | null>(null);
  const [proyectos, setProyectos] = useState<ProyectoResumen[]>([]);
  const primeraVista = useRef(false);
  const [pendientes, setPendientes] = useState(0);
  const [cuotas, setCuotas] = useState<EstadoCuotas | null>(null);
  const [leyendo, setLeyendo] = useState(false);
  const [agentes, setAgentes] = useState<Agente[]>([]);
  const [cargandoLista, setCargandoLista] = useState(true);
  const [elegido, setElegido] = useState<string | null>(null);
  const [detalle, setDetalle] = useState<Detalle | null>(null);
  const [cargandoDetalle, setCargandoDetalle] = useState(false);
  const elegidoRef = useRef<string | null>(null);
  const proyectoRef = useRef("");

  // La vista guardada se lee después de hidratar: al prerenderizar no hay localStorage.
  useEffect(() => {
    const v = leerVista();
    // eslint-disable-next-line react-hooks/set-state-in-effect -- lectura única del almacenamiento del navegador
    setVista(v);
  }, []);

  const cargarCuotas = useCallback(() => {
    api.cuotas().then(setCuotas).catch(() => {});
  }, []);

  const cargarProyectos = useCallback(() => {
    api
      .proyectos()
      .then((ps) => {
        setProyectos(ps);
        // La primera vez (ningún proyecto y el asistente sin terminar ni saltar), se abre Configurar.
        if (!primeraVista.current) {
          primeraVista.current = true;
          let saltado = false;
          try {
            saltado = localStorage.getItem(CLAVE_SALTADO) === "1";
          } catch {}
          if (ps.length === 0 && !window.location.hash && !saltado) {
            api
              .preferencias()
              .then((p) => {
                if (!p.asistente && !window.location.hash) window.location.hash = "configurar";
              })
              .catch(() => {});
          }
        }
        setVista((v) => (v && !v.proyecto && ps.length ? { ...v, proyecto: ps[0].nombre } : v));
      })
      .catch(() => {});
    // Con los proyectos, cuántas decisiones te esperan: el número de la pestaña.
    api
      .decisiones()
      .then((r) => setPendientes(r.pendientes))
      .catch(() => {});
  }, []);

  const cargarDetalle = useCallback((id: string) => {
    api
      .agente(id)
      .then((d) => {
        if (elegidoRef.current === id) setDetalle(d);
      })
      .catch(() => {})
      .finally(() => setCargandoDetalle(false));
  }, []);

  const cargarAgentes = useCallback((proyecto: string) => {
    if (!proyecto) return;
    api
      .agentes(proyecto)
      .then((lista) => {
        if (proyectoRef.current !== proyecto) return;
        setAgentes(lista);
      })
      .catch(() => {})
      .finally(() => setCargandoLista(false));
  }, []);

  // Las alertas de las reglas. Lo visto se guarda en este navegador.
  const [alertas, setAlertas] = useState<RespuestaAlertas>({ reglas: [], alertas: [] });
  const [vistas, setVistas] = useState<Set<string>>(() => new Set());
  const [avisos, setAvisos] = useState<EstadoAvisos | null>(null);

  const cargarAvisos = useCallback(() => {
    api.avisos().then(setAvisos).catch(() => {});
  }, []);

  useEffect(() => {
    /* eslint-disable react-hooks/set-state-in-effect -- lectura única del almacenamiento del navegador */
    setVistas(leerVistas());
    /* eslint-enable react-hooks/set-state-in-effect */
    cargarAvisos();
  }, [cargarAvisos]);

  const cargarAlertas = useCallback((proyecto: string) => {
    if (!proyecto) return;
    api
      .alertas(proyecto)
      .then((r) => {
        if (proyectoRef.current !== proyecto) return;
        setAlertas(r);
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    cargarCuotas();
    cargarProyectos();
  }, [cargarCuotas, cargarProyectos]);

  const proyecto = vista?.proyecto ?? "";
  const filtro = vista?.filtro ?? "activos";
  const seccion = useSeccion();
  // Sube con cada aviso de la datapedia: la sección vuelve a buscar y a pedir el documento abierto.
  const [versionDatapedia, setVersionDatapedia] = useState(0);
  // El turno al que hay que llevar la actividad cuando se llega desde una traza.
  const [irATurno, setIrATurno] = useState<{ agente: string; turno: number; cuando: number } | null>(null);
  // El documento que la datapedia debe abrir cuando se llega desde la paleta.
  const [abrirDoc, setAbrirDoc] = useState<{ id: number; cuando: number } | null>(null);
  const [paleta, setPaleta] = useState(false);
  // El paso al que hay que llevar la actividad (desde «Qué lo llenó» o desde la búsqueda de pasos), de ese agente.
  const [irAPaso, setIrAPaso] = useState<{ agente: string; n: number; cuando: number } | null>(null);
  // Sube con cada consulta nueva a Atalaya por MCP: la sección «Consultas» vuelve a pedir el registro.
  const [versionConsultas, setVersionConsultas] = useState(0);
  // Sube con cada aviso de agentes o de temporales: la sección «Salud» vuelve a pedir el informe.
  const [versionProceso, setVersionProceso] = useState(0);
  const [totalDocs, setTotalDocs] = useState<number | null>(null);

  // Ctrl K (⌘K en Mac) abre la paleta desde cualquier parte, también con el foco en un campo.
  useEffect(() => {
    function alTecla(e: KeyboardEvent) {
      if ((e.ctrlKey || e.metaKey) && !e.altKey && e.key.toLowerCase() === "k") {
        e.preventDefault();
        setPaleta((p) => !p);
      }
    }
    document.addEventListener("keydown", alTecla);
    return () => document.removeEventListener("keydown", alTecla);
  }, []);

  // Cuántos documentos tiene la datapedia, para la pestaña; se vuelve a contar con cada aviso.
  useEffect(() => {
    api
      .buscar({ q: "", tipo: "", proyecto: "" })
      .then((b) => setTotalDocs(b.cuentas.reduce((s, c) => s + c.cantidad, 0)))
      .catch(() => {});
  }, [versionDatapedia]);

  useEffect(() => {
    proyectoRef.current = proyecto;
    if (!proyecto) return;
    guardarVista({ proyecto, filtro });
    cargarAgentes(proyecto);
  }, [proyecto, filtro, cargarAgentes]);

  useEffect(() => {
    cargarAlertas(proyecto);
  }, [proyecto, cargarAlertas]);

  // Si no hay nada elegido (o lo elegido ya no existe), se elige el primero de la lista. Un agente que termina mientras
  // se lo mira sale del filtro «Activos», pero sigue elegido: el cambio de filtro lo decide la persona (`filtrar`).
  useEffect(() => {
    const visibles = agentes.filter((a) => enFiltro(a, filtro));
    if (visibles.length === 0) return;
    if (elegido && agentes.some((a) => a.id === elegido)) return;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- elección por omisión cuando llega la lista
    setElegido(visibles[0].id);
  }, [agentes, filtro, elegido]);

  useEffect(() => {
    elegidoRef.current = elegido;
    if (!elegido) return;
    cargarDetalle(elegido);
  }, [elegido, cargarDetalle]);

  // Los avisos en vivo: cada uno vuelve a pedir solo lo que cambió, con un respiro de 400 ms para juntar ráfagas.
  const pendiente = useRef<ReturnType<typeof setTimeout> | null>(null);
  const conexion = useEventos((e) => {
    if (e.tipo === "cuotas") {
      cargarCuotas();
      return;
    }
    if (e.tipo === "datapedia") {
      setVersionDatapedia((v) => v + 1);
      return;
    }
    if (e.tipo === "alertas") {
      cargarAlertas(proyectoRef.current);
      return;
    }
    if (e.tipo === "consultas") {
      setVersionConsultas((v) => v + 1);
      return;
    }
    if (e.tipo === "proceso") {
      setVersionProceso((v) => v + 1);
      return;
    }
    // Una pregunta nueva por Atalaya, o respondida: la bandeja y el número de la pestaña, al instante.
    if (e.tipo === "decisiones") {
      setVersionProceso((v) => v + 1);
      cargarProyectos();
      return;
    }
    if (e.tipo === "avisos") {
      cargarAvisos();
      return;
    }
    if (e.tipo === "reconectado") setVersionDatapedia((v) => v + 1);
    const ids = e.tipo === "agentes" ? e.ids : undefined;
    if (pendiente.current) clearTimeout(pendiente.current);
    pendiente.current = setTimeout(() => {
      if (e.tipo === "reconectado") {
        cargarCuotas();
        cargarAlertas(proyectoRef.current);
        cargarAvisos();
      }
      cargarAgentes(proyectoRef.current);
      cargarProyectos();
      setVersionProceso((v) => v + 1);
      const id = elegidoRef.current;
      if (id && (!ids || ids.length === 0 || ids.includes(id) || id.startsWith("codex~"))) cargarDetalle(id);
    }, 400);
  });

  function elegirProyecto(p: string) {
    setAgentes([]);
    setCargandoLista(true);
    setElegido(null);
    setDetalle(null);
    setVista((v) => ({ filtro: v?.filtro ?? "activos", proyecto: p }));
  }

  function elegirAgente(id: string) {
    if (id === elegido) return;
    setCargandoDetalle(true);
    setDetalle(null);
    setElegido(id);
  }

  // La persona cambia el filtro: si lo elegido queda fuera, pasa al primero que se ve.
  function filtrar(f: Filtro) {
    setVista((v) => ({ proyecto: v?.proyecto ?? "", filtro: f }));
    const visibles = agentes.filter((a) => enFiltro(a, f));
    if (visibles.length && !visibles.some((a) => a.id === elegido)) elegirAgente(visibles[0].id);
  }

  // La única forma de abrir un agente desde cualquier sección: cambia de proyecto si es de otro, pasa a «Todos» (puede
  // haber entregado ya y quedar fuera de «Activos») y lleva la actividad al paso o al turno pedido.
  function abrirAgente(o: { id: string; proyecto?: string; paso?: number; turno?: number }) {
    const p = o.proyecto || proyecto;
    if (p !== proyecto) {
      setAgentes([]);
      setCargandoLista(true);
    }
    setVista({ proyecto: p, filtro: "todos" });
    if (o.paso !== undefined) setIrAPaso({ agente: o.id, n: o.paso, cuando: Date.now() });
    if (o.turno !== undefined) setIrATurno({ agente: o.id, turno: o.turno, cuando: Date.now() });
    if (o.id !== elegido) {
      setCargandoDetalle(true);
      setDetalle(null);
      setElegido(o.id);
    }
    window.location.hash = "agentes";
    window.scrollTo({ top: 0 });
  }

  // Desde un «Delega», una alerta o la paleta: del proyecto elegido.
  const abrirSubagente = (id: string) => abrirAgente({ id });
  const abrirDesdeAlerta = abrirSubagente;

  function marcarVistas(ids: string[]) {
    setVistas((v) => {
      const n = new Set(v);
      ids.forEach((id) => n.add(id));
      guardarVistas(n);
      return n;
    });
  }

  const banda = alertas.alertas.find((a) => a.nivel === "error" && !vistas.has(a.id));

  // Desde una traza de la datapedia (su turno) o desde la búsqueda de pasos (su paso).
  const abrirTraza = (proyectoTraza: string, agente: string, turno: number) => abrirAgente({ id: agente, proyecto: proyectoTraza, turno });
  const abrirPaso = (proyectoPaso: string, agente: string, n: number) => abrirAgente({ id: agente, proyecto: proyectoPaso, paso: n });

  const accionesPaleta: AccionesPaleta = {
    abrirAgente: abrirSubagente,
    elegirProyecto: elegirProyecto,
    abrirDocumento: (d: DocumentoDatapedia) => {
      // Una traza lleva al agente y su turno; un documento, a la datapedia.
      const m = /^trazas\/(.+)\/turno-(\d+)$/.exec(d.ruta);
      if (m) {
        abrirTraza(d.proyecto, m[1], Number(m[2]));
        return;
      }
      setAbrirDoc({ id: d.id, cuando: Date.now() });
      window.location.hash = "datapedia";
    },
    irA: (s) => {
      window.location.hash = s;
    },
  };

  const visible = detalle && detalle.agente.id === elegido ? detalle : null;

  // «Ver la traza»: el último turno del agente, abierto en la datapedia.
  function verTraza(a: Agente) {
    const turno = Math.max(1, ...(visible?.pasos ?? []).map((p) => p.turno));
    api
      .documentoPorRuta(a.proyecto, `trazas/${a.id}/turno-${turno}`)
      .then((d) => {
        setAbrirDoc({ id: d.id, cuando: Date.now() });
        window.location.hash = "datapedia";
      })
      .catch(() => {
        window.location.hash = "datapedia";
      });
  }

  function leerCuotas() {
    setLeyendo(true);
    api
      .cuotas(true)
      .then(setCuotas)
      .catch(() => {})
      .finally(() => setLeyendo(false));
  }

  return (
    <div className="relative flex min-h-screen flex-col overflow-x-clip">
      {/* El halo ambiental (de Camaleón): un resplandor ciruela muy tenue arriba a la izquierda, solo en oscuro. */}
      <div
        aria-hidden="true"
        className="pointer-events-none absolute -top-44 -left-36 hidden size-[35rem] rounded-full bg-[radial-gradient(closest-side,rgb(167_139_250/0.16),rgb(167_139_250/0)_72%)] blur-[30px] oscuro:block"
      />
      {/* Para el teclado: el primer Tab ofrece saltar la cabecera. No es un enlace con ancla: el ancla elige la sección. */}
      <button
        type="button"
        onClick={() => document.getElementById("contenido")?.focus()}
        className="fixed top-2 left-2 z-50 -translate-y-16 rounded-control bg-acento px-3 py-2 font-medium text-sobre-acento focus:translate-y-0"
      >
        {t("Saltar al contenido")}
      </button>
      <Cabecera
        proyectos={proyectos}
        proyecto={proyecto}
        alElegirProyecto={elegirProyecto}
        conexion={conexion}
        seccion={seccion}
        cuentas={{ proyectos: null, decisiones: pendientes, agentes: proyecto ? agentes.filter((a) => enFiltro(a, filtro)).length : null, linea: null, avance: null, salud: null, estadisticas: null, datapedia: totalDocs, consultas: null, configurar: null }}
        alAbrirPaleta={() => setPaleta(true)}
        campana={
          <Campana
            alertas={alertas.alertas}
            reglas={alertas.reglas}
            vistas={vistas}
            avisos={avisos}
            alMarcarVista={(id) => marcarVistas([id])}
            alMarcarTodas={() => marcarVistas(alertas.alertas.map((a) => a.id))}
            alCambiarAvisos={async (c) => setAvisos(await api.configurarAvisos(c))}
            alProbarAviso={async () => setAvisos(await api.probarAviso())}
            alAbrirAgente={abrirDesdeAlerta}
          />
        }
      />
      {banda && <BandaAlerta key={banda.id} alerta={banda} reglas={alertas.reglas} alVer={() => abrirDesdeAlerta(banda.agente)} alVisto={() => marcarVistas([banda.id])} />}
      <Paleta abierta={paleta} alCerrar={() => setPaleta(false)} agentes={agentes} proyectos={proyectos} proyecto={proyecto} acciones={accionesPaleta} />
      {seccion === "configurar" ? (
        <Configurar alTerminar={() => (window.location.hash = "proyectos")} />
      ) : seccion === "datapedia" ? (
        <Datapedia proyecto={proyecto} version={versionDatapedia} alAbrirTraza={abrirTraza} alAbrirPaso={abrirPaso} abrir={abrirDoc} />
      ) : seccion === "consultas" ? (
        <Consultas version={versionConsultas} alAbrirAgente={(c) => c.agente && abrirAgente({ id: c.agente, proyecto: c.proyecto, paso: c.paso >= 0 ? c.paso : undefined })} />
      ) : seccion === "salud" ? (
        <Salud proyecto={proyecto} version={versionProceso} alAbrirAgente={(id, paso) => abrirAgente({ id, paso })} />
      ) : seccion === "proyectos" ? (
        <Proyectos
          version={versionProceso}
          alAbrirProyecto={(n, s) => {
            if (n !== proyecto) elegirProyecto(n);
            window.location.hash = s || "agentes";
          }}
        />
      ) : seccion === "avance" ? (
        <Avance proyecto={proyecto} version={versionProceso} />
      ) : seccion === "estadisticas" ? (
        <Estadisticas proyecto={proyecto} version={versionProceso} alAbrirAgente={(id) => abrirAgente({ id })} />
      ) : seccion === "decisiones" ? (
        <Decisiones version={versionProceso} alAbrirSesion={(d) => abrirAgente({ id: d.agente, proyecto: d.proyecto, paso: d.paso })} />
      ) : seccion === "linea" ? (
        <Linea proyecto={proyecto} version={versionProceso} alAbrirAgente={abrirDesdeAlerta} />
      ) : (
        <main id="contenido" tabIndex={-1} aria-label={t("Agentes")} className="relative mx-auto w-full max-w-[1520px]">
          <div className="px-5 pt-5">
            <Cuotas estado={cuotas} leyendo={leyendo} alLeer={leerCuotas} />
          </div>
          <div className="flex flex-col items-start gap-5 p-5 lg:flex-row">
            <ListaAgentes
              agentes={agentes}
              cargando={cargandoLista}
              filtro={filtro}
              alFiltrar={filtrar}
              elegido={elegido}
              alElegir={elegirAgente}
              alertas={porAgente(alertas.alertas)}
            />
            <DetalleAgente
              detalle={visible}
              cargando={cargandoDetalle || (!!elegido && !detalle)}
              agentes={agentes}
              alAbrirAgente={abrirSubagente}
              alVerTraza={verTraza}
              alIrAPaso={(n) => elegido && setIrAPaso({ agente: elegido, n, cuando: Date.now() })}
            />
          </div>
          {/* La actividad, a todo el ancho de la página, debajo de la lista y del detalle. */}
          {visible && (
            <div className="px-5 pb-5">
              <Actividad
                key={visible.agente.id}
                idAgente={visible.agente.id}
                pasos={visible.pasos}
                desde={visible.desde ?? 0}
                enVivo={visible.agente.estado === "Trabajando"}
                alAbrirAgente={abrirSubagente}
                irATurno={irATurno && irATurno.agente === visible.agente.id ? irATurno : null}
                irAPaso={irAPaso && irAPaso.agente === visible.agente.id ? irAPaso : null}
              />
            </div>
          )}
        </main>
      )}
    </div>
  );
}
