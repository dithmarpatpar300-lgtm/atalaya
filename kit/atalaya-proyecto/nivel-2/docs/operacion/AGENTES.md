# Agents, models and effort · {{PROYECTO}}

The policy for deciding **which agent, which model, which effort and how much context** each task gets.

- **Principle:** use **the minimum configuration able to solve the task with the reliability its risk demands**, and escalate **only with a written technical reason**.
- **The method:** first classify the task (§2), then assign it with the matrix (§3). That classification is the "scheduler".
- **How it fits with the rest:**
  - `PRIORIZACION.md` (the prioritization guide) decides the **level** of a brief (risk × size), and therefore how many roles join the cycle;
  - this file decides **what each role works with**;
  - the general rules stay in `AGENTS.md`.

## 1. What actually exists (verify at install time)

| Model | Name in Claude Code | Relative weight | What it is for |
| --- | --- | --- | --- |
| Haiku | `haiku` | 1× | Search, reading, mechanical changes, summaries of outputs |
| Sonnet | `sonnet` | ~20× | Implementation, ordinary debugging, tests, mockups |
| Opus | `opus` | ~40× | Architecture, critical audit, hard debugging, coordination |
| Fable | `fable` | ~100× | The most capable. Only as the last step (§4) |

> The weights are the ones measured in the originating project (October 2026). Models and prices change: the installer confirms them with {{DUENO}} and corrects this table. Delete this note at install time.

- **Effort:** all four accept `low`, `medium`, `high`, `xhigh` and `max`. **Here the effort is always set explicitly.** The default value is not trusted.
- **Where it is set:**
  - the frontmatter of `.claude/agents/<role>.md` (`model:` and `effort:`);
  - or each `Agent` call (`model` and `effort`), which takes precedence over the file.

  This document is the express instruction that authorizes the coordinator to set the effort per call.
- **The roles in `.claude/agents/` are loaded when the session opens.** Roles created during a session do not show up in it ("Agent type not found"). In that case, launch `general-purpose` with the `model` and `effort` from the matrix, and start the brief with "Read `.claude/agents/<role>.md` and follow it as your role instructions". A role's frontmatter carries only `name`, `description`, `model` and `effort`; the `description` cannot contain an unquoted ": ", because it breaks the YAML.
- **An agent receives the `AGENTS.md` that the session loaded when it opened, not the one on disk.** If the rules changed during the session, the brief says "read `AGENTS.md` from disk".
- **An agent has a floor of tens of thousands of tokens** even for a trivial task (measured: about 78,000 with Haiku). What fits in a few minutes is done by the coordinator.
- **What the price means:** with a subscription you pay in usage limit; with an API key, in money. The relative weight is a guide; if the limit is consumed in a different proportion, measure it and correct it (§12).

## 2. Classifying tasks

Every brief is classified with a **class**. The class comes from the question "what is left to decide?", not from the size.

**Risk zone** of this project: {{ZONA_DE_RIESGO}} (anything that stores or transforms data, security and permissions, migrations, secrets, deployment scripts, integrations with external systems, production).

