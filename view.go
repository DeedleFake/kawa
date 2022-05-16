package main

import (
	"fmt"
	"image"

	"deedles.dev/kawa/geom"
	"deedles.dev/kawa/internal/util"
	"deedles.dev/kawa/tile"
	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
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

type View struct {
	ViewSurface
	Coords  geom.Point[float64]
	Restore geom.Rect[float64]
	CSD     bool

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

type NewView struct {
	To        *geom.Rect[float64]
	OnStarted func(*View)
}

type Popup struct {
	Surface wlr.XDGSurface

	onDestroyListener wlr.Listener
}

func (p *Popup) Release() {
	p.onDestroyListener.Destroy()
}

type Decoration struct {
	Decoration wlr.ServerDecoration

	onDestroyListener wlr.Listener
	onModeListener    wlr.Listener
}

func (d *Decoration) Release() {
	d.onDestroyListener.Destroy()
	d.onModeListener.Destroy()
}

func (server *Server) viewBounds(out *Output, view *View) geom.Rect[int] {
	var r geom.Rect[int]
	view.ForEachSurface(func(s wlr.Surface, sx, sy int) {
		if server.isPopupSurface(s) {
			return
		}

		sb := server.surfaceBounds(s, geom.Pt(sx, sy)).Add(geom.PConv[int](view.Coords))
		r = r.Union(sb)
	})
	return r
}

func (server *Server) surfaceBounds(s wlr.Surface, p geom.Point[int]) geom.Rect[int] {
	current := s.Current()
	return geom.Rt(0, 0, current.Width(), current.Height()).Add(p)
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

	r := geom.RConv[float64](server.viewBounds(nil, view))
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

func (server *Server) onNewXWaylandSurface(surface wlr.XWaylandSurface) {
	view := View{
		ViewSurface: &viewSurfaceXWayland{s: surface},
		Coords:      geom.Pt[float64](-1, -1),
	}
	view.onDestroyListener = surface.OnDestroy(func(s wlr.XWaylandSurface) {
		server.onDestroyView(&view)
	})
	view.onMapListener = surface.OnMap(func(s wlr.XWaylandSurface) {
		server.onMapView(&view)
	})
	view.onRequestMoveListener = surface.OnRequestMove(func(s wlr.XWaylandSurface) {
		server.startMove(&view)
	})
	view.onRequestResizeListener = surface.OnRequestResize(func(s wlr.XWaylandSurface, edges wlr.Edges) {
		if !server.isViewTiled(&view) {
			server.startBorderResize(&view, edges)
		}
	})
	view.onRequestMinimizeListener = surface.OnRequestMinimize(func(s wlr.XWaylandSurface) {
		server.hideView(&view)
	})
	view.onRequestMaximizeListener = surface.OnRequestMaximize(func(s wlr.XWaylandSurface) {
		server.toggleViewTiling(&view)
	})
	view.onSetTitleListener = surface.OnSetTitle(func(s wlr.XWaylandSurface, title string) {
		server.updateTitles()
	})

	server.addView(&view)
}

func (server *Server) onNewXDGSurface(surface wlr.XDGSurface) {
	switch surface.Role() {
	case wlr.XDGSurfaceRoleTopLevel:
		server.addXDGTopLevel(surface)
	case wlr.XDGSurfaceRolePopup:
		server.addXDGPopup(surface)
	case wlr.XDGSurfaceRoleNone:
		// TODO
	}
}

func (server *Server) addXDGPopup(surface wlr.XDGSurface) {
	p := Popup{
		Surface: surface,
	}
	p.onDestroyListener = surface.OnDestroy(func(s wlr.XDGSurface) {
		server.onDestroyPopup(&p)
	})

	server.popups = append(server.popups, &p)
}

func (server *Server) isPopupSurface(surface wlr.Surface) (ok bool) {
	for _, p := range server.popups {
		p.Surface.ForEachSurface(func(s wlr.Surface, sx, sy int) {
			if s == surface {
				ok = true
			}
		})
		if ok {
			return true
		}
	}
	return false
}

func (server *Server) onDestroyPopup(p *Popup) {
	p.Release()

	i := slices.Index(server.popups, p)
	server.popups = slices.Delete(server.popups, i, i+1)
}

func (server *Server) addXDGTopLevel(surface wlr.XDGSurface) {
	view := View{
		ViewSurface: &viewSurfaceXDG{s: surface},
		Coords:      geom.Pt[float64](-1, -1),
	}
	view.onDestroyListener = surface.OnDestroy(func(s wlr.XDGSurface) {
		server.onDestroyView(&view)
	})
	view.onMapListener = surface.OnMap(func(s wlr.XDGSurface) {
		server.onMapView(&view)
	})
	view.onRequestMoveListener = surface.TopLevel().OnRequestMove(func(t wlr.XDGTopLevel, client wlr.SeatClient, serial uint32) {
		server.startMove(&view)
	})
	view.onRequestResizeListener = surface.TopLevel().OnRequestResize(func(t wlr.XDGTopLevel, client wlr.SeatClient, serial uint32, edges wlr.Edges) {
		if !server.isViewTiled(&view) {
			server.startBorderResize(&view, edges)
		}
	})
	view.onRequestMinimizeListener = surface.TopLevel().OnRequestMinimize(func(t wlr.XDGTopLevel) {
		server.hideView(&view)
	})
	view.onRequestMaximizeListener = surface.TopLevel().OnRequestMaximize(func(t wlr.XDGTopLevel) {
		server.toggleViewTiling(&view)
	})
	view.onSetTitleListener = surface.TopLevel().OnSetTitle(func(t wlr.XDGTopLevel, title string) {
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

	if n, ok := util.Last(server.tiled, server.views); ok {
		server.focusView(n, n.Surface())
	}
}

func (server *Server) onMapView(view *View) {
	pid := view.PID()

	nv, ok := server.newViews[pid]
	if ok {
		delete(server.newViews, pid)

		server.resizeViewTo(nil, view, *nv.To)

		if nv.OnStarted != nil {
			nv.OnStarted(view)
		}

		return
	}

	out := server.outputAt(server.cursorCoords())
	if out == nil {
		if len(server.outputs) == 0 {
			return
		}
		out = server.outputs[0]
	}

	if view.Coords == geom.Pt[float64](-1, -1) {
		server.centerViewOnOutput(out, view)
		return
	}

	server.moveViewTo(out, view, view.Coords)
}

func (server *Server) addView(view *View) {
	server.views = append(server.views, view)
	server.updateCSDs()

	nv, ok := server.newViews[view.PID()]
	if ok {
		view.Resize(int(nv.To.Dx()), int(nv.To.Dy()))
	}
}

func (server *Server) centerViewOnOutput(out *Output, view *View) {
	ob := server.outputBounds(out)
	vb := geom.RConv[float64](server.viewBounds(out, view))
	p := vb.Align(ob.Center())

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

	vb := server.viewBounds(out, view)
	sb := server.surfaceBounds(view.Surface(), geom.PConv[int](view.Coords))
	off := sb.Min.Sub(vb.Min)
	r = r.Add(geom.PConv[float64](off))

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

	prev := server.seat.KeyboardState().FocusedSurface()
	if prev == s {
		return
	}
	pv := server.viewForSurface(prev)
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
}

func (server *Server) viewForSurface(s wlr.Surface) *View {
	for _, view := range server.views {
		if view.Mapped() && view.HasSurface(s) {
			return view
		}
	}
	for _, view := range server.tiled {
		if view.Mapped() && view.HasSurface(s) {
			return view
		}
	}
	for _, view := range server.hidden {
		if view.Mapped() && view.HasSurface(s) {
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
	i := slices.Index(server.views, view)
	if i >= 0 {
		server.views = slices.Delete(server.views, i, i+1)
	}
	i = slices.Index(server.tiled, view)
	if i >= 0 {
		server.tiled = slices.Delete(server.tiled, i, i+1)
		server.layoutTiles(nil)
	}

	// TODO: Remember whether or not a hidden view was tiled.
	server.hidden = append(server.hidden, view)
	view.SetMinimized(true)

	item := NewMenuItem(
		CreateTextTexture(server.renderer, image.White, view.Title()),
		CreateTextTexture(server.renderer, image.Black, view.Title()),
	)
	item.OnSelect = func() {
		server.unhideView(view)
	}
	server.mainMenu.Add(item)
}

func (server *Server) unhideView(view *View) {
	i := slices.Index(server.hidden, view)
	server.hidden = slices.Delete(server.hidden, i, i+1)
	server.mainMenu.Remove(len(mainMenuText) + i)

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
		view.Restore = geom.RConv[float64](server.viewBounds(nil, view))
	}
	server.layoutTiles(nil)
	server.focusView(view, view.Surface())
	view.SetMaximized(true)
}

func (server *Server) untileView(view *View, restore bool) {
	i := slices.Index(server.tiled, view)
	server.tiled = slices.Delete(server.tiled, i, i+1)
	server.views = append(server.views, view)

	if restore {
		server.resizeViewTo(nil, view, view.Restore)
	}
	server.layoutTiles(nil)
	server.focusView(view, view.Surface())
	view.SetMaximized(false)
}

func (server *Server) layoutTiles(out *Output) {
	if len(server.tiled) == 0 {
		return
	}

	if out == nil {
		out = server.outputs[0]
	}

	or := server.outputBounds(out)
	tiles := tile.RightThenDown(or, len(server.tiled))
	for i, tile := range tiles {
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
	deco := Decoration{
		Decoration: d,
	}
	deco.onDestroyListener = d.OnDestroy(func(d wlr.ServerDecoration) {
		server.onDestroyDecoration(&deco)
	})
	deco.onModeListener = d.OnMode(func(d wlr.ServerDecoration) {
		server.updateCSDs()
	})

	server.decorations = append(server.decorations, &deco)
	server.updateCSDs()
}

func (server *Server) onDestroyDecoration(d *Decoration) {
	i := slices.Index(server.decorations, d)
	server.decorations = slices.Delete(server.decorations, i, i+1)
	server.updateCSDs()
}

func (server *Server) updateCSDs() {
	for _, view := range server.views {
		view.CSD = server.isCSDSurface(view.Surface())
	}
	for _, view := range server.tiled {
		view.CSD = server.isCSDSurface(view.Surface())
	}
}

func (server *Server) isCSDSurface(surface wlr.Surface) (ok bool) {
	for _, d := range server.decorations {
		d.Decoration.Surface().ForEachSurface(func(s wlr.Surface, sx, sy int) {
			if s == surface {
				ok = true
			}
		})
		if ok {
			return d.Decoration.Mode() == wlr.ServerDecorationManagerModeClient
		}
	}
	return false
}

func (server *Server) updateTitles() {
	// Not the best way to do this, perhaps...
	//for _, view := range server.hidden {
	//	server.mainMenu.Remove(len(mainMenuText))
	//	server.mainMenu.Add(server, view.Title())
	//}
}
