package main

import (
	"image"
	"time"

	"deedles.dev/wlr"
)

type Framer interface {
	Frame(*Server, *Output, time.Time)
}

func (server *Server) onFrame(out *Output) {
	t := time.Now()

	_, err := out.Output.AttachRender()
	if err != nil {
		wlr.Log(wlr.Error, "output attach render: %v", err)
		return
	}
	defer out.Output.Commit()

	server.renderer.Begin(out.Output, out.Output.Width(), out.Output.Height())
	defer server.renderer.End()

	server.renderer.Clear(ColorBackground)

	server.renderLayer(out, wlr.LayerShellV1LayerBackground, t)
	server.renderLayer(out, wlr.LayerShellV1LayerBottom, t)
	server.renderViews(out, t)
	server.renderLayer(out, wlr.LayerShellV1LayerTop, t)
	server.renderMode(out, t)
	server.renderLayer(out, wlr.LayerShellV1LayerOverlay, t)
	server.renderCursor(out, t)
}

func (server *Server) renderLayer(out *Output, layer wlr.LayerShellV1Layer, t time.Time) {
	// TODO
}

func (server *Server) renderViews(out *Output, t time.Time) {
	for _, view := range server.views {
		if !view.XDGSurface.Mapped() {
			continue
		}

		server.renderView(out, view, t)
	}
}

func (server *Server) renderView(out *Output, view *View, t time.Time) {
	server.renderViewBorder(out, view, t)
	server.renderViewSurfaces(out, view, t)
}

func (server *Server) renderViewBorder(out *Output, view *View, t time.Time) {
	color := ColorInactiveBorder
	if view.XDGSurface.TopLevel().Current().Activated() {
		color = ColorActiveBorder
	}
	if server.targetView() == view {
		color = ColorSelectionBox
	}

	r := server.viewBounds(out, view).Inset(-WindowBorder)
	server.renderer.RenderRect(box(r.Min.X, r.Min.Y, WindowBorder, r.Dy()), color, out.Output.TransformMatrix())
	server.renderer.RenderRect(box(r.Max.X-WindowBorder, r.Min.Y, WindowBorder, r.Dy()), color, out.Output.TransformMatrix())
	server.renderer.RenderRect(box(r.Min.X, r.Min.Y, r.Dx(), WindowBorder), color, out.Output.TransformMatrix())
	server.renderer.RenderRect(box(r.Min.X, r.Max.Y-WindowBorder, r.Dx(), WindowBorder), color, out.Output.TransformMatrix())
}

func (server *Server) renderViewSurfaces(out *Output, view *View, t time.Time) {
	view.XDGSurface.ForEachSurface(func(s wlr.Surface, x, y int) {
		server.renderSurface(out, s, view.X+x, view.Y+y, t)
	})
}

func (server *Server) renderSurface(out *Output, s wlr.Surface, x, y int, t time.Time) {
	texture := s.GetTexture()
	if !texture.Valid() {
		wlr.Log(wlr.Error, "invalid texture for surface")
		return
	}

	r := server.surfaceBounds(out, s, x, y)
	tr := s.Current().Transform().Invert()
	m := wlr.ProjectBoxMatrix(r, tr, 0, out.Output.TransformMatrix())

	server.renderer.RenderTextureWithMatrix(texture, m, 1)
	s.SendFrameDone(t)
}

func (server *Server) renderMode(out *Output, t time.Time) {
	m, ok := server.inputMode.(Framer)
	if !ok {
		return
	}

	m.Frame(server, out, t)
}

func (server *Server) renderCursor(out *Output, t time.Time) {
	out.Output.RenderSoftwareCursors(image.ZR)
}

func (server *Server) renderMenu(out *Output, m *Menu, x, y float64) int {
	cx, cy := server.cursor.X(), server.cursor.Y()
	p := image.Pt(int(cx), int(cy))

	r := box(int(x), int(y), 100, 24*5)
	server.renderer.RenderRect(r.Inset(-WindowBorder), ColorMenuBorder, out.Output.TransformMatrix())
	server.renderer.RenderRect(r, ColorMenuUnselected, out.Output.TransformMatrix())

	sel := -1
	r.Max.Y = r.Min.Y
	for i := range m.inactive {
		t := m.inactive[i]

		r.Min.Y += r.Dy()
		r.Max.Y = r.Min.Y + t.Height()

		if p.In(r) {
			server.renderer.RenderRect(r, ColorMenuSelected, out.Output.TransformMatrix())
			sel = i
		}

		matrix := wlr.ProjectBoxMatrix(r, wlr.OutputTransformNormal, 0, out.Output.TransformMatrix())
		server.renderer.RenderTextureWithMatrix(t, matrix, 1)
	}

	return sel
}
