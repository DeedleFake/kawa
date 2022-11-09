package main

import (
	"math"
	"time"

	"deedles.dev/kawa/geom"
	"deedles.dev/wlr"
	"golang.org/x/exp/slices"
)

type InputMode interface{}

type inputModeNormal struct {
	inView    bool
	prevEdges wlr.Edges
}

func (server *Server) startNormal() {
	server.setCursor("left_ptr")
	server.inputMode = &inputModeNormal{}
}

func (m *inputModeNormal) CursorMoved(server *Server, t time.Time) {
	cc := server.cursorCoords()

	view, edges, surface, sp := server.viewAt(nil, cc)
	if (edges != m.prevEdges) && !server.isViewTiled(view) {
		server.setCursor(edgeCursors[edges])
		m.prevEdges = edges
	}
	if (view == nil) && m.inView {
		server.setCursor("left_ptr")
	}
	m.inView = view != nil
	if !surface.Valid() {
		server.seat.PointerNotifyClearFocus()
		return
	}

	server.seat.PointerNotifyEnter(surface, sp.X, sp.Y)
	server.seat.PointerNotifyMotion(t, sp.X, sp.Y)
}

func (m *inputModeNormal) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	cc := server.cursorCoords()

	//out := server.outputAt(cc)
	//if out != nil {
	//	if cc.In(server.statusBar.Bounds()) {
	//		switch b {
	//		case wlr.BtnLeft:
	//			server.startMenu(server.systemMenu, b)
	//		case wlr.BtnRight:
	//			server.startMenu(server.mainMenu, b)
	//		}
	//		return
	//	}
	//}

	view, edges, surface, _ := server.viewAt(nil, cc)
	if view == nil {
		switch b {
		case wlr.BtnRight:
			server.startMenu(server.mainMenu, b)
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
			if !server.isViewTiled(view) {
				server.startBorderResize(view, edges)
			}
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
	view *View
	off  geom.Point[float64]
}

func (server *Server) startMove(view *View) {
	server.setCursor("grabbing")
	server.focusView(view, view.Surface())

	cc := server.cursorCoords()
	server.inputMode = &inputModeMove{
		view: view,
		off:  cc.Sub(view.Coords),
	}
}

func (m *inputModeMove) CursorMoved(server *Server, t time.Time) {
	cc := server.cursorCoords()

	if server.isViewTiled(m.view) {
		i, _, _, _ := server.viewIndexAt(nil, server.tiled, cc)
		if i >= 0 {
			vi := slices.Index(server.tiled, m.view)
			server.tiled[i], server.tiled[vi] = server.tiled[vi], server.tiled[i]
			server.layoutTiles(nil)
		}
		return
	}

	to := cc.Sub(m.off)

	//out := server.outputAt(cc)
	//if out != nil {
	//	sbb := server.statusBar.Bounds()
	//	sbb.Max.Y += WindowBorder
	//	if cc.In(sbb) {
	//		to.Y = m.view.Coords.Y
	//	}
	//}

	server.moveViewTo(nil, m.view, to)
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
	cur   geom.Rect[float64]
}

func (server *Server) startBorderResize(view *View, edges wlr.Edges) {
	from := view.Bounds()
	server.startBorderResizeFrom(view, edges, from)
}

func (server *Server) startBorderResizeFrom(view *View, edges wlr.Edges, from geom.Rect[float64]) {
	view.SetResizing(true)
	server.focusView(view, view.Surface())
	server.inputMode = &inputModeBorderResize{
		view:  view,
		edges: edges,
		cur:   from,
	}
}

func (m *inputModeBorderResize) CursorMoved(server *Server, t time.Time) {
	cc := server.cursorCoords()

	min := geom.Pt(
		math.Max(MinWidth, m.view.MinWidth()),
		math.Max(MinHeight, m.view.MinHeight()),
	)

	if m.edges&wlr.EdgeTop != 0 {
		m.cur.Min.Y = cc.Y
		if m.cur.Dy() < min.Y {
			m.cur.Min.Y = m.cur.Max.Y - min.Y
		}
	}
	if m.edges&wlr.EdgeBottom != 0 {
		m.cur.Max.Y = cc.Y
		if m.cur.Dy() < min.Y {
			m.cur.Max.Y = m.cur.Min.Y + min.Y
		}
	}
	if m.edges&wlr.EdgeLeft != 0 {
		m.cur.Min.X = cc.X
		if m.cur.Dx() < min.X {
			m.cur.Min.X = m.cur.Max.X - min.X
		}
	}
	if m.edges&wlr.EdgeRight != 0 {
		m.cur.Max.X = cc.X
		if m.cur.Dx() < min.X {
			m.cur.Max.X = m.cur.Min.X + min.X
		}
	}

	if cc.X < m.cur.Min.X {
		m.edges |= wlr.EdgeLeft
		m.edges &^= wlr.EdgeRight
		server.setCursor(edgeCursors[m.edges])
	}
	if cc.X > m.cur.Max.X {
		m.edges |= wlr.EdgeRight
		m.edges &^= wlr.EdgeLeft
		server.setCursor(edgeCursors[m.edges])
	}
	if cc.Y < m.cur.Min.Y {
		m.edges |= wlr.EdgeTop
		m.edges &^= wlr.EdgeBottom
		server.setCursor(edgeCursors[m.edges])
	}
	if cc.Y > m.cur.Max.Y {
		m.edges |= wlr.EdgeBottom
		m.edges &^= wlr.EdgeTop
		server.setCursor(edgeCursors[m.edges])
	}

	server.resizeViewTo(nil, m.view, m.cur)
}

func (m *inputModeBorderResize) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	m.view.SetResizing(false)
	server.startNormal()
}

