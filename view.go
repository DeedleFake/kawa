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

	panic("Not implemented.")
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
