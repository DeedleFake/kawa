package main

import (
	"fmt"
	"slices"

	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
	"deedles.dev/xiter"
)

type ViewTargeter interface {
	TargetView() *View
}

var edgeCursors = [...]string{
	wlr.EdgeNone:                   "",
	wlr.EdgeTop:                    "top_side",
	wlr.EdgeLeft:                   "left_side",
	wlr.EdgeRight:                  "right_side",
	wlr.EdgeBottom:                 "bottom_side",
	wlr.EdgeTop | wlr.EdgeLeft:     "top_left_corner",
	wlr.EdgeTop | wlr.EdgeRight:    "top_right_corner",
	wlr.EdgeBottom | wlr.EdgeLeft:  "bottom_left_corner",
	wlr.EdgeBottom | wlr.EdgeRight: "bottom_right_corner",
}

const (
	moveCursor     = "move"
	interactCursor = "hand"
)

type View struct {
	ViewSurface
	Coords  geom.Point[float64]
	Restore geom.Rect[float64]
	CSD     bool

	popups []*Popup

	onMapListener             wlr.Listener
	onDestroyListener         wlr.Listener
	onRequestMoveListener     wlr.Listener
	onRequestResizeListener   wlr.Listener
	onRequestMinimizeListener wlr.Listener
	onRequestMaximizeListener wlr.Listener
	onSetTitleListener        wlr.Listener
}

func (view *View) Release() {
	view.onDestroyListener.Destroy()
	view.onMapListener.Destroy()
	view.onRequestMoveListener.Destroy()
	view.onRequestResizeListener.Destroy()
	view.onRequestMinimizeListener.Destroy()
	view.onRequestMaximizeListener.Destroy()
	view.onSetTitleListener.Destroy()
}

func (view *View) Bounds() geom.Rect[float64] {
	return geom.RConv[float64](view.Geometry()).Add(view.Coords)
}

func (view *View) addPopup(surface wlr.XDGSurface) {
	p := Popup{
		Surface: surface,
	}
	p.onDestroyListener = surface.OnDestroy(func(s wlr.XDGSurface) {
		view.onDestroyPopup(&p)
	})

	view.popups = append(view.popups, &p)
}

func (view *View) onDestroyPopup(p *Popup) {
	p.Release()

	i := slices.Index(view.popups, p)
	view.popups = slices.Delete(view.popups, i, i+1)
}

func (view *View) isPopupSurface(surface wlr.Surface) (ok bool) {
	for _, p := range view.popups {
		for s := range p.Surface.Surfaces() {
			if s.Surface == surface {
				return true
			}
		}
	}
	return false
}

func surfaceBounds(s wlr.Surface) geom.Rect[int] {
	c := s.Current()
	return geom.Rt(0, 0, c.Width(), c.Height())
}

type Popup struct {
	Surface wlr.XDGSurface

	onDestroyListener wlr.Listener
}

func (p *Popup) Release() {
	p.onDestroyListener.Destroy()
}

func (server *Server) targetView() *View {
	m, ok := server.inputMode.(ViewTargeter)
	if !ok {
		return nil
	}

	return m.TargetView()
}

func (server *Server) viewAt(out *Output, p geom.Point[float64]) (*View, wlr.Edges, wlr.Surface, geom.Point[float64]) {
	if out == nil {
		out = server.outputAt(p)
	}

	i, edges, surface, sp := server.viewIndexAt(out, server.views, p)
	if i >= 0 {
		return server.views[i], edges, surface, sp
	}

	i, edges, surface, sp = server.viewIndexAt(out, server.tiled, p)
	if i >= 0 {
		return server.tiled[i], edges, surface, sp
	}

	return nil, wlr.EdgeNone, wlr.Surface{}, geom.Point[float64]{}
}

func (server *Server) viewIndexAt(out *Output, views []*View, p geom.Point[float64]) (int, wlr.Edges, wlr.Surface, geom.Point[float64]) {
	for i := len(views) - 1; i >= 0; i-- {
		view := views[i]
		if !view.Mapped() {
			continue
		}

		edges, surface, sp, ok := server.isViewAt(out, view, p)
		if ok {
			return i, edges, surface, sp
		}
	}

	return -1, 0, wlr.Surface{}, geom.Point[float64]{}
}

