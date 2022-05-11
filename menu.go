package main

import (
	"fmt"
	"image"

	"deedles.dev/kawa/internal/drm"
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
	})
	if err != nil {
		panic(fmt.Errorf("create font face: %w", err))
	}

	buf := image.NewNRGBA(image.Rect(0, 0, 128, 128))

	fdraw := font.Drawer{
		Dst:  buf,
		Src:  image.Black,
		Face: gomono,
	}
	inactive := make([]wlr.Texture, 0, len(text))
	for _, item := range text {
		inactive = append(inactive, createTextTexture(ren, buf, fdraw, item))
	}

	fdraw.Src = image.White
	active := make([]wlr.Texture, 0, len(text))
	for _, item := range text {
		active = append(active, createTextTexture(ren, buf, fdraw, item))
	}

	return &Menu{
		inactive: inactive,
		active:   active,
	}
}

func createTextTexture(ren wlr.Renderer, buf *image.NRGBA, fdraw font.Drawer, item string) wlr.Texture {
	draw.Copy(buf, image.ZP, image.Transparent, image.Transparent.Bounds(), draw.Src, nil)

	fdraw.Dot = fixed.P(0, 0)
	fdraw.DrawString(item)

	extents, _ := fdraw.BoundString(item)
	return wlr.TextureFromPixels(
		ren,
		drm.FormatRGBA8888,
		uint32(buf.Stride),
		uint32((extents.Max.X-extents.Min.X)+2),
		uint32((extents.Max.Y-extents.Min.Y)+2),
		buf.Pix,
	)
}
