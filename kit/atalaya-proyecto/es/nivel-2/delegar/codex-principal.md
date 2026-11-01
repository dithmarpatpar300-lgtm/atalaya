## Codex coordina, Claude Code ayuda

La sesión de Codex reparte el trabajo. Lo que pide diseño o depuración difícil se le pasa a Claude Code con
`scripts/claude.ps1`: corre `claude -p` sin pantalla en una copia del repositorio, y su informe y su parche quedan en
`docs/trazabilidad/<fase>/claude/`, con una línea en su `REGISTRO.md`. Atalaya lo muestra colgado de la sesión de Codex
que lo lanzó.

| Va a Claude Code | Se queda en Codex |
| --- | --- |
| El diseño y la arquitectura, con el encargo y sus restricciones por escrito | Coordinar, hablar con {{DUENO}} y decidir |
| Depurar con hipótesis cuando la causa no es evidente | Buscar, leer y revisar |
| La segunda opinión de un cambio delicado | Implementar con la especificación cerrada |
| Redactar documentos largos o explicaciones para personas | Lo que toca secretos o seguridad |

Permisos de `claude.ps1`: por omisión solo lee (`Read`, `Grep`, `Glob` y el MCP de Atalaya). Con `-Escribe` también
edita, solo dentro de la copia, y entrega un parche que **nunca se aplica solo**: el coordinador lo lee, lo prueba y lo
aplica o lo rechaza, y lo anota en `REGISTRO.md`. No tiene shell ni web; una orden de prueba se nombra con `-Ordenes`
(por ejemplo `-Ordenes 'npm test'`), una por una. Si Claude Code falla dos veces seguidas en la misma clase de tarea,
esa tarea vuelve a Codex y se anota por qué.
