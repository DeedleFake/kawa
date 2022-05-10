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

func (server *Server) viewAt(out *Output, x, y float64) *View {
	if out == nil {
		out = server.outputAt(x, y)
	}

	p := image.Pt(int(x), int(y))
	for _, view := range server.views {
		r := server.viewBounds(out, view)
		if p.In(r) {
			return view
		}
	}

	return nil
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
