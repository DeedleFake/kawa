#!/bin/sh
set -eu

usage() {
	printf 'usage: capture.sh NAME -- command [args...]\n' >&2
	exit 2
}

if [ -z "${VERIFY_KAWA_EVIDENCE:-}" ]; then
	printf 'capture: VERIFY_KAWA_EVIDENCE is not set\n' >&2
	exit 1
fi

if [ "$#" -lt 3 ]; then
	usage
fi
name=$1
shift
if [ "$1" != "--" ]; then
	usage
fi
shift

case "$name" in
""|*/*|*" "*)
	printf 'capture: NAME must be a single path segment\n' >&2
	exit 2
	;;
esac

if [ "$#" -lt 1 ]; then
	usage
fi

mkdir -p "$VERIFY_KAWA_EVIDENCE"
cmd_file="$VERIFY_KAWA_EVIDENCE/$name.cmd"
out_file="$VERIFY_KAWA_EVIDENCE/$name.stdout"
err_file="$VERIFY_KAWA_EVIDENCE/$name.stderr"
exit_file="$VERIFY_KAWA_EVIDENCE/$name.exit"

printf '%s\n' "$*" >"$cmd_file"

if [ -n "${CAPTURE_CWD:-}" ]; then
	cd "$CAPTURE_CWD"
elif [ -n "${VERIFY_KAWA_ROOT:-}" ]; then
	cd "$VERIFY_KAWA_ROOT"
fi

# Prefer isolated helper when the command is wayland-registry.
set -- "$@"
if [ "$1" = "wayland-registry" ]; then
	if [ -z "${VERIFY_KAWA_HELPER:-}" ]; then
		printf 'capture: VERIFY_KAWA_HELPER is not set\n' >&2
		exit 1
	fi
	shift
	set -- "$VERIFY_KAWA_HELPER" "$@"
fi

set +e
"$@" >"$out_file" 2>"$err_file"
st=$?
set -e
printf '%s\n' "$st" >"$exit_file"
exit "$st"
