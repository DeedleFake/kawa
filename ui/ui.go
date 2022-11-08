package ui

import (
	"deedles.dev/kawa/geom"
	"deedles.dev/wlr"
	"golang.org/x/exp/constraints"
)

// Widget is a piece of a UI hierarchy.
type Widget interface {
	// Layout calculates sizing of a widget based on the given
	// constraints.
	Layout(Constraints) LayoutContext
}

// Constraints contains info for a widget to help it calculate its own
// layout. It is up to the widget it respect it.
type Constraints struct {
	MaxSize geom.Point[float64]
}

func (c Constraints) Rect() geom.Rect[float64] {
	return geom.Rect[float64]{Max: c.MaxSize}
}

// LayoutContext is the results and related context of a widget's
// layout calculation.
type LayoutContext struct {
	// Size is the size that widget wants to be.
	Size geom.Point[float64]

	// Render is a function that will perform the actual rendering of
	// the widget that generated the LayoutContext. into is the bounds
	// that the parent of the widget would like the widget to render
	// into. It is up to the widget to respect the request.
	Render func(rc RenderContext, into geom.Rect[float64])
}

// RenderContext is context for rendering to the screen.
type RenderContext struct {
	R   wlr.Renderer
	Out wlr.Output
}

func max[T constraints.Ordered](v1, v2 T) T {
	if v1 >= v2 {
		return v1
	}
	return v2
}
