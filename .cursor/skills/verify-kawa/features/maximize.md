# Maximize / tiling

Windows can be tiled/maximized under the status bar. The main menu item `Tile` runs `toggleViewTiling`. Maximized/tiled views sit under floating windows (wishful-thinking behavior in README, implemented via tiling).

## Sub-features

- `tile-toggle` — menu `Tile` then select a view.
- `request-maximize` — clients may request maximize via xdg/Xwayland hooks on the view.

## How to get to it (user POV)

- Right-click → `Tile` → click the target window.
- Or use a client that requests maximize.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven; a client window present (window-menu `New`).

- Drive with `xdotool` per SKILL.md Drive: right-press on empty space → move to `Tile` → release → right-click the window.
- Proof: `screenshot.sh maximize-tiled` shows the view filling the area under the status bar, not an internal `SetMaximized` call from a unit test.

## Gotchas

- There is no separate menu label `Maximize`; use `Tile`.
- Status bar remains visible when tiled.
