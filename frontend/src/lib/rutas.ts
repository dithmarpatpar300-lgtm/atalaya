/** Resuelve un enlace relativo de un documento («../adr/0013.md#x», «F5/08.md:12») contra la carpeta del documento actual. */
export function resolverRuta(base: string, enlace: string): string {
  const limpio = enlace.replace(/[#?].*$/, "").replace(/:\d+(-\d+)?$/, "");
  const partes = limpio.startsWith("/") ? [] : base.split("/").slice(0, -1);
  for (const p of limpio.split("/")) {
    if (p === "" || p === ".") continue;
    if (p === "..") partes.pop();
    else partes.push(p);
  }
  return partes.join("/");
}
