# Background image

Optional wallpaper via `-bg PATH` and `-bgscale` (`stretch`, `center`, `fit`, `fill`; default `stretch`). On success kawa logs `loaded %q as background`. The image is placed in the area below the status bar.

## Sub-features

- `bg-load` — decode the image and create a texture.
- `bg-scale` — `stretch` fills the area ignoring aspect; `center` draws it at natural size, centered; `fill` scales it to fit inside the area, keeping aspect, centered (letterboxed); `fit` is `fill` for an image larger than the area and natural size otherwise.
- `bg-errors` — a bad path or undecodable file logs `load "…" as background: …` or `decode "…" as background: …` and kawa runs without a background.

## How to get to it (user POV)

- `kawa -bg /path/to/image.png -bgscale fit`

## Driving it with process + Wayland client

Preconditions: baseline; two test images, e.g. `ffmpeg -f lavfi -i testsrc=size=200x100:rate=1 -frames:v 1 small.png` and the same at `1600x900` as `big.png`.

- For each mode, `cleanup.sh` any prior instance, then `launch.sh -bg "$PWD/small.png" -bgscale MODE`, `sleep 1`, `$shot background-small-MODE`. Also `-bg big.png -bgscale fit` (`$shot background-big-fit`).
- Proof: stderr contains `loaded "..." as background`, the ready line arrives, and the screenshots match the modes above.
- `bg-errors`: `launch.sh -bg /nonexistent.png` still reaches the ready line and logs `load "/nonexistent.png" as background: open /nonexistent.png: no such file or directory`.

## Gotchas

- Take the screenshot a second after launch; right after the ready line the window may not have drawn yet and comes out white.
- With `fit` and an image smaller than the screen, kawa currently draws the image at the output's top-left corner, partly under the status bar, instead of centering it. That is a product bug; `center` places it correctly.
- Unknown `-bgscale` is rejected during flag parsing: kawa prints `invalid value "..." for flag -bgscale: unknown scaling method: "..."` plus usage and exits with status 2 before starting, so `launch.sh` reports `kawa died before client connect`.
