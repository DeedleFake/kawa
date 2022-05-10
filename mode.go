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

	view := server.viewAt(nil, x, y)
	if view != nil {
		surface := view.XDGSurface.Surface()
		focus := server.seat.PointerState().FocusedSurface() != surface

		server.seat.PointerNotifyEnter(surface, x, y)
		if !focus {
			server.seat.PointerNotifyMotion(t, x, y)
		}
		return
	}

	// TODO: Handle moving cursor over window borders.

	server.seat.PointerClearFocus()
}

func (m inputModeNormal) CursorButtonPressed(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	//var view *View
	//var surface wlr.Surface
	//var sx, sy float64
	//var ok bool
	//if server.inputState == InputStateNone {
	//	view, surface, sx, sy, ok = server.viewAt(server.cursor.X(), server.cursor.Y())
	//}
	//if !ok {
	//	if (state == wlr.ButtonPressed) && (b != wlr.BtnRight) {
	//		server.viewEndInteractive()
	//		return
	//	}

	//	server.cursorButtonInternal(dev, t, b, state)
	//	return
	//}

	//view.focus(surface)
	//switch view.Area {
	//case ViewAreaSurface:
	//	server.seat.PointerNotifyButton(t, b, state)
	//default:
	//	switch b {
	//	case wlr.BtnRight:
	//		view.beginInteractive(
	//			surface,
	//			sx,
	//			sy,
	//			"grabbing",
	//			InputStateBorderDrag,
	//		)
	//	default:
	//		server.corner = corners[view.Area]
	//		view.beginInteractive(
	//			surface,
	//			float64(view.X),
	//			float64(view.Y),
	//			server.corner,
	//			InputStateBorderDrag,
	//		)
	//	}
	//}
}

func (m inputModeNormal) CursorButtonReleased(server *Server, dev wlr.InputDevice, b wlr.CursorButton, t time.Time) {
	// TODO
}

func (m inputModeNormal) RequestCursor(server *Server, s wlr.Surface, x, y int) {
	server.cursor.SetSurface(s, int32(x), int32(y))
}
