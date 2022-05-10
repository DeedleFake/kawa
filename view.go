package main

import (
	"image"

	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
)

func (server *Server) viewBounds(out *Output, view *View) image.Rectangle {
	return server.surfaceBounds(out, view.XDGSurface.Surface(), view.X, view.Y)
}

func (server *Server) surfaceBounds(out *Output, surface wlr.Surface, x, y int) image.Rectangle {
	ox, oy := server.outputLayout.OutputCoords(out.Output)
	scale := out.Output.Scale()
	current := surface.Current()

	return box(
		int((ox+float64(x))*float64(scale)),
		int((oy+float64(y))*float64(scale)),
		int(float64(current.Width())*float64(scale)),
		int(float64(current.Height())*float64(scale)),
	)
}

func (server *Server) viewAt(out *Output, x, y float64) (*View, ViewArea) {
	if out == nil {
		out = server.outputAt(x, y)
	}

	p := image.Pt(int(x), int(y))
	for _, view := range server.views {
		r := server.viewBounds(out, view)
		if p.In(r) {
			return view, ViewAreaSurface
		}

		left := image.Rect(r.Min.X-WindowBorder, r.Min.Y, r.Max.X, r.Max.Y)
		top := image.Rect(r.Min.X, r.Min.Y-WindowBorder, r.Max.X, r.Max.Y)
		right := image.Rect(r.Min.X, r.Min.Y, r.Max.X+WindowBorder, r.Max.Y)
		bottom := image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y+WindowBorder)

		if p.In(top.Union(left)) {
			return view, ViewAreaBorderTopLeft
		}
		if p.In(top.Union(right)) {
			return view, ViewAreaBorderTopRight
		}
		if p.In(bottom.Union(left)) {
			return view, ViewAreaBorderBottomLeft
		}
		if p.In(bottom.Union(right)) {
			return view, ViewAreaBorderBottomRight
		}

		if p.In(left) {
			return view, ViewAreaBorderLeft
		}
		if p.In(top) {
			return view, ViewAreaBorderTop
		}
		if p.In(right) {
			return view, ViewAreaBorderRight
		}
		if p.In(bottom) {
			return view, ViewAreaBorderBottom
		}
	}

	return nil, ViewAreaNone
}

func (server *Server) onNewXDGSurface(surface wlr.XDGSurface) {
	if surface.Role() != wlr.XDGSurfaceRoleTopLevel {
		return
	}

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

	server.addView(&view)
}

func (server *Server) onDestroyView(view *View) {
	view.Release()

	i := slices.IndexFunc(server.views, func(v *View) bool {
		return v.XDGSurface == view.XDGSurface
	})
	server.views = slices.Delete(server.views, i, i)
}

func (server *Server) onMapView(view *View) {
	out := server.outputAt(server.cursor.X(), server.cursor.Y())
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

var areaCursors = [...]string{
	"ptr_left",
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
	ViewAreaBorderTopLeft
	ViewAreaBorderTop
	ViewAreaBorderTopRight
	ViewAreaBorderLeft
	ViewAreaSurface
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
