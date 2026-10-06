package kawa

import (
	"math"
	"slices"

	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

// An interaction is a use of the pointer that kawa handles itself: a
// menu, picking a window, drawing a box, or moving or resizing a
// window. While one is active, it gets the pointer instead of clients.
// It can change windows, focus, and the overlay, but only the seat
// router sends anything to clients.
type interaction interface {
	moved(server *Server)
	pressed(server *Server, b wlr.CursorButton)
	// released handles a release of b and reports whether the client
	// under the pointer gets it too.
	released(server *Server, b wlr.CursorButton) bool
}

// begin makes i the active interaction in place of whatever was active.
// view is the window that i acts on, if any. cursor is the image to
// show, or "" to keep the one that's showing.
func (server *Server) begin(i interaction, view *View, cursor string) {
	server.setCursor(cursor)
	server.interaction = i
	server.interactionView = view
	server.overlay = overlay{target: view}
}

// endInteraction gives the pointer back to clients.
func (server *Server) endInteraction() {
	server.setCursor("left_ptr")
	server.interaction = nil
	server.interactionView = nil
	server.overlay = overlay{}
	server.hover = hoverState{}
}

// pressIdle handles a press while no interaction is active, which can
// start one. It reports whether the client under the pointer gets the
// press.
func (server *Server) pressIdle(b wlr.CursorButton) bool {
	h := server.hitAt(server.cursorCoords(), pressPlanes)

	k := server.seat.GetKeyboard()
	if (k.Valid() && (k.GetModifiers()&wlr.KeyboardModifierLogo != 0)) || (h.kind == hitStatusBar) {
		switch b {
		case wlr.BtnLeft:
			server.startMenu(server.systemMenu, b)
		case wlr.BtnRight:
			server.startMenu(server.mainMenu, b)
		}
		return false
	}

	switch h.kind {
	case hitLayer:
		if h.layer.LayerSurface.Current().KeyboardInteractive() != wlr.LayerSurfaceV1KeyboardInteractivityNone {
			server.focus(h.layer)
		}
		return true
	case hitViewContent:
		server.focus(viewContent{h.view, h.surface})
		return true
	case hitViewBorder:
		server.focus(h.view)
		switch b {
		case wlr.BtnLeft:
			if !server.isViewTiled(h.view) {
				server.startBorderResize(h.view, h.edges)
			}
		case wlr.BtnRight:
			server.startMove(h.view)
		}
	case hitDesktop:
		if b == wlr.BtnRight {
			server.startMenu(server.mainMenu, b)
		}
	}
	return false
}

type moveView struct {
	view *View
	off  geom.Point[float64]
}

func (server *Server) startMove(view *View) {
	server.begin(&moveView{view: view, off: server.cursorCoords().Sub(view.Coords)}, view, "grabbing")
	server.focus(view)
}

func (m *moveView) moved(server *Server) {
	cc := server.cursorCoords()

	if server.isViewTiled(m.view) {
		if h := server.hitAt(cc, swapPlanes); h.view != nil {
			i := slices.Index(server.tiled, h.view)
			vi := slices.Index(server.tiled, m.view)
			server.tiled[i], server.tiled[vi] = server.tiled[vi], server.tiled[i]
			server.layoutTiles(nil)
		}
		return
	}

	server.moveViewTo(nil, m.view, cc.Sub(m.off))
}

func (m *moveView) pressed(server *Server, b wlr.CursorButton) {}

func (m *moveView) released(server *Server, b wlr.CursorButton) bool {
	server.endInteraction()
	// The client got the press that started a move it asked for, and
	// the seat won't send it any more of that button until it gets the
	// release too.
	return true
}

type borderResize struct {
	view  *View
	edges wlr.Edges
	cur   geom.Rect[float64]
}

func (server *Server) startBorderResize(view *View, edges wlr.Edges) {
	server.startBorderResizeFrom(view, edges, view.Bounds())
}

func (server *Server) startBorderResizeFrom(view *View, edges wlr.Edges, from geom.Rect[float64]) {
	view.SetResizing(true)
	server.begin(&borderResize{view: view, edges: edges, cur: from}, view, "")
	server.focus(view)
}

func (m *borderResize) moved(server *Server) {
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

func (m *borderResize) pressed(server *Server, b wlr.CursorButton) {}

func (m *borderResize) released(server *Server, b wlr.CursorButton) bool {
	m.view.SetResizing(false)
	server.endInteraction()
	return true
}

type openMenu struct {
	menu *Menu
	at   geom.Point[float64]
	sel  *MenuItem
	btn  wlr.CursorButton
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

	i := &openMenu{menu: m, at: mb.Min, btn: btn}
	server.begin(i, nil, "")
	server.overlay.menu = m
	server.overlay.menuAt = mb.Min
	i.moved(server)
}

func (m *openMenu) moved(server *Server) {
	m.sel = m.menu.ItemAt(server.cursorCoords().Sub(m.at))
	server.overlay.menuSel = m.sel
}

func (m *openMenu) pressed(server *Server, b wlr.CursorButton) {}

func (m *openMenu) released(server *Server, b wlr.CursorButton) bool {
	if b != m.btn {
		return false
	}

	server.endInteraction()
	m.menu.Select(m.sel)
	return false
}

type selectView struct {
	btn  wlr.CursorButton
	then func(*View)
}

func (server *Server) startSelectView(b wlr.CursorButton, then func(*View)) {
	server.begin(&selectView{btn: b, then: then}, nil, "hand1")
}

func (m *selectView) moved(server *Server) {}

func (m *selectView) pressed(server *Server, b wlr.CursorButton) {
	if b != m.btn {
		server.endInteraction()
		return
	}

	if h := server.hitAt(server.cursorCoords(), pickPlanes); h.view != nil {
		m.then(h.view)
		return
	}
	server.endInteraction()
}

func (m *selectView) released(server *Server, b wlr.CursorButton) bool {
	return false
}

// resizeBox is the box that the Resize menu item has the user draw for
// the window that it picked.
type resizeBox struct {
	view    *View
	start   geom.Point[float64]
	drawing bool
}

func (server *Server) startResize(view *View) {
	server.begin(&resizeBox{view: view}, view, "top_left_corner")
}

func (m *resizeBox) moved(server *Server) {
	if !m.drawing {
		return
	}

	r := geom.Rect[float64]{Min: m.start, Max: server.cursorCoords()}
	server.overlay.box = r
	r = r.Canon()
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

func (m *resizeBox) pressed(server *Server, b wlr.CursorButton) {
	if b != wlr.BtnRight {
		server.endInteraction()
		return
	}

	m.start = server.cursorCoords()
	m.drawing = true
}

func (m *resizeBox) released(server *Server, b wlr.CursorButton) bool {
	if m.drawing {
		server.endInteraction()
	}
	return false
}

// newBox is the box that the New menu item has the user draw for the
// window of the program that it starts.
type newBox struct {
	box geom.Rect[float64]
	// area is the usable area of the output that the drag started on.
	// The box stays inside of it.
	area    geom.Rect[float64]
	drawing bool
	started bool
	// pid is the program that got the box once it was big enough.
	pid int
}

func (server *Server) startNew() {
	server.begin(&newBox{}, nil, "top_left_corner")
}

func (m *newBox) moved(server *Server) {
	if !m.drawing {
		return
	}

	cc := m.clamp(server.cursorCoords())
	m.box.Max = cc
	m.show(server)

	if math.Abs(cc.X-m.box.Min.X) < MinWidth {
		return
	}
	if math.Abs(cc.Y-m.box.Min.Y) < MinHeight {
		return
	}

	if !m.started {
		m.pid = server.exec(m.box)
		m.started = true
		m.show(server)
	}
}

func (m *newBox) pressed(server *Server, b wlr.CursorButton) {
	if b != wlr.BtnRight {
		server.endInteraction()
		return
	}

	cc := server.cursorCoords()
	if out := server.outputAt(cc); out != nil {
		m.area = server.outputUsableBounds(out)
	}
	m.box.Min = m.clamp(cc)
	m.box.Max = m.box.Min
	m.drawing = true
	m.show(server)
}

func (m *newBox) released(server *Server, b wlr.CursorButton) bool {
	if m.drawing {
		server.endInteraction()
	}
	return false
}

// show puts the box where it gets drawn: in the overlay until the
// program starts, and then as the program's New box for as long as it
// still has one.
func (m *newBox) show(server *Server) {
	if !m.started {
		server.overlay.box = m.box
		return
	}

	server.overlay.box = geom.Rect[float64]{}
	if _, ok := server.newViews[m.pid]; ok {
		server.newViews[m.pid] = m.box
	}
}

func (m *newBox) clamp(p geom.Point[float64]) geom.Point[float64] {
	if m.area.IsZero() {
		return p
	}
	return geom.Max(m.area.Min, geom.Min(p, m.area.Max))
}
