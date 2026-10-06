package kawa

import (
	"slices"

	"deedles.dev/wlr"
)

// A focusTarget is something that focus can give the keyboard to.
type focusTarget interface {
	// focusSurface returns the surface that gets the keyboard, or false
	// if t can't have it now or already has it.
	focusSurface(server *Server) (wlr.Surface, bool)
	// focused is called once t has the keyboard. prev is the window
	// that had it, if any.
	focused(server *Server, prev *View)
}

// focus gives the keyboard to t. Only focus and clearFocus move the
// keyboard, so they also keep window activation, stacking, and the
// status bar title in step with it.
func (server *Server) focus(t focusTarget) {
	s, ok := t.focusSurface(server)
	if !ok {
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

func (server *Server) clearFocus() {
	server.seat.KeyboardNotifyClearFocus()
	server.updateTitles()
}

func (view *View) focusSurface(server *Server) (wlr.Surface, bool) {
	return viewContent{view, view.Surface()}.focusSurface(server)
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

func (c viewContent) focusSurface(server *Server) (wlr.Surface, bool) {
	if !c.surface.Valid() {
		return wlr.Surface{}, false
	}

	// The window gets the keyboard once the layer surface that has it
	// to itself lets go.
	if server.exclusiveLayer() != nil {
		server.prevFocus = c.View
		return wlr.Surface{}, false
	}

	return c.surface, server.focusedView() != c.View
}

func (ls *LayerSurface) focusSurface(server *Server) (wlr.Surface, bool) {
	if ex := server.exclusiveLayer(); (ex != nil) && (ex != ls) {
		return wlr.Surface{}, false
	}

	s := ls.LayerSurface.Surface()
	return s, server.seat.KeyboardState().FocusedSurface() != s
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
