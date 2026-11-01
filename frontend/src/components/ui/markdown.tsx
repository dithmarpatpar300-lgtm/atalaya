import { Fragment, type ReactNode } from "react";
import { resolverRuta } from "@/lib/rutas";

/**
 * Markdown de los documentos del centro de operaciones, sin dependencias y sin HTML crudo: cada bloque se vuelve un
 * elemento de React, así un documento no puede inyectar nada en la página. Cubre lo que esos documentos usan:
 * encabezados, párrafos, listas, citas, tablas, bloques de código, separadores, frontmatter y, en línea, código,
 * negrita, cursiva y enlaces. Un enlace relativo a otro .md se abre en la datapedia con `alAbrirRuta`.
 */
export function Markdown({ texto, ruta, alAbrirRuta }: { texto: string; ruta: string; alAbrirRuta?: (ruta: string) => void }) {
  return <div className="flex flex-col gap-3 leading-relaxed">{bloques(texto.replace(/\r\n/g, "\n"), ruta, alAbrirRuta)}</div>;
}

type Abrir = ((ruta: string) => void) | undefined;

function enLinea(t: string, ruta: string, abrir: Abrir): ReactNode[] {
  const out: ReactNode[] = [];
  const re = /(`[^`]+`|\*\*[^*]+\*\*|\[[^\]]+\]\([^)\s]+\)|(?<![\w*])\*[^*\s][^*]*\*(?![\w*])|(?<!\w)_[^_\s][^_]*_(?!\w))/g;
  let ultimo = 0;
  let k = 0;
  for (const m of t.matchAll(re)) {
    const s = m[0];
    if (m.index > ultimo) out.push(t.slice(ultimo, m.index));
    ultimo = m.index + s.length;
    if (s.startsWith("`")) {
      out.push(
        <code key={k++} className="dato rounded-chip bg-elevado px-1 py-px text-[0.92em]">
          {s.slice(1, -1)}
        </code>,
      );
    } else if (s.startsWith("**")) {
      out.push(<strong key={k++}>{enLinea(s.slice(2, -2), ruta, abrir)}</strong>);
    } else if (s.startsWith("[")) {
      const [, texto, url] = /^\[([^\]]+)\]\(([^)\s]+)\)$/.exec(s)!;
      if (/^https?:\/\//.test(url)) {
        out.push(
          <a key={k++} href={url} target="_blank" rel="noreferrer noopener" className="text-acento underline underline-offset-2 hover:text-acento-fuerte">
            {texto}
          </a>,
        );
      } else if (abrir && /\.md([#:?].*)?$/i.test(url)) {
        const destino = resolverRuta(ruta, url);
        out.push(
          <button
            key={k++}
            type="button"
            onClick={() => abrir(destino)}
            title={destino}
            className="text-acento underline underline-offset-2 hover:text-acento-fuerte"
          >
            {texto}
          </button>,
        );
      } else {
        out.push(
          <span key={k++} title={url} className="underline decoration-dotted underline-offset-2">
            {texto}
          </span>,
        );
      }
    } else {
      out.push(<em key={k++}>{s.slice(1, -1)}</em>);
    }
  }
  if (ultimo < t.length) out.push(t.slice(ultimo));
  return out;
}

const esTabla = (l: string) => /^\s*\|.*\|\s*$/.test(l);
const esLista = (l: string) => /^\s*([-*+]|\d+[.)])\s+/.test(l);
const celdas = (l: string) =>
  l
    .trim()
    .replace(/^\||\|$/g, "")
    .split(/(?<!\\)\|/)
    .map((c) => c.trim());

