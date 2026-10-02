# Popups

XDG popups (context menus, dropdowns) are configured on their first commit and unconstrained to the usable area of the parent window's output (`view.go`), so a menu opened near the bottom or right edge flips or slides to stay on screen and below the status bar.

## Sub-features

- `popup-map` — a popup from a toplevel maps at its requested position.
- `popup-unconstrain` — a popup that would leave the usable area is flipped or slid back onto it.

## How to get to it (user POV)

- Right-click inside a GTK app near the bottom-right corner of the screen; the menu opens up and to the left instead of off screen.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven with `new-cmd.sh`; `gtk3-demo` installed.

- `echo gtk3-demo >"$VERIFY_KAWA_HOME/new-cmd"`, `New` a box from 420,380 to 1020,760 so the window touches the bottom-right corner (`$shot popup-gtk3-mapped`; the client-decorated window fits the box exactly).
- Right-click in its source text view near the corner: `xdotool mousemove --window "$wid" 990 740 click 3`; `$shot popup-unconstrained` shows the context menu above and to the left of the pointer, fully on screen. Press Escape to close it.

## Gotchas

- A click on empty desktop does not dismiss an open popup (kawa sends no button event there). Use Escape.
- Close popups before driving keyboard focus elsewhere; an open popup grab keeps the keyboard (see layer-shell.md).
