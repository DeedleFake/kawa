package main

import (
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

		view.X = newView.Box.X
		view.Y = newView.Box.Y
		surface.TopLevelSetSize(
			uint32(newView.Box.Width),
			uint32(newView.Box.Height),
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
	panic("Not implemented.")
}
