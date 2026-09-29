# Window menu / new terminal

Right-click opens the main menu (`New`, `Resize`, `Tile`, `Move`, `Close`, `Hide`). Choosing `New` runs the first working entry from `-terms` (default `sakura`, `alacritty`) via `Server.exec`.

## Sub-features

- `main-menu` — right-click → main menu items from `mainMenuText`.
- `system-menu` — left-click on status bar → `Log Out` (shuts down the compositor).
- `new-terminal` — `New` spawns a terminal from `-terms`.

## How to get to it (user POV)

- Right-click empty space or a window decoration area to open the main menu.
- Select `New`, then drag out a rectangle to place the new terminal.
- Pass `-terms 'foot,alacritty'` (comma-separated list via the flag’s string-list parser) to prefer available terminals.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven; `launch.sh -terms TERM` with an installed terminal when the defaults (`sakura`, `alacritty`) are not installed (e.g. `-terms weston-terminal`).

- Drive with `xdotool` per SKILL.md Drive: right-press → `screenshot.sh window-menu-main` → release on `New` → right-drag a rectangle.
- Proof: the terminal is a child of the kawa PID, stderr logs `new xdg_surface`, and `screenshot.sh` shows it in the dragged rectangle.
- Never mark verified by calling `onMainMenuNew` from Go tests alone.

## Gotchas

- If no terminal from `-terms` starts, stderr logs `no valid terminals found for new window`.
- `Log Out` terminates the display — do not select it during other proofs without intending cleanup.
