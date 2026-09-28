---
name: verify-kawa
description: Drive the kawa Wayland compositor (rio-inspired nested session / X11 backend) — boot, client connect, menus, tiling, background. Use when proving compositor behavior after changes to deedles.dev/kawa; never DRM/session.
---

# Verify Kawa

kawa is a Wayland compositor with a rio-inspired UI (status bar, right-click menus, tiling). The primary user surface is the **nested compositor session**: a window under a parent compositor, or the **X11 backend** when `DISPLAY` is set and `WAYLAND_DISPLAY` is unset. It is **not** a web UI. README forbids production DRM/session use (locks input); verification must use nested/X11 only.

This skill is for agents. Commands are literal. Harness is **process + Wayland client** against kawa’s socket — no Playwright.

## Launch

From the repo root (`deedles.dev/kawa`), on a branch with wlroots 0.20 bindings (`wlr-0.20` tip `9fa567c` or later):

```sh
export VERIFY_KAWA_ROOT="$(pwd)"
export RUN_ID="$(date +%Y%m%dT%H%M%S)-$$"
export VERIFY_KAWA_HOME="/tmp/verify-kawa-$RUN_ID"
export VERIFY_KAWA_EVIDENCE="${VERIFY_KAWA_EVIDENCE:-$VERIFY_KAWA_ROOT/.cursor/skills/verify-kawa/artifacts/$RUN_ID}"

# Or use the helper (preferred): builds binary + registry client, starts kawa, waits for ready.
.cursor/skills/verify-kawa/scripts/launch.sh
# Optional kawa flags after -- are not used; pass flags by editing launch or:
#   ... after sourcing env.sh, kill via cleanup and re-exec with flags (see Drive).
source "$VERIFY_KAWA_HOME/env.sh"
```

`launch.sh` does all of the following (do not skip):

1. Sets `PKG_CONFIG_PATH` / `LD_LIBRARY_PATH` to `/opt/wlroots-0.20/...`, `CGO_ENABLED=1`, `GOTOOLCHAIN=auto`.
2. Unsets `WAYLAND_DISPLAY` and keeps/chooses `DISPLAY` (on this box prefer `DISPLAY=:9`) so kawa uses the **X11 backend**, not another agent’s compositor.
3. Creates `XDG_RUNTIME_DIR=$VERIFY_KAWA_HOME/run` with mode **0700**. Without this, kawa dies with `XDG_RUNTIME_DIR is invalid or not set` / `can't auto add wayland socket`. DRM FD / dmabuf messages on stderr are noisy but non-fatal until the socket fails.
4. Builds `go build -o "$VERIFY_KAWA_HOME/bin/kawa" .` and compiles `.cursor/skills/verify-kawa/scripts/wayland-registry.c` → `$VERIFY_KAWA_HOME/bin/wayland-registry`.
5. Starts kawa in the background; writes PID to `$VERIFY_KAWA_HOME/pids/kawa.pid`; captures stdout/stderr under `$VERIFY_KAWA_HOME/logs/`.
6. Waits for ready log line: `Running Wayland compositor on WAYLAND_DISPLAY=...` (from `kawa.go`). May also log Xwayland `DISPLAY=...`.
7. Exports `VERIFY_KAWA_WAYLAND_DISPLAY` (socket **name**, often under `$XDG_RUNTIME_DIR`) via `$VERIFY_KAWA_HOME/env.sh`.

Never run as a DRM session. Never `pkill kawa`. Do not permanently unset the user’s session compositor.

Flags (see `kawa.go` main): `-terms`, `-bg`, `-bgscale`, `-out`.

Ready: stderr contains `Running Wayland compositor on WAYLAND_DISPLAY=` and `kill -0` on the recorded PID succeeds.

## Doctor

Read-only. Run after launch (or whenever something looks off):

```sh
source "$VERIFY_KAWA_HOME/env.sh"
.cursor/skills/verify-kawa/scripts/doctor.sh
```

Pass means: isolated `.../bin/kawa`, `VERIFY_KAWA_HOME` under `/tmp/verify-kawa-*`, `XDG_RUNTIME_DIR=$VERIFY_KAWA_HOME/run` mode 0700, `DISPLAY` set, `wlroots-0.20` via pkg-config, registry helper present, and if a PID file exists the process is alive with the ready line and matching `VERIFY_KAWA_WAYLAND_DISPLAY`.

## Drive

Harness = process control + Wayland client against kawa’s socket. No Playwright.

| Action | Ready / proof handle |
| --- | --- |
| Boot | ready line + PID alive |
| Client connect | `WAYLAND_DISPLAY=$VERIFY_KAWA_WAYLAND_DISPLAY XDG_RUNTIME_DIR=$XDG_RUNTIME_DIR timeout 5 wayland-registry` prints `wl_compositor` (and other globals) and `globals=N` with N>0, exit 0 |
| Status bar | rendered in-compositor (focused title via `SetTitle`); clock/time as in README is planned — needs screenshot / input later |
| Window menu / New | right-click → menu items `New`, `Resize`, `Tile`, `Move`, `Close`, `Hide`; system menu on status bar (`Log Out`). Needs pointer injection |
| Maximize / tiling | menu `Tile` → `toggleViewTiling`. Needs pointer injection |
| Background | launch with `-bg PATH` (`-bgscale` stretch\|center\|fit\|fill); log `loaded ... as background` |

**Must prove end-to-end now:** compositor boot + client connect (`compositor-boot`). Map the rest; mark input-driven features as requiring nested Wayland + an input tool.

On this box, Xwayland can panic kawa shortly after the ready line (`Surface.OnMap` nil in `onNewXwaylandSurface`). `launch.sh` runs `wayland-registry` immediately after ready; prefer that probe (copied into evidence) over a delayed reconnect.

Capture every drive:

```sh
source "$VERIFY_KAWA_HOME/env.sh"
export WAYLAND_DISPLAY="$VERIFY_KAWA_WAYLAND_DISPLAY"
.cursor/skills/verify-kawa/scripts/capture.sh compositor-boot-registry -- wayland-registry
```

Also copy kawa logs into evidence after a successful boot:

```sh
cp "$VERIFY_KAWA_HOME/logs/kawa.stderr" "$VERIFY_KAWA_EVIDENCE/kawa.stderr"
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
- Screenshots optional if grim/X11 capture exists — not required for first prove.
- Mocks: none.

## Cleanup

```sh
source "$VERIFY_KAWA_HOME/env.sh"   # if not already
.cursor/skills/verify-kawa/scripts/cleanup.sh
```

Kills PIDs under `$VERIFY_KAWA_HOME/pids` (SIGTERM then SIGKILL), then `rm -rf "$VERIFY_KAWA_HOME"`. Does **not** delete `$VERIFY_KAWA_EVIDENCE`. Never `pkill kawa`.

## Helpers

All under `.cursor/skills/verify-kawa/scripts/`, executable (except the `.c` source):

```sh
.cursor/skills/verify-kawa/scripts/launch.sh
.cursor/skills/verify-kawa/scripts/doctor.sh
.cursor/skills/verify-kawa/scripts/capture.sh compositor-boot-registry -- wayland-registry
.cursor/skills/verify-kawa/scripts/cleanup.sh
```

`wayland-registry.c` is compiled by `launch.sh` into `$VERIFY_KAWA_HOME/bin/wayland-registry`. `capture.sh` rewrites the token `wayland-registry` to that binary. `NAME` is a single path segment.
