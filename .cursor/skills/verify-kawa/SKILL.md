---
name: verify-kawa
description: Drive the kawa Wayland compositor (rio-inspired nested session / X11 backend) — boot, client connect, menus, window management, tiling, activation, layer shell, popups, Xwayland and clipboard, presentation, background. Use when proving compositor behavior after changes to deedles.dev/kawa; never DRM/session.
---

# Verify Kawa

kawa is a Wayland compositor with a rio-inspired UI (status bar, right-click menus, tiling). The primary user surface is the **nested compositor session**: a window under a parent compositor, or the **X11 backend** when `DISPLAY` is set and `WAYLAND_DISPLAY` is unset. It is **not** a web UI. README forbids production DRM/session use (locks input); verification must use nested/X11 only.

This skill is for agents. Commands are literal. Harness is **process + Wayland client** against kawa’s socket — no Playwright.

## Launch

From the repo root (`deedles.dev/kawa`), on `master` or a branch based on it. kawa's X11 backend needs a parent X server; start a private one so no other session shares it:

```sh
n=49   # any free display number: [ ! -e /tmp/.X11-unix/X$n ]
Xvfb ":$n" -screen 0 1024x768x24 -ac -noreset >/tmp/xvfb-$n.log 2>&1 &
echo $! >/tmp/xvfb-$n.pid
export DISPLAY=":$n"
```

`-noreset` keeps the X server from resetting when its last client goes, which would drop the atoms `launch.sh` creates (step 2). Kill this Xvfb by its PID when the run is over.

```sh
export VERIFY_KAWA_ROOT="$(pwd)"
export RUN_ID="$(date +%Y%m%dT%H%M%S)-$$"
export VERIFY_KAWA_HOME="/tmp/verify-kawa-$RUN_ID"
export VERIFY_KAWA_EVIDENCE="${VERIFY_KAWA_EVIDENCE:-$VERIFY_KAWA_ROOT/.cursor/skills/verify-kawa/artifacts/$RUN_ID}"

# wlroots must be on pkg-config: set PKG_CONFIG_PATH (e.g. source your wlroots build's env script),
# or set VERIFY_KAWA_PKG_CONFIG_PATH to the directory containing wlroots-0.20.pc.
# Or use the helper (preferred): builds binary + registry client, starts kawa, waits for ready.
# Arguments are passed to kawa, e.g. launch.sh -terms weston-terminal -bg PATH -bgscale fit.
.cursor/skills/verify-kawa/scripts/launch.sh
source "$VERIFY_KAWA_HOME/env.sh"
```

`launch.sh` does all of the following (do not skip):

1. Finds `wlroots-0.20` via the caller's `PKG_CONFIG_PATH` (prepending `VERIFY_KAWA_PKG_CONFIG_PATH` if set) and fails if it is missing; prepends its `libdir` to `LD_LIBRARY_PATH`; sets `CGO_ENABLED=1`, `GOTOOLCHAIN=auto`.
2. Unsets `WAYLAND_DISPLAY` and requires `DISPLAY` (it fails rather than guess one) so kawa uses the **X11 backend**, not another agent’s compositor. It then creates the `WM_PROTOCOLS`, `WM_DELETE_WINDOW`, `_NET_WM_NAME` and `UTF8_STRING` atoms on that display: the X11 backend only looks them up, so on a fresh X server its window would stay untitled and `screenshot.sh` could not find it.
3. Creates `XDG_RUNTIME_DIR=$VERIFY_KAWA_HOME/run` with mode **0700**. Without this, kawa dies with `XDG_RUNTIME_DIR is invalid or not set` / `can't auto add wayland socket`. DRM FD / dmabuf messages on stderr are noisy but non-fatal until the socket fails.
4. Builds `go build -o "$VERIFY_KAWA_HOME/bin/kawa" .`, compiles `.cursor/skills/verify-kawa/scripts/wayland-registry.c` → `$VERIFY_KAWA_HOME/bin/wayland-registry`, and, when `wayland-protocols` is on pkg-config, `activation-client.c` → `$VERIFY_KAWA_HOME/bin/activation-client` (protocol code generated with `wayland-scanner`).
5. Starts a private D-Bus session bus at `$XDG_RUNTIME_DIR/bus` (PID in `pids/dbus.pid`). Without it, GTK clients autolaunch a bus tied to the X display that later runs on the same display would share.
6. Starts kawa in the background with `launch.sh`'s arguments; writes PID to `$VERIFY_KAWA_HOME/pids/kawa.pid`; captures stdout/stderr under `$VERIFY_KAWA_HOME/logs/`.
7. Waits for ready log line: `Running Wayland compositor on WAYLAND_DISPLAY=...` (from `kawa.go`). kawa then logs `Running Xwayland on DISPLAY=:N`, the display X clients inside kawa use.
8. Exports `VERIFY_KAWA_WAYLAND_DISPLAY` (socket **name**, under `$XDG_RUNTIME_DIR`), `VERIFY_KAWA_ACTIVATION_CLIENT`, and `DBUS_SESSION_BUS_ADDRESS` via `$VERIFY_KAWA_HOME/env.sh`.

