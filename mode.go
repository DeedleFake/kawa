package main

import (
	"image"
	"time"

	"deedles.dev/wlr"
)

type InputMode interface {
	CursorMoved(*Server, time.Time)
	CursorButtonPressed(*Server, wlr.InputDevice, wlr.CursorButton, time.Time)
	CursorButtonReleased(*Server, wlr.InputDevice, wlr.CursorButton, time.Time)
}

type inputModeNormal struct{}

func (server *Server) startNormal() {
	server.setCursor("left_ptr")
	server.inputMode = &inputModeNormal{}
}

func (m *inputModeNormal) CursorMoved(server *Server, t time.Time) {
	x, y := server.cursor.X(), server.cursor.Y()

	view, edges, surface, sx, sy := server.viewAt(nil, x, y)
	server.setCursor(edgeCursors[edges])
	if view == nil {
		server.setCursor("left_ptr")
	}
	if !surface.Valid() {
		server.seat.PointerNotifyClearFocus()
		return
	}

	focus := server.seat.PointerState().FocusedSurface() != surface
	server.seat.PointerNotifyEnter(surface, sx, sy)
	if !focus {
		server.seat.PointerNotifyMotion(t, sx, sy)
	}
}

func (m *inputModeNormal) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	view, edges, surface, _, _ := server.viewAt(nil, server.cursor.X(), server.cursor.Y())
	if view == nil {
		if b == wlr.BtnRight {
			server.startMenu(server.mainMenu)
		}
		return
	}

	server.focusView(view, surface)

	switch edges {
	case wlr.EdgeNone:
		server.seat.PointerNotifyButton(t, b, wlr.ButtonPressed)
	default:
		switch b {
		case wlr.BtnLeft:
			server.startBorderResize(view, edges)
		case wlr.BtnRight:
			server.startMove(view)
		}
	}
}

func (m *inputModeNormal) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	server.seat.PointerNotifyButton(t, b, wlr.ButtonReleased)
}

func (m *inputModeNormal) RequestCursor(server *Server, s wlr.Surface, x, y int) {
	server.cursor.SetSurface(s, int32(x), int32(y))
}

type inputModeMove struct {
	view   *View
	ox, oy float64
}

func (server *Server) startMove(view *View) {
	x, y := server.cursor.X(), server.cursor.Y()

	server.setCursor("grabbing")
	server.inputMode = &inputModeMove{
		view: view,
		ox:   x - float64(view.X),
		oy:   y - float64(view.Y),
	}
}

func (m *inputModeMove) CursorMoved(server *Server, t time.Time) {
	x, y := server.cursor.X(), server.cursor.Y()
	server.moveViewTo(nil, m.view, int(x-m.ox), int(y-m.oy))
}

func (m *inputModeMove) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	// This can't happen. Move mode is only active while the button is held down.
	panic("If you see this, there's a bug.")
}

func (m *inputModeMove) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	server.startNormal()
}

func (m *inputModeMove) TargetView() *View {
	return m.view
}

type inputModeBorderResize struct {
	view  *View
	edges wlr.Edges
	start image.Rectangle
	off   image.Point
}

func (server *Server) startBorderResize(view *View, edges wlr.Edges) {
	vb := server.viewBounds(nil, view)
	sb := server.surfaceBounds(nil, view.XDGSurface.Surface(), view.X, view.Y)

	server.inputMode = &inputModeBorderResize{
		view:  view,
		edges: edges,
		start: vb,
		off:   sb.Min.Sub(vb.Min),
	}
}

func (m *inputModeBorderResize) CursorMoved(server *Server, t time.Time) {
	x, y := server.cursor.X(), server.cursor.Y()

	r := m.start.Add(m.off)
	if m.edges&wlr.EdgeTop != 0 {
		r.Min.Y = int(y) + m.off.Y
	}
	if m.edges&wlr.EdgeBottom != 0 {
		r.Max.Y = int(y) + m.off.Y
	}
	if m.edges&wlr.EdgeLeft != 0 {
		r.Min.X = int(x) + m.off.X
	}
	if m.edges&wlr.EdgeRight != 0 {
		r.Max.X = int(x) + m.off.X
	}

	switch m.edges {
	case wlr.EdgeTop, wlr.EdgeBottom:
		if int(x) < r.Min.X {
			m.edges |= wlr.EdgeLeft
		}
		if int(x) > r.Max.X {
			m.edges |= wlr.EdgeRight
		}
	case wlr.EdgeLeft, wlr.EdgeRight:
		if int(y) < r.Min.Y {
			m.edges |= wlr.EdgeTop
		}
		if int(y) > r.Max.Y {
			m.edges |= wlr.EdgeBottom
		}
	}

	if r.Dx() < MinWidth {
		r.Max.X = r.Min.X + MinWidth
	}
	if r.Dy() < MinHeight {
		r.Max.Y = r.Min.Y + MinHeight
	}

	server.resizeViewTo(nil, m.view, r.Canon())
	server.setCursor(edgeCursors[m.edges])
}

func (m *inputModeBorderResize) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	panic("If you see this, there's a bug.")
}

func (m *inputModeBorderResize) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	server.startNormal()
}

func (m *inputModeBorderResize) TargetView() *View {
	return m.view
}

type inputModeMenu struct {
	m    *Menu
	x, y float64
}

func (server *Server) startMenu(m *Menu) {
	server.inputMode = &inputModeMenu{
		m: m,
		x: server.cursor.X(),
		y: server.cursor.Y(),
	}
}

func (m *inputModeMenu) CursorMoved(server *Server, t time.Time) {
	// Purposefully do nothing.
}

func (m *inputModeMenu) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	panic("If you see this, there's a bug.")
}

func (m *inputModeMenu) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	// TODO: Activate mode based on menu selection.
	server.startNormal()
}

func (m *inputModeMenu) Frame(server *Server, out *Output, t time.Time) {
	x, y := server.cursor.X(), server.cursor.Y()
	p := image.Pt(int(x), int(y))

	r := box(int(m.x), int(m.y), 100, 24*5)
	server.renderer.RenderRect(r.Inset(-WindowBorder), ColorMenuBorder, out.Output.TransformMatrix())
	server.renderer.RenderRect(r, ColorMenuUnselected, out.Output.TransformMatrix())

	r.Max.Y = r.Min.Y
	for i := range m.m.inactive {
		t := m.m.inactive[i]

		r.Min.Y += r.Dy()
		r.Max.Y = r.Min.Y + t.Height()

		if p.In(r) {
			server.renderer.RenderRect(r, ColorMenuSelected, out.Output.TransformMatrix())
		}

		matrix := wlr.ProjectBoxMatrix(r, wlr.OutputTransformNormal, 0, out.Output.TransformMatrix())
		server.renderer.RenderTextureWithMatrix(t, matrix, 1)
	}
}
