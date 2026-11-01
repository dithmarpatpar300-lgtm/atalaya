# Agent brief template · {{PROYECTO}}

The coordinator writes each brief on this template. The brief is the only thing the agent knows about the task, so it has to stand on its own.

**The fixed blocks do not go here.** "Before starting", "Limits", "Log" and "Delivery" live in each role's file, and Claude Code loads them by itself when the agent is launched: `.claude/agents/ejecutor.md`, `codificador.md`, `depurador.md`, `arquitecto.md`, `auditor.md` and `ensayador.md`.

Each file carries its default model and effort. The coordinator changes them in the call when the matrix of `AGENTES.md` §3 asks for it (for example, `codificador` with `effort: high` for I2).

**For another role** (researcher, mockup designer, writer…) use `general-purpose` and the brief starts with these two lines: "Read `AGENTS.md` and `docs/operacion/HERRAMIENTAS.md` before starting" and "Follow the limits and the log of `.claude/agents/codificador.md`". Then come the rules of this task.

---

**Assignment: level [direct | light | focused | full] · class [M | I1 | I2 | D1-D3 | A | E1 | E2 | V | R1-R3 | U | T1 | T2] · role [ejecutor | codificador | depurador | arquitecto | auditor | ensayador | Explore | general-purpose] · model [model]/[effort] · budget [{{TOPE_MIN}} min] · base `{{RAMA_TRABAJO}}` at [hash]**

That line comes from `PRIORIZACION.md` §1 (the level) and from `AGENTES.md` §2 and §3 (the class, the role and the configuration).

**Budget** (`AGENTES.md` §11), one line per point:
- the concrete problem;
- the context it needs and the one it does not;
- whether it runs in parallel or what it waits for;
- how it will be verified.

[If the role does not load in this session, or if the rules changed since it was opened (`AGENTES.md` §1), the brief starts with: "Read `.claude/agents/<role>.md` and `AGENTS.md` from disk, and follow them".]
- **If you step outside the role's configuration,** say why in one sentence.
- **If you are going to exceed the budget,** say so in the log with what is missing, as soon as you see it.
- **If a trigger from `AGENTES.md` §4 appears,** escalate instead of carrying on.

You are the **[role] of [sub-stage and part]** of the {{PROYECTO}} project ([the stack that applies to you]). [One sentence on what is at stake: data, an irreversible piece, what the user will see…]

## What to read first

[Short, ordered list:
- the sections of the specification;
- the reports and the lines the datapedia returned (if there is one);
- the code files to know before touching anything.

If the list is complete, the agent does not need to consult the datapedia.]

## What to do

[The work, in parts, with the decisions already made and those that are not theirs.
- **For an auditor:** what to look at most carefully.
- **For a fixer:** the list of findings, the decision on each one, and that they return the real cost.]

**Expensive tests, only once** (`PRIORIZACION.md` §2 and `docs/operacion/pruebas.md`): [who runs the complete suite; whether there are mutations and on what; whether performance is measured and why; which screenshots are needed. What another role already ran and did not change is not repeated.]

## Rules of this task

[Only what is specific to this task; the general rules are already in `AGENTS.md` and in the role:
- which folders can be touched and which cannot (the file boundary, if another agent is working in parallel);
- what can be run: databases, ports;
- if the task touches production: {{DOC_DE_PRODUCCION}}.]

## Log and delivery

- **Log:** `docs/trazabilidad/[phase]/avance/[role].md`.
- **Scratch-files record:** `docs/trazabilidad/[phase]/temporales/[yyyy-mm-dd]-[role]-[topic].md`, opened when the working folder is created and closed **before** deleting it (`HERRAMIENTAS.md` §3). The report names it.
- **Report:** `docs/trazabilidad/[phase]/[NN]-[type]-[topic].md`, with [what is specific to the task].
- **The rest** (what the report contains, how it is closed) comes in the role's file.