Never run as a DRM session. Never `pkill kawa`. Do not permanently unset the user’s session compositor.

Flags (see `kawa.go` main): `-terms`, `-bg`, `-bgscale`, `-out`.

Ready: stderr contains `Running Wayland compositor on WAYLAND_DISPLAY=` and `kill -0` on the recorded PID succeeds.

## Doctor

Read-only. Run after launch (or whenever something looks off):

```sh
source "$VERIFY_KAWA_HOME/env.sh"
.cursor/skills/verify-kawa/scripts/doctor.sh
```

Pass means: isolated `.../bin/kawa`, `VERIFY_KAWA_HOME` under `/tmp/verify-kawa-*`, `XDG_RUNTIME_DIR=$VERIFY_KAWA_HOME/run` mode 0700, `DISPLAY` set, `wlroots-0.20` via pkg-config, registry helper present, and if a PID file exists the process is alive with the ready line, matching `VERIFY_KAWA_WAYLAND_DISPLAY`, and a `wlroots - X11-1` window on `DISPLAY`. It warns if `WAYLAND_DISPLAY` points anywhere other than kawa's socket.

## Drive

Harness = process control + Wayland client against kawa’s socket. No Playwright.

| Action | Ready / proof handle |
| --- | --- |
| Boot | ready line + PID alive |
| Client connect | `WAYLAND_DISPLAY=$VERIFY_KAWA_WAYLAND_DISPLAY XDG_RUNTIME_DIR=$XDG_RUNTIME_DIR timeout 5 wayland-registry` prints `wl_compositor` (and other globals) and `globals=N` with N>0, exit 0 |
| Status bar | screenshot shows the focused title; left-click on the bar → system menu (`Log Out`), right-click → main menu. Clock/time (README) is not implemented |
| Window menu / New | right-click → menu items `New`, `Resize`, `Tile`, `Move`, `Close`, `Hide`, then hidden windows; `New` + right-drag starts a `-terms` child and maps its window in the box (screenshot) |
| Window management | `Move` / `Resize` / `Close` / `Hide` + right-click a window (screenshots, `ps --ppid`) |
| Maximize / tiling | menu `Tile` → right-click a window → it fills the area under the status bar (screenshot) |
| Activation | `activation-client` windows; kawa log `activation request for "NAME": allowed=...` + screenshots |
| Layer shell | swaybg, waybar, fuzzel, mako; kawa log `new layer surface ...` + screenshots |
| Popups | GTK context menu near the screen corner stays on screen (screenshot) |
| Xwayland | xclock via `New`; `xclip`/`xsel` ↔ `wl-copy`/`wl-paste` results |
| Presentation | `WAYLAND_DEBUG` count of `wp_presentation_feedback…presented` grows in gtk4-demo |
| Background | launch with `-bg PATH` (`-bgscale` stretch\|center\|fit\|fill); log `loaded ... as background` |