| Class | What it is | Generic examples |
| --- | --- | --- |
| **M · Mechanical** | Everything is decided: it only has to be executed well. **In the risk zone** only a **literal** change is M (copying a fingerprint, a version number, an approved text) and the coordinator reviews the diff line by line. Anything else there is I1 at the least | Bumping the version in the files that carry it; renaming; already-approved texts; formatting a report; extracting data from a log |
| **I1 · Guided implementation** | Closed design: approved mockup, written contract or signature, and defined tests | A component from an approved mockup; an endpoint with a written contract; a validation with its rule; the tests for a change |
| **I2 · Implementation with decisions** | You have to decide how, in the risk zone | Concurrency or save conflicts; encryption or keys; permissions and roles; an integration with an external system |
| **D1 · Obvious debugging** | Clear message and a located cause | A compile or type error; a failing test with a clear diff |
| **D2 · Distributed debugging** | Several components and a cause that is not obvious | A mock that is missing a route; a seed that collides with an old step; a generated document that comes out wrong |
| **D3 · Hard debugging** | Intermittent, load, concurrency, inconsistent data, external system or unknown cause | A test suite that fails only when the machine is under load; something that fails only against the real system |
| **A · Architecture and design** | Decisions that propagate | Schema changes or new migrations, data flow, contracts between components, synchronization, roles and permissions |
| **E1 · Routine audit** | Reviewing something of low or medium risk | Documentation, texts, a visual component, a small refactor |
| **E2 · Critical audit** | Reviewing something that can lose, change or expose data, or touch production | Keys and backups, migrations, saving and conflicts, permissions, deployment scripts |
| **V · Validation and testing** | Proving that it works with a known procedure, separate from the deliverer's own tests | The test run of a release kit (`version-y-kit.md` §4); repeating a measurement with the machine idle |
| **R · Search and research** | R1: locate something inside the repository. R2: research outside, with sources. R3: research to decide architecture | R1: "where is this constant used?". R2: a standard or a library, read and never invented. R3: the study of a new module |
| **U · Mockups and visual design** | Proposing a screen for approval | A new screen; a mockup of an extension |
| **T · Documentation** | T1: update with given data. T2: write guides or methods for {{DUENO}} | T1: status, logs, release notes. T2: the guide to a release, this document |

**Who classifies and how ties are broken:**
- **The coordinator classifies,** and writes it in the assignment line.
- **If a task looks like two classes,** the one that asks for more capability in the matrix §3 wins: first the model, then the effort.
- **When in doubt between E1 and E2, it is E2.**
- **If you do not know what is left to decide,** it is A or R3 until you do.
- **Irreversible** means it cannot be undone with git or with a rehearsed rollback. For example:
  - real data transformed or deleted;
  - changes to keys, encryption or backups;
  - anything sent to an external system that cannot be withdrawn.

## 3. Matrix: class → role → model and effort

| Class | Role (`.claude/agents/`) | Model · effort | Review | Autonomy |
| --- | --- | --- | --- | --- |
| M | `ejecutor`, or the coordinator if it fits in minutes | Haiku · `medium` (`low` if it is copy and paste) | Automatic check: `grep`, build, a test. The coordinator looks at the diff | None: if something is not in the brief, stop |
| I1 | `codificador` | Sonnet · `medium` | The deliverer's tests + the coordinator's cheap checks. The audit depends on the level (`PRIORIZACION.md` §1) | Low: follows the specification and escalates doubts |
| I2 | `codificador` | Sonnet · `high` | E2 audit of the diff | Medium: decides what is local and escalates what is structural |
| D1 | The coordinator, or `codificador` | Sonnet · `medium` | The test that was failing, now green | Medium |
| D2 | `depurador` | Sonnet · `high` | Reproduce before and after | Medium |
| D3 | `depurador` | Opus · `high`. Up to two independent hypotheses are reproduced in parallel, each with a `depurador` on Sonnet · `medium`, with its own port and folder. If both need the same shared resource, they run in series | Demonstrated reproduction; if it touches data or production, E2 audit | Medium, with written hypotheses |
| A | `arquitecto` | Opus · `high`. If irreversible: Opus · `xhigh` | When irreversible, a cross-critique (§6.3) and {{DUENO}}'s approval | High to propose; **none to implement** |
| E1 | `auditor` with `model: sonnet` | Sonnet · `high` | The coordinator confirms the findings | High to search; none to fix |
| E2 | `auditor` | Opus · `high`. **`xhigh` only if the change is irreversible** | The coordinator decides the box for each finding (`PRIORIZACION.md` §3) | Same |
| V | `ensayador` | Sonnet · `medium` for a release with no logic changes; `high` if it brings a migration or new deployment logic, or a failure that cannot be explained | The verdict with evidence | Low: the procedure is fixed |
| R1 | `Explore` | Haiku · `low` (`medium` if the search is wide) | What it returns is checked | — |
| R2 | `general-purpose` | Sonnet · `high` | Sources cited; anything unverified, marked | Medium |
| R3 | `arquitecto` | Opus · `high` | Same as A | High to propose |
| U | `general-purpose`, with design skills | Sonnet · `high` | {{DUENO}}'s approval of the mockup | Medium within the approved identity |
| T1 | `ejecutor` | Haiku · `medium` | The coordinator reads the result | Low |
| T2 | The coordinator, or `general-purpose` | Sonnet · `medium`; Opus · `medium` if it defines rules or methods | The coordinator; {{DUENO}} if it is for them | Medium |

