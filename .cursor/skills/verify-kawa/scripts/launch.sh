#!/bin/sh
# Build kawa + wayland-registry helper, start kawa on X11 backend with isolated XDG_RUNTIME_DIR.
set -eu

fail() {
	printf 'launch: %s\n' "$*" >&2
	exit 1
}

if [ -z "${VERIFY_KAWA_ROOT:-}" ]; then
	SCRIPT_DIR=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
	VERIFY_KAWA_ROOT=$(CDPATH= cd -- "$SCRIPT_DIR/../../../.." && pwd)
	export VERIFY_KAWA_ROOT
fi
[ -f "$VERIFY_KAWA_ROOT/go.mod" ] || fail "VERIFY_KAWA_ROOT has no go.mod: $VERIFY_KAWA_ROOT"
grep -q '^module deedles.dev/kawa$' "$VERIFY_KAWA_ROOT/go.mod" || fail "not deedles.dev/kawa"

if [ -z "${RUN_ID:-}" ]; then
	RUN_ID=$(date +%Y%m%dT%H%M%S)-$$
	export RUN_ID
fi
if [ -z "${VERIFY_KAWA_HOME:-}" ]; then
	VERIFY_KAWA_HOME="/tmp/verify-kawa-$RUN_ID"
	export VERIFY_KAWA_HOME
fi
case "$VERIFY_KAWA_HOME" in
/tmp/verify-kawa-*) ;;
*)
	fail "VERIFY_KAWA_HOME must be /tmp/verify-kawa-* (got $VERIFY_KAWA_HOME)"
	;;
esac

export VERIFY_KAWA_EVIDENCE="${VERIFY_KAWA_EVIDENCE:-$VERIFY_KAWA_ROOT/.cursor/skills/verify-kawa/artifacts/$RUN_ID}"

# wlroots comes from the caller's pkg-config environment; VERIFY_KAWA_PKG_CONFIG_PATH is prepended if set.
if [ -n "${VERIFY_KAWA_PKG_CONFIG_PATH:-}" ]; then
	PKG_CONFIG_PATH="$VERIFY_KAWA_PKG_CONFIG_PATH${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
fi
export PKG_CONFIG_PATH="${PKG_CONFIG_PATH:-}"
pkg-config --exists wlroots-0.20 \
	|| fail "pkg-config cannot find wlroots-0.20; set PKG_CONFIG_PATH (or VERIFY_KAWA_PKG_CONFIG_PATH) to the directory containing wlroots-0.20.pc"
wlr_libdir=$(pkg-config --variable=libdir wlroots-0.20)
export LD_LIBRARY_PATH="$wlr_libdir${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
export CGO_ENABLED=1
export GOTOOLCHAIN=auto

# Prefer X11 backend so we do not nest into another agent's Wayland session.
unset WAYLAND_DISPLAY || true
# Guessing a display could land kawa on another agent's X server, so the
# caller has to name its own.
[ -n "${DISPLAY:-}" ] || fail "DISPLAY is unset; start a private Xvfb (see SKILL.md Launch) and export DISPLAY"
xdpyinfo >/dev/null 2>&1 || fail "cannot connect to DISPLAY=$DISPLAY"
export DISPLAY
unset WLR_BACKENDS || true

# The X11 backend only looks up the atoms that it names its window with and
# never creates them, so on a fresh X server the window stays untitled and
# screenshot.sh cannot find it. Create them first.
python3 - <<'PY' || fail "could not intern the X11 backend's window atoms on DISPLAY=$DISPLAY"
import ctypes
x = ctypes.CDLL("libX11.so.6")
x.XOpenDisplay.restype = ctypes.c_void_p
x.XOpenDisplay.argtypes = [ctypes.c_char_p]
x.XInternAtom.argtypes = [ctypes.c_void_p, ctypes.c_char_p, ctypes.c_int]
x.XCloseDisplay.argtypes = [ctypes.c_void_p]
d = x.XOpenDisplay(None)
if not d:
    raise SystemExit(1)
for name in (b"WM_PROTOCOLS", b"WM_DELETE_WINDOW", b"_NET_WM_NAME", b"UTF8_STRING"):
    x.XInternAtom(d, name, 0)
x.XCloseDisplay(d)
PY

# Do not override the real HOME (module cache / toolchain live there).
case "${HOME:-}" in
/tmp/verify-kawa-*)
	fail "HOME=$HOME looks like VERIFY_KAWA_HOME; unset it so Go uses your real home"
	;;
