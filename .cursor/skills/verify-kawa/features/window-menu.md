# Window menu / new windows

Right-pressing empty desktop or the status bar opens the main menu: `New`, `Resize`, `Tile`, `Move`, `Close`, `Hide`, then one entry per hidden window. `New` asks for a box, starts the first `-terms` entry that will run (default `sakura`, `alacritty`), and puts the program's window in that box. The box stays on screen until the window maps or the program exits.

## Sub-features

- `main-menu` — right-press on empty desktop, on the status bar, or anywhere with Logo held opens the main menu (`mainMenuText` and the `onMainMenu*` callbacks in `internal/kawa/server.go`, `startMenu` in `internal/kawa/mode.go`).
- `system-menu` — left-press on the status bar opens a menu with only `Log Out`, which shuts the compositor down.
- `new-window` — `New`, then right-drag a box. Once the drag passes the minimum size (128x24), kawa starts the program. The child gets `XDG_ACTIVATION_TOKEN` and `DESKTOP_STARTUP_ID` set to a trusted activation token. Its stdout and stderr go to `/dev/null`.
- `new-box` — the red-bordered box stays drawn after the drag until a window whose PID matches the child maps, or the child exits (`startNew` in `internal/kawa/mode.go`). A program that exits without a window clears the box.
- `new-late-map` — a window that maps after the drag ended takes the box and gets keyboard focus. The next click is not swallowed.
- `new-early-map` — a window that maps while the drag is still going border-resizes with the pointer until release.
- `new-grandchild` — a window opened by some other process (for example a grandchild after the started program exited) does not match the box. It opens centered on the output, on top but unfocused.

## How to get to it (user POV)

- Right-click empty desktop or the status bar to open the main menu. A right-press on a window's server-side border starts a move instead, and a press in a window's content goes to that window.
- Choose `New`, then right-drag out a box for the new window.
- Pass `-terms 'foot,alacritty'` (comma-separated). Each entry is split on whitespace, so an entry can carry arguments, e.g. `-terms 'foot -e htop'`.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven; launched with `-terms "$VERIFY_KAWA_ROOT/.cursor/skills/verify-kawa/scripts/new-cmd.sh"` (absolute path) or with an installed terminal (e.g. `-terms weston-terminal`).

- Open menus from the bar so the item positions are fixed (SKILL.md Drive): `xdotool mousemove --window "$wid" 600 10 mousedown 3`, `$shot window-menu-main`, then move to `y=20+19*I` and release to pick item `I`.
- `new-window`: pick `New` (item 0), then right-drag `xdotool mousemove --window "$wid" 200 150 mousedown 3 sleep 0.2 mousemove --window "$wid" 260 200 sleep 0.2 mousemove --window "$wid" 700 520 sleep 0.3 mouseup 3`. Proof: `ps -o pid,cmd --ppid "$(cat "$VERIFY_KAWA_HOME/pids/kawa.pid")"` lists the program, `tr '\0' '\n' </proc/PID/environ | grep -e XDG_ACTIVATION_TOKEN -e DESKTOP_STARTUP_ID` shows both, stderr logs `new xdg_surface`, and `$shot window-menu-new` shows the window in the box with its title in the bar.
- With `new-cmd.sh`, write the command line to `$VERIFY_KAWA_HOME/new-cmd` before each `New` (default `weston-terminal`). It `exec`s the command so the PID is kept, and logs `new-cmd: pid N: CMD` plus the program's output to `$VERIFY_KAWA_HOME/logs/new.log`.
- `new-box`: `echo "sh -c 'sleep 2; exit 1'" >"$VERIFY_KAWA_HOME/new-cmd"`, drag a box, `$shot new-box-waiting` within 2 s (box still drawn), then `$shot new-box-cleared` after 3 s (box gone).
- `new-late-map`: `echo "sh -c 'sleep 2; exec weston-terminal'" >"$VERIFY_KAWA_HOME/new-cmd"`, drag, `$shot new-late-waiting`, wait 4 s, `$shot new-late-mapped` (window in the box, title in the bar). Then a bar right-press still opens the menu (`$shot new-late-menu`).
- `new-grandchild`: `echo "sh -c '(sleep 2; weston-terminal) & exit 0'" >"$VERIFY_KAWA_HOME/new-cmd"`, drag. The box clears when `sh` exits, and the terminal later appears centered without focus (`$shot new-fork-mapped`).
- `system-menu`: `xdotool mousemove --window "$wid" 600 10 mousedown 1`, `$shot status-bar-left`, then `mousemove --window "$wid" 600 300 mouseup 1` to cancel.
- Never mark verified by calling `onMainMenuNew` from Go tests alone.

## Gotchas

- If no terminal from `-terms` starts, stderr logs `no valid terminals found for new window`.
- Use an absolute path for `-terms` entries that are scripts. kawa starts them from its own working directory.
- Because kawa discards the program's output, a `-terms weston-terminal` failure is silent. Use `new-cmd.sh` when you need the output.
- `Log Out` terminates the display. Releasing button 1 on the bar without moving selects it.
- A menu opened away from the bar puts the last-chosen item under the pointer, shifted to stay on screen, so near the screen edges the item under the pointer is not the one you expect. A press near the bottom-right corner can land on `Hide`.
- If the drag never passes the minimum size, nothing is started.
