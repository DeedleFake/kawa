package main

import (
	"image"

	"deedles.dev/kawa/geom"
	"deedles.dev/kawa/ui"
	"deedles.dev/wlr"
)

type StatusBar struct {
	State *StatusBarState
}

func (sb StatusBar) Layout(con ui.Constraints) ui.LayoutContext {
	title := ui.Align{
		Edges: wlr.EdgeLeft,
		Child: ui.Padding{
			Top:    WindowBorder,
			Bottom: WindowBorder,
			Left:   WindowBorder,
			Right:  WindowBorder,
			Child: ui.Label{
				State: &sb.State.title,
			},
		},
	}

	con.MaxSize.Y = StatusBarHeight
	lctitle := title.Layout(con)

	return ui.LayoutContext{
		Size: con.MaxSize,
		Render: func(rc ui.RenderContext, into geom.Rect[float64]) {
			rc.R.RenderRect(into.ImageRect(), ColorMenuBorder, rc.Out.TransformMatrix())
			lctitle.Render(rc, into)
		},
	}
}

type StatusBarState struct {
	title ui.LabelState
}

func (s *StatusBarState) Title() string {
	return s.title.Text()
}

func (s *StatusBarState) SetTitle(r wlr.Renderer, src image.Image, str string) {
	s.title.SetText(r, src, str)
}
