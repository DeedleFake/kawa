package fimg

import (
	"image"
	"image/color"
)

type NABGR struct {
	Pix    []byte
	Stride int
	Rect   image.Rectangle
}

func NewNABGR(r image.Rectangle) *NABGR {
	return &NABGR{
		Pix:    make([]byte, 4*r.Dx()*r.Dy()),
		Stride: 4 * r.Dx(),
		Rect:   r,
	}
}

func (p *NABGR) PixOffset(x, y int) int {
	return ((y - p.Rect.Min.Y) * p.Stride) + (x-p.Rect.Min.X)*4
}

func (p *NABGR) Bounds() image.Rectangle {
	return p.Rect
}

func (p *NABGR) ColorModel() color.Model {
	panic("Not implemented.")
}

func (p *NABGR) At(x, y int) color.Color {
	i := p.PixOffset(x, y)
	return color.NRGBA{p.Pix[i+3], p.Pix[i+2], p.Pix[i+1], p.Pix[i]}
}

func (p *NABGR) Set(x, y int, c color.Color) {
	r, g, b, a := c.RGBA()

	i := p.PixOffset(x, y)
	p.Pix[i] = uint8(a * 255 / 0xFFFF)

	if a == 0 {
		a = 1
	}
	p.Pix[i+1] = uint8(b * 255 / a)
	p.Pix[i+2] = uint8(g * 255 / a)
	p.Pix[i+3] = uint8(r * 255 / a)
}
