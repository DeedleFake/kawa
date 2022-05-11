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

	_, area, surface, sx, sy := server.viewAt(nil, x, y)
	server.setCursor(area.Cursor())
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
	view, area, surface, _, _ := server.viewAt(nil, server.cursor.X(), server.cursor.Y())
	if view != nil {
		server.focusView(view, surface)

		switch area {
		case ViewAreaNone:
			// Huh?
		case ViewAreaSurface:
			server.seat.PointerNotifyButton(t, b, wlr.ButtonPressed)
		default:
			switch b {
			case wlr.BtnLeft:
				server.startBorderResize(view, area)
			case wlr.BtnRight:
				server.startMove(view)
			}
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

type inputModeBorderResize struct {
	view  *View
	edges wlr.Edges
	start image.Rectangle
}

func (server *Server) startBorderResize(view *View, area ViewArea) {
	server.inputMode = &inputModeBorderResize{
		view:  view,
		edges: area.Edges(),
		start: server.surfaceBounds(nil, view.XDGSurface.Surface(), 0, 0),
	}
}

func (m *inputModeBorderResize) CursorMoved(server *Server, t time.Time) {
	x, y := server.cursor.X(), server.cursor.Y()

	r := m.start
	if m.edges&wlr.EdgeTop != 0 {
		r.Min.Y = int(y)
	}
	if m.edges&wlr.EdgeBottom != 0 {
		r.Max.Y = int(y)
	}
	if m.edges&wlr.EdgeLeft != 0 {
		r.Min.X = int(x)
	}
	if m.edges&wlr.EdgeRight != 0 {
		r.Max.X = int(x)
	}

	server.resizeViewTo(nil, m.view, r.Canon())
}

func (m *inputModeBorderResize) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	panic("If you see this, there's a bug.")
}

func (m *inputModeBorderResize) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	server.startNormal()
}
