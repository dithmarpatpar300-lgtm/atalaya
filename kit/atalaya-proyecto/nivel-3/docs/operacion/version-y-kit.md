# Version and delivery · {{PROYECTO}}

The step-by-step for cutting a version and delivering it. Sections 1 to 4 are for a project that is delivered **as a
package** (an executable, an installer, a zip). If the project is **deployed** (a website, a service), the section "If
the project is deployed" applies and the others are deleted on install.

- **The rules around it are in `AGENTS.md`:** delivery to {{DESTINO_DE_ENTREGA}}, scripts with `-Simular` (dry run), never `{{RAMA_PRODUCCION}}` without {{DUENO}}'s yes.

## If the project is deployed

1. **Before:** the code on `{{RAMA_TRABAJO}}`, committed; the cheap tests pass ({{PRUEBAS_BARATAS}}) and the expensive
   battery was run once by whoever delivers (`pruebas.md`). New migrations, with their rollback written.
2. **The version:** the number goes up wherever the project keeps it, and the commit gets a tag (`v<version>`) and its
   notes.
3. **The rehearsal:** the deployment is first run as a dry run (`-Simular` or the tool's equivalent) and you read what
   it would do. If the project has a test environment, it is deployed there and what is new is checked.
4. **Production, with {{DUENO}}'s yes:** it moves to `{{RAMA_PRODUCCION}}` and is deployed with the real command, never
   in the hours {{DUENO}} has forbidden.
5. **The check:** that it answers, that it carries what is new (a text or a number specific to the version) and that
   the logs show no new errors. If something fails, the rollback written in step 1, without improvising.
6. **The close-out:** the version notes in the project status and, if the tester took part, its report.

## 1. Before starting

- **Anything visual has its approved mockup,** and the code is on `{{RAMA_TRABAJO}}`, committed.
- **The cheap tests pass:** {{PRUEBAS_BARATAS}}.
- **The expensive ones are run once** by whoever delivers (`pruebas.md`).
- **When raising the version,** before the long battery, look for the old version written in the tests (`Grep` for the old number): several tend to carry it.
- **New migrations** are stated in the version notes, with their rollback.

## 2. Build from a clean tree

1. **A clean worktree** of the version's commit, in `{{TMP}}\release-<version>\`, with no unsaved changes (with its record: `HERRAMIENTAS.md` §3).
2. **The product:** {{COMO_SE_COMPILA}}. Check:
   - that the binary comes out of an unmodified tree;
   - that it starts (`-help` exits with code 0);
   - that it carries what is new, by searching for a text specific to the version.
3. **The fingerprint:** SHA-256 of the product, kept next to it.
4. **When done,** the worktree's record is closed and only then is it deleted, with the literal path.

## 3. Assemble the kit

**The working copy** goes in {{TMP}} or in the project's kits folder. **It carries:** the product and its fingerprint; the install and update scripts, with the new fingerprint (only that fingerprint and its text change); the guide with its new part; a `README-<version>.txt`.

**What the README says:** what the version brings; what the owner does that night, in order; how to go back and what to watch when doing so; the migrations, or "no migrations".

**The kit check, before delivering it:**
- **Nothing forbidden:** 0 secret files (`.env*`, keys, certificates), database dumps or backups, real data, or logs (`.log`).
- **No secret:** no `PRIVATE KEY` and no long base64 strings.
- **The syntax of each `.ps1`:** `Parser::ParseFile`, with `$t = $null; $e = $null` before it.
- **The fingerprints written** in the README, the guide and the scripts match the calculated ones.

## 4. The rehearsal

- **An independent agent does it,** one that did not assemble the kit and does not fix it: the `ensayador` role (the tester).
- **Its verdict:** "ready to install" or "not ready", in its report (the next number after the version's).
- **What is always rehearsed:**
  1. going from the installed version to the new one: code 0, identical configuration and migrations up to date;
  2. repeating it: nothing changes;
  3. going back by hand and returning;
  4. checking that what is served carries what is new;
  5. the cheap checks of §3, on the delivery folder.
- **What is not rehearsed** (always said in the report): whatever exists only in real production, real accounts, real scheduled tasks and real external systems.
- **The findings** come with severity × cost (`PRIORIZACION.md` §3). If medium or more, it is fixed and what changed is rehearsed again.

## 5. Deliver

- **Destination:** {{DESTINO_DE_ENTREGA}}, in two forms: the folder `Kit {{PROYECTO}} <version>` and its `.zip` with the contents **at the root** (no top folder) and entries with `/`.
- **Check** that the zip opens and that the folder, the zip and the working copy have the same fingerprints (0 differences).

## 6. Close out

1. **The version report** (`docs/trazabilidad/<phase>/NN-release-<version>.md`): what changed, tests with figures, the kit with its fingerprints, the rehearsal and its result, what was not verified and what was learned about tools.
2. **The status to resume,** up to date: only the current section.
3. **Commit on `{{RAMA_TRABAJO}}`** with the version, the fingerprints and the rehearsal result; then push that branch.
4. **Publish on `{{RAMA_PRODUCCION}}`,** with a tag and a release: **only when {{DUENO}} asks**.

## 7. What the owner does (not the coordinator)

- Install with the `README` in the real environment; test with real data; say whether it is published.
