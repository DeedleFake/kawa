package main

import (
	"image"
	"time"

	"deedles.dev/wlr"
)

func (server *Server) onFrame(out *Output) {
	now := time.Now()

	_, err := out.Output.AttachRender()
	if err != nil {
		wlr.Log(wlr.Error, "output attach render: %v", err)
		return
	}
	defer out.Output.Commit()

	server.renderer.Begin(out.Output, out.Output.Width(), out.Output.Height())
	defer server.renderer.End()

	server.renderer.Clear(ColorBackground)

	server.renderLayer(out, wlr.LayerShellV1LayerBackground, now)
	server.renderLayer(out, wlr.LayerShellV1LayerBottom, now)
	server.renderViews(out, now)
	server.renderManipBox(out, now)
	server.renderLayer(out, wlr.LayerShellV1LayerTop, now)
	server.renderMenu(out, now)
	server.renderLayer(out, wlr.LayerShellV1LayerOverlay, now)
	server.renderCursor(out, now)
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
	server.renderer.RenderRect(r, color, out.Output.TransformMatrix())
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

func (server *Server) renderManipBox(out *Output, t time.Time) {
	// TODO
}

func (server *Server) renderMenu(out *Output, t time.Time) {
	// TODO
}

func (server *Server) renderCursor(out *Output, t time.Time) {
	out.Output.RenderSoftwareCursors(image.ZR)
}
