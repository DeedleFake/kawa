package main

import (
	"image/color"

	"deedles.dev/kawa/geom"
)

const (
	MinWidth  = 128
	MinHeight = 24

	WindowBorder    = 5
	StatusBarHeight = 5 * WindowBorder
)

var (
	ColorBackground          = color.NRGBA{0x77, 0x77, 0x77, 0xFF}
	ColorSelectionBox        = color.NRGBA{0xFF, 0x0, 0x0, 0xFF}
	ColorSelectionBackground = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF / 100}
	ColorActiveBorder        = color.NRGBA{0x50, 0xA1, 0xAD, 0xFF}
	ColorInactiveBorder      = color.NRGBA{0x9C, 0xE9, 0xE9, 0xFF}
	ColorMenuSelected        = color.NRGBA{0x3D, 0x7D, 0x42, 0xFF}
	ColorMenuUnselected      = color.NRGBA{0xEB, 0xFF, 0xEC, 0xFF}
	ColorMenuBorder          = color.NRGBA{0x78, 0xAD, 0x84, 0xFF}
	ColorSurface             = color.NRGBA{0xEE, 0xEE, 0xEE, 0xFF}
)

var (
	DefaultRestore = geom.Rt[float64](0, 0, 640, 480).Add(geom.Pt[float64](10, 10))
)

type scaleFunc func(out, r geom.Rect[float64]) geom.Rect[float64]

func scaleStretch(out, r geom.Rect[float64]) geom.Rect[float64] {
	return out
}

func scaleCenter(out, r geom.Rect[float64]) geom.Rect[float64] {
	return r.CenterAt(out.Center())
}

func scaleFit(out, r geom.Rect[float64]) geom.Rect[float64] {
	if (r.Dx() < out.Dx()) && (r.Dy() < out.Dy()) {
		return r
	}
	return scaleFill(out, r)
}

func scaleFill(out, r geom.Rect[float64]) geom.Rect[float64] {
	return scaleCenter(out, r.FitTo(out.Size()))
}

func scaleTile(out, r geom.Rect[float64]) geom.Rect[float64] {
	// TODO
	return scaleCenter(out, r)
}
