# Activation and attention

kawa implements `xdg_activation_v1` (`activation.go`). A window that asks to be activated with a valid token is focused and raised, and shown again if it was hidden. A request kawa refuses marks the window as wanting attention instead: a red border, even on client-decorated windows, and a red entry in the main menu if the window is hidden. Attention clears when the window gets focus. Programs started by `New` get a trusted token in `XDG_ACTIVATION_TOKEN` and `DESKTOP_STARTUP_ID`.

## Sub-features

- `act-new-token` — `New` children get a trusted token, so a program that activates with it on startup gets focus.
- `act-launch` — a client that gets a token from a click on its own focused surface and hands it to a program it starts lets that program take focus, even when the request comes before the window maps (kawa remembers it).
- `act-refused` — a token without a seat serial, or from a surface that never had focus, is refused. The window gets the attention border. Clicking the window clears it.
- `act-hidden-attention` — a refused request from a hidden (minimized) window turns its main-menu entry red.
- `act-unhide` — an allowed request for a hidden window shows and focuses it (a single-instance app raised by a second launch).

## How to get to it (user POV)

- Launch an app from another app (a launcher, a link in a terminal): the new window takes focus.
- An app that tries to grab focus without user action gets a red border instead; click it.
- A hidden app that wants attention shows a red entry under `Hide` in the main menu.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven; launched with `-terms "$VERIFY_KAWA_ROOT/.cursor/skills/verify-kawa/scripts/new-cmd.sh"`; `AC=$VERIFY_KAWA_ACTIVATION_CLIENT` (built by `launch.sh`). Usage and modes are in the header of `scripts/activation-client.c`. Each window is a flat color picked from its name, and the client prints `token done`, `activate`, and `keyboard enter/leave` on stdout. Start clients outside `New` with their output in evidence and the PID in `$VERIFY_KAWA_HOME/pids/` (SKILL.md Drive, `bgc`). kawa's stderr logs `activation request for "NAME": allowed=BOOL mapped=BOOL` for each request.

- `act-launch`: `echo "$AC alpha launch $AC charlie start" >"$VERIFY_KAWA_HOME/new-cmd"`, `New` a box, then click inside `alpha`. `charlie` maps with focus and its title in the bar (`$shot act-launch-C`); the log shows `"charlie": allowed=true mapped=false`.
- `act-refused`: `bgc bravo "$AC" bravo self 2`. After 2 s it has a red border and the bar keeps the old title (`$shot act-B-attention`, log `allowed=false mapped=true`). Click it: border gone, focused (`$shot act-B-clicked`).
- `act-hidden-attention`: `bgc delta env ACTIVATION_CLIENT_MINIMIZE=1 "$AC" delta self 2`. kawa hides it on the minimize request; after 2 s the bar menu shows a red `delta` entry (`$shot act-hidden-attention`).
- `act-unhide`: `mkfifo "$VERIFY_KAWA_HOME/echo.fifo"`, `bgc echo env ACTIVATION_CLIENT_MINIMIZE=1 "$AC" echo fifo "$VERIFY_KAWA_HOME/echo.fifo"` (it hides). Then `New` with `$AC writer launch sh -c 'echo "$XDG_ACTIVATION_TOKEN" >'"$VERIFY_KAWA_HOME"'/echo.fifo'` in `new-cmd` and click `writer`. `echo` comes back focused (`$shot act-fifo-unhidden`, log `allowed=true`).
- Keep `$VERIFY_KAWA_HOME/logs/new.log` (the `New` clients' output) in evidence before cleanup.

## Gotchas

- Use names of several letters. Colors come from a hash of the name, and one-letter names give nearly the same gray.
- `self` gets a token with no serial, so it is always refused. `self-stale` uses the last click's serial and is refused once another window has been focused since.
- The attention border is drawn over client decorations, so it is visible on GTK windows too.
- kawa treats a client's minimize request as `Hide`.
