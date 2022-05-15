package main

import (
	"fmt"
	"image"

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
	X, Y    int
	Restore image.Rectangle
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
	To        *image.Rectangle
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

func (server *Server) viewBounds(out *Output, view *View) image.Rectangle {
	var r image.Rectangle
	view.ForEachSurface(func(s wlr.Surface, sx, sy int) {
		if server.isPopupSurface(s) {
			return
		}
		r = r.Union(server.surfaceBounds(out, s, view.X+sx, view.Y+sy))
	})
	return r
}

func (server *Server) surfaceBounds(out *Output, surface wlr.Surface, x, y int) image.Rectangle {
	var ox, oy float64
	scale := float32(1)
	if out != nil {
		ox, oy = server.outputLayout.OutputCoords(out.Output)
		scale = out.Output.Scale()
	}

	current := surface.Current()
	return box(
		int((ox+float64(x))*float64(scale)),
		int((oy+float64(y))*float64(scale)),
		int(float64(current.Width())*float64(scale)),
		int(float64(current.Height())*float64(scale)),
	)
}

func (server *Server) targetView() *View {
	m, ok := server.inputMode.(ViewTargeter)
	if !ok {
		return nil
	}

	return m.TargetView()
}

func (server *Server) viewAt(out *Output, x, y float64) (*View, wlr.Edges, wlr.Surface, float64, float64) {
	if out == nil {
		out = server.outputAt(x, y)
	}

	i, edges, surface, sx, sy := server.viewIndexAt(out, server.views, x, y)
	if i >= 0 {
		return server.views[i], edges, surface, sx, sy
	}

	i, edges, surface, sx, sy = server.viewIndexAt(out, server.tiled, x, y)
	if i >= 0 {
		return server.tiled[i], edges, surface, sx, sy
	}

	return nil, wlr.EdgeNone, wlr.Surface{}, 0, 0
}

func (server *Server) viewIndexAt(out *Output, views []*View, x, y float64) (int, wlr.Edges, wlr.Surface, float64, float64) {
	for i := len(views) - 1; i >= 0; i-- {
		view := views[i]
		if !view.Mapped() {
			continue
		}

		edges, surface, sx, sy, ok := server.isViewAt(out, view, x, y)
		if ok {
			return i, edges, surface, sx, sy
		}
	}

	return -1, 0, wlr.Surface{}, 0, 0
}

func (server *Server) isViewAt(out *Output, view *View, x, y float64) (edges wlr.Edges, s wlr.Surface, sx, sy float64, ok bool) {
	surface, sx, sy, ok := view.SurfaceAt(x-float64(view.X), y-float64(view.Y))
	if ok {
		return wlr.EdgeNone, surface, sx, sy, true
	}

	// Don't bother checking the borders if there aren't any.
	if view.CSD {
		return 0, wlr.Surface{}, 0, 0, false
	}

	p := image.Pt(int(x), int(y))
	r := server.viewBounds(nil, view)
	if !p.In(r.Inset(-WindowBorder)) {
		return 0, wlr.Surface{}, 0, 0, false
	}

	left := image.Rect(r.Min.X-WindowBorder, r.Min.Y, r.Max.X, r.Max.Y)
	if p.In(left) {
		return wlr.EdgeLeft, wlr.Surface{}, 0, 0, true
	}

	top := image.Rect(r.Min.X, r.Min.Y-WindowBorder, r.Max.X, r.Max.Y)
	if p.In(top) {
		return wlr.EdgeTop, wlr.Surface{}, 0, 0, true
	}

	right := image.Rect(r.Min.X, r.Min.Y, r.Max.X+WindowBorder, r.Max.Y)
	if p.In(right) {
		return wlr.EdgeRight, wlr.Surface{}, 0, 0, true
	}

	bottom := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y+WindowBorder)
	if p.In(bottom) {
		return wlr.EdgeBottom, wlr.Surface{}, 0, 0, true
	}

	if (p.X < r.Min.X) && (p.Y < r.Min.Y) {
		return wlr.EdgeTop | wlr.EdgeLeft, wlr.Surface{}, 0, 0, true
	}
	if (p.X >= r.Max.X) && (p.Y < r.Min.Y) {
		return wlr.EdgeTop | wlr.EdgeRight, wlr.Surface{}, 0, 0, true
	}
	if (p.X < r.Min.X) && (p.Y >= r.Max.Y) {
		return wlr.EdgeBottom | wlr.EdgeLeft, wlr.Surface{}, 0, 0, true
	}
	if (p.X >= r.Max.X) && (p.Y >= r.Max.Y) {
		return wlr.EdgeBottom | wlr.EdgeRight, wlr.Surface{}, 0, 0, true
	}

	// Where else could it possibly be if it gets to here?
	panic(fmt.Errorf("If you see this, there's a bug.\np = %+v\nr = %+v", p, r))
}

