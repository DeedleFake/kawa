package tile

import "image"

func split(r image.Rectangle, half image.Point) (first, second image.Rectangle) {
	first = image.Rectangle{Min: r.Min, Max: r.Max.Sub(half)}
	second = image.Rectangle{Min: r.Min.Add(half), Max: r.Max}
	return
}

func vsplit(r image.Rectangle) (left, right image.Rectangle) {
	half := image.Pt(r.Dx()/2, 0)
	return split(r, half)
}

func hsplit(r image.Rectangle) (top, bottom image.Rectangle) {
	half := image.Pt(0, r.Dy()/2)
	return split(r, half)
}

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
