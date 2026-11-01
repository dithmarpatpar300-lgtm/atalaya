# Registro de herramientas y apoyos · {{PROYECTO}}

La historia de lo que falló, de lo que sirvió y de cada uso de una skill o un plugin que enseñó algo. **Las reglas vigentes no están aquí:** están en `HERRAMIENTAS.md` (herramientas) y en `SKILLS.md` (apoyos).

- **Cómo se usa:**
  - cada fallo nuevo deja **una línea al final de la parte 1**, con el formato `fecha · quién · herramienta · qué pasó · qué funciona`. Si ya tenía regla y volvió a pasar, lleva la palabra **repetido**;
  - cada uso de un apoyo que enseñó algo va **al final de la parte 2**;
  - cada agente lanzado deja su línea `Config:` en la parte 3;
  - al cerrar una sub-etapa, el coordinador sube a `HERRAMIENTAS.md` lo que se repite.
- **Este archivo no se lee entero.** Se busca con Grep. Si crece más de ~100 KB, lo más viejo pasa a `docs/archivo/`.

# Parte 1 · Fallos de herramientas y órdenes que sirvieron

Una línea por hecho nuevo. Formato: `fecha · quién · herramienta · qué pasó · qué funciona`.

- {{FECHA_DE_INSTALACION}} · coordinador · instalación del centro de operaciones neutro · no fue un fallo · el kit se instaló con la skill atalaya-proyecto (su `SKILL.md`); ver su acta en `docs/trazabilidad/` (carpeta `temporales/`).

# Parte 2 · Usos de skills y plugins

Una línea por uso que enseñó algo. Formato: `fecha · quién · skill o plugin · para qué · resultado · ¿se repetiría?`

# Parte 3 · Uso de agentes (`AGENTES.md` §12)

Una línea por agente lanzado, copiada de la línea `Config:` de su informe:
`fecha · rol · clase · modelo/effort · minutos · escaló (sí: motivo / no) · hallazgos o retrabajo`.
