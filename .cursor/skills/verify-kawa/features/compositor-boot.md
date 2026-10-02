# Compositor boot + client connect

Starts kawa as a nested/X11 compositor with an isolated runtime dir, waits for the Wayland socket, and proves a client can enumerate registry globals.

## Sub-features

- `boot-x11` starts with `DISPLAY` set and `WAYLAND_DISPLAY` unset (X11 backend).
- `ready-log` observes `Running Wayland compositor on WAYLAND_DISPLAY=<name>` on stderr, followed by `Running Xwayland on DISPLAY=:N`.
- `client-registry` connects with `wayland-registry` and prints globals including `wl_compositor`. On master it lists 19, among them `zwlr_layer_shell_v1`, `xdg_activation_v1`, `wp_presentation`, `zwlr_data_control_manager_v1`, `zwp_primary_selection_device_manager_v1`, and `xwayland_shell_v1`.

## How to get to it (user POV)

- Build and run `kawa` under another compositor (nested) or on X11.
- Clients set `WAYLAND_DISPLAY` to the name kawa logged and talk to that socket under `XDG_RUNTIME_DIR`.

## Driving it with process + Wayland client

Preconditions: baseline (a private Xvfb exported as `DISPLAY`); `launch.sh` succeeded; `doctor.sh` passed.

- **Launch.** `.cursor/skills/verify-kawa/scripts/launch.sh` then `source "$VERIFY_KAWA_HOME/env.sh"`.
- **Doctor.** `.cursor/skills/verify-kawa/scripts/doctor.sh`.
- **Registry.** `export WAYLAND_DISPLAY="$VERIFY_KAWA_WAYLAND_DISPLAY"` then `.cursor/skills/verify-kawa/scripts/capture.sh compositor-boot-registry -- wayland-registry`. Exit `0`. stdout contains `wl_compositor` and a `globals=N` line with N>0.
- **Evidence.** Copy `$VERIFY_KAWA_HOME/logs/kawa.stderr`, `pids/kawa.pid`, and the socket name into `$VERIFY_KAWA_EVIDENCE/`.
- **Cleanup.** `.cursor/skills/verify-kawa/scripts/cleanup.sh`. Confirm `$VERIFY_KAWA_EVIDENCE` still has the captures.

## Gotchas

- Missing `XDG_RUNTIME_DIR` (or not mode 0700) → `can't auto add wayland socket`; kawa exits. Always use `$VERIFY_KAWA_HOME/run`.
- Leaving `WAYLAND_DISPLAY` set nests into another compositor (possibly another agent's). `launch.sh` unsets it; when running kawa by hand, unset it too or wlroots picks the Wayland backend.
- On a fresh X server the X11 backend's window has no title, because wlroots only looks up the atoms it names the window with. `launch.sh` creates them first, and `doctor.sh` fails if the `wlroots - X11-1` window is missing. Start Xvfb with `-noreset` so the atoms survive the last client disconnecting.
- DRM FD / dmabuf errors on stderr can appear and still be OK if the ready line arrives.
- Never `pkill kawa`; only the PID under `$VERIFY_KAWA_HOME/pids`.
- Socket name is often a short string (not a full path); clients resolve it under `XDG_RUNTIME_DIR`.
