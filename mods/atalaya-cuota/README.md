# atalaya-cuota

A Claude Code mod for [Atalaya](../../README.md). Atalaya reads your plan's usage with `claude -p /usage` every 10
minutes. With this mod installed, Claude Code also tells Atalaya the moment the 5-hour or the weekly window moves a
whole point, so the quota strip and its chart in the panel are never more than one point behind.

It listens to `session.measure` and sends `POST http://127.0.0.1:7420/api/uso-claude/empujar` with the two windows
(`five_hour`, `seven_day`): their percentage and when they reset. Nothing else leaves the session, nothing goes beyond
your own computer, and if Atalaya is not running the request simply fails without affecting the session. The
per-model weekly window still comes from `/usage`.

## Install

`<owner>/atalaya` is this repository as GitHub shows it in its address: the account it lives under, then `/atalaya`.
At the prompt of a terminal session:

```
/plugin install atalaya-cuota --marketplace <owner>/atalaya
```

Answer `y` to add the marketplace, then pick the user scope so it runs in every session. From a shell:

```
claude plugin install atalaya-cuota --marketplace <owner>/atalaya
```

In the desktop app, where `/plugin` does not run: click + next to the prompt box, then Plugins, Add marketplace with
`<owner>/atalaya`, then Add plugin.

## Español

Un mod de Claude Code para Atalaya. Atalaya lee el uso de tu plan con `claude -p /usage` cada 10 minutos; con este mod,
Claude Code además le avisa en el momento en que la ventana de 5 horas o la semanal se mueven un punto entero. Solo
manda esas dos ventanas a `127.0.0.1:7420`, nada sale de tu computadora, y si Atalaya no está corriendo la sesión sigue
igual. Se instala con la misma línea de arriba (`/plugin install atalaya-cuota --marketplace <owner>/atalaya`),
con el alcance de usuario para que corra en todas tus sesiones.
