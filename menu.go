package main

import (
	"fmt"
	"image"

	"deedles.dev/kawa/internal/drm"
	"deedles.dev/kawa/internal/fimg"
	"deedles.dev/wlr"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var monoFont *sfnt.Font

func init() {
	gomonoFont, err := opentype.Parse(gomono.TTF)
	if err != nil {
		panic(fmt.Errorf("parse font: %w", err))
	}

	monoFont = gomonoFont
}

type Menu struct {
	active   []wlr.Texture
	inactive []wlr.Texture
}

func (server *Server) createMenu(text ...string) *Menu {
	ren := server.renderer

	gomono, err := opentype.NewFace(monoFont, &opentype.FaceOptions{
		Size: 24,
		DPI:  72,
	})
	if err != nil {
		panic(fmt.Errorf("create font face: %w", err))
	}

	buf := image.NewNRGBA(image.Rect(0, 0, 128, 128))

	inactive := make([]wlr.Texture, 0, len(text))
	for _, item := range text {
		inactive = append(inactive, createTextTexture(ren, buf, image.Black, gomono, item))
	}

	active := make([]wlr.Texture, 0, len(text))
	for _, item := range text {
		active = append(active, createTextTexture(ren, buf, image.White, gomono, item))
	}

	return &Menu{
		inactive: inactive,
		active:   active,
	}
}

func (m *Menu) Bounds() image.Rectangle {
	var w, h int
	for _, t := range m.active {
		if tw := t.Width(); tw > w {
			w = tw
		}
		h += t.Height()
	}

	return box(0, 0, w, h)
}

func createTextTexture(ren wlr.Renderer, dst *image.NRGBA, src image.Image, face font.Face, item string) wlr.Texture {
	draw.Copy(dst, image.ZP, image.Transparent, image.Transparent.Bounds(), draw.Src, nil)

	fdraw := font.Drawer{
		Dst:  dst,
		Src:  src,
		Face: face,
		Dot:  fixed.P(0, 24),
	}

	extents, _ := fdraw.BoundString(item)
	fdraw.DrawString(item)

	buf := fimg.NewNABGR(image.Rect(
		0,
		0,
		int(extents.Max.X-extents.Min.X),
		int(extents.Max.Y-extents.Min.Y),
	))
	draw.Copy(buf, image.ZP, dst, image.Rect(
		int(extents.Min.X),
		int(extents.Min.Y),
		int(extents.Max.X),
		int(extents.Max.Y),
	), draw.Src, nil)

	return wlr.TextureFromPixels(
		ren,
		drm.FormatABGR8888,
		uint32(buf.Stride),
		uint32(buf.Bounds().Dx()),
		uint32(buf.Bounds().Dy()),
		buf.Pix,
	)
}
