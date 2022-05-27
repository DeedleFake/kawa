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

func NewBox(vert bool, children ...Widget) *Box {
	return &Box{
		children: children,
		vert:     vert,
	}
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

type Stack struct {
	children []Widget
}

func NewStack(children ...Widget) *Stack {
	return &Stack{children: children}
}

func (s *Stack) Add(child Widget) {
	s.children = append(s.children, child)
}

// TODO: Add methods for removing widgets.

func (s *Stack) Children() []Widget {
	return s.children
}

func (s *Stack) Layout(lc LayoutConstraints) (size geom.Point[float64]) {
	for _, c := range s.children {
		cs := c.Layout(lc)
		if cs.X > size.X {
			size.X = cs.X
		}
		if cs.Y > size.Y {
			size.Y = cs.Y
		}
	}
	return size
}

func (s *Stack) Render(server *Server, out *Output, to geom.Rect[float64]) {
	for _, c := range s.children {
		c.Render(server, out, to)
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

type Align struct {
	child Widget
	edges wlr.Edges
	size  geom.Point[float64]
}

func NewAlign(edges wlr.Edges, child Widget) *Align {
	return &Align{
		child: child,
		edges: edges,
	}
}

func (a *Align) alignmentRect(to geom.Rect[float64]) geom.Rect[float64] {
	panic("Not implemented.")
}

func (a *Align) Layout(lc LayoutConstraints) geom.Point[float64] {
	a.size = a.child.Layout(lc)
	return a.size
}

func (a *Align) Render(server *Server, out *Output, to geom.Rect[float64]) {
	a.child.Render(
		server,
		out,
		a.alignmentRect(to),
	)
}

type Label struct {
	r   wlr.Renderer
	src image.Image
	s   string
	t   wlr.Texture
}

func NewLabel(r wlr.Renderer, src image.Image, text string) *Label {
	label := Label{
		r:   r,
		src: src,
		s:   text,
	}
	label.SetText(text)
	return &label
}

func (label *Label) Text() string {
	return label.s
}

func (label *Label) SetText(text string) {
	if label.t.Valid() {
		label.t.Destroy()
	}

	if text == "" {
		label.t = wlr.Texture{}
		return
	}

	label.s = text
	label.t = CreateTextTexture(label.r, label.src, text)
}

func (label *Label) Layout(lc LayoutConstraints) geom.Point[float64] {
	if !label.t.Valid() {
		return geom.Point[float64]{}
	}

	return geom.Pt(
		float64(label.t.Width()),
		float64(label.t.Height()),
	)
}

func (label *Label) Render(server *Server, out *Output, to geom.Rect[float64]) {
	if !label.t.Valid() {
		return
	}

	m := wlr.ProjectBoxMatrix(to.ImageRect(), wlr.OutputTransformNormal, 0, out.Output.TransformMatrix())
	server.renderer.RenderTextureWithMatrix(label.t, m, 1)
}
