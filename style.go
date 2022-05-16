package main

import (
	"fmt"
	"image"
	"image/color"

	"deedles.dev/kawa/geom"
	"deedles.dev/wlr"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
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

var (
	fontOptions = opentype.FaceOptions{
		Size: 14,
		DPI:  72,
	}

	gomonoFont *sfnt.Font
	gomonoFace font.Face
)

func init() {
	var err error
	gomonoFont, err = opentype.Parse(gomono.TTF)
	if err != nil {
		panic(fmt.Errorf("parse font: %w", err))
	}

	gomonoFace, err = opentype.NewFace(gomonoFont, &fontOptions)
	if err != nil {
		panic(fmt.Errorf("create font face: %w", err))
	}
}

func CreateTextTexture(renderer wlr.Renderer, src image.Image, item string) wlr.Texture {
	fdraw := font.Drawer{
		Src:  src,
		Face: gomonoFace,
		Dot:  fixed.P(0, int(fontOptions.Size)),
	}

	extents, _ := fdraw.BoundString(item)
	buf := image.NewNRGBA(image.Rect(
		0,
		0,
		(extents.Max.X - extents.Min.X).Floor(),
		int(fontOptions.Size),
	))
	fdraw.Dst = buf
	fdraw.DrawString(item)

	return wlr.TextureFromImage(renderer, buf)
}

type scaleFunc func(out, r geom.Rect[float64]) geom.Rect[float64]

func scaleStretch(out, r geom.Rect[float64]) geom.Rect[float64] {
	return out
}

func scaleCenter(out, r geom.Rect[float64]) geom.Rect[float64] {
	return r.Align(out.Center())
}

func scaleFit(out, r geom.Rect[float64]) geom.Rect[float64] {
	return scaleCenter(out, r).Intersect(out)
}

func scaleFill(out, r geom.Rect[float64]) geom.Rect[float64] {
	return scaleCenter(out, r.FitTo(out.Size()))
}

func scaleTile(out, r geom.Rect[float64]) geom.Rect[float64] {
	// TODO
	return scaleCenter(out, r)
}
