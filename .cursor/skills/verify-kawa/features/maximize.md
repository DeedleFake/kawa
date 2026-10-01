# Maximize / tiling

Tiled windows share the usable area below the status bar (and any layer-shell exclusive zones) in rows, 15 px in from its edges, and sit under floating windows. The main menu item `Tile` toggles tiling for the window you pick; a client's maximize request does the same. Untiling puts the window back where it was.

## Sub-features

- `tile-toggle` — menu `Tile`, then right-click a window. Again on a tiled window restores its old geometry.
- `tile-rows` — several tiled windows split the area side by side (`TiledRows`).
- `tile-csd` — client-decorated windows (GTK) are tiled by their window geometry, so the visible frame fits the tile exactly.
- `tile-swap` — `Move` a tiled window onto another tiled window to swap them.
- `request-maximize` — a client maximize request (GTK headerbar double-click, `xdg_toplevel.set_maximized`) toggles tiling.
- `tile-zones` — layer-shell exclusive zones shrink the tile area (layer-shell.md).

## How to get to it (user POV)

- Right-click empty desktop → `Tile` → right-click the target window.
- Or double-click a client-decorated window's title bar, or use its maximize button if it has one.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven; one window from `New` (window-menu.md), and a GTK window (`gtk3-demo` via `new-cmd.sh`) for `tile-csd` and `tile-rows`.

- `tile-toggle`: from the bar, pick `Tile` (item 2), then `xdotool mousemove --window "$wid" 480 380 click 3`; `$shot maximize-tiled`. Repeat to untile; `$shot maximize-untiled` shows the old position and size.
- `tile-rows` and `tile-csd`: tile a second, GTK window; `$shot maximize-two-tiled` shows both side by side with the GTK frame filling its tile.
- `tile-swap`: pick `Move` (item 3), right-press the left tile, drag onto the right one and release; `$shot maximize-tile-swap` shows them swapped.
- `request-maximize`: `New` a `gtk3-demo` window, double-click its headerbar (`xdotool mousemove --window "$wid" X Y click --repeat 2 --delay 80 1`); it tiles (`$shot maximize-request`). Double-click again; it returns to its box (`$shot maximize-request-untiled`).
- Proof is the screenshot, not an internal `SetMaximized` call from a unit test.

## Gotchas

- There is no menu label `Maximize`; use `Tile`.
- The status bar stays visible when tiled.
- Border resize does nothing on a tiled window.
