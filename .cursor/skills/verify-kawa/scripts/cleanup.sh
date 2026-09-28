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

rm -rf "$VERIFY_KAWA_HOME"
printf 'cleanup: removed %s (evidence kept at %s)\n' "$VERIFY_KAWA_HOME" "${VERIFY_KAWA_EVIDENCE:-unset}"
