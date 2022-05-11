package main

import (
	"image"
	"math"
	"time"

	"deedles.dev/wlr"
)

type InputMode interface{}

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
	server.setCursor("grabbing")
	server.focusView(view, view.XDGSurface.Surface())

	x, y := server.cursor.X(), server.cursor.Y()
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

func (m *inputModeMove) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	server.startNormal()
}

func (m *inputModeMove) TargetView() *View {
	return m.view
}

type inputModeBorderResize struct {
	view  *View
	edges wlr.Edges
	cur   image.Rectangle
}

func (server *Server) startBorderResize(view *View, edges wlr.Edges) {
	vb := server.viewBounds(nil, view)
	server.startBorderResizeFrom(view, edges, vb)
}

func (server *Server) startBorderResizeFrom(view *View, edges wlr.Edges, from image.Rectangle) {
	server.focusView(view, view.XDGSurface.Surface())
	server.inputMode = &inputModeBorderResize{
		view:  view,
		edges: edges,
		cur:   from,
	}
}

func (m *inputModeBorderResize) CursorMoved(server *Server, t time.Time) {
	x, y := server.cursor.X(), server.cursor.Y()
	ox, oy := int(x), int(y)

	r := m.cur
	if m.edges&wlr.EdgeTop != 0 {
		r.Min.Y = oy
		if r.Dy() < MinHeight {
			r.Min.Y = r.Max.Y - MinHeight
		}
	}
	if m.edges&wlr.EdgeBottom != 0 {
		r.Max.Y = oy
		if r.Dy() < MinHeight {
			r.Max.Y = r.Min.Y + MinHeight
		}
	}
	if m.edges&wlr.EdgeLeft != 0 {
		r.Min.X = ox
		if r.Dx() < MinWidth {
			r.Min.X = r.Max.X - MinWidth
		}
	}
	if m.edges&wlr.EdgeRight != 0 {
		r.Max.X = ox
		if r.Dx() < MinWidth {
			r.Max.X = r.Min.X + MinWidth
		}
	}

	if ox < r.Min.X {
		m.edges |= wlr.EdgeLeft
		m.edges &^= wlr.EdgeRight
	}
	if ox > r.Max.X {
		m.edges |= wlr.EdgeRight
		m.edges &^= wlr.EdgeLeft
	}
	if oy < r.Min.Y {
		m.edges |= wlr.EdgeTop
		m.edges &^= wlr.EdgeBottom
	}
	if oy > r.Max.Y {
		m.edges |= wlr.EdgeBottom
		m.edges &^= wlr.EdgeTop
	}

	m.cur = r
	server.resizeViewTo(nil, m.view, r.Canon())
	server.setCursor(edgeCursors[m.edges])
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
	sel  int
}

func (server *Server) startMenu(m *Menu) {
	off := m.StartOffset()
	server.inputMode = &inputModeMenu{
		m: m,
		x: server.cursor.X() + float64(off.X),
		y: server.cursor.Y() + float64(off.Y),
	}
}

func (m *inputModeMenu) CursorMoved(server *Server, t time.Time) {
	cx, cy := server.cursor.X(), server.cursor.Y()

	p := image.Pt(int(cx), int(cy))
	r := m.m.Bounds().Add(image.Pt(int(m.x), int(m.y)))

	m.sel = -1
	if p.In(r) {
		m.sel = (p.Y - r.Min.Y) / int(fontOptions.Size+WindowBorder*2)
	}
}

func (m *inputModeMenu) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if b != wlr.BtnRight {
		return
	}

	server.startNormal()
	m.m.Select(m.sel)
}

func (m *inputModeMenu) Frame(server *Server, out *Output, t time.Time) {
	server.renderMenu(out, m.m, m.x, m.y, m.sel)
}

type inputModeSelectView struct {
	startBtn wlr.CursorButton
	then     func(*View)
}

func (server *Server) startSelectView(b wlr.CursorButton, then func(*View)) {
	server.setCursor("hand1")
	server.inputMode = &inputModeSelectView{
		startBtn: b,
		then:     then,
	}
}

func (m *inputModeSelectView) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if b != m.startBtn {
		server.startNormal()
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	view, _, _, _, _ := server.viewAt(nil, x, y)
	if view != nil {
		m.then(view)
		return
	}
	server.startNormal()
}

type inputModeResize struct {
	view     *View
	sx, sy   float64
	resizing bool
}

func (server *Server) startResize(view *View) {
	server.setCursor("top_left_corner")
	server.inputMode = &inputModeResize{
		view: view,
	}
}

func (m *inputModeResize) CursorMoved(server *Server, t time.Time) {
	if !m.resizing {
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	if math.Abs(x-m.sx) < MinWidth {
		return
	}
	if math.Abs(y-m.sy) < MinHeight {
		return
	}

	r := image.Rect(
		int(m.sx),
		int(m.sy),
		int(x),
		int(y),
	)
	server.startBorderResizeFrom(m.view, wlr.EdgeNone, r)
}

func (m *inputModeResize) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if b != wlr.BtnRight {
		server.startNormal()
		return
	}

	m.sx, m.sy = server.cursor.X(), server.cursor.Y()
	m.resizing = true
}

func (m *inputModeResize) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if !m.resizing {
		return
	}

	server.startNormal()
}

func (m *inputModeResize) Frame(server *Server, out *Output, t time.Time) {
	if !m.resizing {
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	r := image.Rect(
		int(m.sx),
		int(m.sy),
		int(x),
		int(y),
	).Canon()
	server.renderSelectionBox(out, r, t)
}

func (m *inputModeResize) TargetView() *View {
	return m.view
}

type inputModeNew struct {
	n        image.Rectangle
	starting bool
	started  bool
}

func (server *Server) startNew() {
	server.setCursor("top_left_corner")
	server.inputMode = &inputModeNew{}
}

func (m *inputModeNew) CursorMoved(server *Server, t time.Time) {
	if !m.starting {
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	if math.Abs(x-float64(m.n.Min.X)) < MinWidth {
		return
	}
	if math.Abs(y-float64(m.n.Min.Y)) < MinHeight {
		return
	}

	m.n.Max.X = int(x)
	m.n.Max.Y = int(y)

	if !m.started {
		server.exec(&m.n)
		m.started = true
	}
}

func (m *inputModeNew) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if b != wlr.BtnRight {
		server.startNormal()
		return
	}

	m.n.Min.X, m.n.Min.Y = int(server.cursor.X()), int(server.cursor.Y())
	m.starting = true
}

func (m *inputModeNew) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if !m.starting {
		return
	}

	server.startNormal()
}

func (m *inputModeNew) Frame(server *Server, out *Output, t time.Time) {
	if !m.starting {
		return
	}

	x, y := server.cursor.X(), server.cursor.Y()
	r := image.Rect(
		int(m.n.Min.X),
		int(m.n.Min.Y),
		int(x),
		int(y),
	)
	server.renderSelectionBox(out, r, t)
}
