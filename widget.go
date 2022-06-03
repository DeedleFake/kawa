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
	// Size returns the size that the widget wants given the provided
	// constraints.
	Size(min, max geom.Point[float64]) (size geom.Point[float64])

	// Position instructs the widget to position itself on the screen,
	// using the provided base as a guide. It is entirely the widget's
	// decision on how to use the provided base, and some might even
	// ignore it completely.
	//
	// The returned rectangle are the bounds that the widget should draw
	// itself into the next time that Render is called.
	Position(base geom.Rect[float64]) (actual geom.Rect[float64])

	// Render renders the widget onto the screen.
	Render(server *Server, out *Output)
}

type Container struct {
	children []Widget
	sizes    []geom.Rect[float64]
	bounds   []geom.Rect[float64]
	layout   ContainerLayout
}

func NewContainer(layout ContainerLayout, children ...Widget) *Container {
	return &Container{
		children: children,
		layout:   layout,
	}
}

func (c *Container) Add(child Widget) {
	c.children = append(c.children, child)
}

func (c *Container) Remove(child Widget) {
	i := slices.IndexFunc(c.children, util.Match(child))
	c.RemoveIndex(i)
}

func (c *Container) RemoveIndex(i int) {
	c.children = slices.Delete(c.children, i, i+1)
}

func (c *Container) Children() []Widget {
	return c.children
}

func (c *Container) Size(min, max geom.Point[float64]) geom.Point[float64] {
	c.sizes = c.layout.LayoutChildren(c, min, max)

	var bounds geom.Rect[float64]
	for _, b := range c.sizes {
		bounds = bounds.Union(b)
	}
	return bounds.Size()
}

func (c *Container) Position(base geom.Rect[float64]) (bounds geom.Rect[float64]) {
	c.bounds = c.bounds[:0]
	for i, child := range c.Children() {
		cb := child.Position(c.layout.Position(base, c.sizes[i]))
		c.bounds = append(c.bounds, cb)
		bounds = bounds.Union(cb)
	}

	return bounds
}

func (c *Container) Render(server *Server, out *Output) {
	for _, c := range c.children {
		c.Render(server, out)
	}
}

type ContainerLayout interface {
	LayoutChildren(c *Container, min, max geom.Point[float64]) []geom.Rect[float64]
	Position(base, layout geom.Rect[float64]) geom.Rect[float64]
}

var (
	VBoxLayout  ContainerLayout = boxLayout{vert: true}
	HBoxLayout  ContainerLayout = boxLayout{vert: false}
	StackLayout ContainerLayout = stackLayout{}
)

type boxLayout struct {
	vert bool
}

func (b boxLayout) addOffset(base geom.Rect[float64], off geom.Point[float64]) geom.Rect[float64] {
	if b.vert {
		return base.Add(geom.Pt(0, off.Y))
	}
	return base.Add(geom.Pt(off.X, 0))
}

func (b boxLayout) childMax(base, layout geom.Rect[float64]) geom.Point[float64] {
	if b.vert {
		return geom.Pt(base.Max.X, layout.Max.Y)
	}
	return geom.Pt(layout.Max.X, base.Max.Y)
}

func (b boxLayout) LayoutChildren(c *Container, min, max geom.Point[float64]) (sizes []geom.Rect[float64]) {
	var bounds geom.Rect[float64]
	for _, c := range c.Children() {
		off := b.addOffset(geom.Rect[float64]{}, bounds.Max)

		cmax := max.Sub(off.Min)
		csize := c.Size(min, cmax)
		sizes = append(sizes, off.Resize(csize))
	}

	return sizes
}

func (b boxLayout) Position(base, layout geom.Rect[float64]) geom.Rect[float64] {
	return geom.Rect[float64]{Min: base.Min.Add(layout.Min), Max: b.childMax(base, layout)}
}

type stackLayout struct{}

func (s stackLayout) LayoutChildren(c *Container, min, max geom.Point[float64]) (sizes []geom.Rect[float64]) {
	for _, c := range c.Children() {
		sizes = append(sizes, geom.Rect[float64]{Max: c.Size(min, max)})
	}
	return sizes
}

func (s stackLayout) Position(base, layout geom.Rect[float64]) geom.Rect[float64] {
	return geom.Rect[float64]{Min: base.Min, Max: layout.Max}
}

