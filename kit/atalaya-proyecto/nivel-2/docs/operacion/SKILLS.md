# Skills, plugins and aids · {{PROYECTO}}

Which aids **really** exist on this machine, when they are worth it and when they are not.

- **The exact names** are those on the session's skill list: they are not guessed. If a name here does not appear on the list, say so and do not use it.
- **The usage log** (what helped and what did not) is in `HERRAMIENTAS-registro.md`, part 2.
- **This file is born almost empty on purpose:** every team has different plugins. The installer fills it with what {{DUENO}}'s session really offers, and not before.

## 1. How to decide

1. **Does the task fall in any row below?** If not, none is used.
2. **Does the benefit outweigh the cost?** A skill loads its instructions into the context (from hundreds to thousands of tokens). For a one-line check it does not pay off.
3. **The decision is noted** in the report ("X was used because…" or "none, because…"). If it taught something, a line goes in the log.

**Who invokes a skill:** the coordinator, always; an agent, if it has the `Skill` tool and the skill appears on its list. If not, it says so in its report and the coordinator runs it on its delivery.

**Looking for new skills** (only when a task falls in no row): with the words of the task and of the real stack, not with generic searches. Quality filter: more than 1,000 installs, a known source and a repository with more than 100 stars; anything else, with distrust. **It must not clash:** if it does the same as one already there, it is not proposed; it cannot contradict `AGENTS.md` §4. **Nobody installs anything without {{DUENO}}'s yes**, and an agent never installs. Every search leaves a line in the log, also when it found nothing useful.

## 2. Critical (always used)

| Aid | What for | Caution |
| --- | --- | --- |
| Read, Edit, Write, Grep, Glob | Reading, changing and searching | Rules in `HERRAMIENTAS.md` §2 |
| Bash and the machine's console | Commands, scripts, git | `HERRAMIENTAS.md` §4 and §5 |
| Agents (`.claude/agents/`, `Explore`, `general-purpose`) | The swarm by role and wide searches | Only for what is big and parallelizable (`PRIORIZACION.md` §0) |
| Monitor | Waiting for an end file without a long `sleep` | `HERRAMIENTAS.md` §6 |
| The `sin-heredoc.mjs` hook | Stops heredoc and `sed -i` in Bash | Only in Claude Code and with the session open in the repository |
| Project memory and `AGENTS.md` | Continuity between sessions, and rules | `AGENTS.md` §5 |

## 3. Highly recommended (if they exist in the session)

| Aid | Use it when | Do not use it for |
| --- | --- | --- |
| `code-review`, `security-review` | When closing a release; changes to session, permissions, encryption or protected data | Appearance-only or document-only changes |
| `superpowers:verification-before-completion` | Before calling something done | — |
| Migration or schema review skills | A new schema change | Migrations already applied, which are not edited |
| `gh` (in the console) | Querying the remote and the state of branches | Publishing to `{{RAMA_PRODUCCION}}` without {{DUENO}}'s yes |

## 4. Useful in specific cases

| Aid | Use it when | Caution |
| --- | --- | --- |
| Design and accessibility skills | New screens, mockups, notice texts | The approved identity rules |
| Debugging and testing-strategy skills | A failure that does not reproduce the first time; deciding what to test | Errors with a clear message |
| Architecture skills | A new piece with data or boundary decisions | What is already decided in the specification |
| `simplify` | After a round of fixes | Finding defects: it does not look for them |
| `update-config` | Changing hooks or permissions of `.claude/settings.json` | Only with {{DUENO}}'s yes |
| WebSearch and WebFetch | External sources | Sites that answer 403/418: `curl` (`HERRAMIENTAS.md` §10) |

## 5. Do not apply to this project

[List of plugins and connectors that the team has but this project does not use. The installer writes it after seeing the real list.]

## 6. What no longer exists

[Names that some old document mentions and that are no longer there. Empty at install time.]
