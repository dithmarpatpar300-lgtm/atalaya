/** Clave de localStorage con la preferencia de tema ("claro" u "oscuro"; sin valor = el del sistema). */
export const CLAVE_TEMA = "atalaya:tema";
export const CONSULTA_OSCURO = "(prefers-color-scheme: dark)";

/**
 * Se ejecuta en <head>, antes del primer pintado, para que la página no parpadee en el tema equivocado. Debe decidir
 * lo mismo que lib/tema.ts. Un valor desconocido vale como «sistema». También pone el idioma de la página (lib/i18n.ts:
 * el guardado o, sin elegir, el del navegador), para que el lector de pantalla lo sepa desde el principio.
 */
export const GUION_ARRANQUE = `(function(){var d=document.documentElement,p=null;try{p=localStorage.getItem("${CLAVE_TEMA}")}catch(e){}var o=p==="oscuro"||(p!=="claro"&&matchMedia("${CONSULTA_OSCURO}").matches);d.dataset.tema=o?"oscuro":"claro";var i=null;try{i=localStorage.getItem("atalaya:idioma")}catch(e){}d.lang=i==="es"||i==="en"?i:(navigator.language||"es").toLowerCase().indexOf("es")===0?"es":"en"})()`;
