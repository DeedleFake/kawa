package main

import (
	"image"

	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
)

func (server *Server) onNewXDGSurface(surface wlr.XDGSurface) {
	if surface.Role() != wlr.XDGSurfaceRoleTopLevel {
		return
	}

	view := View{
		X:          -1,
		Y:          -1,
		XDGSurface: surface,
		Server:     server,
	}
	view.Destroy = surface.OnDestroy(view.onDestroy)
	view.Map = surface.OnMap(view.onMap)

	surface.TopLevelSetTiled(wlr.EdgeLeft | wlr.EdgeRight | wlr.EdgeTop | wlr.EdgeBottom)

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

func (view *View) onDestroy(surface wlr.XDGSurface) {
	view.Release()

	server := view.Server
	i := slices.IndexFunc(server.views, func(v *View) bool {
		return v.XDGSurface == surface
	})
	server.views = slices.Delete(server.views, i, i)
}

func (view *View) onMap(surface wlr.XDGSurface) {
	server := view.Server
	view.Focus(surface.Surface())

	output := server.outputLayout.OutputAt(server.cursor.X(), server.cursor.Y())
	layout := server.outputLayout.Get(output)
	if (view.X != -1) || (view.Y != -1) {
		view.Move(view.X, view.Y)
		return
	}

	current := view.XDGSurface.Surface().Current()
	ow, oh := output.EffectiveResolution()
	view.Move(
		layout.X()+(ow/2-current.Width()/2),
		layout.Y()+(oh/2-current.Height()/2),
	)
}

func (view *View) Focus(surface wlr.Surface) {
	server := view.Server
	prevSurface := server.seat.KeyboardState().FocusedSurface()
	if prevSurface == surface {
		return
	}
	if prevSurface.Valid() {
		prev := wlr.XDGSurfaceFromWLRSurface(prevSurface)
		prev.TopLevelSetActivated(false)
	}

	keyboard := server.seat.GetKeyboard()
	view.XDGSurface.TopLevelSetActivated(true)
	server.seat.KeyboardNotifyEnter(view.XDGSurface.Surface(), keyboard.Keycodes(), keyboard.Modifiers())

	i := slices.Index(server.views, view)
	server.views = slices.Delete(server.views, i, i)
	server.views = append(server.views, view)
}

func (view *View) Move(x, y int) {
	view.X = x
	view.Y = y

	// TODO: Do this properly. The view isn't entering every single
	// output.
	for _, out := range view.Server.outputs {
		view.XDGSurface.Surface().SendEnter(out.Output)
	}
}

func (server *Server) viewAt(lx, ly float64) (view *View, surface wlr.Surface, sx, sy float64, ok bool) {
	for _, view := range server.views {
		surface, sx, sy, ok := view.XDGSurface.SurfaceAt(lx, ly)
		if ok {
			view.Area = ViewAreaSurface
			return view, surface, sx, sy, true
		}

		current := view.XDGSurface.Surface().Current()
		border := box(
			view.X-WindowBorder,
			view.Y-WindowBorder,
			current.Width()+WindowBorder*2,
			current.Width()+WindowBorder*2,
		)
		if image.Pt(int(lx), int(ly)).In(border) {
			view.Area = whichCorner(border, lx, ly)
			return view, wlr.Surface{}, lx - float64(view.X), ly - float64(view.Y), true
		}
	}
	return nil, wlr.Surface{}, 0, 0, false
}

func whichCorner(r image.Rectangle, lx, ly float64) ViewArea {
	portion := func(x, lo, width int) ViewArea {
		x -= lo
		if x < 20 {
			return 0
		}
		if x > width-20 {
			return 2
		}
		return 1
	}

	i := portion(int(lx), r.Min.X, r.Dx())
	j := portion(int(ly), r.Min.Y, r.Dy())
	return 3*j + i
}