Pointer input: `xdotool` on the parent X display (`DISPLAY` from `env.sh`) drives kawa's X11 backend window. Screenshots: `screenshot.sh NAME` grabs that window with `xwd -id` and converts it with `ffmpeg` to `$VERIFY_KAWA_EVIDENCE/NAME.png`.

```sh
source "$VERIFY_KAWA_HOME/env.sh"
wid=$(xwininfo -root -tree | sed -n 's/^ *\(0x[0-9a-f]*\) "wlroots - X11-1".*/\1/p' | head -n1)
shot=.cursor/skills/verify-kawa/scripts/screenshot.sh
```

Menus open on button press with the last-chosen item (initially the first) under the pointer, shifted to stay on screen; releasing the same button selects the item under the pointer. Items are ~19 px apart. Release off the menu to cancel. Because of the shift, a menu opened near a screen edge does not have the expected item under the pointer. From the status bar the menu is pushed below the bar with item 0 at the pointer whatever was chosen last, so open menus there when the item matters:

```sh
# barmenu I: pick main-menu item I (0 New, 1 Resize, 2 Tile, 3 Move, 4 Close, 5 Hide, 6+ hidden windows)
barmenu() { xdotool mousemove --window "$wid" 600 10 mousedown 3 sleep 0.3 mousemove --window "$wid" 600 $((20+19*$1)) sleep 0.3 mouseup 3 sleep 0.3; }
# bgc NAME CMD...: start a client against kawa, output in evidence, PID where cleanup.sh finds it
bgc() { n=$1; shift; "$@" >"$VERIFY_KAWA_EVIDENCE/$n.stdout" 2>"$VERIFY_KAWA_EVIDENCE/$n.stderr" & echo $! >"$VERIFY_KAWA_HOME/pids/$n.pid"; }
```

Start clients with `WAYLAND_DISPLAY=$VERIFY_KAWA_WAYLAND_DISPLAY` exported. For `New`, launch kawa with `-terms "$VERIFY_KAWA_ROOT/.cursor/skills/verify-kawa/scripts/new-cmd.sh"` and write the command line to start into `$VERIFY_KAWA_HOME/new-cmd` before each `New` (window-menu.md).

- **Main menu:** `xdotool mousemove --window "$wid" 400 300 mousedown 3`, then `$shot window-menu-main`.
- **New:** with the pointer still on `New`, `xdotool mouseup 3`; then `xdotool mousemove --window "$wid" 200 150 mousedown 3 sleep 0.2 mousemove --window "$wid" 260 200 sleep 0.2 mousemove --window "$wid" 700 520 sleep 0.3 mouseup 3`. After ~2 s, `ps -o pid,cmd --ppid "$(cat "$VERIFY_KAWA_HOME/pids/kawa.pid")"` lists the terminal; `$shot window-menu-new` shows it in the dragged rectangle and its title in the status bar.
- **Tile:** `barmenu 2`, then right-click the window: `xdotool mousemove --window "$wid" 480 380 click 3`; `$shot maximize-tiled`. A right-press on a window's border starts a move instead of opening the menu.
- **Status bar:** `xdotool mousemove --window "$wid" 600 10 mousedown 3` → main menu (`$shot status-bar-right`); `mousedown 1` on the bar → `Log Out` (`$shot status-bar-left`). Move below the menu before `mouseup` unless you intend to log out; releasing button 1 in place selects `Log Out` and shuts kawa down.

Capture every drive:

```sh
source "$VERIFY_KAWA_HOME/env.sh"
export WAYLAND_DISPLAY="$VERIFY_KAWA_WAYLAND_DISPLAY"
.cursor/skills/verify-kawa/scripts/capture.sh compositor-boot-registry -- wayland-registry
```

