# Background image

Optional wallpaper via `-bg PATH` and `-bgscale` (`stretch`, `center`, `fit`, `fill`). On success kawa logs `loaded %q as background`.

## Sub-features

- `bg-load` — decode image and create texture.
- `bg-scale` — scaling mode from `-bgscale`.

## How to get to it (user POV)

- `kawa -bg /path/to/image.png -bgscale fit`

## Driving it with process + Wayland client

Preconditions: baseline; a small PNG/JPEG under `$VERIFY_KAWA_HOME`.

- Stop any prior instance (`cleanup.sh`), then start with background:
  `VERIFY_KAWA_HOME=...` and run kawa with `-bg` (extend launch by starting manually after build, or re-run with flags after adjusting the launch command).
- Proof: stderr contains `loaded "...\" as background` and compositor still reaches the ready line; optional screenshot later.
- First skill run may skip after documenting the log assert — boot without `-bg` is enough for `compositor-boot`.

## Gotchas

- Bad path/decode logs an error and continues without a background.
- Unknown `-bgscale` logs `unknown scaling method` and may leave scale unset.
