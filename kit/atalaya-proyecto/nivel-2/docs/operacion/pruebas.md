# Testing guide · {{PROYECTO}}

What to run and which traps have already been stepped on. This file is born with **the principles** that held in the originating project; the concrete commands for {{PROYECTO}} are filled in by the installer (§2) and completed by the team with use. Each new rule carries in parentheses the log entry it came from (`HERRAMIENTAS-registro.md`).

## 1. Cheap and expensive

- **Cheap:** {{PRUEBAS_BARATAS}}.
- **Expensive:** {{PRUEBAS_CARAS}}.
- **The expensive ones are run only once by whoever delivers the code** (`PRIORIZACION.md` §2). When integrating, the coordinator runs the complete suite **of the cheap ones**, not the expensive one: whoever delivered already ran it; the coder runs only their own packages.
- **If an approved rule changes,** its test is brought up to date in the same commit, not when preparing a release.

## 2. This project's test suites

| Suite | Command | Takes | Notes |
| --- | --- | --- | --- |
| {{NOMBRE_DE_LA_BATERIA}} | {{ORDEN}} | {{DURACION}} | |

**Each suite runs alone:** no agent working, no other load, and two suites never together. A test that fails in the suite and passes alone is repeated first with the machine idle and with several repetitions, before opening a diagnosis.

## 3. Mutations

The code is broken on purpose and some test must fail; if it "survives", the test is useless. Only for new tests of medium risk or more, one pass.

1. **Never in the repository:** in a copy in the working folder (or with the language's overlay facility, if there is one). Always one **control mutation that must die**.
2. **Check that the mutated build was compiled.** If the mutation breaks the types, the build fails, the earlier good output stays and the test "passes" with the unmutated code. After mutating, the build must finish without error before running the test.
3. **After a cut-off halfway, recompile:** the build output may still be compiled with the mutation.
4. **The program that runs the tests and counts mutants** decodes the output as UTF-8, runs the **unmutated baseline** first, and also counts the **cancelled** tests (a mutant that hangs is dead, but it is reported as having died by timeout).
5. **A mutation with only one of two defenses survives for good reason:** mutate both together and note that each one separately is equivalent.

## 4. Screenshots

- Screenshots are taken only if what is seen changes, and only those that the owner or the auditor need to look at.
- A variable that asks for "a screenshots folder" is a folder, not a switch.
- With the complete suite, the screenshots of all phases get rewritten: use only the filter for your own steps, and before delivering, `git status` on `capturas/` shows no files that belong to others.

## 5. Mock servers and seeds

- Mocks are entirely fictional and listen only on `127.0.0.1`, on free high ports.
- **Before seeding a record** in a mock, search (Grep) whether an old step uses it: if not, an old step that types it as "new" starts saying "already exists".
- **New route that the interface queries:** add it to the mock, or it answers 404 and the suite notes "console errors that the step did not expect".
- Test cleanups delete **by exact value, never by pattern**, and count afterwards.

## 6. Browser traps (if the project has a web interface)

- **The first key is lost** if it arrives before hydration: the waits retry the action, not just the check.
- **Notices cover clicks:** a notice enters after mounting and pushes down what is below. Before a click in a table, close the notices.
- **A click that retries hides a lost click.** For a click that must land on what is visible now: measure and click without retry, with a pause between press and release.
- **The open dialog is `dialog[open]`,** not the first `<dialog>`.
- What is read from the DOM does not come back as it was written (`outline: 3px solid …` → `rgb(…) solid 3px`): compare it by parts. Inside a scaled container, the measurements come out scaled.

## 7. Database traps (if there is one)

- **An old test database keeps what the code retired.** If a catalog test fails on one PC and not on another, search for the name in the git history.
- **New table with a foreign key:** add it to the `TRUNCATE` of the load tests.
- **Locks:** to see who is waiting, look at the locks view, not the activity view.
- **The temporary cluster** is created with messages in a fixed language (`C`), because it takes the Windows one and a test that looks for the text of an error fails.

## 8. "A test that did not run did not pass"

**Zero tests run is a filter failure, not an "all good".** After each run with a filter, check how many ran.

1. **The filter is copied with its accents.**
2. **A `.ps1` written with the file tool is UTF-8 without BOM** and PowerShell 5.1 reads it as ANSI: a filter with non-ASCII characters runs 0 steps without warning. Inside the `.ps1`, write the non-ASCII as `[char]0x00B7`; and check the syntax before running it.
3. **Before trusting a 0,** check that the detector finds something known to exist. Do not throw to `/dev/null` the output of what will be read afterwards.
4. **Without the variable that enables the database tests,** "ok" can mean "skipped". Count the skipped ones.
5. **Invisible or ambiguous characters** in a test (U+0301, U+00A0, U+2028): they go as an escape.
6. **A declared result** (a `verificacion.json`, a screenshot from another role) **is not a test you just ran.**
