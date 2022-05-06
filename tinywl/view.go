package main

import (
	"deedles.dev/wlr"
)

type View struct {
	surface wlr.XDGSurface
	Mapped  bool
	X       float64
	Y       float64
}

func NewView(surface wlr.XDGSurface) *View {
	return &View{surface: surface}
}

func (v *View) Surface() wlr.Surface {
	return v.surface.Surface()
}

func (v *View) XDGSurface() wlr.XDGSurface {
	return v.surface
}
