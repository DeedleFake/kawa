# Status bar

A top status bar is drawn on the primary output. The focused window title is pushed into the bar via `StatusBar.SetTitle`. README plans a clock/time display; that is not separately logged today and needs a visual capture later.

## Sub-features

- `bar-chrome` — bar region (`StatusBarHeight`) rendered each frame on the status-bar output.
- `focused-title` — bar title tracks the focused view title.
- `time-display` — planned (README); not asserted by log today.

## How to get to it (user POV)

- Start kawa; look at the top of the compositor window.
- Focus a window; the bar title updates.
- Left-click the status bar for the system menu (`Log Out`); right-click it for the main menu.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven.

- `bar-chrome`: `screenshot.sh status-bar` after boot shows the bar across the top.
- `focused-title`: map a window (window-menu `New`), then `screenshot.sh`; the bar shows its title.
- Bar menus: `xdotool mousemove --window "$wid" 600 10 mousedown 3` → main menu; `mousedown 1` → `Log Out`. Screenshot each, then move below the menu before `mouseup` to cancel.
- `time-display`: not implemented; report as such.
- Do not call `SetTitle` from a test helper and call that proof.

## Gotchas

- Title updates are render-path side effects; there is no stdout clock line to grep.
- Any press with Y ≤ `StatusBarHeight` (or with Logo held) opens a menu: left → system, right → main (`mode.go`).
- Releasing button 1 without moving selects `Log Out` and shuts kawa down.
