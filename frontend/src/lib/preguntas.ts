// Responder desde Atalaya: una sesión pregunta por el
// MCP `preguntar` y espera; aquí se responde por pasos, con texto libre e imágenes. La lógica de qué preguntas se ven
// y si se puede enviar es pura, para probarla sin navegador.

export interface OpcionCentro {
  etiqueta: string;
  nota?: string;
  recomendada?: boolean;
  abre?: string[];
}

export interface PreguntaDelCentro {
  id: string;
  texto: string;
  encabezado?: string;
  multiple?: boolean;
  opciones: OpcionCentro[];
  depende?: { pregunta: string; opcion: string };
}

export interface PreguntaCentro {
  id: string;
  proyecto: string;
  cliente: string;
  titulo: string;
  contexto?: string;
  preguntas: PreguntaDelCentro[];
  creada: string;
  estado: "esperando" | "respondida" | "entregada";
}

/** Lo que va armando la persona: por pregunta, las etiquetas elegidas y su texto propio («Escribir otra respuesta»). */
export interface Borrador {
  elecciones: Record<string, string[]>;
  textos: Record<string, string>;
  otra: Record<string, boolean>;
}

export const borradorVacio = (): Borrador => ({ elecciones: {}, textos: {}, otra: {} });

/** Las preguntas que se hacen con lo elegido: una que depende de una opción no elegida no aparece. */
export function visibles(qs: PreguntaDelCentro[], elecciones: Record<string, string[]>): PreguntaDelCentro[] {
  const ver = new Set<string>();
  const out: PreguntaDelCentro[] = [];
  for (const q of qs) {
    if (q.depende && (!ver.has(q.depende.pregunta) || !(elecciones[q.depende.pregunta] ?? []).includes(q.depende.opcion))) continue;
    ver.add(q.id);
    out.push(q);
  }
  return out;
}

/** ¿Está respondida? Una opción suya, o «otra» con texto. */
export function respondida(q: PreguntaDelCentro, b: Borrador): boolean {
  const el = b.elecciones[q.id] ?? [];
  return el.length > 0 || (!!b.otra[q.id] && (b.textos[q.id] ?? "").trim() !== "");
}

/** Elegir una opción: en una de opción única reemplaza (y apaga «otra»); en una múltiple la suma o la quita. */
export function elegir(b: Borrador, q: PreguntaDelCentro, etiqueta: string): Borrador {
  const actual = b.elecciones[q.id] ?? [];
  const nuevas = q.multiple ? (actual.includes(etiqueta) ? actual.filter((x) => x !== etiqueta) : [...actual, etiqueta]) : [etiqueta];
  return { ...b, elecciones: { ...b.elecciones, [q.id]: nuevas }, otra: q.multiple ? b.otra : { ...b.otra, [q.id]: false } };
}

/** «Escribir otra respuesta»: en una de opción única quita la opción elegida; en una múltiple se suma a ellas. */
export function elegirOtra(b: Borrador, q: PreguntaDelCentro): Borrador {
  return { ...b, otra: { ...b.otra, [q.id]: true }, elecciones: q.multiple ? b.elecciones : { ...b.elecciones, [q.id]: [] } };
}

/** Lo que se envía: solo lo de las preguntas que se ven (lo de una que se cerró al cambiar de opción, no). */
export function cuerpo(qs: PreguntaDelCentro[], b: Borrador, nota: string, imagenes: { nombre: string; base64: string }[]) {
  const vis = visibles(qs, b.elecciones);
  const elecciones: Record<string, string[]> = {};
  const textos: Record<string, string> = {};
  for (const q of vis) {
    if ((b.elecciones[q.id] ?? []).length) elecciones[q.id] = b.elecciones[q.id];
    if (b.otra[q.id] && (b.textos[q.id] ?? "").trim()) textos[q.id] = b.textos[q.id].trim();
  }
  return { elecciones, textos, nota: nota.trim(), imagenes };
}

/** Cómo se lee la respuesta de una pregunta en el resumen. */
export function textoRespuesta(q: PreguntaDelCentro, b: Borrador): string {
  const partes = [...(b.elecciones[q.id] ?? [])];
  if (b.otra[q.id] && (b.textos[q.id] ?? "").trim()) partes.push(`«${b.textos[q.id].trim()}»`);
  return partes.join(" · ") || "—";
}

export const TIPOS_IMAGEN = ["image/png", "image/jpeg", "image/gif", "image/webp"];
export const MAX_IMAGENES = 4;
export const MAX_BYTES_IMAGEN = 5 * 1024 * 1024;

/** Por qué no entra una imagen, o null si entra. */
export function motivoImagen(tipo: string, bytes: number, yaHay: number): string | null {
  if (yaHay >= MAX_IMAGENES) return `Van hasta ${MAX_IMAGENES} imágenes.`;
  if (!TIPOS_IMAGEN.includes(tipo)) return "Solo PNG, JPEG, GIF o WebP.";
  if (bytes > MAX_BYTES_IMAGEN) return "Esa imagen pasa de 5 MB.";
  return null;
}