type Padding struct {
	child                    Widget
	bounds                   geom.Rect[float64]
	top, bottom, left, right float64
}

func NewPadding(top, bottom, left, right float64, child Widget) *Padding {
	return &Padding{
		child:  child,
		top:    top,
		bottom: bottom,
		left:   left,
		right:  right,
	}
}

func NewUniformPadding(amount float64, child Widget) *Padding {
	return &Padding{
		child:  child,
		top:    amount,
		bottom: amount,
		left:   amount,
		right:  amount,
	}
}

func (p *Padding) SetChild(child Widget) {
	p.child = child
}

func (p *Padding) Child() Widget {
	return p.child
}

func (p *Padding) Size(min, max geom.Point[float64]) geom.Point[float64] {
	pad := geom.Pt(p.top+p.bottom, p.left+p.right)
	max = max.Sub(pad)
	return p.child.Size(min, max).Add(pad)
}

func (p *Padding) Position(base geom.Rect[float64]) geom.Rect[float64] {
	p.bounds = p.child.Position(base.Pad(p.top, p.bottom, p.left, p.right))
	return p.bounds
}

func (p *Padding) Render(server *Server, out *Output) {
	p.child.Render(server, out)
}

type Center struct {
	child  Widget
	size   geom.Point[float64]
	bounds geom.Rect[float64]
}

func NewCenter(child Widget) *Center {
	return &Center{child: child}
}

func (c *Center) Size(min, max geom.Point[float64]) geom.Point[float64] {
	c.size = c.child.Size(min, max)
	return c.size
}

func (c *Center) Position(base geom.Rect[float64]) geom.Rect[float64] {
	c.bounds = c.child.Position(geom.Rect[float64]{Max: c.size}.Align(base.Center()))
	return c.bounds
}

func (c *Center) Render(server *Server, out *Output) {
	c.child.Render(server, out)
}

type Align struct {
	child  Widget
	edges  wlr.Edges
	size   geom.Point[float64]
	bounds geom.Rect[float64]
}

func NewAlign(edges wlr.Edges, child Widget) *Align {
	return &Align{
		child: child,
		edges: edges,
	}
}

func (a *Align) alignmentRect(to geom.Rect[float64]) geom.Rect[float64] {
	r := geom.Rect[float64]{Max: a.size}.Align(to.Center())
	if a.edges&wlr.EdgeTop != 0 {
		r.Min.Y, r.Max.Y = to.Min.Y, to.Min.Y+r.Dy()
	}
	if a.edges&wlr.EdgeBottom != 0 {
		r.Min.Y, r.Max.Y = to.Max.Y-r.Dy(), to.Max.Y
	}
	if a.edges&wlr.EdgeLeft != 0 {
		r.Min.X, r.Max.X = to.Min.X, to.Min.X+r.Dx()
	}
	if a.edges&wlr.EdgeRight != 0 {
		r.Min.X, r.Max.X = to.Max.X-r.Dx(), to.Max.X
	}

	return r
}

func (a *Align) Size(min, max geom.Point[float64]) geom.Point[float64] {
	a.size = a.child.Size(min, max)
	return a.size
}

func (a *Align) Position(base geom.Rect[float64]) geom.Rect[float64] {
	a.bounds = a.child.Position(a.alignmentRect(base))
	return a.bounds
}

func (a *Align) Render(server *Server, out *Output) {
	a.child.Render(server, out)
}

type Label struct {
	r      wlr.Renderer
	src    image.Image
	s      string
	t      wlr.Texture
	bounds geom.Rect[float64]
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

func (label *Label) Size(min, max geom.Point[float64]) geom.Point[float64] {
	if !label.t.Valid() {
		return geom.Point[float64]{}
	}

	return geom.Pt(
		float64(label.t.Width()),
		float64(label.t.Height()),
	)
}

func (label *Label) Position(base geom.Rect[float64]) geom.Rect[float64] {
	label.bounds = base
	return label.bounds
}

func (label *Label) Render(server *Server, out *Output) {
	if !label.t.Valid() {
		return
	}

	m := wlr.ProjectBoxMatrix(label.bounds.ImageRect(), wlr.OutputTransformNormal, 0, out.Output.TransformMatrix())
	server.renderer.RenderTextureWithMatrix(label.t, m, 1)
}