**The six profiles and their roles:** Explorer = `Explore`; Architect = `arquitecto`; Implementer = `codificador` (and `ejecutor` for mechanical work); Debugger = `depurador`; Auditor = `auditor`; Validator = `ensayador` on releases, and the deliverer runs the expensive tests only once.

**The coordinator** (the main session) runs on Opus.
- **Effort:** `medium` for coordinating and reviewing deliveries; `high` for designing, deciding or reviewing something critical.
- **`max`:** never by default. **Fable:** only by escalation (§4).
- **Orchestrating many agents at once** ("workflow"): only if {{DUENO}} asks for it.

## 4. Escalation: when and how to move up

**An agent does not change model midway.** Escalating means **stopping and reporting**: the agent writes down what it found and the coordinator relaunches the task one step higher, handing over that finding so it does not start from zero.

**Valid triggers.** Write down which one it was and its evidence:
1. **There is more than one reasonable interpretation** of the specification, and the choice changes the result.
2. **The scope goes beyond the brief:** an implementer needs to change a file the brief does not name, or an auditor/debugger/architect finds the cause or the risk elsewhere in the system.
3. **A contradiction** between the specification, the code, the tests or what was approved.
4. **Risk of data loss or corruption,** or of exposing a secret or protected data.
5. **The solution changes the architecture:** a contract, the schema, the data flow, the boundary with another system.
6. **Concurrency or races:** two writers, locks, timers, out-of-order events.
7. **It touches production** or a deployment script.
8. **A hypothesis that cannot be verified** with confidence, or two failed attempts with the same idea.
9. **The time budget of the level** is about to be exceeded with no verifiable result. The usual response is to report and cut the scope; relaunch on another step only if the cause of the delay is one of triggers 1 to 8.

**"It is hard", "I want to be sure" or "just in case" are not triggers.**

**The ladder.** Move up **one step** at a time:

`Haiku medium → Sonnet medium → Sonnet high → Opus high → Opus xhigh → Fable high`

- **How to choose what to raise:** within the same model, the effort first, when the problem is one of **depth** (more steps, more cases); the model, when it is one of **judgment** (triggers 1, 3, 5 and 8).
- **Fable** comes in only if all three hold: (1) Opus `xhigh` left a contradiction unresolved; (2) it is an **irreversible** A or E2 decision; (3) {{DUENO}} gave **their yes**. It carries a budget of 30 minutes.
- **Cap:** two escalations per task. If two do not resolve it, the coordinator takes the task or asks {{DUENO}}.
- **The format of the notice,** in the log and in the final message, one line:

  `ESCALATE · trigger N · evidence (file:line or output) · what is already done · what is missing`

## 5. De-escalation: when to move down

Once an expensive role has closed the decision, the execution passes to a cheaper one. **A decision is closed** if there is a document in `docs/trazabilidad/<phase>/` that holds: (1) the files to be touched; (2) the contracts or signatures; (3) the acceptance tests; (4) no open questions.

**The pattern:** `arquitecto` (Opus · `high`) decides and writes the design → `codificador` (Sonnet · `medium`) implements by modules, and the repetitive parts go to `ejecutor` (Haiku) → `auditor` reviews **only the diff** against the design.

