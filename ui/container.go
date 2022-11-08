package ui

import (
	"image"

	"deedles.dev/kawa/geom"
	"deedles.dev/wlr"
)

type Padding struct {
	Child                    Widget
	Top, Bottom, Left, Right float64
}

func (p Padding) toChild(con Constraints) Constraints {
	return Constraints{
		MaxSize: p.pad(con.Rect()).Size(),
	}
}

func (p Padding) fromChild(lc LayoutContext) geom.Point[float64] {
	return geom.Pt(
		lc.Size.X+p.Left+p.Right,
		lc.Size.Y+p.Top+p.Bottom,
	)
}

func (p Padding) pad(r geom.Rect[float64]) geom.Rect[float64] {
	return r.Pad(p.Top, p.Bottom, p.Left, p.Right)
}

func (p Padding) Layout(con Constraints) LayoutContext {
	lc := p.Child.Layout(p.toChild(con))
	return LayoutContext{
		Size: p.fromChild(lc),
		Render: func(rc RenderContext, into geom.Rect[float64]) {
			lc.Render(rc, p.pad(into))
		},
	}
}

type Center struct {
	Child Widget
}

func (c Center) Layout(con Constraints) LayoutContext {
	lc := c.Child.Layout(con)
	return LayoutContext{
		Size: con.MaxSize,
		Render: func(rc RenderContext, into geom.Rect[float64]) {
			lc.Render(rc, geom.Rect[float64]{Max: lc.Size}.Align(into.Center()))
		},
	}
}

type Align struct {
	Child Widget
	Edges wlr.Edges
}

func (a Align) alignmentRect(lc LayoutContext, into geom.Rect[float64]) geom.Rect[float64] {
	r := geom.Rect[float64]{Max: lc.Size}.Align(into.Center())
	if a.Edges&wlr.EdgeTop != 0 {
		r.Min.Y, r.Max.Y = into.Min.Y, into.Min.Y+r.Dy()
	}
	if a.Edges&wlr.EdgeBottom != 0 {
		r.Min.Y, r.Max.Y = into.Max.Y-r.Dy(), into.Max.Y
	}
	if a.Edges&wlr.EdgeLeft != 0 {
		r.Min.X, r.Max.X = into.Min.X, into.Min.X+r.Dx()
	}
	if a.Edges&wlr.EdgeRight != 0 {
		r.Min.X, r.Max.X = into.Max.X-r.Dx(), into.Max.X
	}

	return r
}

func (a Align) Layout(con Constraints) LayoutContext {
	lc := a.Child.Layout(con)
	return LayoutContext{
		Size: con.MaxSize,
		Render: func(rc RenderContext, into geom.Rect[float64]) {
			lc.Render(rc, a.alignmentRect(lc, into))
		},
	}
}

type Texture struct {
	Tex          wlr.Texture
	Transparency float64
}

func (t Texture) render(rc RenderContext, into geom.Rect[float64]) {
	m := wlr.ProjectBoxMatrix(
		into.ImageRect(),
		wlr.OutputTransformNormal,
		0,
		rc.Out.TransformMatrix(),
	)
	rc.R.RenderTextureWithMatrix(t.Tex, m, float32(1-t.Transparency))
}

func (t Texture) Layout(con Constraints) LayoutContext {
	return LayoutContext{
		Size:   geom.Pt(float64(t.Tex.Width()), float64(t.Tex.Height())),
		Render: t.render,
	}
}

type Label struct {
	State *LabelState
}

func (label Label) Layout(con Constraints) LayoutContext {
	tex := label.State.tex
	if !tex.Valid() {
		return LayoutContext{
			Render: func(RenderContext, geom.Rect[float64]) {},
		}
	}

	return Texture{Tex: tex}.Layout(con)
}

type LabelState struct {
	tex wlr.Texture
	str string
}

func (ls *LabelState) update(rc RenderContext, src image.Image) {
	if ls.str == "" {
		ls.tex = wlr.Texture{}
		return
	}

	ls.tex = CreateTextTexture(rc, src, ls.str)
}

func (ls *LabelState) Text() string {
	return ls.str
}

func (ls *LabelState) SetText(rc RenderContext, src image.Image, str string) {
	ls.str = str
	ls.update(rc, src)
}
