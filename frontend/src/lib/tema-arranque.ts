/** Clave de localStorage con la preferencia de tema ("claro" u "oscuro"; sin valor = el del sistema). */
export const CLAVE_TEMA = "atalaya:tema";
export const CONSULTA_OSCURO = "(prefers-color-scheme: dark)";

/**
 * Se ejecuta en <head>, antes del primer pintado, para que la página no parpadee en el tema equivocado. Debe decidir
 * lo mismo que lib/tema.ts. Un valor desconocido vale como «sistema».
 */
export const GUION_ARRANQUE = `(function(){var d=document.documentElement,p=null;try{p=localStorage.getItem("${CLAVE_TEMA}")}catch(e){}var o=p==="oscuro"||(p!=="claro"&&matchMedia("${CONSULTA_OSCURO}").matches);d.dataset.tema=o?"oscuro":"claro"})()`;
