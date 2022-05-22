package main

import (
	"fmt"
	"image"

	"deedles.dev/kawa/geom"
	"deedles.dev/kawa/geom/tile"
	"deedles.dev/kawa/internal/util"
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

func (server *Server) viewBounds(out *Output, view *View) geom.Rect[float64] {
	var r geom.Rect[int]
	view.ForEachSurface((s, sx, sy) => {
		if server.isPopupSurface(s) {
			return
		}

		sb := server.surfaceBounds(s, geom.Pt(sx, sy))
		r = r.Union(sb)
	})
	return geom.RConv[float64](r).Add(view.Coords)
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

	r := server.viewBounds(nil, view)
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
	}
	view.onDestroyListener = surface.OnDestroy((s) => { server.onDestroyView(&view) })
	view.onMapListener = surface.OnMap((s) => { server.onMapView(&view) })
	view.onRequestMoveListener = surface.OnRequestMove((s) => { server.startMove(&view) })
	view.onRequestResizeListener = surface.OnRequestResize((s, edges) => {
		if !server.isViewTiled(&view) {
			server.startBorderResize(&view, edges)
		}
	})
	view.onRequestMinimizeListener = surface.OnRequestMinimize((s) => { server.hideView(&view) })
	view.onRequestMaximizeListener = surface.OnRequestMaximize((s) => { server.toggleViewTiling(&view) })
	view.onSetTitleListener = surface.OnSetTitle((s, title) => { server.updateTitles() })

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
	p.onDestroyListener = surface.OnDestroy((s) => { server.onDestroyPopup(&p) })

	server.popups = append(server.popups, &p)
}

func (server *Server) isPopupSurface(surface wlr.Surface) (ok bool) {
	for _, p := range server.popups {
		p.Surface.ForEachSurface((s, sx, sy) => {
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
	}
	view.onDestroyListener = surface.OnDestroy((s) => { server.onDestroyView(&view) })
	view.onMapListener = surface.OnMap((s) => { server.onMapView(&view) })
	view.onRequestMoveListener = surface.TopLevel().OnRequestMove((t, client, serial) => { server.startMove(&view) })
	view.onRequestResizeListener = surface.TopLevel().OnRequestResize((t, client, serial, edges) => {
		if !server.isViewTiled(&view) {
			server.startBorderResize(&view, edges)
		}
	})
	view.onRequestMinimizeListener = surface.TopLevel().OnRequestMinimize((t) => { server.hideView(&view) })
	view.onRequestMaximizeListener = surface.TopLevel().OnRequestMaximize((t) => { server.toggleViewTiling(&view) })
	view.onSetTitleListener = surface.TopLevel().OnSetTitle((t, title) => { server.updateTitles() })

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

	server.centerViewOnOutput(out, view)
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
	vb := server.viewBounds(out, view)
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
	off := view.Coords.Sub(vb.Min)
	r = r.Add(off)

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
	item.OnSelect = () => { server.unhideView(view) }
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
		view.Restore = server.viewBounds(nil, view)
	}
	server.layoutTiles(nil)
	server.focusView(view, view.Surface())
	view.SetMaximized(true)
}

func (server *Server) untileView(view *View, restore bool) {
	i := slices.Index(server.tiled, view)
	server.tiled = slices.Delete(server.tiled, i, i+1)
	server.views = append(server.views, view)

	if restore && !view.Restore.IsZero() {
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
	tiles := tile.TwoThirdsSidebar(or, len(server.tiled))
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
	deco.onDestroyListener = d.OnDestroy((d) => { server.onDestroyDecoration(&deco) })
	deco.onModeListener = d.OnMode((d) => { server.updateCSDs() })

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
		d.Decoration.Surface().ForEachSurface((s, sx, sy) => {
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
	for _, view := range server.hidden {
		item := server.mainMenu.Item(len(mainMenuText))
		item.Release()

		n := NewTextMenuItem(server.renderer, view.Title())
		n.OnSelect = item.OnSelect

		server.mainMenu.Remove(item)
		server.mainMenu.Add(n)
	}

	if server.focusedTitle.Valid() {
		server.focusedTitle.Destroy()
		server.focusedTitle = wlr.Texture{}
	}
	if fv := server.focusedView(); fv != nil {
		server.focusedTitle = CreateTextTexture(server.renderer, image.White, fv.Title())
	}
}
