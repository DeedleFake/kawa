package main

import (
	"image"

	"deedles.dev/kawa/geom"
	"deedles.dev/kawa/internal/util"
	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
)

// Widget is a piece of the compositor's UI.
type Widget interface {
	// Layout returns the size that the widget wants given the provided
	// constraints.
	Layout(LayoutConstraints) (size geom.Point[float64])

	// Render renders the widget onto the screen.
	Render(server *Server, out *Output, to geom.Rect[float64])
}

type LayoutConstraints struct {
	MinSize, MaxSize geom.Point[float64]
}

type Box struct {
	bounds   []geom.Rect[float64]
	children []Widget
	vert     bool
}

func NewBox(vert bool) *Box {
	return &Box{vert: vert}
}

func (b *Box) Add(child Widget) {
	b.children = append(b.children, child)
}

func (b *Box) Remove(child Widget) {
	i := slices.IndexFunc(b.children, util.Match(child))
	b.RemoveIndex(i)
}

func (b *Box) RemoveIndex(i int) {
	b.children = slices.Delete(b.children, i, i+1)
}

func (b *Box) Children() []Widget {
	return b.children
}

func (b *Box) Layout(lc LayoutConstraints) (size geom.Point[float64]) {
	var bounds geom.Rect[float64]
	cbounds := b.bounds[:0]
	for _, c := range b.children {
		off := geom.Pt(bounds.Dx(), 0)
		if b.vert {
			off = geom.Pt(0, bounds.Dy())
		}

		clc := lc
		clc.MaxSize = clc.MaxSize.Sub(off)
		csize := c.Layout(clc)
		r := geom.Rect[float64]{Max: csize}.Add(off)

		bounds = bounds.Union(r)
		cbounds = append(cbounds, r)
	}

	b.bounds = cbounds
	return bounds.Size()
}

func (b *Box) Render(server *Server, out *Output, to geom.Rect[float64]) {
	for i, c := range b.children {
		r := b.bounds[i]
		c.Render(server, out, r.Add(to.Min))
	}
}

type Padding struct {
	child  Widget
	amount geom.Point[float64]
}

func NewPadding(amount geom.Point[float64], child Widget) *Padding {
	return &Padding{child: child, amount: amount}
}

func (p *Padding) SetChild(child Widget) {
	p.child = child
}

func (p *Padding) Child() Widget {
	return p.child
}

func (p *Padding) Layout(lc LayoutConstraints) geom.Point[float64] {
	pad := p.amount.Mul(2)
	lc.MaxSize = lc.MaxSize.Sub(pad)
	return p.child.Layout(lc).Add(pad)
}

func (p *Padding) Render(server *Server, out *Output, to geom.Rect[float64]) {
	p.child.Render(server, out, to.Inset2(p.amount))
}

type Center struct {
	child Widget
	size  geom.Point[float64]
}

func NewCenter(child Widget) *Center {
	return &Center{child: child}
}

func (c *Center) Layout(lc LayoutConstraints) geom.Point[float64] {
	c.size = c.child.Layout(lc)
	return c.size
}

func (c *Center) Render(server *Server, out *Output, to geom.Rect[float64]) {
	c.child.Render(
		server,
		out,
		geom.Rect[float64]{Max: c.size}.Align(to.Center()),
	)
}

type Label struct {
	r   wlr.Renderer
	src image.Image
	s   string
	t   wlr.Texture
}

func NewLabel(r wlr.Renderer, src image.Image, text string) *Label {
	return &Label{
		r:   r,
		src: src,
		s:   text,
		t:   CreateTextTexture(r, src, text),
	}
}

func (label *Label) Text() string {
	return label.s
}

func (label *Label) SetText(text string) {
	label.t.Destroy()

	label.s = text
	label.t = CreateTextTexture(label.r, label.src, text)
}

func (label *Label) Layout(lc LayoutConstraints) geom.Point[float64] {
	return geom.Pt(
		float64(label.t.Width()),
		float64(label.t.Height()),
	)
}

func (label *Label) Render(server *Server, out *Output, to geom.Rect[float64]) {
	m := wlr.ProjectBoxMatrix(to.ImageRect(), wlr.OutputTransformNormal, 0, out.Output.TransformMatrix())
	server.renderer.RenderTextureWithMatrix(label.t, m, 1)
}
