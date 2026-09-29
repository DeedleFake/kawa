#!/bin/sh
# Grab kawa's X11 backend window into $VERIFY_KAWA_EVIDENCE/NAME.png.
set -eu

if [ -z "${VERIFY_KAWA_EVIDENCE:-}" ]; then
	printf 'screenshot: VERIFY_KAWA_EVIDENCE is not set\n' >&2
	exit 1
fi
if [ "$#" -ne 1 ]; then
	printf 'usage: screenshot.sh NAME\n' >&2
	exit 2
fi
case "$1" in
""|*/*|*" "*)
	printf 'screenshot: NAME must be a single path segment\n' >&2
	exit 2
	;;
esac

wid=$(xwininfo -root -tree | sed -n 's/^ *\(0x[0-9a-f]*\) "wlroots - X11-1".*/\1/p' | head -n1)
[ -n "$wid" ] || { printf 'screenshot: no "wlroots - X11-1" window on DISPLAY=%s\n' "${DISPLAY:-}" >&2; exit 1; }

mkdir -p "$VERIFY_KAWA_EVIDENCE"
xwd -silent -id "$wid" | ffmpeg -loglevel error -y -f xwd_pipe -i - "$VERIFY_KAWA_EVIDENCE/$1.png"
printf '%s\n' "$VERIFY_KAWA_EVIDENCE/$1.png"
