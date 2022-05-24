package main

import (
	"deedles.dev/kawa/geom"
	"deedles.dev/wlr"
)

type StatusBar struct {
	b     geom.Rect[float64]
	title wlr.Texture
}

func NewStatusBar(server *Server, out *Output) *StatusBar {
	var b StatusBar
	b.MoveToOutput(server, out)
	return &b
}

func (b *StatusBar) Bounds() geom.Rect[float64] {
	return b.b
}

func (b *StatusBar) MoveToOutput(server *Server, out *Output) {
	ob := server.outputBounds(out)
	b.b = geom.Rt(ob.Min.X, ob.Min.Y-StatusBarHeight, ob.Max.X, ob.Min.Y)
}

func (b *StatusBar) SetTitle(title wlr.Texture) {
	if b.title.Valid() {
		b.title.Destroy()
	}
	b.title = title
}
