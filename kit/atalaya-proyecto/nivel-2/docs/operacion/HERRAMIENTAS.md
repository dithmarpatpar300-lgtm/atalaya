# Tools · what works on this machine · {{PROYECTO}}

What is said here to fail **is not attempted**: it is not retried "just in case". Every rule comes from a measured failure.

- **The history of each rule,** with its date, is in `HERRAMIENTAS-registro.md` (the tools log). The general rules are in `AGENTS.md`.
- **Main tool:** Claude Code. Machine: {{SISTEMA_Y_CONSOLA}}.
- **Tests:** `pruebas.md`. **Production:** {{DOC_DE_PRODUCCION}}.
- **This file is born with the lessons of the originating project** (Windows 11, Git Bash and PowerShell 5.1). The installer deletes what does not apply to your machine and adds your own; new findings are noted first in `HERRAMIENTAS-registro.md` and move up here when they repeat.

## 1. Datapedia (optional)

If the project has an index or search tool for its documentation, it is consulted when the brief does not bring the reading list and before answering about something already decided. What it returns is checked against the file. {{ORDEN_DE_LA_DATAPEDIA}}

## 2. Reading and writing files

- **Read to read,** with `offset` and `limit` if the file is long (the cap is about 25,000 tokens). Independent reads go in the same message.
- **Before an Edit:**
  - a **Read of the file by the same path** and in this conversation. After compacting the conversation, changing paths (including a project move) or a `git pull`, the section is read again;
  - **Grep, `cat` and `sed -n` do not count as a read;**
  - the anchor is **a single line**, copied as is (tabs and spaces included; without the line-number column);
  - **do not mix** `sed` and Edit on the same file.
- **`replace_all` on a short class or word** also touches the ones that contain it (`p-6` is inside `gap-6`): replace it with its context, or line by line.
- **Writing files with heredoc, `sed -i`, `perl -i`, `Set-Content` for code, `python -c` or `node -e` is forbidden.**
  - In Bash a project hook stops it: `.claude/hooks/sin-heredoc.mjs`. If it triggers, that is not a failure: write the program with Write in the working folder and run it, or use Edit.
  - Still allowed: `printf … > file` for the log, the here-string `<<<` and `git commit -F - <<'EOF'`.
- **Escapes:** invisible or ambiguous characters (U+00A0, U+2028, a combining accent) always go as an escape in the code, and are verified with `od -c` after writing them.
- **Python on Windows** writes CRLF in text mode, and `-I` ignores `PYTHONIOENCODING`. For accented characters in the output: `PYTHONIOENCODING=utf-8` without `-I`.

## 3. Scratch files: create, use, leave a record, destroy

Everything temporary goes in `{{TMP}}`; it does not go into the repository or the system temp folder, which is often shared between agents.

**The owner's rule.** Scratch files leave gaps: forgotten folders, live processes, tests with no trace. So **nothing temporary is left without a record or without being destroyed**, and the record is saved **before** deleting.

