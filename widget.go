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
	// Layout causes the Widget to size and position itself into the
	// available area.
	//
	// available should always be canonical.
	Layout(available geom.Rect[float64])

	// Bounds is the position and size of the widget on screen. If
	// Bounds().IsZero() == true, the parent should call Layout.
	Bounds() geom.Rect[float64]

	// Render renders the widget onto the screen.
	Render(server *Server, out *Output)
}

type Box struct {
	bounds   geom.Rect[float64]
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

func (b *Box) Layout(r geom.Rect[float64]) {
	need := geom.Rt(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y)
	if b.vert {
		need = geom.Rt(r.Min.X, r.Min.Y, r.Min.X, r.Max.Y)
	}

	for _, c := range b.children {
		c.Layout(r)
		need = need.Union(c.Bounds())
	}

	b.bounds = need
}

func (b *Box) Bounds() geom.Rect[float64] {
	return b.bounds
}

func (b *Box) Render(server *Server, out *Output) {
	for _, c := range b.children {
		c.Render(server, out)
	}
}

type Padding struct {
	bounds geom.Rect[float64]
	child  Widget
	amount float64
}

func NewPadding(amount float64, child Widget) *Padding {
	return &Padding{child: child, amount: amount}
}

func (p *Padding) SetChild(child Widget) {
	p.child = child
}

func (p *Padding) Child() Widget {
	return p.child
}

func (p *Padding) Layout(r geom.Rect[float64]) {
	b := r.Inset(p.amount)
	p.child.Layout(b)
	p.bounds = p.child.Bounds().Inset(-p.amount)
}

func (p *Padding) Bounds() geom.Rect[float64] {
	return p.bounds
}

func (p *Padding) Render(server *Server, out *Output) {
	p.child.Render(server, out)
}

type Label struct {
	bounds geom.Rect[float64]
	r      wlr.Renderer
	src    image.Image
	s      string
	t      wlr.Texture
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
	label.bounds = geom.Rect[float64]{}

	label.s = text
	label.t = CreateTextTexture(label.r, label.src, text)
}

func (label *Label) Layout(r geom.Rect[float64]) {
	// TODO: This isn't going to work like this...
	label.bounds = geom.Rt(
		0,
		0,
		float64(label.t.Width()),
		float64(label.t.Height()),
	).Align(r.Center())
}

func (label *Label) Bounds() geom.Rect[float64] {
	return label.bounds
}

func (label *Label) Render(server *Server, out *Output) {
	m := wlr.ProjectBoxMatrix(label.bounds.ImageRect(), wlr.OutputTransformNormal, 0, out.Output.TransformMatrix())
	server.renderer.RenderTextureWithMatrix(label.t, m, 1)
}
