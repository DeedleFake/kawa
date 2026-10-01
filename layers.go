package main

import (
	"slices"

	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

type LayerSurface struct {
	LayerSurface wlr.LayerSurfaceV1
	Output       *Output
	// Layer is the layer whose list in Output.Layers holds the
	// surface. A client can move the surface to another layer, which
	// takes effect when the commit is handled.
	Layer wlr.LayerShellV1Layer
	// Geo is where the surface was last arranged, relative to its
	// output.
	Geo geom.Rect[int]

	onDestroyListener wlr.Listener
	onMapListener     wlr.Listener
	onUnmapListener   wlr.Listener
	onCommitListener  wlr.Listener
}

func (ls *LayerSurface) Release() {
	ls.onDestroyListener.Destroy()
	ls.onMapListener.Destroy()
	ls.onUnmapListener.Destroy()
	ls.onCommitListener.Destroy()
}

func (ls *LayerSurface) Mapped() bool {
	return ls.LayerSurface.Surface().Mapped()
}

func (server *Server) onNewLayerSurface(surface wlr.LayerSurfaceV1) {
	out := server.outputFor(surface.Output())
	if out == nil {
		out = server.outputAt(server.cursorCoords())
	}
	if (out == nil) && (len(server.outputs) > 0) {
		out = server.outputs[0]
	}
	if out == nil {
		surface.Destroy()
		return
	}
	surface.SetOutput(out.Output)

	ls := LayerSurface{
		LayerSurface: surface,
		Output:       out,
		Layer:        surface.Pending().Layer(),
	}
	ls.onDestroyListener = surface.OnDestroy(func(wlr.LayerSurfaceV1) {
		server.onDestroyLayerSurface(&ls)
	})
	ls.onMapListener = surface.Surface().OnMap(func(wlr.Surface) {
		server.arrangeLayers(ls.Output)
	})
	ls.onUnmapListener = surface.Surface().OnUnmap(func(wlr.Surface) {
		server.arrangeLayers(ls.Output)
	})
	ls.onCommitListener = surface.Surface().OnCommit(func(wlr.Surface) {
		server.onLayerSurfaceCommit(&ls)
	})

	out.Layers[ls.Layer] = append(out.Layers[ls.Layer], &ls)
	wlr.Log(wlr.Debug, "new layer surface %q on %v in layer %v", surface.Namespace(), out.Output.Name(), ls.Layer)
}

func (server *Server) onLayerSurfaceCommit(ls *LayerSurface) {
	s := ls.LayerSurface
	if !s.Initialized() {
		return
	}

	current := s.Current()
	if layer := current.Layer(); layer != ls.Layer {
		ls.Output.removeLayerSurface(ls)
		ls.Layer = layer
		ls.Output.Layers[layer] = append(ls.Output.Layers[layer], ls)
	}

	// The client won't map the surface until it gets a configure in
	// response to its initial commit.
	if s.InitialCommit() || (current.Committed() != 0) {
		server.arrangeLayers(ls.Output)
	}
}

func (server *Server) onDestroyLayerSurface(ls *LayerSurface) {
	ls.Release()
	ls.Output.removeLayerSurface(ls)
	server.arrangeLayers(ls.Output)
}

func (out *Output) removeLayerSurface(ls *LayerSurface) {
	layer := out.Layers[ls.Layer]
	if i := slices.Index(layer, ls); i >= 0 {
		out.Layers[ls.Layer] = slices.Delete(layer, i, i+1)
	}
}

// closeLayerSurfaces closes every layer surface on out. They're
// destroyed right away, which removes them from out.
func (out *Output) closeLayerSurfaces() {
	for _, layer := range out.Layers {
		for _, ls := range slices.Clone(layer) {
			ls.LayerSurface.Destroy()
		}
	}
}

// arrangeLayers places the layer surfaces on out and works out what is
// left for windows, starting from the part of the output that the
// status bar doesn't cover. Surfaces with an exclusive zone are placed
// first, from the top layer down, and each takes its zone out of the
// area that the surfaces after it get. This is the same order that
// wlroots' scene helper and sway use.
func (server *Server) arrangeLayers(out *Output) {
	origin := server.outputBounds(out).Min
	full := geom.RConv[int](server.outputBounds(out).Sub(origin))
	usable := geom.RConv[int](server.outputVisibleBounds(out).Sub(origin))

	for _, exclusive := range []bool{true, false} {
		for layer := wlr.LayerShellV1LayerOverlay; layer >= wlr.LayerShellV1LayerBackground; layer-- {
			for _, ls := range out.Layers[layer] {
				s := ls.LayerSurface
				if !s.Initialized() || ((s.Current().ExclusiveZone() > 0) != exclusive) {
					continue
				}

				ls.Geo = placeLayerSurface(s.Current(), full, usable)
				s.Configure(uint32(ls.Geo.Dx()), uint32(ls.Geo.Dy()))
				if ls.Mapped() {
					usable = excludeZone(usable, s.Current(), s.ExclusiveEdge())
				}
			}
		}
	}

	if usable == out.usable {
		return
	}
	out.usable = usable
	server.layoutTiles(nil)
}

// placeLayerSurface finds where a layer surface goes inside of usable,
// or inside of full if its exclusive zone is -1. A desired size of 0
// stretches it between its margins on that axis.
func placeLayerSurface(state wlr.LayerSurfaceV1State, full, usable geom.Rect[int]) geom.Rect[int] {
	bounds := usable
	if state.ExclusiveZone() == -1 {
		bounds = full
	}

	anchor := state.Anchor()
	top, right, bottom, left := state.Margin()
	x0, x1 := placeSpan(
		bounds.Min.X, bounds.Max.X,
		int(state.DesiredWidth()), int(left), int(right),
		anchor&wlr.LayerSurfaceV1AnchorLeft != 0, anchor&wlr.LayerSurfaceV1AnchorRight != 0,
	)
	y0, y1 := placeSpan(
		bounds.Min.Y, bounds.Max.Y,
		int(state.DesiredHeight()), int(top), int(bottom),
		anchor&wlr.LayerSurfaceV1AnchorTop != 0, anchor&wlr.LayerSurfaceV1AnchorBottom != 0,
	)
	return geom.Rt(x0, y0, x1, y1).Canon()
}

// placeSpan places a span of length n between lo and hi on one axis.
// It's pushed against whichever end it is anchored to alone, and
// centered otherwise.
func placeSpan(lo, hi, n, before, after int, toLo, toHi bool) (int, int) {
	switch {
	case n == 0:
		return lo + before, max(lo+before, hi-after)
	case toLo && !toHi:
		return lo + before, lo + before + n
	case toHi && !toLo:
		return hi - after - n, hi - after
	default:
		start := lo + (hi-lo)/2 - n/2
		return start, start + n
	}
}

// excludeZone removes a mapped layer surface's exclusive zone from the
// edge of usable that it's anchored to.
func excludeZone(usable geom.Rect[int], state wlr.LayerSurfaceV1State, edge wlr.Edges) geom.Rect[int] {
	zone := int(state.ExclusiveZone())
	top, right, bottom, left := state.Margin()
	switch edge {
	case wlr.EdgeTop:
		return usable.Pad(zone+int(top), 0, 0, 0)
	case wlr.EdgeBottom:
		return usable.Pad(0, zone+int(bottom), 0, 0)
	case wlr.EdgeLeft:
		return usable.Pad(0, 0, zone+int(left), 0)
	case wlr.EdgeRight:
		return usable.Pad(0, 0, 0, zone+int(right))
	default:
		return usable
	}
}

// layerSurfaceCoords returns the position of a layer surface in the
// layout.
func (server *Server) layerSurfaceCoords(ls *LayerSurface) geom.Point[float64] {
	return server.outputBounds(ls.Output).Min.Add(geom.PConv[float64](ls.Geo.Min))
}

func (server *Server) layerForSurface(s wlr.Surface) *LayerSurface {
	for _, out := range server.outputs {
		for _, layer := range out.Layers {
			for _, ls := range layer {
				for sub := range ls.LayerSurface.Surfaces() {
					if sub.Surface == s {
						return ls
					}
				}
			}
		}
	}
	return nil
}

// unconstrainLayerPopup keeps a popup of a layer surface inside the
// part of the output that the status bar doesn't cover. A layer surface
// can sit outside of the usable area, like a panel in its own exclusive
// zone, so its popups aren't held to it either.
func (server *Server) unconstrainLayerPopup(ls *LayerSurface, popup wlr.XDGPopup) {
	box := server.outputVisibleBounds(ls.Output).Sub(server.layerSurfaceCoords(ls))
	popup.UnconstrainFromBox(box.ImageRect())
}