esac

mkdir -p "$VERIFY_KAWA_HOME/bin" "$VERIFY_KAWA_HOME/pids" "$VERIFY_KAWA_HOME/logs" "$VERIFY_KAWA_EVIDENCE"

# Required: without a private XDG_RUNTIME_DIR, wl_display_add_socket_auto fails with
# "XDG_RUNTIME_DIR is invalid or not set" / "can't auto add wayland socket".
export XDG_RUNTIME_DIR="$VERIFY_KAWA_HOME/run"
mkdir -p "$XDG_RUNTIME_DIR"
chmod 0700 "$XDG_RUNTIME_DIR"

(
	cd "$VERIFY_KAWA_ROOT"
	go build -o "$VERIFY_KAWA_HOME/bin/kawa" ./cmd/kawa
)
export VERIFY_KAWA_BIN="$VERIFY_KAWA_HOME/bin/kawa"

helper_src="$VERIFY_KAWA_ROOT/.cursor/skills/verify-kawa/scripts/wayland-registry.c"
helper_bin="$VERIFY_KAWA_HOME/bin/wayland-registry"
gcc -O2 -o "$helper_bin" "$helper_src" $(pkg-config --cflags --libs wayland-client)
export VERIFY_KAWA_HELPER="$helper_bin"

# activation-client drives xdg-activation (features/activation.md). It needs
# wayland-protocols for the protocol XML; without it only that helper is missing.
act_bin="$VERIFY_KAWA_HOME/bin/activation-client"
if protocols=$(pkg-config --variable=pkgdatadir wayland-protocols 2>/dev/null) && [ -n "$protocols" ]; then
	gen="$VERIFY_KAWA_HOME/bin/gen"
	mkdir -p "$gen"
	for xml in "$protocols/stable/xdg-shell/xdg-shell.xml" "$protocols/staging/xdg-activation/xdg-activation-v1.xml"; do
		base=$(basename "$xml" .xml)
		wayland-scanner client-header "$xml" "$gen/$base-client.h"
		wayland-scanner private-code "$xml" "$gen/$base-protocol.c"
	done
	gcc -O2 -Wall -I"$gen" -o "$act_bin" \
		"$VERIFY_KAWA_ROOT/.cursor/skills/verify-kawa/scripts/activation-client.c" \
		"$gen/xdg-shell-protocol.c" "$gen/xdg-activation-v1-protocol.c" \
		$(pkg-config --cflags --libs wayland-client)
else
	printf 'launch: warning: wayland-protocols not found by pkg-config; activation-client not built\n' >&2
fi

if [ -f "$VERIFY_KAWA_HOME/pids/kawa.pid" ]; then
	old=$(cat "$VERIFY_KAWA_HOME/pids/kawa.pid")
	if [ -n "$old" ] && kill -0 "$old" 2>/dev/null; then
		fail "kawa already running under this home (pid=$old); run cleanup.sh first"
	fi
fi

# A private session bus. Without one, GTK clients autolaunch a bus on the X
# display that every later run on that display shares, and the portals it
# starts mount $XDG_RUNTIME_DIR/doc. cleanup.sh stops it with the rest.
unset DBUS_SESSION_BUS_ADDRESS || true
if command -v dbus-daemon >/dev/null 2>&1; then
	bus="unix:path=$XDG_RUNTIME_DIR/bus"
	dbus-daemon --session --fork --address="$bus" --print-pid >"$VERIFY_KAWA_HOME/pids/dbus.pid" \
		|| fail "could not start a private D-Bus session"
	export DBUS_SESSION_BUS_ADDRESS="$bus"
else
	printf 'launch: warning: dbus-daemon not found; GTK clients will autolaunch a shared session bus\n' >&2
fi

stdout_log="$VERIFY_KAWA_HOME/logs/kawa.stdout"
stderr_log="$VERIFY_KAWA_HOME/logs/kawa.stderr"
: >"$stdout_log"
: >"$stderr_log"

(
	cd "$VERIFY_KAWA_ROOT"
	exec "$VERIFY_KAWA_BIN" "$@"
) >>"$stdout_log" 2>>"$stderr_log" &
kawa_pid=$!
printf '%s\n' "$kawa_pid" >"$VERIFY_KAWA_HOME/pids/kawa.pid"

