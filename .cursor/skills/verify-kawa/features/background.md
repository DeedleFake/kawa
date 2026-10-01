# Background image

Optional wallpaper via `-bg PATH` and `-bgscale` (`stretch`, `center`, `fit`, `fill`). On success kawa logs `loaded %q as background`.

## Sub-features

- `bg-load` — decode image and create texture.
- `bg-scale` — scaling mode from `-bgscale`.

## How to get to it (user POV)

- `kawa -bg /path/to/image.png -bgscale fit`

## Driving it with process + Wayland client

Preconditions: baseline; a small PNG/JPEG under `$VERIFY_KAWA_HOME`.

- Stop any prior instance (`cleanup.sh`), then `launch.sh -bg "$VERIFY_KAWA_HOME/bg.png" -bgscale fit`.
- Proof: stderr contains `loaded "..." as background`, the compositor reaches the ready line, and `screenshot.sh background` shows the image.

## Gotchas

- Bad path/decode logs an error and continues without a background.
- Unknown `-bgscale` is rejected during flag parsing: kawa prints `invalid value "..." for flag -bgscale: unknown scaling method: "..."` plus usage to stderr and exits with status 2 before starting, so `launch.sh` reports `kawa died before client connect`.
