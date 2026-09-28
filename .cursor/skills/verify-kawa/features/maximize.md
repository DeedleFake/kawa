# Maximize / tiling

Windows can be tiled/maximized under the status bar. The main menu item `Tile` runs `toggleViewTiling`. Maximized/tiled views sit under floating windows (wishful-thinking behavior in README, implemented via tiling).

## Sub-features

- `tile-toggle` — menu `Tile` then select a view.
- `request-maximize` — clients may request maximize via xdg/Xwayland hooks on the view.

## How to get to it (user POV)

- Right-click → `Tile` → click the target window.
- Or use a client that requests maximize.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven; pointer injection; a client window present. **Deferred** for the first skill run.

- Proof would be: tiled geometry under the status bar (screenshot or layout query), not an internal `SetMaximized` call from a unit test.

## Gotchas

- There is no separate menu label `Maximize`; use `Tile`.
- Status bar remains visible when tiled.
