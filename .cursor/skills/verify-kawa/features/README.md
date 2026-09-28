# Kawa verification map

This directory is the maintained source for verifying user-facing kawa compositor behavior. Read this index before driving, then use the matching feature file.

## Baseline preconditions

- Repo root is `deedles.dev/kawa` (`go.mod`), on wlroots 0.20 bindings (`wlr-0.20`).
- `VERIFY_KAWA_BIN` is a `go build` of `.` from this checkout into `$VERIFY_KAWA_HOME/bin/kawa` (`CGO_ENABLED=1`, `GOTOOLCHAIN=auto`, wlroots 0.20 on `PKG_CONFIG_PATH` / `LD_LIBRARY_PATH`).
- `VERIFY_KAWA_HOME` is a fresh `/tmp/verify-kawa-$RUN_ID`.
- `XDG_RUNTIME_DIR=$VERIFY_KAWA_HOME/run` exists with mode `0700`.
- `DISPLAY` is set (prefer `:9` on this box); `WAYLAND_DISPLAY` is **unset** for the compositor process so the X11 backend is used.
- Never DRM/session (`WLR_BACKENDS` must not force drm/libinput session).
- `VERIFY_KAWA_EVIDENCE` is set for this run.
- `doctor.sh` passed after `launch.sh`.
- Never drive a `kawa` this run did not build. Never `pkill kawa`.

## Driving conventions

- Start from baseline unless a feature lists extra preconditions.
- Harness is process + Wayland client (`wayland-registry`). Pointer/menu features need an input injection tool on a nested session — document and skip rather than fake via internal hooks.
- Capture via `capture.sh`. Keep PID, logs, and registry dump in evidence.
- Restore nothing outside `$VERIFY_KAWA_HOME` and evidence.

## Proof and skip reporting

- Boot proof: ready log line + live PID + client registry globals including `wl_compositor`.
- Input-driven features: report unmet precondition (no pointer injector) — do not mark verified via `go test`.
- Record the feature ID in artifact names.

## Feature entry contract

Each feature file: H1, one paragraph, then exactly four H2s: `Sub-features`, `How to get to it (user POV)`, `Driving it with process + Wayland client`, `Gotchas`.

## Features

- [Compositor boot + client connect](./compositor-boot.md) — launch, ready line, registry dump (**prove this**).
- [Status bar](./status-bar.md) — bar chrome / focused title; time display deferred to visual proof.
- [Window menu / new terminal](./window-menu.md) — right-click menu, `-terms`, `New`.
- [Maximize / tiling](./maximize.md) — menu `Tile` / maximize paths.
- [Background image](./background.md) — `-bg` / `-bgscale`.
