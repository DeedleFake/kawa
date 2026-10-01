#!/bin/sh
set -eu

if [ -z "${VERIFY_KAWA_HOME:-}" ]; then
	printf 'cleanup: VERIFY_KAWA_HOME is not set\n' >&2
	exit 1
fi

case "$VERIFY_KAWA_HOME" in
/tmp/verify-kawa-*) ;;
*)
	printf 'cleanup: refusing to remove unexpected VERIFY_KAWA_HOME=%s\n' "$VERIFY_KAWA_HOME" >&2
	exit 1
	;;
esac

kawa_pid=$(cat "$VERIFY_KAWA_HOME/pids/kawa.pid" 2>/dev/null || true)

if [ -d "$VERIFY_KAWA_HOME/pids" ]; then
	for f in "$VERIFY_KAWA_HOME/pids"/*; do
		[ -f "$f" ] || continue
		pid=$(cat "$f")
		if [ -n "$pid" ] && kill -0 "$pid" 2>/dev/null; then
			kill "$pid" 2>/dev/null || true
			# Give the Wayland display a moment to unwind.
			n=0
			while [ "$n" -lt 20 ] && kill -0 "$pid" 2>/dev/null; do
				sleep 0.1
				n=$((n + 1))
			done
			kill -0 "$pid" 2>/dev/null && kill -9 "$pid" 2>/dev/null || true
		fi
	done
fi

# Programs can outlive their recorded PIDs: grandchildren of New, and the
# D-Bus session that GTK clients autolaunch on the X display along with the
# portals it starts. They all carry this run's XDG_RUNTIME_DIR, so stop the
# caller's processes that still have it, except for this script, the shells
# that started it, and its own children.
run="$VERIFY_KAWA_HOME/run"
ppid_of() {
	sed -n 's/^[0-9]* (.*) [A-Za-z] \([0-9]*\) .*/\1/p' "/proc/$1/stat" 2>/dev/null
}
keep=" "
p=$$
while [ -n "$p" ] && [ "$p" -gt 1 ]; do
	keep="$keep$p "
	p=$(ppid_of "$p")
done
mine() {
	case "$keep" in *" $1 "*) return 0 ;; esac
	p=$(ppid_of "$1")
	while [ -n "$p" ] && [ "$p" -gt 1 ]; do
		[ "$p" != "$$" ] || return 0
		p=$(ppid_of "$p")
	done
	return 1
}
residue() {
	for env in /proc/[0-9]*/environ; do
		pid=${env#/proc/}
		pid=${pid%/environ}
		[ -O "$env" ] || continue
		tr '\0' '\n' <"$env" 2>/dev/null | grep -qxF "XDG_RUNTIME_DIR=$run" || continue
		mine "$pid" || printf '%s\n' "$pid"
	done
}
for sig in TERM KILL; do
	pids=$(residue)
	[ -n "$pids" ] || break
	printf 'cleanup: stopping leftover processes (SIG%s): %s\n' "$sig" "$(echo $pids)"
	kill -s "$sig" $pids 2>/dev/null || true
	sleep 1
done

# xdg-document-portal mounts $XDG_RUNTIME_DIR/doc; rm cannot remove a mount.
if grep -qF " $run/doc " /proc/self/mounts; then
	fusermount3 -u "$run/doc" 2>/dev/null || fusermount -u "$run/doc" 2>/dev/null || true
	n=0
	while [ "$n" -lt 20 ] && grep -qF " $run/doc " /proc/self/mounts; do
		sleep 0.1
		n=$((n + 1))
	done
fi

# kawa does not clean up after itself on SIGTERM, so the X lock and socket
# of the Xwayland it started are left behind. Remove the ones that name it.
if [ -n "$kawa_pid" ] && ! kill -0 "$kawa_pid" 2>/dev/null; then
	for lock in /tmp/.X*-lock; do
		[ -O "$lock" ] || continue
		[ "$(tr -d ' \0\n' <"$lock" 2>/dev/null)" = "$kawa_pid" ] || continue
		n=${lock#/tmp/.X}
		n=${n%-lock}
		rm -f "/tmp/.X11-unix/X$n" "$lock"
		printf 'cleanup: removed stale Xwayland display :%s\n' "$n"
	done
fi

rm -rf "$VERIFY_KAWA_HOME"
printf 'cleanup: removed %s (evidence kept at %s)\n' "$VERIFY_KAWA_HOME" "${VERIFY_KAWA_EVIDENCE:-unset}"
