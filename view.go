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
	// Coords is the position of the view's window geometry, not of its
	// surface, so that the view stays in place when a client changes
	// its geometry offset, such as by dropping CSD shadows on maximize.
	Coords  geom.Point[float64]
	Restore geom.Rect[float64]
	CSD     bool

	popups []*Popup

	onMapListener             wlr.Listener
	onCommitListener          wlr.Listener
	onDestroyListener         wlr.Listener
	onAssociateListener       wlr.Listener
	onDissociateListener      wlr.Listener
	onRequestMoveListener     wlr.Listener
	onRequestResizeListener   wlr.Listener
	onRequestMinimizeListener wlr.Listener
	onRequestMaximizeListener wlr.Listener
	onSetTitleListener        wlr.Listener
}

func (view *View) Release() {
	view.onDestroyListener.Destroy()
	view.onMapListener.Destroy()
	view.onCommitListener.Destroy()
	view.onAssociateListener.Destroy()
	view.onDissociateListener.Destroy()
	view.onRequestMoveListener.Destroy()
	view.onRequestResizeListener.Destroy()
	view.onRequestMinimizeListener.Destroy()
	view.onRequestMaximizeListener.Destroy()
	view.onSetTitleListener.Destroy()
}

func (view *View) Bounds() geom.Rect[float64] {
	g := view.Geometry()
	return geom.RConv[float64](g.Sub(g.Min)).Add(view.Coords)
}

// surfaceCoords returns the position of the view's main surface.
func (view *View) surfaceCoords() geom.Point[float64] {
	return view.Coords.Sub(geom.PConv[float64](view.Geometry().Min))
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
	surface, sp, ok := view.SurfaceAt(p.Sub(view.surfaceCoords()))
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
	panic(fmt.Errorf("this should not have happened\np = %+v\nr = %+v", p, r))
}