Also copy kawa logs (and `new.log` if `new-cmd.sh` ran) into evidence after a successful boot and again before cleanup:

```sh
cp "$VERIFY_KAWA_HOME/logs/kawa.stderr" "$VERIFY_KAWA_EVIDENCE/kawa.stderr"
cp "$VERIFY_KAWA_HOME/logs/new.log" "$VERIFY_KAWA_EVIDENCE/new.log" 2>/dev/null || true
cp "$VERIFY_KAWA_HOME/pids/kawa.pid" "$VERIFY_KAWA_EVIDENCE/kawa.pid"
printf '%s\n' "$VERIFY_KAWA_WAYLAND_DISPLAY" >"$VERIFY_KAWA_EVIDENCE/wayland-display.txt"
```

Read the [feature map](features/README.md) before driving.

## Evidence

Directory: `$VERIFY_KAWA_EVIDENCE` (default `.cursor/skills/verify-kawa/artifacts/$RUN_ID`). Gitignored. Survives cleanup.

Each `capture.sh` writes `NAME.cmd`, `NAME.stdout`, `NAME.stderr`, `NAME.exit`. Boot proof also keeps `kawa.stderr`, `kawa.pid`, `wayland-display.txt`.

Proof standards:

- Drive the real compositor binary built into `$VERIFY_KAWA_HOME/bin/kawa`, not a unit test double.
- Assert the ready line and a successful client registry dump (`wl_compositor` present), not only exit 0 on build.
- Record PID and socket name.
- Never treat DRM FD warnings alone as failure; socket failure / missing ready line is failure.
- Screenshot every visual feature with `screenshot.sh`. `xwd` grabs what is on screen, so windows overlapping kawa's appear in the image.
- Mocks: none.

## Cleanup

```sh
source "$VERIFY_KAWA_HOME/env.sh"   # if not already
.cursor/skills/verify-kawa/scripts/cleanup.sh
```

Kills PIDs under `$VERIFY_KAWA_HOME/pids` (SIGTERM then SIGKILL). Then it stops your remaining processes whose environment has `XDG_RUNTIME_DIR=$VERIFY_KAWA_HOME/run` (grandchildren of `New`, portals started over the bus), except itself and the shells that started it, unmounts `$XDG_RUNTIME_DIR/doc` if a document portal mounted it, removes the X lock and socket that kawa's Xwayland leaves behind (kawa does not clean them up on SIGTERM), and runs `rm -rf "$VERIFY_KAWA_HOME"`. Does **not** delete `$VERIFY_KAWA_EVIDENCE`. Never `pkill kawa`. Run it as the last step of a drive, from a shell that is not also running clients you still need for this run. Then kill your private Xvfb by the PID you saved.

## Helpers

All under `.cursor/skills/verify-kawa/scripts/`, executable (except the `.c` sources):

```sh
.cursor/skills/verify-kawa/scripts/launch.sh
.cursor/skills/verify-kawa/scripts/doctor.sh
.cursor/skills/verify-kawa/scripts/capture.sh compositor-boot-registry -- wayland-registry
.cursor/skills/verify-kawa/scripts/screenshot.sh NAME
.cursor/skills/verify-kawa/scripts/cleanup.sh
.cursor/skills/verify-kawa/scripts/new-cmd.sh   # as a -terms entry, not run by hand
```

`new-cmd.sh` runs the command line in `$VERIFY_KAWA_HOME/new-cmd` (default `weston-terminal`) with `exec`, so the window keeps the PID kawa started, and logs it with the program's output to `$VERIFY_KAWA_HOME/logs/new.log`. `activation-client.c` is compiled by `launch.sh` into `$VERIFY_KAWA_ACTIVATION_CLIENT`; its modes are in its header comment and activation.md.

`wayland-registry.c` is compiled by `launch.sh` into `$VERIFY_KAWA_HOME/bin/wayland-registry`. `capture.sh` rewrites the token `wayland-registry` to that binary. `NAME` is a single path segment.