# Poll for the socket and a successful registry connect.
ready=0
sock=""
probe_out="$VERIFY_KAWA_HOME/logs/registry-probe.stdout"
probe_err="$VERIFY_KAWA_HOME/logs/registry-probe.stderr"
: >"$probe_out"
: >"$probe_err"
i=0
while [ "$i" -lt 200 ]; do
	if ! kill -0 "$kawa_pid" 2>/dev/null; then
		printf 'launch: kawa exited early; stderr:\n' >&2
		cat "$stderr_log" >&2 || true
		fail "kawa died before client connect (pid=$kawa_pid)"
	fi
	if [ -z "$sock" ] && grep -q 'Running Wayland compositor on WAYLAND_DISPLAY=' "$stderr_log" 2>/dev/null; then
		sock=$(sed -n 's/.*Running Wayland compositor on WAYLAND_DISPLAY=\([^[:space:]]*\).*/\1/p' "$stderr_log" | tail -n1)
	fi
	if [ -z "$sock" ]; then
		for cand in "$XDG_RUNTIME_DIR"/wayland-*; do
			[ -S "$cand" ] || continue
			case "$cand" in
			*.lock) continue ;;
			esac
			sock=$(basename "$cand")
			break
		done
	fi
	if [ -n "$sock" ]; then
		set +e
		WAYLAND_DISPLAY="$sock" XDG_RUNTIME_DIR="$XDG_RUNTIME_DIR" \
			"$helper_bin" >"$probe_out" 2>"$probe_err"
		probe_st=$?
		set -e
		if [ "$probe_st" -eq 0 ] && grep -q 'wl_compositor' "$probe_out"; then
			printf '%s\n' "$probe_st" >"$VERIFY_KAWA_HOME/logs/registry-probe.exit"
			ready=1
			break
		fi
	fi
	i=$((i + 1))
	sleep 0.02
done
[ "$ready" = 1 ] || {
	printf 'launch: timed out racing client connect; stderr:\n' >&2
	cat "$stderr_log" >&2 || true
	printf 'launch: last probe stderr:\n' >&2
	cat "$probe_err" >&2 || true
	fail "client could not talk to kawa in time"
}

export VERIFY_KAWA_WAYLAND_DISPLAY="$sock"
printf '%s\n' "$sock" >"$VERIFY_KAWA_HOME/wayland-display"

{
	printf "export VERIFY_KAWA_ROOT='%s'\n" "$VERIFY_KAWA_ROOT"
	printf "export RUN_ID='%s'\n" "$RUN_ID"
	printf "export VERIFY_KAWA_HOME='%s'\n" "$VERIFY_KAWA_HOME"
	printf "export VERIFY_KAWA_EVIDENCE='%s'\n" "$VERIFY_KAWA_EVIDENCE"
	printf "export VERIFY_KAWA_BIN='%s'\n" "$VERIFY_KAWA_BIN"
	printf "export VERIFY_KAWA_HELPER='%s'\n" "$VERIFY_KAWA_HELPER"
	printf "export VERIFY_KAWA_ACTIVATION_CLIENT='%s'\n" "$act_bin"
	printf "export VERIFY_KAWA_WAYLAND_DISPLAY='%s'\n" "$sock"
	printf "export XDG_RUNTIME_DIR='%s'\n" "$XDG_RUNTIME_DIR"
	printf "export DISPLAY='%s'\n" "$DISPLAY"
	if [ -n "${DBUS_SESSION_BUS_ADDRESS:-}" ]; then
		printf "export DBUS_SESSION_BUS_ADDRESS='%s'\n" "$DBUS_SESSION_BUS_ADDRESS"
	fi
	printf "export PKG_CONFIG_PATH='%s'\n" "$PKG_CONFIG_PATH"
	printf "export LD_LIBRARY_PATH='%s'\n" "$LD_LIBRARY_PATH"
	printf "export CGO_ENABLED=1\n"
	printf "export GOTOOLCHAIN=auto\n"
} >"$VERIFY_KAWA_HOME/env.sh"

printf 'launch: ok pid=%s WAYLAND_DISPLAY=%s DISPLAY=%s home=%s\n' \
	"$kawa_pid" "$sock" "$DISPLAY" "$VERIFY_KAWA_HOME"
printf 'launch: source %s/env.sh to restore env\n' "$VERIFY_KAWA_HOME"
printf 'launch: registry probe ok (see %s)\n' "$probe_out"
