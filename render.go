package main

import (
	"image"
	"image/color"
	"time"

	"deedles.dev/kawa/geom"
	"deedles.dev/kawa/ui"
	"deedles.dev/wlr"
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

	b := server.outputBounds(out)
	lc := out.Child.Layout(ui.Constraints{MaxSize: b.Size()})
	lc.Render(ui.RenderContext{
		R:   server.renderer,
		Out: out.Output,
	}, geom.Rect[float64]{Max: lc.Size}.Align(b.Center()))

	server.renderMode(out)
	server.renderCursor(out)
}

func (server *Server) renderLayer(out wlr.Output, layer wlr.LayerShellV1Layer) {
	// TODO
}

func (server *Server) renderRectBorder(out wlr.Output, r geom.Rect[float64], color color.Color) {
	server.renderer.RenderRect(geom.Rt(0, 0, WindowBorder, r.Dy()).Add(r.Min).ImageRect(), color, out.TransformMatrix())
	server.renderer.RenderRect(geom.Rt(0, 0, WindowBorder, r.Dy()).Add(geom.Pt(r.Max.X-WindowBorder, r.Min.Y)).ImageRect(), color, out.TransformMatrix())
	server.renderer.RenderRect(geom.Rt(0, 0, r.Dx(), WindowBorder).Add(r.Min).ImageRect(), color, out.TransformMatrix())
	server.renderer.RenderRect(geom.Rt(0, 0, r.Dx(), WindowBorder).Add(geom.Pt(r.Min.X, r.Max.Y-WindowBorder)).ImageRect(), color, out.TransformMatrix())
}

func (server *Server) renderSelectionBox(out wlr.Output, r geom.Rect[float64]) {
	r = r.Canon()
	server.renderRectBorder(out, r, ColorSelectionBox)
	server.renderer.RenderRect(r.Inset(WindowBorder).ImageRect(), ColorSelectionBackground, out.TransformMatrix())
}

func (server *Server) renderSurface(out wlr.Output, s wlr.Surface, p geom.Point[int]) {
	texture := s.GetTexture()
	if !texture.Valid() {
		wlr.Log(wlr.Error, "invalid texture for surface")
		return
	}

	r := surfaceBounds(s).Add(geom.PConv[int](p))
	tr := s.Current().Transform().Invert()
	m := wlr.ProjectBoxMatrix(r.ImageRect(), tr, 0, out.TransformMatrix())

	server.renderer.RenderTextureWithMatrix(texture, m, 1)
	s.SendFrameDone(time.Now())
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
