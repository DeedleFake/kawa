package main

import (
	"image"

	"deedles.dev/wlr"
)

type LayerSurface struct {
	LayerSurface wlr.LayerSurfaceV1
	Geo          image.Rectangle

	onDestroyListener       wlr.Listener
	onMapListener           wlr.Listener
	onSurfaceCommitListener wlr.Listener
	onOutputDestroyListener wlr.Listener
}

func (server *Server) onNewLayerSurface(surface wlr.LayerSurfaceV1) {
	panic("Not implemented.")
}
