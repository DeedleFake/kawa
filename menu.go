package main

import (
	"fmt"
	"image"

	"deedles.dev/kawa/internal/drm"
	"deedles.dev/kawa/internal/fimg"
	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
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
	OnSelect func(int)

	prev     int
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

	buf := fimg.NewNABGR(image.Rect(0, 0, 128, 128))
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

func (m *Menu) Len() int {
	return len(m.active)
}

func (m *Menu) Bounds() image.Rectangle {
	var w int
	for _, t := range m.active {
		if tw := t.Width(); tw > w {
			w = tw
		}
	}

	return box(0, 0, w, len(m.active)*24)
}

func (m *Menu) StartOffset() image.Point {
	b := m.Bounds()
	return image.Pt(
		-b.Dx()/2,
		-24*m.prev-12,
	)
}

func (m *Menu) Select(n int) {
	if (n >= 0) && (n < m.Len()) {
		m.prev = n
	}
	if m.OnSelect != nil {
		m.OnSelect(n)
	}
}

func (m *Menu) Add(server *Server, item string) {
	gomono, err := opentype.NewFace(monoFont, &opentype.FaceOptions{
		Size: 24,
		DPI:  72,
	})
	if err != nil {
		panic(fmt.Errorf("create font face: %w", err))
	}

	buf := fimg.NewNABGR(image.Rect(0, 0, 128, 128))
	m.inactive = append(m.inactive, createTextTexture(server.renderer, buf, image.Black, gomono, item))
	m.active = append(m.active, createTextTexture(server.renderer, buf, image.Black, gomono, item))
}

func (m *Menu) Remove(i int) {
	m.inactive[i].Destroy()
	m.active[i].Destroy()

	m.inactive = slices.Delete(m.inactive, i, i+1)
	m.active = slices.Delete(m.active, i, i+1)

	if m.prev >= m.Len() {
		m.prev = m.Len() - 1
	}
}

func createTextTexture(ren wlr.Renderer, dst draw.Image, src image.Image, face font.Face, item string) wlr.Texture {
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
		(extents.Max.X - extents.Min.X).Floor(),
		24,
	))
	draw.Copy(buf, image.ZP, dst, buf.Bounds(), draw.Src, nil)

	return wlr.TextureFromPixels(
		ren,
		drm.FormatABGR8888,
		uint32(buf.Stride),
		uint32(buf.Bounds().Dx()),
		uint32(buf.Bounds().Dy()),
		buf.Pix,
	)
}
