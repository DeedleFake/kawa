package main

import (
	"image"
	"image/color"
	"time"

	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

type Framer interface {
	Frame(*Server, *Output)
}

func (server *Server) onFrame(out *Output) {
	_, err := out.Output.AttachRender()
	if err != nil {
		wlr.Log(wlr.Error, "output attach render: %v", err)
		return
	}
	defer out.Output.Commit()

	server.renderer.Begin(out.Output, out.Output.Width(), out.Output.Height())
	defer server.renderer.End()

	server.renderer.Clear(ColorBackground)
	server.renderBG(out)
	server.renderLayer(out, wlr.LayerShellV1LayerBackground)
	server.renderLayer(out, wlr.LayerShellV1LayerBottom)
	server.renderViews(out)
	server.renderNewViews(out)
	server.renderLayer(out, wlr.LayerShellV1LayerTop)
	server.renderLayer(out, wlr.LayerShellV1LayerOverlay)
	if server.statusBar.Output() == out {
		server.renderStatusBar()
	}
	server.renderMode(out)
	server.renderCursor(out)
}

func (server *Server) renderBG(out *Output) {
	if !server.bg.Valid() {
		return
	}

	to := server.outputTilingBounds(out)
	r := geom.RConv[float64](geom.Rt(0, 0, server.bg.Width(), server.bg.Height()))

	m := wlr.ProjectBoxMatrix(
		server.bgScale(to, r).ImageRect(),
		wlr.OutputTransformNormal,
		0,
		out.Output.TransformMatrix(),
	)
	server.renderer.RenderTextureWithMatrix(server.bg, m, 1)
}

func (server *Server) renderViews(out *Output) {
	for _, view := range server.tiled {
		if !view.Mapped() {
			continue
		}

		server.renderView(out, view)
	}

	for _, view := range server.views {
		if !view.Mapped() {
			continue
		}

		server.renderView(out, view)
	}
}

func (server *Server) renderView(out *Output, view *View) {
	if !view.CSD {
		server.renderViewBorder(out, view)
	}
	server.renderViewSurfaces(out, view)
}

func (server *Server) renderViewBorder(out *Output, view *View) {
	color := ColorInactiveBorder
	if view.Activated() {
		color = ColorActiveBorder
	}
	if server.targetView() == view {
		color = ColorSelectionBox
	}

	r := view.Bounds().Inset(-WindowBorder)
	server.renderRectBorder(out, geom.RConv[float64](r), color)
}

func (server *Server) renderViewSurfaces(out *Output, view *View) {
	view.ForEachSurface(func(s wlr.Surface, x, y int) {
		p := geom.Pt(x, y)
		server.renderSurface(out, s, geom.PConv[int](view.Coords).Add(p))
	})
}

func (server *Server) renderNewViews(out *Output) {
	for _, nv := range server.newViews {
		server.renderSelectionBox(out, *nv)
	}
}

func (server *Server) renderLayer(out *Output, layer wlr.LayerShellV1Layer) {
	// TODO
}

func (server *Server) renderRectBorder(out *Output, r geom.Rect[float64], color color.Color) {
	server.renderer.RenderRect(geom.Rt(0, 0, WindowBorder, r.Dy()).Add(r.Min).ImageRect(), color, out.Output.TransformMatrix())
	server.renderer.RenderRect(geom.Rt(0, 0, WindowBorder, r.Dy()).Add(geom.Pt(r.Max.X-WindowBorder, r.Min.Y)).ImageRect(), color, out.Output.TransformMatrix())
	server.renderer.RenderRect(geom.Rt(0, 0, r.Dx(), WindowBorder).Add(r.Min).ImageRect(), color, out.Output.TransformMatrix())
	server.renderer.RenderRect(geom.Rt(0, 0, r.Dx(), WindowBorder).Add(geom.Pt(r.Min.X, r.Max.Y-WindowBorder)).ImageRect(), color, out.Output.TransformMatrix())
}

func (server *Server) renderSelectionBox(out *Output, r geom.Rect[float64]) {
	r = r.Canon()
	server.renderRectBorder(out, r, ColorSelectionBox)
	server.renderer.RenderRect(r.Inset(WindowBorder).ImageRect(), ColorSelectionBackground, out.Output.TransformMatrix())
}

func (server *Server) renderSurface(out *Output, s wlr.Surface, p geom.Point[int]) {
	texture := s.GetTexture()
	if !texture.Valid() {
		wlr.Log(wlr.Error, "invalid texture for surface")
		return
	}

	r := surfaceBounds(s).Add(geom.PConv[int](p))
	tr := s.Current().Transform().Invert()
	m := wlr.ProjectBoxMatrix(r.ImageRect(), tr, 0, out.Output.TransformMatrix())

	server.renderer.RenderTextureWithMatrix(texture, m, 1)
	s.SendFrameDone(time.Now())
}

func (server *Server) renderStatusBar() {
	out := server.statusBar.Output()
	tm := out.Output.TransformMatrix()

	b := server.statusBarBounds()
	server.renderer.RenderRect(b.ImageRect(), ColorMenuBorder, tm)

	if title := server.statusBar.Title(); title.Valid() {
		tb := geom.Rt(0, 0, float64(title.Width()), float64(title.Height()))
		tb = geom.Align(b, tb, geom.EdgeLeft)
		tb = tb.Add(geom.Pt[float64](WindowBorder, 0))
		m := wlr.ProjectBoxMatrix(tb.ImageRect(), wlr.OutputTransformNormal, 0, tm)
		server.renderer.RenderTextureWithMatrix(title, m, 1)
	}
}

func (server *Server) renderMode(out *Output) {
	m, ok := server.inputMode.(Framer)
	if !ok {
		return
	}

	m.Frame(server, out)
}

func (server *Server) renderCursor(out *Output) {
	out.Output.RenderSoftwareCursors(image.ZR)
}

func (server *Server) renderMenu(out *Output, m *Menu, p geom.Point[float64], sel *MenuItem) {
	r := m.Bounds().Add(p)
	server.renderer.RenderRect(r.Inset(-WindowBorder/2).ImageRect(), ColorMenuBorder, out.Output.TransformMatrix())
	server.renderer.RenderRect(r.ImageRect(), ColorMenuUnselected, out.Output.TransformMatrix())

	for _, item := range m.items {
		ar := m.ItemBounds(item).Add(p)
		tr := geom.Rt(0, 0, float64(item.active.Width()), float64(item.active.Height())).CenterAt(ar.Center())

		t := item.inactive
		if item == sel {
			t = item.active
			server.renderer.RenderRect(ar.ImageRect(), ColorMenuSelected, out.Output.TransformMatrix())
		}

		matrix := wlr.ProjectBoxMatrix(tr.ImageRect(), wlr.OutputTransformNormal, 0, out.Output.TransformMatrix())
		server.renderer.RenderTextureWithMatrix(t, matrix, 1)
	}
}
