package main

import (
	"image"
	"image/color"
	"time"

	"deedles.dev/kawa/geom"
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

	server.renderBG(out, t)
	server.renderLayer(out, wlr.LayerShellV1LayerBackground, t)
	server.renderLayer(out, wlr.LayerShellV1LayerBottom, t)
	server.renderViews(out, t)
	server.renderNewViews(out, t)
	server.renderLayer(out, wlr.LayerShellV1LayerTop, t)
	server.renderStatusBar(out, t)
	server.renderMode(out, t)
	server.renderLayer(out, wlr.LayerShellV1LayerOverlay, t)
	server.renderCursor(out, t)
}

func (server *Server) renderBG(out *Output, t time.Time) {
	if !server.bg.Valid() {
		return
	}

	m := wlr.ProjectBoxMatrix(
		image.Rect(0, 0, out.Output.Width(), out.Output.Height()),
		wlr.OutputTransformNormal,
		0,
		out.Output.TransformMatrix(),
	)
	server.renderer.RenderTextureWithMatrix(server.bg, m, 1)
}

func (server *Server) renderLayer(out *Output, layer wlr.LayerShellV1Layer, t time.Time) {
	// TODO
}

func (server *Server) renderViews(out *Output, t time.Time) {
	for _, view := range server.tiled {
		if !view.Mapped() {
			continue
		}

		server.renderView(out, view, t)
	}

	for _, view := range server.views {
		if !view.Mapped() {
			continue
		}

		server.renderView(out, view, t)
	}
}

func (server *Server) renderView(out *Output, view *View, t time.Time) {
	if !view.CSD {
		server.renderViewBorder(out, view, t)
	}
	server.renderViewSurfaces(out, view, t)
}

func (server *Server) renderViewBorder(out *Output, view *View, t time.Time) {
	color := ColorInactiveBorder
	if view.Activated() {
		color = ColorActiveBorder
	}
	if server.targetView() == view {
		color = ColorSelectionBox
	}

	r := server.viewBounds(out, view).Inset(-WindowBorder)
	server.renderRectBorder(out, geom.RConv[float64](r), color, t)
}

func (server *Server) renderRectBorder(out *Output, r geom.Rect[float64], color color.Color, t time.Time) {
	server.renderer.RenderRect(geom.Rt(0, 0, WindowBorder, r.Dy()).Add(r.Min).ImageRect(), color, out.Output.TransformMatrix())
	server.renderer.RenderRect(geom.Rt(0, 0, WindowBorder, r.Dy()).Add(geom.Pt(r.Max.X-WindowBorder, r.Min.Y)).ImageRect(), color, out.Output.TransformMatrix())
	server.renderer.RenderRect(geom.Rt(0, 0, r.Dx(), WindowBorder).Add(r.Min).ImageRect(), color, out.Output.TransformMatrix())
	server.renderer.RenderRect(geom.Rt(0, 0, r.Dx(), WindowBorder).Add(geom.Pt(r.Min.X, r.Max.Y-WindowBorder)).ImageRect(), color, out.Output.TransformMatrix())
}

func (server *Server) renderSelectionBox(out *Output, r geom.Rect[float64], t time.Time) {
	r = r.Canon()
	server.renderRectBorder(out, r, ColorSelectionBox, t)
	server.renderer.RenderRect(r.Inset(WindowBorder).ImageRect(), ColorSelectionBackground, out.Output.TransformMatrix())
}

func (server *Server) renderViewSurfaces(out *Output, view *View, t time.Time) {
	view.ForEachSurface(func(s wlr.Surface, x, y int) {
		p := geom.Pt(x, y)
		server.renderSurface(out, s, geom.PConv[int](view.Coords).Add(p), t)
	})
}

func (server *Server) renderSurface(out *Output, s wlr.Surface, p geom.Point[int], t time.Time) {
	texture := s.GetTexture()
	if !texture.Valid() {
		wlr.Log(wlr.Error, "invalid texture for surface")
		return
	}

	r := server.surfaceBounds(s, geom.PConv[int](p))
	tr := s.Current().Transform().Invert()
	m := wlr.ProjectBoxMatrix(r.ImageRect(), tr, 0, out.Output.TransformMatrix())

	server.renderer.RenderTextureWithMatrix(texture, m, 1)
	s.SendFrameDone(t)
}

func (server *Server) renderNewViews(out *Output, t time.Time) {
	for _, nv := range server.newViews {
		server.renderSelectionBox(out, *nv.To, t)
	}
}

func (server *Server) renderStatusBar(out *Output, t time.Time) {
	r := server.statusBarBounds(out)
	server.renderer.RenderRect(r.ImageRect(), ColorMenuBorder, out.Output.TransformMatrix())
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

func (server *Server) renderMenu(out *Output, m *Menu, p geom.Point[float64], sel *MenuItem) {
	r := m.Bounds().Add(p)
	server.renderer.RenderRect(r.Inset(-WindowBorder).ImageRect(), ColorMenuBorder, out.Output.TransformMatrix())
	server.renderer.RenderRect(r.ImageRect(), ColorMenuUnselected, out.Output.TransformMatrix())

	for _, item := range m.items {
		ar := m.ItemBounds(item).Add(p)
		tr := geom.Rt(0, 0, float64(item.active.Width()), float64(item.active.Height())).Align(ar.Center())

		t := item.inactive
		if item == sel {
			t = item.active
			server.renderer.RenderRect(ar.ImageRect(), ColorMenuSelected, out.Output.TransformMatrix())
		}

		matrix := wlr.ProjectBoxMatrix(tr.ImageRect(), wlr.OutputTransformNormal, 0, out.Output.TransformMatrix())
		server.renderer.RenderTextureWithMatrix(t, matrix, 1)
	}
}
