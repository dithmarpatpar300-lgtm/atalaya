# Operations center · {{PROYECTO}}

This is **how work is done** in this project. The rules are in **`AGENTS.md`**, at the root: it is their only copy and it loads by itself in every session. This folder keeps the detail that those rules link to.

- **What is being built:** {{ESPECIFICACION}}.
- **Where the work stands:** `docs/trazabilidad/ESTADO-para-retomar.md`.

## The files

| File | What for | Who reads it |
| --- | --- | --- |
| `HERRAMIENTAS.md` | What works and what fails on this machine, tool by tool, with the command that works; the cycle of scratch files (§3) | Every agent at the start; the coordinator |
| `HERRAMIENTAS-registro.md` | The history: one line for each tool failure, for each use of a skill that taught something and for each agent launched | Whoever notes it; the coordinator when closing a sub-stage |
| `PRIORIZACION.md` | Proportional effort: level of the brief (risk × size), expensive tests only once, findings by severity × cost | The coordinator when assigning; the auditor |
| `AGENTES.md` | Agent policy: task classes, class → role → model and effort matrix, escalation and de-escalation, audit, parallelism, context per agent and measurement | The coordinator before launching any agent |
| `ENCARGO-PLANTILLA.md` | The variable part of each brief (the fixed block is in `.claude/agents/`) | The coordinator |
| `SKILLS.md` | Skills, plugins and tools that really exist: when yes and when no | The coordinator; an agent if its brief asks for it |
| `pruebas.md` | How testing is done: test suites, caps, mutations, screenshots, mocks and known traps | Coder, auditor and tester |
| `version-y-kit.md` | (optional) The step-by-step of a release: build, kit, test run, delivery and close-out | The coordinator and the tester |

**The swarm's roles** are in `.claude/agents/` (`ejecutor`, `codificador`, `depurador`, `arquitecto`, `auditor`, `ensayador`), each with its default model and effort (`AGENTES.md` §3).

**The templates** (the scratch-files record) are in `docs/trazabilidad/_plantillas/`.

## How it is maintained

- **A rule is changed in its single place.** If it is general, in `AGENTS.md`; if it belongs to a tool, in `HERRAMIENTAS.md`. It is never copied to another file: it is linked.
- **Every tool failure** leaves a line in `HERRAMIENTAS-registro.md`. When closing a sub-stage, the coordinator turns what repeats into a rule.
- **Size caps,** so that nobody reads halfway: `HERRAMIENTAS.md` about 20 KB; `SKILLS.md` about 8 KB; `AGENTES.md` about 28 KB (the coordinator reads it, not each agent); the status to resume about 6 KB. What grows without stopping goes to a log or to the history.
- **Old material is not deleted:** it goes to `docs/archivo/`.
