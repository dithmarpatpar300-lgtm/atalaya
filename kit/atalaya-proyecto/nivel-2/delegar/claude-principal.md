## Claude Code coordinates, Codex helps

The Claude Code session assigns the work. Before launching a Claude subagent, ask whether Codex can do it; it is launched
with `scripts/codex.ps1`, and its report and its patch are kept in `docs/trazabilidad/<phase>/codex/`, with a line in
its `REGISTRO.md`.

| Goes to Codex | Stays with Claude Code |
| --- | --- |
| Searching and reading: where it is, who uses it, what changed | Coordinating, talking to {{DUENO}} and deciding |
| Reviewing and auditing a change, and the second opinion | Design and architecture |
| Implementing with a closed specification (`-Escribe`): tests, types, documentation, a piece with its contract written | Debugging with hypotheses, and implementation with open decisions |
| Reviewing texts and comparing documents | Anything that touches secrets or security |

With `-Escribe`, Codex works on a copy and delivers a patch that is **never applied by itself**: the coordinator reads it,
tests it and applies or rejects it, and notes it in `REGISTRO.md`. If Codex fails twice in a row on the same class of
task, that task goes back to Claude Code and the reason is noted.
