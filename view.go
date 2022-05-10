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
		server.onDestroyView(view)
	})
	view.Map = surface.OnMap(func(s wlr.XDGSurface) {
		server.onMapView(view)
	})

	server.addView(&view)

	client := surface.Resource().GetClient()
	pid, _, _ := client.GetCredentials()

	for i, newView := range server.newViews {
		if newView.PID != pid {
			continue
		}

		view.X = newView.Box.Min.X
		view.Y = newView.Box.Min.Y
		surface.TopLevelSetSize(
			uint32(newView.Box.Dx()),
			uint32(newView.Box.Dy()),
		)

		slices.Delete(server.newViews, i, i)
		break
	}

	server.views = append(server.views, &view)
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

func (server *Server) centerViewOnOutput(out *Output, view *View) {
	layout := server.outputLayout.Get(out.Output)
	current := view.XDGSurface.Surface().Current()
	ow, oh := out.Output.EffectiveResolution()

	server.moveViewTo(
		out,
		view,
		layout.X()+(ow/2-current.Width()/2),
		layout.Y()+(oh/2-current.Height/2),
	)
}

func (server *Server) moveViewTo(out *Output, view *View, x, y int) {
	view.X = x
	view.Y = y

	if out == nil {
		out = server.outputAt(x, y)
	}
	view.XDGSurface.Surface().SendEnter(out.Output)
}

//func (view *View) focus(surface wlr.Surface) {
//	server := view.Server
//	prevSurface := server.seat.KeyboardState().FocusedSurface()
//	if prevSurface == surface {
//		return
//	}
//	if prevSurface.Valid() {
//		prev := wlr.XDGSurfaceFromWLRSurface(prevSurface)
//		prev.TopLevelSetActivated(false)
//	}
//
//	keyboard := server.seat.GetKeyboard()
//	view.XDGSurface.TopLevelSetActivated(true)
//	server.seat.KeyboardNotifyEnter(view.XDGSurface.Surface(), keyboard.Keycodes(), keyboard.Modifiers())
//
//	i := slices.Index(server.views, view)
//	server.views = slices.Delete(server.views, i, i)
//	server.views = append(server.views, view)
//}
//
//func (server *Server) viewAt(lx, ly float64) (view *View, surface wlr.Surface, sx, sy float64, ok bool) {
//	for _, view := range server.views {
//		surface, sx, sy, ok := view.XDGSurface.SurfaceAt(lx-float64(view.X), ly-float64(view.Y))
//		if ok {
//			view.Area = ViewAreaSurface
//			return view, surface, sx, sy, true
//		}
//
//		current := view.XDGSurface.Surface().Current()
//		border := box(
//			view.X-WindowBorder,
//			view.Y-WindowBorder,
//			current.Width()+WindowBorder*2,
//			current.Width()+WindowBorder*2,
//		)
//		if image.Pt(int(lx), int(ly)).In(border) {
//			view.Area = whichCorner(border, lx, ly)
//			return view, wlr.Surface{}, lx - float64(view.X), ly - float64(view.Y), true
//		}
//	}
//	return nil, wlr.Surface{}, 0, 0, false
//}
//
//func whichCorner(r image.Rectangle, lx, ly float64) ViewArea {
//	portion := func(x, lo, width int) ViewArea {
//		x -= lo
//		if x < 20 {
//			return 0
//		}
//		if x > width-20 {
//			return 2
//		}
//		return 1
//	}
//
//	i := portion(int(lx), r.Min.X, r.Dx())
//	j := portion(int(ly), r.Min.Y, r.Dy())
//	return 3*j + i
//}