func (server *Server) isViewAt(out *Output, view *View, p geom.Point[float64]) (edges wlr.Edges, s wlr.Surface, sp geom.Point[float64], ok bool) {
	surface, sp, ok := view.SurfaceAt(p.Sub(view.Coords))
	if ok {
		return wlr.EdgeNone, surface, sp, true
	}

	// Don't bother checking the borders if there aren't any.
	if view.CSD {
		return 0, wlr.Surface{}, geom.Point[float64]{}, false
	}

	r := view.Bounds()
	if !p.In(r.Inset(-WindowBorder)) {
		return 0, wlr.Surface{}, geom.Point[float64]{}, false
	}

	left := geom.Rt(r.Min.X-WindowBorder, r.Min.Y, r.Max.X, r.Max.Y)
	if p.In(left) {
		return wlr.EdgeLeft, wlr.Surface{}, geom.Point[float64]{}, true
	}

	top := geom.Rt(r.Min.X, r.Min.Y-WindowBorder, r.Max.X, r.Max.Y)
	if p.In(top) {
		return wlr.EdgeTop, wlr.Surface{}, geom.Point[float64]{}, true
	}

	right := geom.Rt(r.Min.X, r.Min.Y, r.Max.X+WindowBorder, r.Max.Y)
	if p.In(right) {
		return wlr.EdgeRight, wlr.Surface{}, geom.Point[float64]{}, true
	}

	bottom := geom.Rt(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y+WindowBorder)
	if p.In(bottom) {
		return wlr.EdgeBottom, wlr.Surface{}, geom.Point[float64]{}, true
	}

	if (p.X < r.Min.X) && (p.Y < r.Min.Y) {
		return wlr.EdgeTop | wlr.EdgeLeft, wlr.Surface{}, geom.Point[float64]{}, true
	}
	if (p.X >= r.Max.X) && (p.Y < r.Min.Y) {
		return wlr.EdgeTop | wlr.EdgeRight, wlr.Surface{}, geom.Point[float64]{}, true
	}
	if (p.X < r.Min.X) && (p.Y >= r.Max.Y) {
		return wlr.EdgeBottom | wlr.EdgeLeft, wlr.Surface{}, geom.Point[float64]{}, true
	}
	if (p.X >= r.Max.X) && (p.Y >= r.Max.Y) {
		return wlr.EdgeBottom | wlr.EdgeRight, wlr.Surface{}, geom.Point[float64]{}, true
	}

	// Where else could it possibly be if it gets to here?
	panic(fmt.Errorf("If you see this, there's a bug.\np = %+v\nr = %+v", p, r))
}

func (server *Server) onNewXwaylandSurface(surface wlr.XwaylandSurface) {
	view := View{
		CSD:         false,
		ViewSurface: &viewSurfaceXwayland{s: surface},
	}
	view.onDestroyListener = surface.OnDestroy(func(s wlr.XwaylandSurface) {
		server.onDestroyView(&view)
	})
	view.onMapListener = surface.Surface().OnMap(func(s wlr.Surface) {
		server.onMapView(&view)
	})
	view.onRequestMoveListener = surface.OnRequestMove(func(s wlr.XwaylandSurface) {
		server.startMove(&view)
	})
	view.onRequestResizeListener = surface.OnRequestResize(func(s wlr.XwaylandSurface, edges wlr.Edges) {
		if !server.isViewTiled(&view) {
			server.startBorderResize(&view, edges)
		}
	})
	view.onRequestMinimizeListener = surface.OnRequestMinimize(func(s wlr.XwaylandSurface) {
		server.hideView(&view)
	})
	view.onRequestMaximizeListener = surface.OnRequestMaximize(func(s wlr.XwaylandSurface) {
		server.toggleViewTiling(&view)
	})
	view.onSetTitleListener = surface.OnSetTitle(func(s wlr.XwaylandSurface, title string) {
		server.updateTitles()
	})

	server.addView(&view)
}

func (server *Server) onNewXDGSurface(surface wlr.XDGSurface) {
	switch surface.Role() {
	case wlr.XDGSurfaceRoleToplevel:
		server.addXDGToplevel(surface)
	case wlr.XDGSurfaceRolePopup:
		server.addXDGPopup(surface)
	case wlr.XDGSurfaceRoleNone:
		// TODO
	}
}

func (server *Server) addXDGPopup(surface wlr.XDGSurface) {
	parent := server.viewForSurface(surface.Popup().Parent())
	if parent == nil {
		wlr.Log(wlr.Debug, "parent of popup could not be found")
		return
	}

	parent.addPopup(surface)
}

