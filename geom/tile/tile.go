// Package tile provides utilities to help with laying out tiles in an
// area.
package tile

import (
	"deedles.dev/kawa/geom"
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
//    RightThenDown(r, 4)
//
// will produce
//
//    ------------
//    |    |     |
//    |    -------
//    |    |  |  |
//    ------------
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
