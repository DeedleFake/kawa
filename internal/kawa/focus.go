package kawa

import (
	"slices"

	"deedles.dev/wlr"
)

// A focusTarget is something that focus can give the keyboard to.
type focusTarget interface {
	// focusSurface returns the surface that gets the keyboard. It may
	// be invalid, in which case focus does nothing.
	focusSurface() wlr.Surface
	// view returns the window behind t, or nil if t isn't part of one.
	view() *View
	// focused is called once t has the keyboard. prev is the window
	// that had it, if any.
	focused(server *Server, prev *View)
}

// focus gives the keyboard to t. Only focus and clearFocus move the
// keyboard, so they also keep window activation, stacking, and the
// status bar title in step with it.
func (server *Server) focus(t focusTarget) {
	s := t.focusSurface()
	if !s.Valid() {
		return
	}

	if ex := server.exclusiveLayer(); (ex != nil) && (ex.LayerSurface.Surface() != s) {
		// A window gets the keyboard once the layer surface that has
		// it to itself lets go.
		if v := t.view(); v != nil {
			server.prevFocus = v
		}
		return
	}

	if server.hasFocus(t, s) {
		return
	}

	prev := server.focusedView()
	if prev != nil {
		prev.SetActivated(false)
	}
	server.keyboardEnter(s)
	t.focused(server, prev)
	server.updateTitles()
}

// hasFocus reports whether t already has the keyboard. A window has it
// if any of its surfaces do.
func (server *Server) hasFocus(t focusTarget, s wlr.Surface) bool {
	if v := t.view(); v != nil {
		return server.focusedView() == v
	}
	return server.seat.KeyboardState().FocusedSurface() == s
}

func (server *Server) clearFocus() {
	server.seat.KeyboardNotifyClearFocus()
	server.updateTitles()
}

func (view *View) focusSurface() wlr.Surface {
	return view.Surface()
}

func (view *View) view() *View {
	return view
}

func (view *View) focused(server *Server, prev *View) {
	view.attention = false
	view.SetActivated(true)
	server.bringViewToFront(view)
}

// viewContent is a surface of a window, such as a subsurface or a
// popup, that gets the keyboard instead of the window's own surface.
type viewContent struct {
	*View
	surface wlr.Surface
}

func (c viewContent) focusSurface() wlr.Surface {
	return c.surface
}

func (ls *LayerSurface) focusSurface() wlr.Surface {
	return ls.LayerSurface.Surface()
}

func (ls *LayerSurface) view() *View {
	return nil
}

func (ls *LayerSurface) focused(server *Server, prev *View) {
	if prev != nil {
		server.prevFocus = prev
	}
}

func (server *Server) keyboardEnter(s wlr.Surface) {
	if k := server.seat.GetKeyboard(); k.Valid() {
		server.seat.KeyboardNotifyEnter(s, k.Keycodes(), k.Modifiers())
	} else {
		server.seat.KeyboardNotifyEnter(s, nil, wlr.KeyboardModifiers{})
	}
}

func (server *Server) focusedView() *View {
	s := server.seat.KeyboardState().FocusedSurface()
	return server.viewForSurface(s)
}

func (server *Server) focusedLayer() *LayerSurface {
	s := server.seat.KeyboardState().FocusedSurface()
	if !s.Valid() {
		return nil
	}
	return server.layerForSurface(s)
}

// topView returns the window on top, preferring floating windows to
// tiled ones, or nil if there are no windows.
func (server *Server) topView() *View {
	if n := len(server.views); n > 0 {
		return server.views[n-1]
	}
	if n := len(server.tiled); n > 0 {
		return server.tiled[n-1]
	}
	return nil
}

// exclusiveLayer returns the topmost mapped layer surface in the top or
// overlay layer that wants the keyboard to itself. The protocol only
// lets those two layers lock the keyboard.
func (server *Server) exclusiveLayer() *LayerSurface {
	for _, layer := range []wlr.LayerShellV1Layer{wlr.LayerShellV1LayerOverlay, wlr.LayerShellV1LayerTop} {
		for _, out := range server.outputs {
			list := out.Layers[layer]
			for _, ls := range slices.Backward(list) {
				if ls.Mapped() && (ls.LayerSurface.Current().KeyboardInteractive() == wlr.LayerSurfaceV1KeyboardInteractivityExclusive) {
					return ls
				}
			}
		}
	}
	return nil
}

// updateLayerFocus gives the keyboard to a layer surface that wants it
// to itself, and gives it back once the layer surface that has it
// can't keep it.
func (server *Server) updateLayerFocus() {
	if ls := server.exclusiveLayer(); ls != nil {
		server.focus(ls)
		return
	}

	ls := server.focusedLayer()
	if ls == nil {
		return
	}
	if ls.Mapped() && (ls.LayerSurface.Current().KeyboardInteractive() != wlr.LayerSurfaceV1KeyboardInteractivityNone) {
		return
	}
	server.restoreFocus()
}

// restoreFocus gives the keyboard back to the window that had it before
// a layer surface took it. If that window is gone or hidden, the
// topmost window gets it instead.
func (server *Server) restoreFocus() {
	view := server.prevFocus
	server.prevFocus = nil
	if !slices.Contains(server.views, view) && !slices.Contains(server.tiled, view) {
		view = server.topView()
	}
	if view == nil {
		server.clearFocus()
		return
	}
	server.focus(view)
}
