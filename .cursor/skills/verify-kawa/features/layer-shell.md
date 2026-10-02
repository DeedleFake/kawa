# Layer shell

kawa implements `zwlr_layer_shell_v1` (`internal/kawa/layers.go`) for wallpapers, panels, launchers, and notifications. Surfaces with an exclusive zone shrink the area used for new windows and tiling. Top and overlay surfaces that ask for exclusive keyboard get it and give it back when they let go, and layer popups (panel menus) are kept on screen below the status bar and drawn above everything.

## Sub-features

- `layer-background` — background-layer surfaces (`swaybg`) draw under windows. kawa's own `-bg` is not needed.
- `layer-zone` — exclusive zones (a `waybar` panel) are taken out of the usable area, so the New box, tiling, and popups stay clear of them. Zones are arranged starting from the overlay layer, and the usable area starts below kawa's status bar. `exclusive_zone -1` surfaces use the whole output.
- `layer-popup` — popups from layer surfaces (`waybar` GTK menus) open on screen and above all layers.
- `layer-keyboard` — exclusive keyboard on the top and overlay layers (`fuzzel`). Typing goes to it, the bar title is empty, and when it closes focus returns to the window that had it. `on_demand` surfaces get the keyboard when clicked.
- `layer-overlay` — overlay notifications (`mako`) sit above windows.
- Input order: layer popups, then overlay and top layers, then windows, then bottom and background layers.

## How to get to it (user POV)

- Run a wallpaper (`swaybg`), a panel (`waybar`), a launcher (`fuzzel`), or a notification daemon (`mako`) in kawa.

## Driving it with process + Wayland client

Preconditions: `compositor-boot` proven with `-terms weston-terminal` or `new-cmd.sh`; `swaybg`, `waybar`, `fuzzel`, `mako`, and `notify-send` installed. kawa's stderr logs `new layer surface "NAMESPACE" on X11-1 in layer N` for each one. Give each run its own config directory, `C=$VERIFY_KAWA_HOME/cfg`, and start every client with `bgc` (SKILL.md Drive) so `cleanup.sh` stops it.

- `layer-background`: `bgc swaybg swaybg -c '#336633'`; `$shot layer-swaybg`. A right-press on it still opens kawa's menu.
- `layer-zone`: write `$C/waybar/config` as `{"layer":"top","position":"top","height":30,"exclusive":true,"modules-left":["custom/menu"],"modules-right":["clock"],"custom/menu":{"format":"MENU","menu":"on-click","menu-file":"$C/waybar/menu.xml","menu-actions":{"one":"true"}},"clock":{"format":"{:%H:%M:%S}","interval":1}}` (with `$C` expanded) and `$C/waybar/menu.xml` as a GtkBuilder `GtkMenu` with id `menu` and a few `GtkMenuItem`s. `bgc waybar env XDG_CONFIG_HOME="$C" waybar`. Drag a `New` box that starts above the panel: the window is clamped below it (`$shot layer-waybar-zone`). Tile a window: it starts below the panel (`$shot layer-waybar-tiled`).
- `layer-popup`: click `MENU` on the panel; the menu opens below it (`$shot layer-waybar-popup`). Press Escape to close it.
- `layer-keyboard`: with a terminal focused and no popup open, `printf '[main]\nlayer=overlay\n' >"$C/fuzzel/fuzzel.ini"` and `bgc fuzzel env XDG_CONFIG_HOME="$C" fuzzel`. Type `zzq`: fuzzel filters and the bar title is empty (`$shot layer-fuzzel-exclusive`). Press Escape, type `echo back`: it lands in the terminal (`$shot layer-fuzzel-restored`).
- `layer-overlay`: `printf 'layer=overlay\nanchor=top-right\ndefault-timeout=0\n' >"$C/mako/config"`, `bgc mako mako -c "$C/mako/config"`, `notify-send hello`; the notification is at the top right, below the panel (`$shot layer-mako`).

## Gotchas

- Close any popup (waybar menu) before mapping an exclusive-keyboard surface. If an xdg_popup grab is open when fuzzel maps, the keyboard stays with the old window even after the popup closes. That is a product bug; don't record it as a layer-shell pass.
- A click on empty desktop does not dismiss an open popup. Press Escape, or click the popup's own surface.
- waybar and other GTK clients need a session bus. `launch.sh` starts a private one and exports it in `env.sh`; without it GTK autolaunches a bus on the X display that later runs share.
- Don't point clients at `~/.config`. Keep configs under `$VERIFY_KAWA_HOME` so runs don't depend on each other.