function bloques(texto: string, ruta: string, abrir: Abrir): ReactNode[] {
  const lineas = texto.split("\n");
  const out: ReactNode[] = [];
  let i = 0;
  let k = 0;

  // Frontmatter (la memoria): se muestra aparte, en pequeño.
  if (lineas[0]?.trim() === "---") {
    const fin = lineas.indexOf("---", 1);
    if (fin > 0) {
      out.push(
        <pre key={k++} className="dato m-0 rounded-panel border border-borde bg-fondo p-3 text-ayuda whitespace-pre-wrap text-suave">
          {lineas.slice(1, fin).join("\n")}
        </pre>,
      );
      i = fin + 1;
    }
  }

  while (i < lineas.length) {
    const l = lineas[i];
    if (l.trim() === "") {
      i++;
      continue;
    }
    // Bloque de código.
    if (/^\s*```/.test(l)) {
      const cuerpo: string[] = [];
      i++;
      while (i < lineas.length && !/^\s*```/.test(lineas[i])) cuerpo.push(lineas[i++]);
      i++;
      out.push(
        <pre key={k++} className="dato scroll-interno m-0 overflow-x-auto rounded-panel border border-borde bg-fondo p-3 text-ayuda leading-relaxed">
          {cuerpo.join("\n")}
        </pre>,
      );
      continue;
    }
    const h = /^(#{1,6})\s+(.*)$/.exec(l);
    if (h) {
      const nivel = h[1].length;
      const clases = ["text-titulo font-semibold", "text-[1.125rem] font-semibold mt-3", "text-encabezado font-semibold mt-2", "font-semibold mt-1"];
      const Etiqueta = (["h2", "h3", "h4", "h5", "h6", "h6"] as const)[nivel - 1];
      out.push(
        <Etiqueta key={k++} className={`m-0 ${clases[Math.min(nivel, 4) - 1]}`}>
          {enLinea(h[2], ruta, abrir)}
        </Etiqueta>,
      );
      i++;
      continue;
    }
    if (/^\s*(-{3,}|\*{3,}|_{3,})\s*$/.test(l)) {
      out.push(<hr key={k++} className="my-1 border-borde" />);
      i++;
      continue;
    }
    if (esTabla(l)) {
      const filas: string[][] = [];
      while (i < lineas.length && esTabla(lineas[i])) {
        if (!/^\s*\|[\s:|-]+\|\s*$/.test(lineas[i])) filas.push(celdas(lineas[i]));
        i++;
      }
      const [cabeza, ...resto] = filas;
      out.push(
        <div key={k++} className="scroll-interno overflow-x-auto rounded-panel border border-borde">
          <table className="w-full border-collapse text-rotulo">
            <thead className="bg-elevado text-left">
              <tr>
                {cabeza.map((c, j) => (
                  <th key={j} className="border-b border-borde px-3 py-2 font-medium">
                    {enLinea(c, ruta, abrir)}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {resto.map((f, n) => (
                <tr key={n} className="border-b border-borde last:border-0">
                  {f.map((c, j) => (
                    <td key={j} className="px-3 py-2 align-top">
                      {enLinea(c, ruta, abrir)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>,
      );
      continue;
    }
    if (/^\s*>/.test(l)) {
      const cita: string[] = [];
      while (i < lineas.length && /^\s*>/.test(lineas[i])) cita.push(lineas[i++].replace(/^\s*>\s?/, ""));
      out.push(
        <blockquote key={k++} className="m-0 border-l-2 border-acento pl-3 text-suave">
          {bloques(cita.join("\n"), ruta, abrir)}
        </blockquote>,
      );
      continue;
    }
    if (esLista(l)) {
      const ordenada = /^\s*\d/.test(l);
      const items: { nivel: number; texto: string }[] = [];
      while (i < lineas.length && (esLista(lineas[i]) || (/^\s{2,}\S/.test(lineas[i]) && items.length > 0))) {
        const m = /^(\s*)([-*+]|\d+[.)])\s+(.*)$/.exec(lineas[i]);
        if (m) items.push({ nivel: Math.min(3, Math.floor(m[1].length / 2)), texto: m[3] });
        else items[items.length - 1].texto += " " + lineas[i].trim();
        i++;
      }
      const Lista = ordenada ? "ol" : "ul";
      out.push(
        <Lista key={k++} className={`m-0 flex flex-col gap-1 pl-5 ${ordenada ? "list-decimal" : "list-disc"}`}>
          {items.map((it, n) => (
            <li key={n} style={{ marginLeft: `${it.nivel * 1.25}rem` }}>
              {enLinea(it.texto, ruta, abrir)}
            </li>
          ))}
        </Lista>,
      );
      continue;
    }
    // Párrafo: las líneas seguidas hasta un blanco u otro bloque.
    const parrafo: string[] = [];
    while (
      i < lineas.length &&
      lineas[i].trim() !== "" &&
      !/^(#{1,6}\s|\s*```|\s*>)/.test(lineas[i]) &&
      !esTabla(lineas[i]) &&
      !esLista(lineas[i])
    ) {
      parrafo.push(lineas[i++].trim());
    }
    // Una línea que ningún bloque tomó (no debería pasar) se muestra tal cual y no detiene el recorrido.
    if (parrafo.length === 0) parrafo.push(lineas[i++]);
    out.push(
      <p key={k++} className="m-0">
        {parrafo.map((p, n) => (
          <Fragment key={n}>
            {n > 0 && " "}
            {enLinea(p, ruta, abrir)}
          </Fragment>
        ))}
      </p>,
    );
  }
  return out;
}
