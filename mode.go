package main

import (
	"time"

	"deedles.dev/wlr"
)

type InputMode interface {
	CursorMoved(*Server, time.Time)
	CursorButtonPressed(*Server, wlr.InputDevice, wlr.CursorButton, time.Time)
	CursorButtonReleased(*Server, wlr.InputDevice, wlr.CursorButton, time.Time)
}

type inputModeNormal struct{}

func (m inputModeNormal) CursorMoved(server *Server, t time.Time) {
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

func (m inputModeNormal) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	view, area, surface, _, _ := server.viewAt(nil, server.cursor.X(), server.cursor.Y())
	if view != nil {
		server.focusView(view, surface)

		switch area {
		case ViewAreaSurface:
			server.seat.PointerNotifyButton(t, b, wlr.ButtonPressed)
		default:
			// TODO: Handle clicking on the border.
		}
	}
}

func (m inputModeNormal) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	server.seat.PointerNotifyButton(t, b, wlr.ButtonReleased)
}

func (m inputModeNormal) RequestCursor(server *Server, s wlr.Surface, x, y int) {
	server.cursor.SetSurface(s, int32(x), int32(y))
}