func (server *Server) addXDGToplevel(surface wlr.XDGSurface) {
	view := View{
		CSD:         true,
		ViewSurface: &viewSurfaceXDG{s: surface},
	}
	view.onDestroyListener = surface.OnDestroy(func(s wlr.XDGSurface) {
		server.onDestroyView(&view)
	})
	view.onMapListener = surface.Surface().OnMap(func(s wlr.Surface) {
		server.onMapView(&view)
	})
	view.onRequestMoveListener = surface.Toplevel().OnRequestMove(func(t wlr.XDGToplevel, client wlr.SeatClient, serial uint32) {
		server.startMove(&view)
	})
	view.onRequestResizeListener = surface.Toplevel().OnRequestResize(func(t wlr.XDGToplevel, client wlr.SeatClient, serial uint32, edges wlr.Edges) {
		if !server.isViewTiled(&view) {
			server.startBorderResize(&view, edges)
		}
	})
	view.onRequestMinimizeListener = surface.Toplevel().OnRequestMinimize(func(t wlr.XDGToplevel) {
		server.hideView(&view)
	})
	view.onRequestMaximizeListener = surface.Toplevel().OnRequestMaximize(func(t wlr.XDGToplevel) {
		server.toggleViewTiling(&view)
	})
	view.onSetTitleListener = surface.Toplevel().OnSetTitle(func(t wlr.XDGToplevel, title string) {
		server.updateTitles()
	})

	server.addView(&view)
}

func (server *Server) onDestroyView(view *View) {
	view.Release()

	i := slices.Index(server.views, view)
	if i >= 0 {
		server.views = slices.Delete(server.views, i, i+1)
	}
	i = slices.Index(server.tiled, view)
	if i >= 0 {
		server.tiled = slices.Delete(server.tiled, i, i+1)
		server.layoutTiles(nil)
	}

	server.updateTitles()
	allviews := xiter.Concat(slices.Values(server.tiled), slices.Values(server.views))
	if n, ok := xiter.Drain(allviews); ok {
		server.focusView(n, n.Surface())
	}
}

func (server *Server) onMapView(view *View) {
	pid := view.PID()

	nv, ok := server.newViews[pid]
	if ok {
		delete(server.newViews, pid)
		server.startBorderResizeFrom(view, wlr.EdgeNone, *nv)
		return
	}

	out := server.outputAt(server.cursorCoords())
	if out == nil {
		if len(server.outputs) == 0 {
			return
		}
		out = server.outputs[0]
	}

	server.centerViewOnOutput(out, view)
}

func (server *Server) addView(view *View) {
	server.views = append(server.views, view)

	nv, ok := server.newViews[view.PID()]
	if ok {
		server.resizeViewTo(nil, view, *nv)
	}
}

func (server *Server) centerViewOnOutput(out *Output, view *View) {
	ob := server.outputBounds(out)
	vb := view.Bounds()
	p := vb.CenterAt(ob.Center())

	server.moveViewTo(out, view, p.Min)
}

func (server *Server) moveViewTo(out *Output, view *View, p geom.Point[float64]) {
	if out == nil {
		out = server.outputAt(p)
	}

	view.Coords = p

	if out != nil {
		view.Surface().SendEnter(out.Output)
	}
}

func (server *Server) resizeViewTo(out *Output, view *View, r geom.Rect[float64]) {
	if out == nil {
		out = server.outputAt(r.Min)
	}

	vb := view.Bounds()
	off := view.Coords.Sub(vb.Min)
	r = r.Add(off).Canon()

	view.Coords = r.Min
	view.Resize(int(r.Dx()), int(r.Dy()))

	if out != nil {
		view.Surface().SendEnter(out.Output)
	}
}

func (server *Server) focusView(view *View, s wlr.Surface) {
	if !s.Valid() {
		if !view.Mapped() {
			return
		}
		s = view.Surface()
	}

	pv := server.focusedView()
	if pv == view {
		return
	}
	if pv != nil {
		pv.SetActivated(false)
	}

	k := server.seat.GetKeyboard()
	server.seat.KeyboardNotifyEnter(s, k.Keycodes(), k.Modifiers())

	view.SetActivated(true)
	server.bringViewToFront(view)

	server.updateTitles()
}

func (server *Server) focusedView() *View {
	s := server.seat.KeyboardState().FocusedSurface()
	return server.viewForSurface(s)
}

func (server *Server) viewForSurface(s wlr.Surface) *View {
	for _, view := range server.views {
		if view.HasSurface(s) {
			return view
		}
	}
	for _, view := range server.tiled {
		if view.HasSurface(s) {
			return view
		}
	}
	for _, view := range server.hidden {
		if view.HasSurface(s) {
			return view
		}
	}

	return nil
}

func (server *Server) bringViewToFront(view *View) {
	if server.isViewTiled(view) {
		return
	}

	i := slices.Index(server.views, view)
	server.views = slices.Delete(server.views, i, i+1)
	server.views = append(server.views, view)
}