func (server *Server) onNewXWaylandSurface(surface wlr.XWaylandSurface) {
	view := View{
		ViewSurface: &viewSurfaceXWayland{s: surface},
		X:           -1,
		Y:           -1,
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
		X:           -1,
		Y:           -1,
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

	out := server.outputAt(server.cursor.X(), server.cursor.Y())
	if out == nil {
		if len(server.outputs) == 0 {
			return
		}
		out = server.outputs[0]
	}

	if (view.X == -1) || (view.Y == -1) {
		server.centerViewOnOutput(out, view)
		return
	}

	server.moveViewTo(out, view, view.X, view.Y)
}

func (server *Server) addView(view *View) {
	server.views = append(server.views, view)
	server.updateCSDs()

	nv, ok := server.newViews[view.PID()]
	if ok {
		view.Resize(nv.To.Dx(), nv.To.Dy())
	}
}

func (server *Server) centerViewOnOutput(out *Output, view *View) {
	layout := server.outputLayout.Get(out.Output)
	current := view.Surface().Current()
	ow, oh := out.Output.EffectiveResolution()

	server.moveViewTo(
		out,
		view,
		layout.X()+(ow/2-current.Width()/2),
		layout.Y()+(oh/2-current.Height()/2),
	)
}

func (server *Server) moveViewTo(out *Output, view *View, x, y int) {
	if out == nil {
		out = server.outputAt(float64(x), float64(y))
	}

	view.X = x
	view.Y = y

	if out != nil {
		view.Surface().SendEnter(out.Output)
	}
}

func (server *Server) resizeViewTo(out *Output, view *View, r image.Rectangle) {
	if out == nil {
		out = server.outputAt(float64(r.Min.X), float64(r.Min.Y))
	}

	vb := server.viewBounds(out, view)
	sb := server.surfaceBounds(out, view.Surface(), view.X, view.Y)
	off := sb.Min.Sub(vb.Min)
	r = r.Add(off)

	view.X = r.Min.X
	view.Y = r.Min.Y
	view.Resize(r.Dx(), r.Dy())

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
	server.mainMenu.Add(server, view.Title())
	view.SetMinimized(true)
}

func (server *Server) unhideView(view *View) {
	i := slices.Index(server.hidden, view)
	server.hidden = slices.Delete(server.hidden, i, i+1)
	server.mainMenu.Remove(len(mainMenuItems) + i)

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

	view.Restore = box(10, 10, 640, 480)
	if s := view.Surface(); s.Valid() {
		view.Restore = box(view.X, view.Y, s.Current().Width(), s.Current().Height())
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

	x, y := server.outputLayout.OutputCoords(out.Output)
	or := box(int(x), int(y), out.Output.Width(), out.Output.Height())

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
	for _, view := range server.hidden {
		server.mainMenu.Remove(len(mainMenuItems))
		server.mainMenu.Add(server, view.Title())
	}
}
