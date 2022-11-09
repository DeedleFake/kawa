package main

import (
	"image"

	"deedles.dev/kawa/draw"
	"deedles.dev/wlr"
)

type StatusBar struct {
	out   *Output
	title wlr.Texture
}

func NewStatusBar(out *Output) *StatusBar {
	return &StatusBar{
		out: out,
	}
}

func (s *StatusBar) SetTitle(r wlr.Renderer, str string) {
	if s.title.Valid() {
		s.title.Destroy()
		s.title = wlr.Texture{}
	}
	if str == "" {
		return
	}

	s.title = draw.CreateTextTexture(r, image.White, str)
}

func (s *StatusBar) Title() wlr.Texture {
	return s.title
}

func (s *StatusBar) Output() *Output {
	return s.out
}
