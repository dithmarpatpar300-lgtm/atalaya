## Codex coordinates, Claude Code helps

The Codex session assigns the work. Anything that calls for design or hard debugging is passed to Claude Code with
`scripts/claude.ps1`: it runs `claude -p` headless on a copy of the repository, and its report and its patch are kept in
`docs/trazabilidad/<phase>/claude/`, with a line in its `REGISTRO.md`. Atalaya shows it attached to the Codex session
that launched it.

| Goes to Claude Code | Stays with Codex |
| --- | --- |
| Design and architecture, with the brief and its constraints in writing | Coordinating, talking to {{DUENO}} and deciding |
| Debugging with hypotheses when the cause is not obvious | Searching, reading and reviewing |
| The second opinion on a delicate change | Implementing with a closed specification |
| Writing long documents or explanations for people | Anything that touches secrets or security |

Permissions of `claude.ps1`: by default it only reads (`Read`, `Grep`, `Glob` and the Atalaya MCP). With `-Escribe` it also
edits, only inside the copy, and delivers a patch that is **never applied by itself**: the coordinator reads it, tests it and
applies or rejects it, and notes it in `REGISTRO.md`. It has no shell or web; a test command is named with `-Ordenes`
(for example `-Ordenes 'npm test'`), one by one. If Claude Code fails twice in a row on the same class of task,
that task goes back to Codex and the reason is noted.
