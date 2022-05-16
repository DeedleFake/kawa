// Package tile provides utilities to help with laying out tiles in an
// area.
package tile

import (
	"deedles.dev/kawa/geom"
)

// hsplit splits a rectangle into two rectangles arranged
// horizontally.
func hsplit(r geom.Rect[float64], w float64) (left, right geom.Rect[float64]) {
	left = r.Resize(geom.Pt(w, r.Dy()))
	right = r.Resize(geom.Pt(r.Dx()-w, r.Dy())).Add(geom.Pt(w, 0))
	return left, right
}

func hsplitHalf(r geom.Rect[float64]) (left, right geom.Rect[float64]) {
	return hsplit(r, r.Dx()/2)
}

// vsplit splits a rectangle into two rectangles arranged vertically.
func vsplit(r geom.Rect[float64], h float64) (top, bottom geom.Rect[float64]) {
	top = r.Resize(geom.Pt(r.Dx(), h))
	bottom = r.Resize(geom.Pt(r.Dx(), r.Dy()-h)).Add(geom.Pt(0, h))
	return top, bottom
}

func vsplitHalf(r geom.Rect[float64]) (top, bottom geom.Rect[float64]) {
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
func RightThenDown(r geom.Rect[float64], n int) []geom.Rect[float64] {
	tiles := make([]geom.Rect[float64], n)
	return tiles
}

func rightThenDown(tiles []geom.Rect[float64], r geom.Rect[float64]) {
	tiles[0] = r

	split, next := hsplitHalf, vsplitHalf
	for i := 1; i < len(tiles); i++ {
		tiles[i-1], tiles[i] = split(tiles[i-1])
		split, next = next, split
	}
}

// TwoThirdsSidebar produces a layout where the first rectangle is
// two-thirds the width of r and the rest are arranged vertically in
// an even split in the remaining space.
func TwoThirdsSidebar(r geom.Rect[float64], n int) []geom.Rect[float64] {
	tiles := make([]geom.Rect[float64], n)
	twoThirdsSidebar(tiles, r)
	return tiles
}

func twoThirdsSidebar(tiles []geom.Rect[float64], r geom.Rect[float64]) {
	var rem geom.Rect[float64]
	tiles[0], rem = hsplit(r, 2*r.Dx()/3)
	evenVertically(tiles[1:], rem)
}

// EvenVertically splits r into n rectangles arranged vertically each
// with the full width of r.
func EvenVertically(r geom.Rect[float64], n int) []geom.Rect[float64] {
	tiles := make([]geom.Rect[float64], n)
	evenVertically(tiles, r)
	return tiles
}

func evenVertically(tiles []geom.Rect[float64], r geom.Rect[float64]) {
	size := geom.Pt(0, r.Dy()/float64(len(tiles)))
	c, _ := vsplit(r, size.Y)
	for i := range tiles {
		tiles[i] = c
		c = c.Add(size)
	}
}
