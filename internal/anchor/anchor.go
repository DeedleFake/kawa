// Package anchor places layer surfaces on an output.
package anchor

import (
	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

// Margins are the distances that a surface keeps from the edges that
// it is anchored to.
type Margins struct {
	Top, Right, Bottom, Left int
}

// State is what a layer surface asks for.
type State struct {
	// Anchor is the edges that the surface is anchored to.
	Anchor wlr.Edges
	// Width and Height are the surface's desired size. 0 stretches it
	// between its margins on that axis.
	Width, Height int
	Margin        Margins
	// Zone is the surface's exclusive zone. -1 places it in the whole
	// output, ignoring the zones of other surfaces.
	Zone int
}

// Place finds where a layer surface goes inside of usable, or inside of
// full if its exclusive zone is -1.
func Place(s State, full, usable geom.Rect[int]) geom.Rect[int] {
	bounds := usable
	if s.Zone == -1 {
		bounds = full
	}

	x0, x1 := placeSpan(
		bounds.Min.X, bounds.Max.X,
		s.Width, s.Margin.Left, s.Margin.Right,
		s.Anchor&wlr.EdgeLeft != 0, s.Anchor&wlr.EdgeRight != 0,
	)
	y0, y1 := placeSpan(
		bounds.Min.Y, bounds.Max.Y,
		s.Height, s.Margin.Top, s.Margin.Bottom,
		s.Anchor&wlr.EdgeTop != 0, s.Anchor&wlr.EdgeBottom != 0,
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

// Exclude removes a layer surface's exclusive zone from edge, the edge
// of usable that the zone applies to.
func Exclude(usable geom.Rect[int], s State, edge wlr.Edges) geom.Rect[int] {
	switch edge {
	case wlr.EdgeTop:
		return usable.Pad(s.Zone+s.Margin.Top, 0, 0, 0)
	case wlr.EdgeBottom:
		return usable.Pad(0, s.Zone+s.Margin.Bottom, 0, 0)
	case wlr.EdgeLeft:
		return usable.Pad(0, 0, s.Zone+s.Margin.Left, 0)
	case wlr.EdgeRight:
		return usable.Pad(0, 0, 0, s.Zone+s.Margin.Right)
	default:
		return usable
	}
}
