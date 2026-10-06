package kawa

import (
	"slices"

	"deedles.dev/kawa/internal/anchor"
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
		server.updateLayerFocus()
	})
	ls.onUnmapListener = surface.Surface().OnUnmap(func(wlr.Surface) {
		server.arrangeLayers(ls.Output)
		server.updateLayerFocus()
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
	if current.Committed()&wlr.LayerSurfaceV1StateKeyboardInteractivity != 0 {
		server.updateLayerFocus()
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

				state := anchorState(s.Current())
				ls.Geo = anchor.Place(state, full, usable)
				s.Configure(uint32(ls.Geo.Dx()), uint32(ls.Geo.Dy()))
				if ls.Mapped() {
					usable = anchor.Exclude(usable, state, s.ExclusiveEdge())
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

// anchorState returns what a layer surface asks for in its state.
func anchorState(state wlr.LayerSurfaceV1State) anchor.State {
	top, right, bottom, left := state.Margin()
	return anchor.State{
		Anchor: wlr.Edges(state.Anchor()),
		Width:  int(state.DesiredWidth()),
		Height: int(state.DesiredHeight()),
		Margin: anchor.Margins{Top: int(top), Right: int(right), Bottom: int(bottom), Left: int(left)},
		Zone:   int(state.ExclusiveZone()),
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

// layerSurfaceAt returns the topmost mapped layer surface on out in
// one of layers that takes input at p, along with the surface there
// and p relative to that surface.
func (server *Server) layerSurfaceAt(out *Output, p geom.Point[float64], layers ...wlr.LayerShellV1Layer) (*LayerSurface, wlr.Surface, geom.Point[float64], bool) {
	for _, layer := range layers {
		list := out.Layers[layer]
		for _, ls := range slices.Backward(list) {
			if !ls.Mapped() {
				continue
			}

			lp := p.Sub(server.layerSurfaceCoords(ls))
			s, sx, sy, ok := ls.LayerSurface.Surface().SurfaceAt(lp.X, lp.Y)
			if ok {
				return ls, s, geom.Pt(sx, sy), true
			}
		}
	}
	return nil, wlr.Surface{}, geom.Point[float64]{}, false
}

// layerPopupAt is like layerSurfaceAt, but for the popups of every
// layer surface on out.
func (server *Server) layerPopupAt(out *Output, p geom.Point[float64]) (*LayerSurface, wlr.Surface, geom.Point[float64], bool) {
	for layer := wlr.LayerShellV1LayerOverlay; layer >= wlr.LayerShellV1LayerBackground; layer-- {
		list := out.Layers[layer]
		for _, ls := range slices.Backward(list) {
			if !ls.Mapped() {
				continue
			}

			lp := p.Sub(server.layerSurfaceCoords(ls))
			s, sx, sy, ok := ls.LayerSurface.PopupSurfaceAt(lp.X, lp.Y)
			if ok {
				return ls, s, geom.Pt(sx, sy), true
			}
		}
	}
	return nil, wlr.Surface{}, geom.Point[float64]{}, false
}

// unconstrainLayerPopup keeps a popup of a layer surface inside the
// part of the output that the status bar doesn't cover. A layer surface
// can sit outside of the usable area, like a panel in its own exclusive
// zone, so its popups aren't held to it either.
func (server *Server) unconstrainLayerPopup(ls *LayerSurface, popup wlr.XDGPopup) {
	box := server.outputVisibleBounds(ls.Output).Sub(server.layerSurfaceCoords(ls))
	popup.UnconstrainFromBox(box.ImageRect())
}
