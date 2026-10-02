# Xwayland

kawa starts Xwayland at startup and runs X11 programs as ordinary windows with kawa's own border and title bar. X programs started by `New` land in the New box (kawa matches them by PID), `Close` sends them `WM_DELETE_WINDOW`, and the clipboard and primary selection are shared between X and Wayland clients through the seat.

## Sub-features

- `xwayland-start` — stderr logs `Running Xwayland on DISPLAY=:N`. kawa exports that `DISPLAY`, so everything it starts, `New` included, talks to Xwayland.
- `xwayland-new` — an X program started by `New` maps in the box with server-side decorations and its title in the bar.
- `xwayland-close` — `Close` on an X window makes it exit, and kawa reaps it.
- `xwayland-clipboard` — the CLIPBOARD selection copies X → Wayland and Wayland → X.
- `xwayland-primary` — the PRIMARY selection copies both ways too. `zwp_primary_selection_device_manager_v1` and `zwlr_data_control_manager_v1` are advertised.

## How to get to it (user POV)

- Start an X11 program (from `New` or a terminal inside kawa). Copy in an X program and paste in a Wayland one, or the reverse.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven with `new-cmd.sh` as `-terms`; `xclock`, `xclip`, `xsel`, `wl-copy`, `wl-paste`, `xdotool` installed. Get the Xwayland display with `XD=$(sed -n 's/.*Running Xwayland on DISPLAY=\(:[0-9]*\).*/\1/p' "$VERIFY_KAWA_HOME/logs/kawa.stderr" | tail -n1)`. It is not the parent `DISPLAY` that `xdotool` and `screenshot.sh` use.

- `xwayland-new`: `echo xclock >"$VERIFY_KAWA_HOME/new-cmd"`, `New` a box. `$shot xwayland-xclock-new` shows the clock in the box with kawa's border and `xclock` in the bar.
- `xwayland-clipboard` and `xwayland-primary`, with xclock focused (click it first). X → Wayland: `printf x2w | DISPLAY=$XD xclip -selection clipboard -i`, then `wl-paste -n` prints `x2w`. Wayland → X: `wl-copy w2x`, then `DISPLAY=$XD xclip -selection clipboard -o` (and `xsel -b -o`) print `w2x`. Repeat with `-selection primary`, `wl-paste -p`, `wl-copy -p`, and `xsel -p`. Save the four results to `$VERIFY_KAWA_EVIDENCE/xwayland-clipboard.txt`. Run `xclip -i` in the background with its PID recorded; it stays up to serve the selection.
- `xwayland-close`: `Close`, right-click xclock; its PID exits and `ps --ppid` no longer lists it (`$shot xwayland-closed`).

## Gotchas

- X → Wayland copies only work while an X window has keyboard focus. With a Wayland window focused, `wl-paste` reports `Nothing is copied`; this is how wlroots hands selections to Xwayland, not a kawa bug. Wayland → X works either way.