func (server *Server) hideView(view *View) {
	// TODO: Remember whether or not a hidden view was tiled.
	if server.isViewTiled(view) {
		server.untileView(view, true)
	}
	i := slices.Index(server.views, view)
	if i >= 0 {
		server.views = slices.Delete(server.views, i, i+1)
	}

	server.hidden = append(server.hidden, view)
	view.SetMinimized(true)

	item := NewTextMenuItem(server.renderer, view.Title())
	item.OnSelect = func() {
		server.unhideView(view)
	}
	server.mainMenu.Add(item)
}

func (server *Server) unhideView(view *View) {
	i := slices.Index(server.hidden, view)
	server.hidden = slices.Delete(server.hidden, i, i+1)

	mi := server.mainMenu.Item(len(mainMenuText) + i)
	server.mainMenu.Remove(mi)
	mi.Release()

	server.views = append(server.views, view)
	server.focusView(view, view.Surface())
	view.SetMinimized(false)
}

func (server *Server) toggleViewTiling(view *View) {
	if server.isViewTiled(view) {
		server.untileView(view, true)
		return
	}
	server.tileView(view)
}

func (server *Server) tileView(view *View) {
	if !view.Mapped() {
		return
	}

	i := slices.Index(server.views, view)
	server.views = slices.Delete(server.views, i, i+1)
	server.tiled = append(server.tiled, view)

	view.Restore = DefaultRestore
	if s := view.Surface(); s.Valid() {
		view.Restore = view.Bounds()
	}
	view.SetMaximized(true) // TODO: Fix the race condition between this and resizing the view.

	server.layoutTiles(nil)
	server.focusView(view, view.Surface())
}

func (server *Server) untileView(view *View, restore bool) {
	i := slices.Index(server.tiled, view)
	server.tiled = slices.Delete(server.tiled, i, i+1)
	server.views = append(server.views, view)

	server.layoutTiles(nil)
	server.focusView(view, view.Surface())

	view.SetMaximized(false)
	if restore && !view.Restore.IsZero() {
		server.resizeViewTo(nil, view, view.Restore)
	}
}

func (server *Server) layoutTiles(out *Output) {
	if len(server.tiled) == 0 {
		return
	}

	if out == nil {
		out = server.outputs[0]
	}

	or := server.outputTilingBounds(out)
	tiles := geom.TiledRows(len(server.tiled), or, 4)
	for i, tile := range xiter.Enumerate(tiles) {
		tile = tile.Inset(3 * WindowBorder)
		server.resizeViewTo(out, server.tiled[i], tile)
	}
}

func (server *Server) isViewTiled(view *View) bool {
	return slices.Contains(server.tiled, view)
}

func (server *Server) closeView(view *View) {
	view.Close()
}

func (server *Server) onNewDecoration(dm wlr.ServerDecorationManager, d wlr.ServerDecoration) {
	var view *View
	for s := range d.Surface().Surfaces() {
		if view == nil {
			view = server.viewForSurface(s.Surface)
		}
	}
	if view == nil {
		return
	}

	view.CSD = d.Mode() != wlr.ServerDecorationManagerModeServer

	var onModeListener, onDestroyListener wlr.Listener
	onModeListener = d.OnMode(func(d wlr.ServerDecoration) {
		view.CSD = d.Mode() != wlr.ServerDecorationManagerModeServer
	})
	onDestroyListener = d.OnDestroy(func(d wlr.ServerDecoration) {
		onModeListener.Destroy()
		onDestroyListener.Destroy()
	})
}

func (server *Server) onNewToplevelDecoration(dm wlr.XDGDecorationManagerV1, d wlr.XDGToplevelDecorationV1) {
	var view *View
	for s := range d.Toplevel().Base().Surfaces() {
		if view == nil {
			view = server.viewForSurface(s.Surface)
		}
	}
	if view == nil {
		// If there's no view, there's probably no point.
		return
	}

	view.CSD = false
	d.SetMode(wlr.XDGToplevelDecorationV1ModeServerSide)

	var onDestroyListener wlr.Listener
	onDestroyListener = d.OnDestroy(func(d wlr.XDGToplevelDecorationV1) {
		onDestroyListener.Destroy()
	})
}

func (server *Server) updateTitles() {
	// Not the best way to do this, perhaps...
	for _, view := range server.hidden {
		item := server.mainMenu.Item(len(mainMenuText))
		item.Release()

		n := NewTextMenuItem(server.renderer, view.Title())
		n.OnSelect = item.OnSelect

		server.mainMenu.Remove(item)
		item.Release()
		server.mainMenu.Add(n)
	}

	var focusedTitle string
	if fv := server.focusedView(); fv != nil {
		focusedTitle = fv.Title()
	}
	server.statusBar.SetTitle(server.renderer, focusedTitle)
}
