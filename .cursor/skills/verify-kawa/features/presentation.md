# Presentation time and GTK4

kawa advertises `wp_presentation` and reports each surface as presented when its texture is drawn on an output (`internal/kawa/render.go`). GTK4 uses that feedback to pace frames, so GTK4 apps animate and respond normally.

## Sub-features

- `presentation-global` — `wp_presentation` is in the registry.
- `presentation-feedback` — clients get `presented` events while they draw.
- `gtk4-app` — a GTK4 app maps, redraws on input, and its title follows its state in the bar.

## How to get to it (user POV)

- Run any GTK4 app (`gtk4-demo`) in kawa and click around.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven; `gtk4-demo` installed (`/usr/bin/gtk4-demo`, or the `org.gtk.Demo4` flatpak).

- `bgc gtk4 env WAYLAND_DEBUG=1 GDK_BACKEND=wayland gtk4-demo`; `$shot gtk4-demo`.
- Count `grep -c 'wp_presentation_feedback#[0-9]*\.presented' "$VERIFY_KAWA_EVIDENCE/gtk4.stderr"`, click a few demo entries (`$shot gtk4-demo-clicked`), count again; the number grows and the bar shows the new title.
- `presentation-global`: the `wayland-registry` capture lists `wp_presentation`.

## Gotchas

- `WAYLAND_DEBUG` output is large; keep it in evidence, not on the terminal.
- GTK4 apps need the session bus that `launch.sh` exports.
