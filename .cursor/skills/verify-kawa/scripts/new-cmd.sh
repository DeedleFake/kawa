#!/bin/sh
# A -terms entry that lets one kawa run start different programs from New.
# It runs the shell command line in $VERIFY_KAWA_HOME/new-cmd, or
# weston-terminal if that file is missing or empty. The command replaces
# this process (eval exec), so the window's PID is the one kawa started,
# which is how kawa matches a window to its New box.
# kawa throws away the output of what it starts, so it goes to
# $VERIFY_KAWA_HOME/logs/new.log instead.
set -eu
f="${VERIFY_KAWA_HOME:?}/new-cmd"
exec >>"$VERIFY_KAWA_HOME/logs/new.log" 2>&1
printf 'new-cmd: pid %s: %s\n' "$$" "$(cat "$f" 2>/dev/null || echo weston-terminal)"
if [ -s "$f" ]; then
	eval "exec $(cat "$f")"
fi
exec weston-terminal
