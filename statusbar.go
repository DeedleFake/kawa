package main

import (
	"image"

	"deedles.dev/kawa/geom"
)

type StatusBar struct {
	title *Label
	tpad  Widget
	tsize geom.Point[float64]
}

func NewStatusBar(server *Server) *StatusBar {
	title := NewLabel(server.renderer, image.White, "")
	return &StatusBar{
		title: title,
		tpad:  NewPadding(geom.Pt[float64](WindowBorder, WindowBorder), title),
	}
}

func (sb *StatusBar) SetTitle(title string) {
	sb.title.SetText(title)
}

func (sb *StatusBar) Layout(lc LayoutConstraints) geom.Point[float64] {
	sb.tsize = sb.tpad.Layout(lc)
	return geom.Pt(lc.MaxSize.X, StatusBarHeight)
}

func (sb *StatusBar) Render(server *Server, out *Output, to geom.Rect[float64]) {
	server.renderer.RenderRect(to.ImageRect(), ColorMenuBorder, out.Output.TransformMatrix())
	sb.tpad.Render(server, out, geom.Rect[float64]{Max: sb.tsize}.Add(to.Min))
}