**What is never lowered because of history:** the class of an audit is set by the risk of the change. An E2 is still an E2 even if the earlier ones came out clean: zero findings do not prove the code is right.

**What can be adjusted with history:** the effort within the same class and model (for example, an irreversible E2 with three rounds in a row without medium findings can drop from `xhigh` to `high`, if the coordinator writes it in the assignment), and every unnecessary escalation is noted for calibration.

## 6. Audit

1. **The auditor's question** is not "does it seem to work?", but **"in what ways could it be wrong even though it seems to work?"**: false assumptions, edge cases, side effects, contradictions with what was approved, security and data, maintainability, what the test does not cover.
2. **Independence:**
   - the auditor wrote nothing of what they review and does not fix it;
   - they receive the specification, the design and the **exact commit** being audited;
   - they read the implementer's report **after** forming their own list, and treat it as claims to verify;
   - each important claim is demonstrated with a test or a mutation on a copy.
3. **Cross-critique** is done only for irreversible A or E2. A second auditor reviews with a **different focus** (one looks at security and secrets, the other at data and rollback). Two identical auditors are not launched "to get another opinion".
4. **Each finding** carries severity × cost and its box (`PRIORIZACION.md` §3). The report says what it could **not** verify.
5. **When there is an audit:** the level says so (`PRIORIZACION.md` §1). Its class (E1 or E2) comes from the risk of the change, not its size. When in doubt, E2.
6. **If an auditor launched as E1 ends up reviewing something that is E2,** they stop and write `ESCALATE · reclassify to E2 · evidence`.

## 7. Implementation

The implementer (`codificador`, `ejecutor`): **follows the specification;** **changes the minimum** and respects the contracts; **introduces no new architecture** (no dependencies, no abstractions nobody asked for, no "while I was at it"); **runs the tests that fall to them** and reports deviations; and **if the specification is not enough or contradicts itself, does not improvise in silence:** they stop and escalate (§4, triggers 1 and 3).

## 8. Parallelization

**Do parallelize:** independent modules with a written folder boundary; `Explore` searches on different questions; independent hypotheses of a D3; audits of different areas of the same change; writing documents that do not touch each other. Afterwards, **the coordinator consolidates**.

**Do not parallelize:** anything with a dependency (the order is `discovery → decision → implementation → validation`; nothing is implemented while the design is open); two writers on the same file or folder; two agents on the same shared resource (test database, port); two long test suites at once, or a test suite while an agent is working.

**If two writers need neighboring folders,** each one goes in its own worktree (`isolation: "worktree"`), and the coordinator integrates.

## 9. Context per agent: the minimum necessary

| Role | Receives | Does not receive |
| --- | --- | --- |
| `Explore` | The question and the folders to search | Rules, history, a long brief |
| `ejecutor` | The brief with the exact steps, the files and the final check | Full specification, history, status |
| `codificador` | The brief, the sections of the specification or design, the files, the contracts, the constraints, `HERRAMIENTAS.md` (the tools guide) and `pruebas.md` | The project history, the tools log, reports that do not concern it |
| `depurador` | The symptom, how to reproduce it, what has already been ruled out and the suspect files | The full design, unless the cause is there |
| `auditor` | The specification and design, the exact commit, the earlier decisions that apply and the rules that are not negotiable | The implementer's reasoning before their own review |
| `arquitecto` | The problem, the status, the specification and the limits of the code | Implementation details that do not matter yet |
| `ensayador` | The kit, its release report, the last test run and `version-y-kit.md` | The source code, except to check what is served |

**Common rules:**
- `AGENTS.md` loads by itself (§1). There is no need to copy it into the brief.
- The roles are told to read `HERRAMIENTAS.md` and `PRIORIZACION.md` in full, as long as the two together fit in about 8,000 tokens. If they grow, trim the roles' preambles.
- Name files and lines instead of pasting their content.
- No agent reads the whole status history or the tools log.

