// Package tile provides utilities to help with laying out tiles in an
// area.
package tile

import "image"

// split splits a rectangle in two around the provided point. The
// results are undefined if both the X and Y coordinates of the point
// are non-zero.
func split(r image.Rectangle, half image.Point) (first, second image.Rectangle) {
	first = image.Rectangle{Min: r.Min, Max: r.Max.Sub(half)}
	second = image.Rectangle{Min: r.Min.Add(half), Max: r.Max}
	return
}

// vsplit splits a rectangle in half vertically.
func vsplit(r image.Rectangle) (left, right image.Rectangle) {
	half := image.Pt(r.Dx()/2, 0)
	return split(r, half)
}

// hsplit splits a rectangle in half horizontally.
func hsplit(r image.Rectangle) (top, bottom image.Rectangle) {
	half := image.Pt(0, r.Dy()/2)
	return split(r, half)
}

// RightThenDown produces a series of n rectangles the union of which recomposes r. The rectangles are produced by splitting the right-most and then the bottom-most rectangles in half recusrively. In other words,
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
func RightThenDown(r image.Rectangle, n int) []image.Rectangle {
	tiles := make([]image.Rectangle, n)
	tiles[0] = r

	split, next := vsplit, hsplit
	for i := 1; i < len(tiles); i++ {
		tiles[i-1], tiles[i] = split(tiles[i-1])
		split, next = next, split
	}

	return tiles
}
