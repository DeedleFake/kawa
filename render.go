package main

import (
	"image"
	"image/color"
	"time"

	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

type Framer interface {
	Frame(*Server, *Output, wlr.RenderPass)
}

func (server *Server) onFrame(out *Output) {
	state := wlr.NewOutputState()
	defer state.Finish()

	pass, err := out.Output.BeginRenderPass(state)
	if err != nil {
		wlr.Log(wlr.Error, "begin render pass: %v", err)
		return
	}

	pass.AddRect(
		image.Rect(0, 0, out.Output.Width(), out.Output.Height()),
		ColorBackground,
		wlr.BlendModePremultiplied,
	)
	server.renderBG(out, pass)
	server.renderLayer(out, pass, wlr.LayerShellV1LayerBackground)
	server.renderLayer(out, pass, wlr.LayerShellV1LayerBottom)
	server.renderViews(out, pass)
	server.renderNewViews(out, pass)
	server.renderLayer(out, pass, wlr.LayerShellV1LayerTop)
	server.renderLayer(out, pass, wlr.LayerShellV1LayerOverlay)
	if server.statusBar.Output() == out {
		server.renderStatusBar(out, pass)
	}
	server.renderMode(out, pass)
	server.renderCursor(out, pass)

	pass.Submit()
	out.Output.CommitState(state)
}

// toOutputLocal converts a layout-space rect to an output-local image.Rectangle.
func (server *Server) toOutputLocal(out *Output, r geom.Rect[float64]) image.Rectangle {
	ox, oy := server.outputLayout.OutputCoords(out.Output)
	return r.Sub(geom.Pt(ox, oy)).ImageRect()
}

func (server *Server) renderBG(out *Output, pass wlr.RenderPass) {
	if !server.bg.Valid() {
		return
	}

	to := server.outputTilingBounds(out)
	r := geom.RConv[float64](geom.Rt(0, 0, server.bg.Width(), server.bg.Height()))
	dst := server.toOutputLocal(out, server.bgScale(to, r))
	pass.AddTexture(
		server.bg,
		image.Rectangle{},
		dst,
		1,
		wlr.OutputTransformNormal,
		wlr.FilterBilinear,
		wlr.BlendModePremultiplied,
	)
}

func (server *Server) renderViews(out *Output, pass wlr.RenderPass) {
	for _, view := range server.tiled {
		if !view.Mapped() {
			continue
		}

		server.renderView(out, pass, view)
	}

	for _, view := range server.views {
		if !view.Mapped() {
			continue
		}

		server.renderView(out, pass, view)
	}
}

func (server *Server) renderView(out *Output, pass wlr.RenderPass, view *View) {
	if !view.CSD {
		server.renderViewBorder(out, pass, view)
	}
	server.renderViewSurfaces(out, pass, view)
}

func (server *Server) renderViewBorder(out *Output, pass wlr.RenderPass, view *View) {
	color := ColorInactiveBorder
	if view.Activated() {
		color = ColorActiveBorder
	}
	if server.targetView() == view {
		color = ColorSelectionBox
	}

	r := view.Bounds().Inset(-WindowBorder)
	server.renderRectBorder(out, pass, geom.RConv[float64](r), color)
}

func (server *Server) renderViewSurfaces(out *Output, pass wlr.RenderPass, view *View) {
	for s := range view.Surfaces() {
		p := geom.Pt(s.X, s.Y)
		server.renderSurface(out, pass, s.Surface, geom.PConv[int](view.Coords).Add(p))
	}
}

func (server *Server) renderNewViews(out *Output, pass wlr.RenderPass) {
	for _, nv := range server.newViews {
		server.renderSelectionBox(out, pass, *nv)
	}
}

func (server *Server) renderLayer(out *Output, pass wlr.RenderPass, layer wlr.LayerShellV1Layer) {
	// TODO
}

func (server *Server) renderRectBorder(out *Output, pass wlr.RenderPass, r geom.Rect[float64], color color.Color) {
	pass.AddRect(server.toOutputLocal(out, geom.Rt(0, 0, WindowBorder, r.Dy()).Add(r.Min)), color, wlr.BlendModePremultiplied)
	pass.AddRect(server.toOutputLocal(out, geom.Rt(0, 0, WindowBorder, r.Dy()).Add(geom.Pt(r.Max.X-WindowBorder, r.Min.Y))), color, wlr.BlendModePremultiplied)
	pass.AddRect(server.toOutputLocal(out, geom.Rt(0, 0, r.Dx(), WindowBorder).Add(r.Min)), color, wlr.BlendModePremultiplied)
	pass.AddRect(server.toOutputLocal(out, geom.Rt(0, 0, r.Dx(), WindowBorder).Add(geom.Pt(r.Min.X, r.Max.Y-WindowBorder))), color, wlr.BlendModePremultiplied)
}

func (server *Server) renderSelectionBox(out *Output, pass wlr.RenderPass, r geom.Rect[float64]) {
	r = r.Canon()
	server.renderRectBorder(out, pass, r, ColorSelectionBox)
	pass.AddRect(server.toOutputLocal(out, r.Inset(WindowBorder)), ColorSelectionBackground, wlr.BlendModePremultiplied)
}

func (server *Server) renderSurface(out *Output, pass wlr.RenderPass, s wlr.Surface, p geom.Point[int]) {
	texture := s.GetTexture()
	if !texture.Valid() {
		wlr.Log(wlr.Error, "invalid texture for surface")
		return
	}

	r := surfaceBounds(s).Add(geom.PConv[int](p))
	tr := s.Current().Transform().Invert()
	pass.AddTexture(
		texture,
		image.Rectangle{},
		server.toOutputLocal(out, geom.RConv[float64](r)),
		1,
		tr,
		wlr.FilterBilinear,
		wlr.BlendModePremultiplied,
	)
	s.SendFrameDone(time.Now())
}

func (server *Server) renderStatusBar(out *Output, pass wlr.RenderPass) {
	b := server.statusBarBounds()
	pass.AddRect(server.toOutputLocal(out, b), ColorMenuBorder, wlr.BlendModePremultiplied)

	if title := server.statusBar.Title(); title.Valid() {
		tb := geom.Rt(0, 0, float64(title.Width()), float64(title.Height()))
		tb = geom.Align(b, tb, geom.EdgeLeft)
		tb = tb.Add(geom.Pt[float64](WindowBorder, 0))
		pass.AddTexture(
			title,
			image.Rectangle{},
			server.toOutputLocal(out, tb),
			1,
			wlr.OutputTransformNormal,
			wlr.FilterBilinear,
			wlr.BlendModePremultiplied,
		)
	}
}

func (server *Server) renderMode(out *Output, pass wlr.RenderPass) {
	m, ok := server.inputMode.(Framer)
	if !ok {
		return
	}

	m.Frame(server, out, pass)
}

func (server *Server) renderCursor(out *Output, pass wlr.RenderPass) {
	out.Output.AddSoftwareCursorsToRenderPass(pass, image.Rectangle{})
}

func (server *Server) renderMenu(out *Output, pass wlr.RenderPass, m *Menu, p geom.Point[float64], sel *MenuItem) {
	r := m.Bounds().Add(p)
	pass.AddRect(server.toOutputLocal(out, r.Inset(-WindowBorder/2)), ColorMenuBorder, wlr.BlendModePremultiplied)
	pass.AddRect(server.toOutputLocal(out, r), ColorMenuUnselected, wlr.BlendModePremultiplied)

	for item, bounds := range m.Items() {
		ar := bounds.Add(p)
		tr := geom.Rt(0, 0, float64(item.active.Width()), float64(item.active.Height())).CenterAt(ar.Center())

		t := item.inactive
		if item == sel {
			t = item.active
			pass.AddRect(server.toOutputLocal(out, ar), ColorMenuSelected, wlr.BlendModePremultiplied)
		}

		pass.AddTexture(
			t,
			image.Rectangle{},
			server.toOutputLocal(out, tr),
			1,
			wlr.OutputTransformNormal,
			wlr.FilterBilinear,
			wlr.BlendModePremultiplied,
		)
	}
}
