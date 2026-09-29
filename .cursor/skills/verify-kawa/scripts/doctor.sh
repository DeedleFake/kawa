#!/bin/sh
set -eu

fail() {
	printf 'doctor: %s\n' "$*" >&2
	exit 1
}

if [ -z "${VERIFY_KAWA_BIN:-}" ]; then
	fail "VERIFY_KAWA_BIN is not set"
fi
if [ ! -x "$VERIFY_KAWA_BIN" ]; then
	fail "VERIFY_KAWA_BIN is not executable: $VERIFY_KAWA_BIN"
fi
case "$VERIFY_KAWA_BIN" in
*/bin/kawa) ;;
*)
	fail "VERIFY_KAWA_BIN should be the isolated build (.../bin/kawa), got: $VERIFY_KAWA_BIN"
	;;
esac

if [ -z "${VERIFY_KAWA_HOME:-}" ]; then
	fail "VERIFY_KAWA_HOME is not set"
fi
case "$VERIFY_KAWA_HOME" in
/tmp/verify-kawa-*) ;;
*)
	fail "VERIFY_KAWA_HOME must be /tmp/verify-kawa-* (got $VERIFY_KAWA_HOME)"
	;;
esac

if [ -z "${XDG_RUNTIME_DIR:-}" ]; then
	fail "XDG_RUNTIME_DIR is unset (launch.sh must set it to \$VERIFY_KAWA_HOME/run)"
fi
case "$XDG_RUNTIME_DIR" in
"$VERIFY_KAWA_HOME"/run) ;;
*)
	fail "XDG_RUNTIME_DIR must be \$VERIFY_KAWA_HOME/run (got $XDG_RUNTIME_DIR)"
	;;
esac
if [ ! -d "$XDG_RUNTIME_DIR" ]; then
	fail "XDG_RUNTIME_DIR does not exist: $XDG_RUNTIME_DIR"
fi
# wl_display_add_socket_auto requires mode 0700
mode=$(stat -c '%a' "$XDG_RUNTIME_DIR" 2>/dev/null || stat -f '%Lp' "$XDG_RUNTIME_DIR")
case "$mode" in
700) ;;
*)
	fail "XDG_RUNTIME_DIR mode must be 0700 (got $mode)"
	;;
esac

if [ -z "${DISPLAY:-}" ]; then
	fail "DISPLAY is unset; X11 backend needs a parent X display (e.g. DISPLAY=:9)"
fi
if [ -n "${WAYLAND_DISPLAY:-}" ]; then
	# Nested under another compositor is allowed for interactive use, but
	# verification prefers X11 so we do not attach to another agent's session.
	printf 'doctor: warning WAYLAND_DISPLAY=%s is set; prefer unsetting it so kawa uses the X11 backend\n' "$WAYLAND_DISPLAY" >&2
fi

# Refuse DRM/session backends for verification (README: locks input).
case "${WLR_BACKENDS:-}" in
*drm*|*libinput*)
	fail "WLR_BACKENDS=$WLR_BACKENDS looks like a session/DRM backend; use nested/X11 only"
	;;
esac

if [ -n "${VERIFY_KAWA_ROOT:-}" ]; then
	mod="$VERIFY_KAWA_ROOT/go.mod"
	[ -f "$mod" ] || fail "no go.mod at VERIFY_KAWA_ROOT"
	grep -q '^module deedles.dev/kawa$' "$mod" || fail "go.mod is not deedles.dev/kawa"
fi

# wlroots 0.20 must be findable for any rebuild / CGO link checks.
if [ -n "${VERIFY_KAWA_PKG_CONFIG_PATH:-}" ]; then
	PKG_CONFIG_PATH="$VERIFY_KAWA_PKG_CONFIG_PATH${PKG_CONFIG_PATH:+:$PKG_CONFIG_PATH}"
	export PKG_CONFIG_PATH
fi
if ! pkg-config --exists wlroots-0.20; then
	fail "pkg-config cannot find wlroots-0.20; set PKG_CONFIG_PATH (or VERIFY_KAWA_PKG_CONFIG_PATH) to the directory containing wlroots-0.20.pc and LD_LIBRARY_PATH to its libdir, or source \$VERIFY_KAWA_HOME/env.sh"
fi
if ldd "$VERIFY_KAWA_BIN" | grep -q 'not found'; then
	fail "kawa cannot load its libraries; add $(pkg-config --variable=libdir wlroots-0.20) to LD_LIBRARY_PATH"
fi
ver=$(pkg-config --modversion wlroots-0.20)
case "$ver" in
0.20.*) ;;
*)
	fail "expected wlroots 0.20.x, got $ver"
	;;
esac

helper="${VERIFY_KAWA_HELPER:-${VERIFY_KAWA_HOME}/bin/wayland-registry}"
if [ ! -x "$helper" ]; then
	fail "wayland-registry helper missing: $helper (run launch.sh first)"
fi

# If kawa is supposed to be running, check PID + ready log.
if [ -f "$VERIFY_KAWA_HOME/pids/kawa.pid" ]; then
	pid=$(cat "$VERIFY_KAWA_HOME/pids/kawa.pid")
	if [ -z "$pid" ] || ! kill -0 "$pid" 2>/dev/null; then
		fail "kawa PID file present but process not running (pid=$pid)"
	fi
	log="$VERIFY_KAWA_HOME/logs/kawa.stderr"
	[ -f "$log" ] || fail "missing kawa stderr log: $log"
	grep -q 'Running Wayland compositor on WAYLAND_DISPLAY=' "$log" \
		|| fail "ready line missing in $log"
	sock=$(sed -n 's/.*Running Wayland compositor on WAYLAND_DISPLAY=\([^[:space:]]*\).*/\1/p' "$log" | tail -n1)
	[ -n "$sock" ] || fail "could not parse WAYLAND_DISPLAY from ready line"
	if [ -z "${VERIFY_KAWA_WAYLAND_DISPLAY:-}" ]; then
		fail "VERIFY_KAWA_WAYLAND_DISPLAY unset but kawa is running (expected $sock)"
	fi
	[ "$VERIFY_KAWA_WAYLAND_DISPLAY" = "$sock" ] \
		|| fail "VERIFY_KAWA_WAYLAND_DISPLAY=$VERIFY_KAWA_WAYLAND_DISPLAY != log sock $sock"
fi

printf 'doctor: ok bin=%s home=%s display=%s wlroots=%s\n' \
	"$VERIFY_KAWA_BIN" "$VERIFY_KAWA_HOME" "$DISPLAY" "$ver"