func (m *inputModeBorderResize) TargetView() *View {
	return m.view
}

type inputModeMenu struct {
	m   *Menu
	p   geom.Point[float64]
	sel *MenuItem
	btn wlr.CursorButton
}

func (server *Server) startMenu(m *Menu, btn wlr.CursorButton) {
	cc := server.cursorCoords()
	ob := server.outputBounds(server.outputAt(cc)).Inset(2 * WindowBorder)

	ib := m.ItemBounds(server.mainMenu.Prev())
	if ib.IsZero() {
		ib = m.ItemBounds(m.Item(0))
	}
	mb := m.Bounds().Sub(ib.Center()).Add(cc)
	mb = mb.ClosestIn(ob)

	mode := inputModeMenu{
		m:   m,
		p:   mb.Min,
		btn: btn,
	}
	mode.CursorMoved(server, time.Now())
	server.inputMode = &mode
}

func (m *inputModeMenu) CursorMoved(server *Server, t time.Time) {
	cc := server.cursorCoords().Sub(m.p)
	m.sel = m.m.ItemAt(cc)
}

func (m *inputModeMenu) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if b != m.btn {
		return
	}

	server.startNormal()
	m.m.Select(m.sel)
}

func (m *inputModeMenu) Frame(server *Server, out *Output) {
	server.renderMenu(out, m.m, m.p, m.sel)
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

	cc := server.cursorCoords()
	view, _, _, _ := server.viewAt(nil, cc)
	if view != nil {
		m.then(view)
		return
	}
	server.startNormal()
}

type inputModeResize struct {
	view     *View
	s        geom.Point[float64]
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

	cc := server.cursorCoords()
	r := geom.Rect[float64]{Min: m.s, Max: cc}.Canon()
	if r.Dx() < math.Max(MinWidth, m.view.MinWidth()) {
		return
	}
	if r.Dy() < math.Max(MinHeight, m.view.MinHeight()) {
		return
	}

	if server.isViewTiled(m.view) {
		server.untileView(m.view, false)
	}

	server.startBorderResizeFrom(m.view, wlr.EdgeNone, r)
}

func (m *inputModeResize) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if b != wlr.BtnRight {
		server.startNormal()
		return
	}

	m.s = server.cursorCoords()
	m.resizing = true
}

func (m *inputModeResize) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if !m.resizing {
		return
	}

	server.startNormal()
}

func (m *inputModeResize) Frame(server *Server, out *Output) {
	if !m.resizing {
		return
	}

	cc := server.cursorCoords()
	r := geom.Rect[float64]{Min: m.s, Max: cc}
	server.renderSelectionBox(out.Output, r)
}

func (m *inputModeResize) TargetView() *View {
	return m.view
}

type inputModeNew struct {
	n        geom.Rect[float64]
	dragging bool
	started  bool
}

func (server *Server) startNew() {
	server.setCursor("top_left_corner")
	server.inputMode = &inputModeNew{}
}

func (m *inputModeNew) CursorMoved(server *Server, t time.Time) {
	if !m.dragging {
		return
	}

	cc := server.cursorCoords()
	m.n.Max = cc

	if math.Abs(cc.X-float64(m.n.Min.X)) < MinWidth {
		return
	}
	if math.Abs(cc.Y-float64(m.n.Min.Y)) < MinHeight {
		return
	}

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

	m.n.Min = server.cursorCoords()
	m.n.Max = m.n.Min
	m.dragging = true
}

func (m *inputModeNew) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	if !m.dragging {
		return
	}

	server.startNormal()
}

func (m *inputModeNew) Frame(server *Server, out *Output) {
	if !m.dragging || m.started {
		return
	}

	server.renderSelectionBox(out.Output, m.n)
}
