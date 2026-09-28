# Window menu / new terminal

Right-click opens the main menu (`New`, `Resize`, `Tile`, `Move`, `Close`, `Hide`). Choosing `New` runs the first working entry from `-terms` (default `sakura`, `alacritty`) via `Server.exec`.

## Sub-features

- `main-menu` — right-click → main menu items from `mainMenuText`.
- `system-menu` — right-click on status bar → `Log Out` (shuts down the compositor).
- `new-terminal` — `New` spawns a terminal from `-terms`.

## How to get to it (user POV)

- Right-click empty space or a window decoration area to open the main menu.
- Select `New`, then drag out a rectangle to place the new terminal.
- Pass `-terms 'foot,alacritty'` (comma-separated list via the flag’s string-list parser) to prefer available terminals.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven; nested session with a pointer injection tool (ydotool/wtype/etc. or compositor-specific test input). **Not proved in the first skill run.**

- Map the path: boot → inject right-click → select `New` → confirm a child process from `-terms` and a new xdg/Xwayland surface.
- Until an input tool is wired, report skip: requires nested Wayland + pointer injection.
- Never mark verified by calling `onMainMenuNew` from Go tests alone.

## Gotchas

- If no terminal from `-terms` starts, stderr logs `no valid terminals found for new window`.
- `Log Out` terminates the display — do not select it during other proofs without intending cleanup.
