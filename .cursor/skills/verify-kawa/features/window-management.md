# Window management

The main menu's `Move`, `Resize`, `Close`, and `Hide` act on a window you pick with a right-click after choosing the item. Windows can also be moved and resized directly: right-press a server-side border to move, left-press it to resize, and client-side decorations (a GTK headerbar) ask kawa to move or resize. Hidden windows get main-menu entries that bring them back.

## Sub-features

- `move` — `Move`, then right-press a window and drag; it follows the pointer until release. Right-press on a server-side border does the same without the menu.
- `resize` — `Resize`, then right-click a window and right-drag a new box. The window takes the box on release.
- `border-resize` — left-press on a server-side border drags that edge or corner (not while tiled).
- `client-move` — a client move or resize request (GTK headerbar drag, CSD edge) starts the same interaction.
- `close` — `Close`, then right-click a window: XDG windows get `close`, X11 windows `WM_DELETE_WINDOW`. kawa reaps its exited children.
- `hide` — `Hide`, then right-click a window: it disappears and its title is added to the end of the main menu. Choosing that entry shows it again.
- `tile-swap` — moving a tiled window onto another tiled window swaps their places (see maximize.md).

## How to get to it (user POV)

- Right-click empty desktop, choose the action, then right-click the window to act on. For `Move` and `Resize` keep the button held and drag.
- Drag a window by its title bar (client decorations) or its border (server decorations).
- Hidden windows are listed below `Hide` in the main menu.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven; one or two windows from `New` (window-menu.md). Open menus from the bar with the `barmenu` pattern in SKILL.md Drive: item 1 `Resize`, 3 `Move`, 4 `Close`, 5 `Hide`, 6 and up hidden windows.

- `move`: pick `Move`, then `xdotool mousemove --window "$wid" 400 300 mousedown 3 sleep 0.2 mousemove --window "$wid" 420 320 sleep 0.2 mousemove --window "$wid" 300 250 sleep 0.3 mouseup 3`; `$shot wm-move`.
- `resize`: pick `Resize`, `xdotool mousemove --window "$wid" 400 300 click 3`, then right-drag a box `mousedown 3 … mousemove … mouseup 3` with one more small `mousemove` just before `mouseup 3`; `$shot wm-resize`. Repeat it to prove it works again.
- `client-move`: left-drag a GTK headerbar in several steps (`mousedown 1`, moves of 10, 30, 100, 50 px, `mouseup 1`), then move the pointer again; the window stays where it was dropped (`$shot wm-client-move`).
- `close`: pick `Close`, right-click the window; `kill -0 PID` fails afterwards and `ps --ppid` no longer lists it (`$shot wm-closed`).
- `hide`: pick `Hide`, right-click the window (`$shot wm-hidden`), open the menu again (`$shot wm-hidden-menu` shows the title at item 6), pick item 6 (`$shot wm-unhidden`).

## Gotchas

- `Resize` shows a box until it passes the minimum size, then switches to resizing the window, which only applies on the next motion. Without one extra pointer motion before release, the window keeps its old size.
- A right-press on a window's border starts a move, not the menu. Open the menu from the bar or empty desktop.
- After `Hide`, the hidden window keeps keyboard focus and its title stays in the bar; keys typed then still reach it. That is a product bug, not harness drift; click another window before typing.
