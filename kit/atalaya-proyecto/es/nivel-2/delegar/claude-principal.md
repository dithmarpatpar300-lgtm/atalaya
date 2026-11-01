## Claude Code coordina, Codex ayuda

La sesión de Claude Code reparte el trabajo. Antes de lanzar un subagente de Claude, se pregunta si Codex lo puede
hacer; se lanza con `scripts/codex.ps1`, y su informe y su parche quedan en `docs/trazabilidad/<fase>/codex/`, con una
línea en su `REGISTRO.md`.

| Va a Codex | Se queda en Claude Code |
| --- | --- |
| Buscar y leer: dónde está, quién lo usa, qué cambió | Coordinar, hablar con {{DUENO}} y decidir |
| Revisar y auditar un cambio, y la segunda opinión | El diseño y la arquitectura |
| Implementar con la especificación cerrada (`-Escribe`): pruebas, tipos, documentación, una pieza con su contrato escrito | Depurar con hipótesis y la implementación con decisiones abiertas |
| Repasar textos y comparar documentos | Lo que toca secretos o seguridad |

Con `-Escribe`, Codex trabaja en una copia y entrega un parche que **nunca se aplica solo**: el coordinador lo lee, lo
prueba y lo aplica o lo rechaza, y lo anota en `REGISTRO.md`. Si Codex falla dos veces seguidas en la misma clase de
tarea, esa tarea vuelve a Claude Code y se anota por qué.
