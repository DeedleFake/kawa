// Package layout provides utilities to help with laying out
// rectangles inside of other rectangles.
package layout

import (
	"deedles.dev/kawa/geom"
	"deedles.dev/wlr"
	"golang.org/x/exp/constraints"
)

// hsplit splits a rectangle into two rectangles arranged
// horizontally.
func hsplit[T constraints.Integer | constraints.Float](r geom.Rect[T], w T) (left, right geom.Rect[T]) {
	left = r.Resize(geom.Pt(w, r.Dy()))
	right = r.Resize(geom.Pt(r.Dx()-w, r.Dy())).Add(geom.Pt(w, 0))
	return left, right
}

func hsplitHalf[T constraints.Integer | constraints.Float](r geom.Rect[T]) (left, right geom.Rect[T]) {
	return hsplit(r, r.Dx()/2)
}

// vsplit splits a rectangle into two rectangles arranged vertically.
func vsplit[T constraints.Integer | constraints.Float](r geom.Rect[T], h T) (top, bottom geom.Rect[T]) {
	top = r.Resize(geom.Pt(r.Dx(), h))
	bottom = r.Resize(geom.Pt(r.Dx(), r.Dy()-h)).Add(geom.Pt(0, h))
	return top, bottom
}

func vsplitHalf[T constraints.Integer | constraints.Float](r geom.Rect[T]) (top, bottom geom.Rect[T]) {
	return vsplit(r, r.Dy()/2)
}

// RightThenDown produces a series of n rectangles the union of which
// recomposes r. The rectangles are produced by splitting the
// right-most and then the bottom-most rectangles in half recusrively.
// In other words,
//
//	RightThenDown(r, 4)
//
// will produce
//
//	------------
//	|    |     |
//	|    -------
//	|    |  |  |
//	------------
func RightThenDown[T constraints.Integer | constraints.Float](r geom.Rect[T], n int) []geom.Rect[T] {
	tiles := make([]geom.Rect[T], n)
	return tiles
}

func rightThenDown[T constraints.Integer | constraints.Float](tiles []geom.Rect[T], r geom.Rect[T]) {
	tiles[0] = r

	split, next := hsplitHalf[T], vsplitHalf[T]
	for i := 1; i < len(tiles); i++ {
		tiles[i-1], tiles[i] = split(tiles[i-1])
		split, next = next, split
	}
}

// TwoThirdsSidebar produces a layout where the first rectangle is
// two-thirds the width of r and the rest are arranged vertically in
// an even split in the remaining space.
func TwoThirdsSidebar[T constraints.Integer | constraints.Float](r geom.Rect[T], n int) []geom.Rect[T] {
	tiles := make([]geom.Rect[T], n)
	twoThirdsSidebar(tiles, r)
	return tiles
}

func twoThirdsSidebar[T constraints.Integer | constraints.Float](tiles []geom.Rect[T], r geom.Rect[T]) {
	var rem geom.Rect[T]
	tiles[0], rem = hsplit(r, 2*r.Dx()/3)
	evenVertically(tiles[1:], rem)
}

// EvenVertically splits r into n rectangles arranged vertically each
// with the full width of r.
func EvenVertically[T constraints.Integer | constraints.Float](r geom.Rect[T], n int) []geom.Rect[T] {
	tiles := make([]geom.Rect[T], n)
	evenVertically(tiles, r)
	return tiles
}

func evenVertically[T constraints.Integer | constraints.Float](tiles []geom.Rect[T], r geom.Rect[T]) {
	size := geom.Pt(0, r.Dy()/T(len(tiles)))
	c, _ := vsplit(r, size.Y)
	for i := range tiles {
		tiles[i] = c
		c = c.Add(size)
	}
}

// VerticalStack returns len(sizes) rectangles stacked vertically. Each
// rectangles height can differ but they are all the same width, specifically
// the width of the widest provided size. The top-left corner of the first
// rectangle is positioned at start.
func VerticalStack[T constraints.Integer | constraints.Float](start geom.Point[T], sizes []geom.Point[T]) []geom.Rect[T] {
	rects := make([]geom.Rect[T], 0, len(sizes))

	prev := geom.Rt(start.X, start.Y, start.X, start.Y)
	for _, size := range sizes {
		if size.X > prev.Dx() {
			prev.Max.X = prev.Min.X + size.X
		}
	}

	for i := range sizes {
		prev = geom.Rt(prev.Min.X, prev.Max.Y, prev.Max.X, prev.Max.Y+sizes[i].Y)
		rects = append(rects, prev)
	}

	return rects
}

// Align shifts the specified edges of inner to align with the
// corresponding edges of outer, stretching the rectangle as
// necessary if opposite edges are specified.
func Align[T constraints.Integer | constraints.Float](outer, inner geom.Rect[T], edges wlr.Edges) geom.Rect[T] {
	inner = inner.Align(outer.Center())
	switch {
	case edges&wlr.EdgeTop != 0:
		inner.Min.Y, inner.Max.Y = outer.Min.Y, outer.Min.Y+inner.Dy()
		if edges&wlr.EdgeBottom != 0 {
			inner.Max.Y = outer.Max.Y
		}
	case edges&wlr.EdgeBottom != 0:
		inner.Min.Y, inner.Max.Y = outer.Max.Y-inner.Dy(), outer.Max.Y
	}
	switch {
	case edges&wlr.EdgeLeft != 0:
		inner.Min.X, inner.Max.X = outer.Min.X, outer.Min.X+inner.Dx()
		if edges&wlr.EdgeRight != 0 {
			inner.Max.X = outer.Max.X
		}
	case edges&wlr.EdgeRight != 0:
		inner.Min.X, inner.Max.X = outer.Max.X-inner.Dx(), outer.Max.X
	}

	return inner
}
