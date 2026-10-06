package kawa

import (
	"slices"

	"deedles.dev/wlr"
)

// focusTarget is what focus gives the keyboard to: a window if view is
// set, a layer surface if layer is set, or nothing if neither is.
type focusTarget struct {
	view *View
	// surface is the surface of view that gets the keyboard. If it's
	// the zero value, the view's own surface does.
	surface wlr.Surface
	layer   *LayerSurface
}

// focus gives the keyboard to t. Nothing else moves the keyboard, so
// it also keeps window activation, stacking, and the status bar title
// in step with it.
func (server *Server) focus(t focusTarget) {
	switch {
	case t.layer != nil:
		if ex := server.exclusiveLayer(); (ex != nil) && (ex != t.layer) {
			return
		}

		s := t.layer.LayerSurface.Surface()
		if server.seat.KeyboardState().FocusedSurface() == s {
			return
		}

		if fv := server.focusedView(); fv != nil {
			server.prevFocus = fv
			fv.SetActivated(false)
		}
		server.keyboardEnter(s)
		server.updateTitles()

	case t.view != nil:
		view, s := t.view, t.surface
		if !s.Valid() {
			s = view.Surface()
		}
		if !s.Valid() && !view.Mapped() {
			return
		}

		// The window gets the keyboard once the layer surface that has
		// it to itself lets go.
		if server.exclusiveLayer() != nil {
			server.prevFocus = view
			return
		}

		pv := server.focusedView()
		if pv == view {
			return
		}
		if pv != nil {
			pv.SetActivated(false)
		}

		server.keyboardEnter(s)

		view.attention = false
		view.SetActivated(true)
		server.bringViewToFront(view)

		server.updateTitles()

	default:
		server.seat.KeyboardNotifyClearFocus()
		server.updateTitles()
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
		server.focus(focusTarget{layer: ls})
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
	server.focus(focusTarget{view: view})
}
