package kawa

import (
	"fmt"
	"slices"

	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

// planes is a set of the levels that kawa stacks things in, from the
// top down. hitAt only finds things in the planes that it's given.
type planes uint8

const (
	planeStatusBar planes = 1 << iota
	planeLayerPopups
	planeUpperLayers
	planeFloating
	planeTiled
	planeLowerLayers
)

// What each kind of pointer event can hit.
const (
	pressPlanes = planeStatusBar | planeLayerPopups | planeUpperLayers | planeFloating | planeTiled | planeLowerLayers
	hoverPlanes = pressPlanes &^ planeStatusBar
	pickPlanes  = planeFloating | planeTiled
	swapPlanes  = planeTiled
)

type hitKind uint8

const (
	hitDesktop hitKind = iota
	hitStatusBar
	hitLayer
	hitViewContent
	hitViewBorder
)

// hit is what hitAt found under a point.
type hit struct {
	kind hitKind
	// layer is set for hitLayer.
	layer *LayerSurface
	// view is set for hitViewContent and hitViewBorder.
	view *View
	// edges are the edges of view that a hitViewBorder is on.
	edges wlr.Edges
	// surface is the surface under the point for hitLayer and
	// hitViewContent, and sp is the point relative to it.
	surface wlr.Surface
	sp      geom.Point[float64]
}

// hitAt returns the topmost thing in see at p, in layout coordinates.
func (server *Server) hitAt(p geom.Point[float64], see planes) hit {
	out := server.outputAt(p)
	if out != nil {
		if (see&planeStatusBar != 0) && (out == server.statusBar.Output()) && (p.Y <= StatusBarHeight) {
			return hit{kind: hitStatusBar}
		}
		if see&planeLayerPopups != 0 {
			if ls, s, sp, ok := server.layerPopupAt(out, p); ok {
				return hit{kind: hitLayer, layer: ls, surface: s, sp: sp}
			}
		}
		if see&planeUpperLayers != 0 {
			if ls, s, sp, ok := server.layerSurfaceAt(out, p, wlr.LayerShellV1LayerOverlay, wlr.LayerShellV1LayerTop); ok {
				return hit{kind: hitLayer, layer: ls, surface: s, sp: sp}
			}
		}
	}
	if see&planeFloating != 0 {
		if h, ok := viewsHit(server.views, p); ok {
			return h
		}
	}
	if see&planeTiled != 0 {
		if h, ok := viewsHit(server.tiled, p); ok {
			return h
		}
	}
	if (out != nil) && (see&planeLowerLayers != 0) {
		if ls, s, sp, ok := server.layerSurfaceAt(out, p, wlr.LayerShellV1LayerBottom, wlr.LayerShellV1LayerBackground); ok {
			return hit{kind: hitLayer, layer: ls, surface: s, sp: sp}
		}
	}
	return hit{kind: hitDesktop}
}

// viewsHit finds the topmost mapped view in views at p. Later views
// are on top.
func viewsHit(views []*View, p geom.Point[float64]) (hit, bool) {
	for _, view := range slices.Backward(views) {
		if !view.Mapped() {
			continue
		}
		if h, ok := viewHit(view, p); ok {
			return h, true
		}
	}
	return hit{}, false
}

func viewHit(view *View, p geom.Point[float64]) (hit, bool) {
	if s, sp, ok := view.SurfaceAt(p.Sub(view.surfaceCoords())); ok {
		return hit{kind: hitViewContent, view: view, surface: s, sp: sp}, true
	}

	// Don't bother checking the borders if there aren't any.
	if view.CSD {
		return hit{}, false
	}

	r := view.Bounds()
	if !p.In(r.Inset(-WindowBorder)) {
		return hit{}, false
	}
	return hit{kind: hitViewBorder, view: view, edges: borderEdges(r, p)}, true
}

// borderEdges returns the edges of r that p is on, given that p is in
// the border around r.
func borderEdges(r geom.Rect[float64], p geom.Point[float64]) wlr.Edges {
	left := geom.Rt(r.Min.X-WindowBorder, r.Min.Y, r.Max.X, r.Max.Y)
	if p.In(left) {
		return wlr.EdgeLeft
	}

	top := geom.Rt(r.Min.X, r.Min.Y-WindowBorder, r.Max.X, r.Max.Y)
	if p.In(top) {
		return wlr.EdgeTop
	}

	right := geom.Rt(r.Min.X, r.Min.Y, r.Max.X+WindowBorder, r.Max.Y)
	if p.In(right) {
		return wlr.EdgeRight
	}

	bottom := geom.Rt(r.Min.X, r.Min.Y, r.Max.X, r.Max.Y+WindowBorder)
	if p.In(bottom) {
		return wlr.EdgeBottom
	}

	if (p.X < r.Min.X) && (p.Y < r.Min.Y) {
		return wlr.EdgeTop | wlr.EdgeLeft
	}
	if (p.X >= r.Max.X) && (p.Y < r.Min.Y) {
		return wlr.EdgeTop | wlr.EdgeRight
	}
	if (p.X < r.Min.X) && (p.Y >= r.Max.Y) {
		return wlr.EdgeBottom | wlr.EdgeLeft
	}
	if (p.X >= r.Max.X) && (p.Y >= r.Max.Y) {
		return wlr.EdgeBottom | wlr.EdgeRight
	}

	// Where else could it possibly be if it gets to here?
	panic(fmt.Errorf("this should not have happened\np = %+v\nr = %+v", p, r))
}
