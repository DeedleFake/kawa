# Kawa verification map

This directory is the maintained source for verifying user-facing kawa compositor behavior. Read this index before driving, then use the matching feature file.

## Baseline preconditions

- Repo root is `deedles.dev/kawa` (`go.mod`), on `master` or a branch based on it (wlroots 0.20 bindings).
- `VERIFY_KAWA_BIN` is a `go build` of `.` from this checkout into `$VERIFY_KAWA_HOME/bin/kawa` (`CGO_ENABLED=1`, `GOTOOLCHAIN=auto`, wlroots 0.20 on `PKG_CONFIG_PATH` / `LD_LIBRARY_PATH`).
- `VERIFY_KAWA_HOME` is a fresh `/tmp/verify-kawa-$RUN_ID`.
- `XDG_RUNTIME_DIR=$VERIFY_KAWA_HOME/run` exists with mode `0700`.
- `DISPLAY` names an X server this run started (a private Xvfb, see SKILL.md Launch); `WAYLAND_DISPLAY` is **unset** for the compositor process so the X11 backend is used.
- Clients are started with `WAYLAND_DISPLAY=$VERIFY_KAWA_WAYLAND_DISPLAY` and the `DBUS_SESSION_BUS_ADDRESS` from `env.sh`, and their PIDs are written under `$VERIFY_KAWA_HOME/pids/`.
- Never DRM/session (`WLR_BACKENDS` must not force drm/libinput session).
- `VERIFY_KAWA_EVIDENCE` is set for this run.
- `doctor.sh` passed after `launch.sh`.
- Never drive a `kawa` this run did not build. Never `pkill kawa`.

## Driving conventions

- Start from baseline unless a feature lists extra preconditions.
- Harness is process + Wayland client (`wayland-registry`) + `xdotool` on the parent X display for pointer/menu features (see SKILL.md Drive). Never fake input via internal hooks.
- Capture via `capture.sh` and `screenshot.sh`. Keep PID, logs, registry dump, and screenshots in evidence.
- Restore nothing outside `$VERIFY_KAWA_HOME` and evidence.
- One long-lived instance can serve most features. Relaunch for features that need different flags (`-bg`, `-terms`).

## Proof and skip reporting

- Boot proof: ready log line + live PID + client registry globals including `wl_compositor`.
- Input-driven features: screenshot after driving with `xdotool` — do not mark verified via `go test`.
- Record the feature ID in artifact names.

## Feature entry contract

Each feature file: H1, one paragraph, then exactly four H2s: `Sub-features`, `How to get to it (user POV)`, `Driving it with process + Wayland client`, `Gotchas`.

## Features

- [Compositor boot + client connect](./compositor-boot.md) — launch, ready line, registry dump (**prove this**).
- [Status bar](./status-bar.md) — focused title, bar menus; time display not implemented.
- [Window menu / new windows](./window-menu.md) — main and system menus, `-terms`, `New`, the New box, late and foreign windows.
- [Window management](./window-management.md) — `Move`, `Resize`, `Close`, `Hide` and unhide, border and client moves.
- [Maximize / tiling](./maximize.md) — `Tile`, rows, swaps, maximize requests.
- [Activation and attention](./activation.md) — `xdg_activation_v1`, New tokens, attention border and menu entry.
- [Layer shell](./layer-shell.md) — wallpapers, panels and exclusive zones, layer popups, launchers, notifications.
- [Popups](./popups.md) — XDG popups kept on screen.
- [Xwayland](./xwayland.md) — X11 windows, `Close`, clipboard and primary selection both ways.
- [Presentation time and GTK4](./presentation.md) — `wp_presentation` feedback, GTK4 apps.
- [Background image](./background.md) — `-bg` / `-bgscale`.
