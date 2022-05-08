kawa
====

kawa is planned to be a Wayland compositor with an interface inspired by, though not directly copying, [Plan 9's rio](https://en.wikipedia.org/wiki/Rio_(windowing_system)) window manager. It isn't yet, however, so feel free to either send a pull request or come back later.

Planned Features
----------------

* Minimal interface besides windows, but not completely blank like rio. There should be, for example, a status bar with the current time and other useful global pieces of info.
* Ability to maximize windows. The status bar will still display, allowing it to be right-clicked to access the window management menu.
* Ability to access the global menus from anywhere by holding a key (Super?) and clicking.
* Window overview similar to GNOME shell's.
* Window starting system similar to rio's with terminal takeover, but with more capability for handling multi-window clients. This could be tricky, however, and will heavily depend on how far Wayland can be stretched to handle something like this.
* An exit feature. Maybe something in the status bar? It shouldn't be too easy to do accidentally, obviously.
* Support for fullscreen apps, such as games.
* Auto-focus of windows.

Wishful Thinking
----------------

* Touchscreen support. I'm not entirely sure how this would work, but since rio's design is heavily mouse-oriented, if it _does_ work it could be quite nice.
* Theming support.

Prior Art
---------

* [wio](https://gitlab.com/Rubo/wio)
* [wio+](https://notabug.org/Leon_Plickat/wio-plus)