**The cycle, always in this order:**
1. **Create.** One subfolder per task: `{{TMP}}\<yyyy-mm-dd>-<role>-<topic>\`. The coordinator too. Nobody touches another's. It is created just before use, not "just in case".
2. **Open the record,** at the same moment: `docs/trazabilidad/<phase>/temporales/<yyyy-mm-dd>-<role>-<topic>.md` (template in `docs/trazabilidad/_plantillas/ACTA-PLANTILLA.md`), with the reason. It is completed **in parts, from early on**: an agent cut off halfway loses everything it had not written down.
3. **Use.** Helper scripts, copies for mutations, mock servers, browser profiles, temporary clusters and worktrees, long outputs and drafts go here.
4. **Close the record before deleting.** It holds, in this order:
   - the reason and the base (`{{RAMA_TRABAJO}}` at `<hash>`);
   - what was created: folders, processes, ports, databases, worktrees;
   - what was done and run, each with its command and its result **in figures** (counts, times, passed or failed), not the whole output;
   - what survives and where it went: report, `deploy/`, `scripts/` or memory;
   - what failed or taught something (that line also goes to `HERRAMIENTAS-registro.md`);
   - what was not verified.
5. **Destroy** with the literal path (with variables, the deletion is blocked: `rm -rf "$VAR/…"` in Bash, `rd` or `Remove-Item` with `$variable` in PowerShell). First close your own processes and ports, **finding them by the folder path in their command line** (a background process stays alive even when it is no longer needed). A folder with half-finished work is not deleted: complete the record or ask the owner.
6. **Confirm.** `Test-Path` (or `ls`) says it does not exist, no process of your own is left and no port is listening, and at the end of the record you add `Deletion confirmed: <date and time> · folder absent · 0 processes · 0 ports`. The time comes from the clock (`Get-Date`, `date`), not from memory.

**Around the cycle:**
- **Never goes there,** not even for a moment, and the record does not copy it either: secrets, keys, backups or protected data.
- **Scope:** it applies to `{{TMP}}`, the release worktrees (`version-y-kit.md` §2), the system temp folder (for example the browser profiles of the tests) and the working folder (scratchpad) of the Claude Code session.
- **The task report** names the record with its path; saying "I deleted my folder" is no longer enough.
- **If there was a cut-off** (usage limit, agent stopped, crash), the first thing on resuming is the gap: read what was left in the folder, write the record marking it "reconstructed" and only then delete.
- **Who watches:** the coordinator looks at `{{TMP}}` when opening each session and when closing each sub-stage (`AGENTS.md` §2 and §7). Any folder without a closed record is reported to the owner.
- **What is reused does not live here.** What must survive goes to the repository (report, `deploy/`, `scripts/`) or, if it is from one session to another, to memory.

## 4. Bash

- **Commands are under about 6 KB.** Anything longer is written with Write in the working folder and run.
- **`\\` arrives as `\`:** no double backslashes inside a command.
- **A check that ends in `grep` with no matches** gives "Exit code 1": add `; true`.
- **`find` only in the folders that matter.** `-not -path` does not prune: it walks all of `node_modules`. The same for `du` on huge folders: on Windows, measure with PowerShell and `Measure-Object`.
- **Long output** is ended with `| head -60` or `| tail`.
- **Paths for Node on Windows:** `/d/…` reaches `node` wrongly; use `D:/…`.

## 5. PowerShell 5.1 (if the machine is Windows)

- **No prose inside the command:** a word that is also an alias, such as `del` (in Spanish, the common word "del"), is taken as the delete command and gets blocked. There is no `&&` either: use `; if ($?) { … }`.
- **A `Remove-Item` inside a longer command** makes the brake block the whole command (even with the `/L` of a `robocopy` before it): the deletion goes separately, with a literal path.
- **Query commands end with `exit 0`:** the odd exit code is usually the last executable's.
- **No `Get-ChildItem -Recurse`** over the whole repository (`node_modules`, worktrees). Summarize per folder, or use `Test-Path`.
- **Encoding:**
  - without a BOM, PowerShell reads as ANSI: to read, `Get-Content -Encoding UTF8`;
  - a `.ps1` is ASCII or, if it has accented characters, UTF-8 **with** BOM. Check with `[regex]::Matches($t,'[^\x00-\x7F]').Count`;
  - `*>` and `Tee-Object` may write UTF-16 or UTF-8 with BOM: look at the encoding before reading.
- **Syntax:**
  - `"${variable}:"` inside double quotes (`"$variable:"` is read as a variable with a drive);
  - every script of your own is checked before running it: `$t = $null; $e = $null; [System.Management.Automation.Language.Parser]::ParseFile($path, [ref]$t, [ref]$e)`;
  - a function of your own is not named `H` or with one or two letters (aliases win);
  - `.Substring(0,[Math]::Min(N,$s.Length))`; check for `$null` before calling a method;
  - variables are case-insensitive (`$t` overwrites `$T`);
  - `$ErrorActionPreference = 'Stop'` cuts a script on the normal warning that `git` writes to stderr: in scripts that call programs, use `Continue` and look at `$LASTEXITCODE`.
- **Executables:**
  - do not use `2>&1` on them: PowerShell turns their error output into `NativeCommandError`;
  - a function that calls an executable sends its output to `| Out-Host` and returns only `$LASTEXITCODE`;
  - `Start-Process -ArgumentList` splits an argument with spaces: pass it quoted inside the string;
  - `Start-Process … -Wait` on a service that never ends hangs: start it without `-Wait` and wait with the check that it is listening.
- **`[IO.File]::ReadAllText`** uses the console's starting directory, not that of `Set-Location`: always with an absolute path.
- **A freshly extended PATH** does not reach a console opened earlier: `$env:Path = [Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [Environment]::GetEnvironmentVariable('Path','User')`.

## 6. Waits and background processes

- **`sleep N; command` with N of 25 or more is blocked** in Claude Code (measured: 24 passes, 30 does not). To wait, use `run_in_background` or the **Monitor** tool with an `until` loop that signals when the end file appears.
- **`run_in_background` with an `&` inside** ends as "completed" instantly even though the process is still alive: the whole command goes without `&`.
- **Anything that takes more than 2 minutes** is launched in the background, and the script writes a `FIN-*.txt` (end marker) when it finishes.
- **Before relaunching a test suite, kill the previous one entirely:** the parent process and its children, finding them by the folder in the command line.
- **When finished, none of your own processes stays alive.** Check it (§3, step 6).

## 7. Browser (if the project has a web interface)

- **The built-in one:** start with `navigate` to a real address (`http://127.0.0.1:PORT/…`; `about:blank` and `data:` do not work). For `next dev` use `localhost`.
  - If the panel is hidden, screenshots time out and there are no real keys: read with `get_page_text`, `find` or `javascript_tool`, and the report says there were no real keys.
  - `computer type` does not fire `keydown`: for anything that reacts to each key, use `computer key`.
- **For real keys and reliable screenshots:** a windowless browser driven by the debugging protocol. **No browser that you did not open is closed.**
- **A PDF cannot be viewed with Read** if there is no `pdftoppm`: read its text, or print it and open it in the same windowless browser. Nothing is installed without the owner's decision.
- **Reserved ports** (the only list; agents use none unless their brief says so, and their mock servers go on free high ports):

  | Port | Whose |
  | --- | --- |
  | {{PUERTOS_RESERVADOS}} | |

## 8. Git and `gh`

- **Commits:** the coordinator makes them, on `{{RAMA_TRABAJO}}`.
  - With `git add` of your own files, never a blind `-A`: there may be untracked files that belong to others.
  - Before the commit, look at what is staged (`git diff --cached --stat`) and that nothing sensitive goes in.
- **The commit message:** in Bash, `git commit -F - <<'EOF' … EOF`; in PowerShell, the message goes to a temporary file and you use `git commit -F <file>`.
- **"Dubious ownership"** (the repository was created by another account or copied from another PC): solve it per command, `git -c "safe.directory=<path>" …`, without touching the global configuration; or by copying the project with the current account.
- **Copying or moving large folders:** `robocopy source destination /E /COPY:DAT /DCOPY:DAT /R:1 /W:1 /NFL /NDL /NP /MT:16 /XJ` (code 1 means "copied"). Verify with `robocopy … /L` (0 differences) and the SHA-256 fingerprint of a few key files; the original is not deleted until the owner says yes.

## 9. Databases

- **The key** is read from `.env` into a process variable, without printing it. Never `cat` or `type` the `.env`.
- **Before assuming the name of a table or column,** query the catalog.
- **Cleanups delete by exact value, never by pattern** (a `LIKE` deleted real data from a test database), and count afterwards.
- **No test data is created in the development database;** the tests use their own database, whose name the code verifies.

## 10. Web and documents

- **WebFetch gives 418 or 403 on some official sites and PDFs.** Read with `curl` and a browser User-Agent.
- **An external standard or figure** is cited from its current official source, with the date it was read.

## 11. Every tool failure goes to review

It applies to the coordinator and to each agent.

- **What counts as a failure:** a call that returns an error, gets blocked, times out or delivers useless output (cut off, too long, empty when it should not be).
- **What the one who suffers it does:** does not repeat the same call. Looks for the cause, finds the command that does work and notes it: an agent, in its report, under "what you learned about tools"; the coordinator, at the end of part 1 of `HERRAMIENTAS-registro.md`.
- **What the line says:** the tool, what was attempted, the message, the cause (**verified** or **assumed**) and the replacement command, written so that someone else can copy it. If it already had a rule and happened again, it carries the word **repeated**.
- **When closing each sub-stage,** the coordinator compares the log and **moves up here** what repeats. If a written rule is not enough (the same failure in several agents in a row), they propose to the owner an automatic brake, like the hook in §2.