func (server *Server) onNewXwaylandSurface(surface wlr.XwaylandSurface) {
	view := View{
		CSD:         false,
		ViewSurface: &viewSurfaceXwayland{s: surface},
	}
	view.onDestroyListener = surface.OnDestroy(func(s wlr.XwaylandSurface) {
		server.onDestroyView(&view)
	})
	// The wlr_surface doesn't exist until the X11 window is associated
	// with one, and it can go away again before the Xwayland surface is
	// destroyed.
	view.onAssociateListener = surface.OnAssociate(func(s wlr.XwaylandSurface) {
		view.onMapListener = s.Surface().OnMap(func(s wlr.Surface) {
			server.onMapView(&view)
		})
	})
	view.onDissociateListener = surface.OnDissociate(func(s wlr.XwaylandSurface) {
		view.onMapListener.Destroy()
		view.onMapListener = wlr.Listener{}
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

func (server *Server) onNewXDGToplevel(toplevel wlr.XDGToplevel) {
	server.addXDGToplevel(toplevel.Base())
}

func (server *Server) onNewXDGSurface(surface wlr.XDGSurface) {
	// Popups only; toplevels come from OnNewToplevel. The role isn't
	// assigned yet when new_surface fires, so check on the initial
	// commit instead.
	var onCommit, onDestroy wlr.Listener
	onCommit = surface.Surface().OnCommit(func(wlr.Surface) {
		if surface.InitialCommit() && (surface.Role() == wlr.XDGSurfaceRolePopup) {
			server.addXDGPopup(surface)
		}
	})
	onDestroy = surface.OnDestroy(func(wlr.XDGSurface) {
		onCommit.Destroy()
		onDestroy.Destroy()
	})
}

func (server *Server) addXDGPopup(surface wlr.XDGSurface) {
	// wlroots leaves configuring popups to the compositor, and the
	// client won't map one until it has been configured.
	surface.ScheduleConfigure()

	popup := surface.Popup()
	parent := server.viewForSurface(popup.Parent())
	if parent == nil {
		wlr.Log(wlr.Debug, "parent of popup could not be found")
		return
	}

	server.unconstrainPopup(parent, popup)
	parent.addPopup(surface)
}

// unconstrainPopup keeps a popup inside the usable part of the output
// that its toplevel is on, flipping or sliding it as its positioner
// allows. wlroots wants that area relative to the toplevel's surface,
// and works out where nested popups are from there.
func (server *Server) unconstrainPopup(view *View, popup wlr.XDGPopup) {
	out := server.outputAt(view.Bounds().Center())
	if out == nil {
		out = server.outputAt(server.cursorCoords())
	}
	if out == nil {
		return
	}

	box := server.outputTilingBounds(out).Sub(view.surfaceCoords())
	popup.UnconstrainFromBox(box.ImageRect())
}

func (server *Server) addXDGToplevel(surface wlr.XDGSurface) {
	vs := &viewSurfaceXDG{s: surface}
	view := View{
		CSD:         true,
		ViewSurface: vs,
	}
	// The toplevel is destroyed before the XDGSurface, and its listeners
	// have to be removed by then.
	view.onDestroyListener = surface.Toplevel().OnDestroy(func(t wlr.XDGToplevel) {
		server.onDestroyView(&view)
	})
	view.onMapListener = surface.Surface().OnMap(func(s wlr.Surface) {
		server.onMapView(&view)
	})
	// The toplevel can't be configured until the initial commit, and it
	// won't map until it is. A 0x0 size lets the client pick its own
	// unless it was started from New.
	view.onCommitListener = surface.Surface().OnCommit(func(s wlr.Surface) {
		if surface.InitialCommit() {
			// A client that unmaps starts over, and a size it never
			// acked must not hold back its first configure.
			vs.sizeSerial, vs.hasPending = 0, false
			if !server.resizeNewView(&view) {
				view.Resize(0, 0)
			}
			return
		}
		vs.onCommit()
	})
	// A client asks for a move or resize after it gets a button press.
	// If the button is already up by the time the request arrives, the
	// drag is over and there would be no release to end it.
	view.onRequestMoveListener = surface.Toplevel().OnRequestMove(func(t wlr.XDGToplevel, client wlr.SeatClient, serial uint32) {
		if len(server.pressed) > 0 {
			server.startMove(&view)
		}
	})
	view.onRequestResizeListener = surface.Toplevel().OnRequestResize(func(t wlr.XDGToplevel, client wlr.SeatClient, serial uint32, edges wlr.Edges) {
		if (len(server.pressed) > 0) && !server.isViewTiled(&view) {
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

	// A mode that is still holding on to the view, such as an
	// in-progress move or resize, would otherwise use it after it's
	// gone.
	if server.targetView() == view {
		server.startNormal()
	}
	if i := slices.Index(server.hidden, view); i >= 0 {
		server.removeHidden(i)
	}

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
	server.resizeNewView(view)
}

func (server *Server) resizeNewView(view *View) bool {
	nv, ok := server.newViews[view.PID()]
	if ok {
		server.resizeViewTo(nil, view, *nv)
	}
	return ok
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

	if s := view.Surface(); (out != nil) && s.Valid() {
		s.SendEnter(out.Output)
	}
}

func (server *Server) resizeViewTo(out *Output, view *View, r geom.Rect[float64]) {
	if out == nil {
		out = server.outputAt(r.Min)
	}

	r = r.Canon()

	view.Coords = r.Min
	view.Resize(int(r.Dx()), int(r.Dy()))

	if s := view.Surface(); (out != nil) && s.Valid() {
		s.SendEnter(out.Output)
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

	if k := server.seat.GetKeyboard(); k.Valid() {
		server.seat.KeyboardNotifyEnter(s, k.Keycodes(), k.Modifiers())
	} else {
		server.seat.KeyboardNotifyEnter(s, nil, wlr.KeyboardModifiers{})
	}

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
	server.removeHidden(slices.Index(server.hidden, view))

	server.views = append(server.views, view)
	server.focusView(view, view.Surface())
	view.SetMinimized(false)
}

// removeHidden removes the view at index i from the hidden list, along
// with its main menu item.
func (server *Server) removeHidden(i int) {
	server.hidden = slices.Delete(server.hidden, i, i+1)

	mi := server.mainMenu.Item(len(mainMenuText) + i)
	server.mainMenu.Remove(mi)
	mi.Release()
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
	view.SetMaximized(true, true)

	server.layoutTiles(nil)
	server.focusView(view, view.Surface())
}

func (server *Server) untileView(view *View, restore bool) {
	i := slices.Index(server.tiled, view)
	server.tiled = slices.Delete(server.tiled, i, i+1)
	server.views = append(server.views, view)

	server.layoutTiles(nil)
	server.focusView(view, view.Surface())

	view.SetMaximized(false, false)
	if restore && !view.Restore.IsZero() {
		server.resizeViewTo(nil, view, view.Restore)
	}
}

func (server *Server) layoutTiles(out *Output) {
	if len(server.tiled) == 0 {
		return
	}

	if out == nil {
		if len(server.outputs) == 0 {
			return
		}
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

	// The mode can't be sent until the toplevel's initial commit.
	base := d.Toplevel().Base()
	setMode := func() bool {
		if !base.Initialized() {
			return false
		}
		d.SetMode(wlr.XDGToplevelDecorationV1ModeServerSide)
		return true
	}

	var onCommitListener, onDestroyListener wlr.Listener
	if !setMode() {
		onCommitListener = base.Surface().OnCommit(func(s wlr.Surface) {
			if setMode() {
				onCommitListener.Destroy()
				onCommitListener = wlr.Listener{}
			}
		})
	}
	onDestroyListener = d.OnDestroy(func(d wlr.XDGToplevelDecorationV1) {
		onCommitListener.Destroy()
		onDestroyListener.Destroy()
	})
}

func (server *Server) updateTitles() {
	// Not the best way to do this, perhaps...
	for _, view := range server.hidden {
		item := server.mainMenu.Item(len(mainMenuText))

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
