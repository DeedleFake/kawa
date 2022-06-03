package main

import (
	"image"

	"deedles.dev/kawa/geom"
)

type StatusBar struct {
	bounds geom.Rect[float64]

	title   *Label
	tpad    Widget
	tsize   geom.Point[float64]
	tbounds geom.Rect[float64]
}

func NewStatusBar(server *Server) *StatusBar {
	title := NewLabel(server.renderer, image.White, "")
	return &StatusBar{
		title: title,
		tpad:  NewCenter(NewUniformPadding(WindowBorder, title)), // TODO: Bottom align this?
	}
}

func (sb *StatusBar) SetTitle(title string) {
	sb.title.SetText(title)
}

func (sb *StatusBar) Size(min, max geom.Point[float64]) geom.Point[float64] {
	sb.tsize = sb.tpad.Size(min, max)
	return geom.Pt(max.X, StatusBarHeight)
}

func (sb *StatusBar) Position(base geom.Rect[float64]) geom.Rect[float64] {
	sb.tbounds = sb.tpad.Position(geom.Rect[float64]{Max: sb.tsize}.Add(base.Min))

	sb.bounds = base
	return base
}

func (sb *StatusBar) Render(server *Server, out *Output) {
	server.renderer.RenderRect(sb.bounds.ImageRect(), ColorMenuBorder, out.Output.TransformMatrix())
	sb.tpad.Render(server, out)
}
