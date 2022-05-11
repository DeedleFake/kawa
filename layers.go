package main

import (
	"image"

	"deedles.dev/wlr"
)

type LayerSurface struct {
	LayerSurface wlr.LayerSurfaceV1

	Destroy       wlr.Listener
	Map           wlr.Listener
	SurfaceCommit wlr.Listener
	OutputDestroy wlr.Listener

	Geo image.Rectangle
}

func (server *Server) onNewLayerSurface(surface wlr.LayerSurfaceV1) {
	panic("Not implemented.")
}