## 10. Coordination: how errors between agents are avoided

- **Fixed base.** Every brief says which commit it works on (`{{RAMA_TRABAJO}}` at `<hash>`). The auditor audits that commit or that diff, not "whatever is there".
- **One writer per file.** The file boundary is written in the brief, and the coordinator checks it with `git status` before integrating.
- **Decisions live in a single document.** If two agents could decide the same thing, the `arquitecto` or the coordinator decides before launching them.
- **No duplicated work.** Before launching, the coordinator checks the active logs (`scripts/avance.ps1`).
- **No lost context.** What one agent found and another needs goes in its report, and the next brief names it.
- **Outdated information.** If the tree changed since the brief, the agent says so before continuing: `git log` from the brief's hash.

## 11. Before launching agents: the budget

The coordinator answers in writing in the brief, below the assignment line:
1. What concrete problem does this agent solve?
2. What class is it and what configuration does it get?
3. What context does it need, and which does it not?
4. Can it be done in parallel, or does it depend on something?
5. How will the result be verified?

**If there is no clear function, it is not launched.**

## 12. Measurement, without bureaucracy

- **Every agent report** ends with one line: `Config: role · class · model/effort · minutes · escalated (yes: reason / no) · findings or rework`. The implementer does not know the rework at delivery: they leave "—".
- **The coordinator** copies it into `HERRAMIENTAS-registro.md` (the tools log), part 3, and fills in the findings and rework when they are known.
- **When closing a sub-stage** they look at these signals. Only classes with **5 tasks or more** count:

| Signal | What it indicates | What to do |
| --- | --- | --- |
| More than 30 % of a class's tasks escalated | That configuration is insufficient | Raise that class's configuration one step |
| A class never escalates and its audits give 0 findings | It may be over-provisioned | Try one step lower on the next one |
| A fix or rollback after a "verified" delivery | A defect was introduced | Review the class and the auditor |
| More than two rounds of correction | Rework: insufficient design or specification | Look at class A and the brief, not just the implementer |
| Many **new** tokens (not from cache) per agent | Too much context | Trim what is passed to it (§9) |
| Percentage of agents on Opus | Cost mix | Suggested initial target: under 40 % without defects rising |

## 13. Generic examples

1. **Bumping a version** (number, fingerprints, notes): M. `ejecutor` on Haiku · `medium`, with the exact list from `version-y-kit.md`; the coordinator reviews the literal diff.
2. **A new screen:** U mockup (Sonnet · `high`) and {{DUENO}}'s approval; if it touches roles or permissions, A (`arquitecto`); the server route is I2; the screen is I1; at the end, E2 on the diff.
3. **Changing a constant that decides who does what:** I1 with high risk, `codificador` on Sonnet · `medium`, plus a 20-minute E2 only on what changes.
4. **A schema change or migration:** A (`arquitecto`, Opus · `high`), I2 (`codificador`, Sonnet · `high`), E2 (`auditor`, Opus · `high`; `xhigh` and cross-critique if it transforms or deletes real data) and V with the rollback rehearsed.
5. **A compile error after a change:** D1; the coordinator fixes it.
6. **A test that fails only with the full suite:** D2; `depurador` on Sonnet · `high`, reproducing with artificial load.
7. **Something that fails only against the real system:** D3; `depurador` on Opus · `high`, two hypotheses in parallel against a mock. {{DUENO}} runs the real test.
8. **Review of keys and backups before a major release:** irreversible E2; `auditor` on Opus · `xhigh` with cross-critique.
9. **Studying a new module:** R3/A with `arquitecto` on Opus · `high`; the searches in the repository are done by `Explore` on Haiku · `low`.
10. **"Where is this constant used?":** R1; `Explore` on Haiku · `low`, or the coordinator's Grep.
