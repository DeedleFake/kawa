package main

import (
	"fmt"
	"image"

	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
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

type Menu struct {
	OnSelect func(int)

	prev     int
	active   []wlr.Texture
	inactive []wlr.Texture
}

func (server *Server) createMenu(text ...string) *Menu {
	m := Menu{
		inactive: make([]wlr.Texture, 0, len(text)),
		active:   make([]wlr.Texture, 0, len(text)),
	}
	for _, item := range text {
		m.Add(server, item)
	}

	return &m
}

func (m *Menu) Len() int {
	return len(m.active)
}

func (m *Menu) Bounds() image.Rectangle {
	var w int
	for _, t := range m.active {
		if tw := t.Width() + WindowBorder*2; tw > w {
			w = tw
		}
	}

	return box(0, 0, w, len(m.active)*int(fontOptions.Size+WindowBorder*2))
}

func (m *Menu) StartOffset() image.Point {
	b := m.Bounds()
	return image.Pt(
		-b.Dx()/2,
		-int(fontOptions.Size+WindowBorder*2)*m.prev-int(fontOptions.Size+WindowBorder*2)/2,
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
	m.inactive = append(m.inactive, server.createTextTexture(image.Black, gomonoFace, item))
	m.active = append(m.active, server.createTextTexture(image.White, gomonoFace, item))
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

func (server *Server) createTextTexture(src image.Image, face font.Face, item string) wlr.Texture {
	fdraw := font.Drawer{
		Src:  src,
		Face: face,
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

	return wlr.TextureFromImage(server.renderer, buf)
}
