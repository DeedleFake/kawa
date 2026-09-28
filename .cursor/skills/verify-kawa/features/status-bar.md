# Status bar

A top status bar is drawn on the primary output. The focused window title is pushed into the bar via `StatusBar.SetTitle`. README plans a clock/time display; that is not separately logged today and needs a visual capture later.

## Sub-features

- `bar-chrome` — bar region (`StatusBarHeight`) rendered each frame on the status-bar output.
- `focused-title` — bar title tracks the focused view title.
- `time-display` — planned (README); not asserted by log today.

## How to get to it (user POV)

- Start kawa; look at the top of the compositor window.
- Focus a window; the bar title updates.
- Right-click the status bar for the system menu (`Log Out`).

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven.

- Without pointer injection or screenshot tooling, only confirm the compositor stays up after boot (PID + ready line). Optional: X11/`grim` capture of the kawa window if available — not required for first prove.
- Mark `time-display` deferred until a screenshot or on-screen OCR path exists.
- Do not call `SetTitle` from a test helper and call that proof.

## Gotchas

- Title updates are render-path side effects; there is no stdout clock line to grep.
- System menu vs main menu depends on click Y relative to the status bar (`mode.go`).
