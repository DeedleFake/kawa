# Status bar

A bar across the top of the primary output shows the focused window's title, kept up to date when the title changes. Pressing on it opens kawa's menus. README plans a clock; it is not implemented.

## Sub-features

- `bar-chrome` — bar region (`StatusBarHeight`) drawn each frame on the status-bar output.
- `focused-title` — the bar shows the focused window's title and follows title changes. It is empty while a layer surface (a launcher) has the keyboard.
- `bar-menus` — left-press → system menu (`Log Out`); right-press → main menu.
- `time-display` — planned in README; not implemented.

## How to get to it (user POV)

- Start kawa; look at the top of the compositor window.
- Focus a window, or let it change its title; the bar follows.
- Left-click the status bar for the system menu (`Log Out`); right-click it for the main menu.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven.

- `bar-chrome`: `$shot status-bar-boot` after boot shows the bar across the top.
- `focused-title`: map a window (window-menu `New`), then `$shot`; the bar shows its title. A GTK app that changes its title (clicking entries in `gtk4-demo`) changes the bar.
- `bar-menus`: `xdotool mousemove --window "$wid" 600 10 mousedown 3` → main menu (`$shot status-bar-right`); `mousedown 1` → `Log Out` (`$shot status-bar-left`). Move below the menu before releasing to cancel (`$shot after-bar-cancel`).
- `time-display`: not implemented; report it as such.
- Do not call `SetTitle` from a test helper and call that proof.

## Gotchas

- Title updates are render-path side effects; there is no log line to grep.
- Any press with Y ≤ `StatusBarHeight` (or with Logo held) opens a menu: left → system, right → main (`mode.go`).
- Releasing button 1 without moving selects `Log Out` and shuts kawa down.
- A layer-shell panel (waybar) is placed below kawa's bar, not over it.
