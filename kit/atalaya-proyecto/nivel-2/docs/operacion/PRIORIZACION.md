# Proportional effort and prioritization · {{PROYECTO}}

The owner's rule: an agent that takes an hour or more on something simple is lost time that cannot be recovered. Before every brief and every decision, two questions are asked: **is it worth doing this much for something this simple or short?** and **where is the clear goal on which the effort is concentrated?** The quality of the output does not drop: the repetition does.

What does **not** change with this rule: "verified" still means "I ran it and saw the result"; the rules without exception in `AGENTS.md` §4 are not negotiable; secrets, protected data and what the user already wrote are never postponed because they are expensive.

## 0. Strict time rule

"If we have to wait more than {{TOPE_MIN}} minutes per agent we will never make progress." No exceptions:

1. **By default the coordinator does it,** directly, if it fits in about 20 minutes of their work. An agent only when the task is big **and** can be done in parallel with something else useful.
2. **No agent runs longer than {{TOPE_MIN}} minutes.** The brief is cut to fit; if it does not fit, it is split into pieces or the coordinator does it.
3. **A single round:** code, and the coordinator verifies. A separate audit only for real high risk (data, secrets, migrations, the production entry point), and of 20 minutes.
4. **No "run again everything the previous one claimed".** The coordinator repeats the cheap things; nobody repeats the expensive ones.
5. If something is running late, the coordinator tells {{DUENO}} **before** the cap is reached, with what is missing and a shorter way out.

## 1. The level of the brief: risk × size

The coordinator puts the level on the first line of each brief and chooses the cycle by it. **What** each role works with (class, model, effort and context) is decided by `AGENTES.md`: the level says how many roles join, and the class says how much capability each one carries.

| | **Small** (one topic, few files, tested with a few assertions) | **Large** (new logic, several files or a whole part of a sub-stage) |
| --- | --- | --- |
| **Low risk** (appearance, texts, comments, tests, documents; nothing that can lose, change or expose data) | **Direct:** one agent, or the coordinator if it is trivial. No audit: the coordinator verifies. Budget: {{TOPE_MIN}} min | **Light:** coder and **one** focused audit. Fixes only if there are highs or blockers; the coordinator verifies them. Budget: {{TOPE_MIN}} min per agent |
| **High risk** ({{ZONA_DE_RIESGO}}) | **Focused:** coder and an audit **of only what can fail**, with mutations on that part. Budget: {{TOPE_MIN}} min per agent | **Full:** code → audit → fix → audit again, {{TOPE_MIN}} min per agent |

No budget goes past the cap (point 2): a higher level puts in more agents, not longer agents.

If an agent goes past its budget, it does not carry on in silence: it updates the log with what is missing and why, and the coordinator decides whether to cut or extend. A job that grows in level midway is reported as soon as it is seen.

## 2. Each expensive test, only once

The expensive: {{PRUEBAS_CARAS}}. The cheap: {{PRUEBAS_BARATAS}}.

1. **The expensive ones are run once by whoever delivers the code,** at the end and with the tree finished. The next role does not repeat them: it reads the output and runs them again only if something does not add up or if its brief expressly asks for it. The coordinator repeats only the cheap ones.
2. **Mutations** only for the new tests that guard something of medium risk or more. One pass; they are not repeated if nothing changed afterwards.
3. **Performance** only if the change touches a hot path (code that runs on every keystroke, every request, over a whole list). One alternating run, before and after; measure again only if the difference passes 10 %.
4. **Screenshots** only if what is seen changes, and only those that {{DUENO}} or the auditor need to look at.
5. **Short reports:** what was done, what was measured with its figure, what was not verified, what needs a decision. Without repeating the brief or what another report says: cite it.

## 3. Audit findings: severity × cost to fix

Each finding carries, besides its severity (blocker, high, medium, low), its **cost to fix**, from 1 to 4, counting the change **and** its test:

| Cost | What it means |
| --- | --- |
| 1 | Text, comment or constant; tested by reading or compiling |
| 2 | Local change, up to about 20 lines in one file, with one assertion or one test step |
| 3 | New logic, several files or a measurement that has to be set up |
| 4 | Touches something closed or approved, needs something we do not have, or a decision from {{DUENO}} |

| Severity \ Cost | **Cheap (1–2)** | **Expensive (3–4)** |
| --- | --- | --- |
| **Medium or higher** | **Fix now** | **Plan:** measure before changing; if it depends on something else, postpone with an owner and a date |
| **Low** | **In passing,** if it does not demand a big new test | **Accept or postpone in writing,** with the reason |

Three rules on top of the grid:
1. A **regression** or anything that contradicts what was approved is **medium** at the least.
2. **Secrets, protected data or what the user already wrote:** not postponed because of cost.
3. If the severity or the cost is **unverified**, the box is **measure first**.

Who does what: the auditor puts severity, cost, evidence (measured, read, unverified) and the box they propose, and ends their tally with one line per box. The coordinator confirms or changes it and writes the decision with its reason in the fixer's brief. The fixer returns the real cost for calibration. {{DUENO}} only sees what is theirs: appearance items, what is postponed with medium severity or higher, and what contradicts the specification.

## 4. What is postponed

What is postponed goes to the "Postponed" table of the status file, with an owner and a review date, not into a loose paragraph.
