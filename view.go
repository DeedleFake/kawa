package main

import (
	"fmt"
	"image"

	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
)

type View struct {
	X, Y        int
	XDGSurface  wlr.XDGSurface
	Map         wlr.Listener
	Destroy     wlr.Listener
	RequestMove wlr.Listener
}

func (view *View) Release() {
	view.Destroy.Destroy()
	view.Map.Destroy()
}

func (server *Server) viewBounds(out *Output, view *View) image.Rectangle {
	var r image.Rectangle
	view.XDGSurface.ForEachSurface(func(s wlr.Surface, sx, sy int) {
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
	m, ok := server.inputMode.(interface{ TargetView() *View })
	if !ok {
		return nil
	}

	return m.TargetView()
}

func (server *Server) viewAt(out *Output, x, y float64) (*View, ViewArea, wlr.Surface, float64, float64) {
	if out == nil {
		out = server.outputAt(x, y)
	}

	p := image.Pt(int(x), int(y))
	for _, view := range server.views {
		surface, sx, sy, ok := view.XDGSurface.SurfaceAt(x-float64(view.X), y-float64(view.Y))
		if ok {
			return view, ViewAreaSurface, surface, sx, sy
		}

		r := server.viewBounds(nil, view)
		if !p.In(r.Inset(-WindowBorder)) {
			continue
		}

		left := image.Rect(r.Min.X-WindowBorder, r.Min.Y, r.Max.X, r.Max.Y)
		if p.In(left) {
			return view, ViewAreaBorderLeft, wlr.Surface{}, 0, 0
		}

		top := image.Rect(r.Min.X, r.Min.Y-WindowBorder, r.Max.X, r.Max.Y)
		if p.In(top) {
			return view, ViewAreaBorderTop, wlr.Surface{}, 0, 0
		}

		right := image.Rect(r.Min.X, r.Min.Y, r.Max.X+WindowBorder, r.Max.Y)
		if p.In(right) {
			return view, ViewAreaBorderRight, wlr.Surface{}, 0, 0
		}

		bottom := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y+WindowBorder)
		if p.In(bottom) {
			return view, ViewAreaBorderBottom, wlr.Surface{}, 0, 0
		}

		if (p.X < r.Min.X) && (p.Y < r.Min.Y) {
			return view, ViewAreaBorderTopLeft, wlr.Surface{}, 0, 0
		}
		if (p.X > r.Max.X) && (p.Y < r.Min.Y) {
			return view, ViewAreaBorderTopRight, wlr.Surface{}, 0, 0
		}
		if (p.X < r.Min.X) && (p.Y > r.Min.Y) {
			return view, ViewAreaBorderBottomLeft, wlr.Surface{}, 0, 0
		}
		if (p.X > r.Max.X) && (p.Y > r.Min.Y) {
			return view, ViewAreaBorderBottomRight, wlr.Surface{}, 0, 0
		}

		// Where else could it possibly be if it gets to here?
		panic(fmt.Errorf("If you see this, there's a bug.\np = %+v\nr = %+v", p, r))
	}

	return nil, ViewAreaNone, wlr.Surface{}, 0, 0
}

func (server *Server) onNewXDGSurface(surface wlr.XDGSurface) {
	if surface.Role() != wlr.XDGSurfaceRoleTopLevel {
		server.addXDGPopup(surface)
		return
	}

	server.addXDGTopLevel(surface)
}

func (server *Server) addXDGPopup(surface wlr.XDGSurface) {
	// TODO
}

func (server *Server) addXDGTopLevel(surface wlr.XDGSurface) {
	view := View{
		X:          -1,
		Y:          -1,
		XDGSurface: surface,
	}
	view.Destroy = surface.OnDestroy(func(s wlr.XDGSurface) {
		server.onDestroyView(&view)
	})
	view.Map = surface.OnMap(func(s wlr.XDGSurface) {
		server.onMapView(&view)
	})
	view.RequestMove = surface.TopLevel().OnRequestMove(func(client wlr.SeatClient, serial uint32) {
		server.startMove(&view)
	})

	server.addView(&view)
}

func (server *Server) onDestroyView(view *View) {
	view.Release()

	i := slices.IndexFunc(server.views, func(v *View) bool {
		return v.XDGSurface == view.XDGSurface
	})
	server.views = slices.Delete(server.views, i, i+1)
}

func (server *Server) onMapView(view *View) {
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
	client := view.XDGSurface.Resource().GetClient()
	pid, _, _ := client.GetCredentials()

	nv, ok := server.newViews[pid]
	if ok {
		delete(server.newViews, pid)

		view.X = nv.Min.X
		view.Y = nv.Min.Y
		view.XDGSurface.TopLevelSetSize(
			uint32(nv.Dx()),
			uint32(nv.Dy()),
		)
	}

	server.views = append(server.views, view)
}

func (server *Server) centerViewOnOutput(out *Output, view *View) {
	layout := server.outputLayout.Get(out.Output)
	current := view.XDGSurface.Surface().Current()
	ow, oh := out.Output.EffectiveResolution()

	server.moveViewTo(
		out,
		view,
		layout.X()+(ow/2-current.Width()/2),
		layout.Y()+(oh/2-current.Height()/2),
	)
}

func (server *Server) moveViewTo(out *Output, view *View, x, y int) {
	view.X = x
	view.Y = y

	if out == nil {
		out = server.outputAt(float64(x), float64(y))
	}
	view.XDGSurface.Surface().SendEnter(out.Output)
}

func (server *Server) resizeViewTo(out *Output, view *View, r image.Rectangle) {
	view.X = r.Min.X
	view.Y = r.Min.Y
	view.XDGSurface.TopLevelSetSize(uint32(r.Dx()), uint32(r.Dy()))

	if out == nil {
		out = server.outputAt(float64(view.X), float64(view.Y))
	}
	view.XDGSurface.Surface().SendEnter(out.Output)
}

func (server *Server) focusView(view *View, s wlr.Surface) {
	if !s.Valid() {
		s = view.XDGSurface.Surface()
	}

	prev := server.seat.KeyboardState().FocusedSurface()
	if prev == s {
		return
	}
	if prev.Valid() && (prev.Type() == wlr.SurfaceTypeXDG) {
		xdg := wlr.XDGSurfaceFromWLRSurface(prev)
		xdg.TopLevelSetActivated(false)
	}

	k := server.seat.GetKeyboard()
	server.seat.KeyboardNotifyEnter(s, k.Keycodes(), k.Modifiers())

	view.XDGSurface.TopLevelSetActivated(true)
	server.bringViewToFront(view)
}

func (server *Server) bringViewToFront(view *View) {
	i := slices.Index(server.views, view)
	server.views = slices.Delete(server.views, i, i+1)
	server.views = append(server.views, view)
}

var areaCursors = [...]string{
	"left_ptr",
	"",
	"top_left_corner",
	"top_side",
	"top_right_corner",
	"left_side",
	"right_side",
	"bottom_left_corner",
	"bottom_side",
	"bottom_right_corner",
}

type ViewArea int

const (
	ViewAreaNone ViewArea = iota
	ViewAreaSurface
	ViewAreaBorderTopLeft
	ViewAreaBorderTop
	ViewAreaBorderTopRight
	ViewAreaBorderLeft
	ViewAreaBorderRight
	ViewAreaBorderBottomLeft
	ViewAreaBorderBottom
	ViewAreaBorderBottomRight
)

func (area ViewArea) Cursor() string {
	if (area < 0) || (int(area) >= len(areaCursors)) {
		return ""
	}
	return areaCursors[area]
}

func (area ViewArea) Edges() (e wlr.Edges) {
	switch area {
	case ViewAreaBorderTopLeft, ViewAreaBorderTop, ViewAreaBorderTopRight:
		e |= wlr.EdgeTop
	case ViewAreaBorderBottomLeft, ViewAreaBorderBottom, ViewAreaBorderBottomRight:
		e |= wlr.EdgeBottom
	}

	switch area {
	case ViewAreaBorderTopLeft, ViewAreaBorderLeft, ViewAreaBorderBottomLeft:
		e |= wlr.EdgeLeft
	case ViewAreaBorderTopRight, ViewAreaBorderRight, ViewAreaBorderBottomRight:
		e |= wlr.EdgeRight
	}

	return e
}
