// Las decisiones que las sesiones te piden. Solo lectura:
// Atalaya muestra la pregunta, sus opciones y la recomendada; respondes en la app.

export interface OpcionDecision {
  etiqueta: string;
  nota?: string;
  recomendada?: boolean;
}

export interface PreguntaDecision {
  texto: string;
  encabezado?: string;
  multiple?: boolean;
  opciones: OpcionDecision[];
  respuesta?: string;
}

export interface Decision {
  id: string;
  agente: string;
  proyecto: string;
  sesion: string;
  tipo: "botones" | "preguntas" | "atalaya";
  hora: string;
  paso: number;
  contexto?: string;
  preguntas: PreguntaDecision[];
  estado: "pendiente" | "respondida" | "reemplazada";
  respondida?: string;
  estado_sesion: string;
  vieja?: boolean;
}

export interface RespuestaDecisiones {
  pendientes: number;
  decisiones: Decision[];
}

export interface Grupos {
  esperando: Decision[];
  sinRespuesta: Decision[];
  respondidas: Decision[];
}

/** Lo que espera (de hoy o ayer), lo que quedó sin respuesta (más de un día) y lo ya respondido; lo reemplazado no se lista. */
export function agrupar(ds: Decision[]): Grupos {
  return {
    esperando: ds.filter((d) => d.estado === "pendiente" && !d.vieja),
    sinRespuesta: ds.filter((d) => d.estado === "pendiente" && d.vieja),
    respondidas: ds.filter((d) => d.estado === "respondida"),
  };
}

/** El título de una tarjeta: la pregunta, o «2 decisiones: medición y d2» cuando son varias. */
export function tituloDecision(d: Decision): string {
  if (d.preguntas.length === 1) return d.preguntas[0].texto;
  const temas = d.preguntas.map((q) => (q.encabezado || q.texto).toLowerCase());
  return `${d.preguntas.length} decisiones: ${temas.slice(0, -1).join(", ")} y ${temas[temas.length - 1]}`;
}

/** Lo que elegiste, corto, para la fila de las respondidas. */
export function eleccion(d: Decision): string {
  return d.preguntas
    .map((q) => q.respuesta)
    .filter(Boolean)
    .join(" · ");
}

/** ¿Esta opción es la que elegiste? Con varias, la respuesta trae las etiquetas separadas por coma. */
export function elegida(q: PreguntaDecision, o: OpcionDecision): boolean {
  if (!q.respuesta) return false;
  if (q.respuesta === o.etiqueta) return true;
  return q.multiple === true && q.respuesta.split(/,\s*/).includes(o.etiqueta);
}

/** «hace 6 min», «hace 2 h», «hace 3 d». */
export function hace(iso: string, ahora: number): string {
  const min = Math.max(0, Math.round((ahora - Date.parse(iso)) / 60_000));
  if (min < 60) return `hace ${min} min`;
  if (min < 48 * 60) return `hace ${Math.round(min / 60)} h`;
  return `hace ${Math.round(min / 1440)} d`;
}
