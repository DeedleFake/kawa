package kawa

import (
	"time"

	"deedles.dev/wlr"
)

// This file is the seat router, the only code that sends pointer events
// to clients. Clients get them while no interaction is active. While
// one is, they get axis events and the releases that the interaction
// passes on, and nothing else.

func (server *Server) onCursorMotion(dev wlr.Pointer, t time.Time, dx, dy float64) {
	server.cursor.Move(dev.Base(), dx, dy)
	server.routeMotion(t)
}

func (server *Server) onCursorMotionAbsolute(dev wlr.Pointer, t time.Time, x, y float64) {
	server.cursor.WarpAbsolute(dev.Base(), x, y)
	server.routeMotion(t)
}

func (server *Server) routeMotion(t time.Time) {
	if server.interaction != nil {
		server.interaction.moved(server)
		return
	}

	h := server.hitAt(server.cursorCoords(), hoverPlanes)
	server.updateHoverCursor(h)
	if !h.surface.Valid() {
		server.seat.PointerNotifyClearFocus()
		return
	}

	server.seat.PointerNotifyEnter(h.surface, h.sp.X, h.sp.Y)
	server.seat.PointerNotifyMotion(t, h.sp.X, h.sp.Y)
}

func (server *Server) onCursorButton(dev wlr.Pointer, t time.Time, b wlr.CursorButton, state wlr.ButtonState) {
	switch state {
	case wlr.ButtonPressed:
		server.pressed[b] = struct{}{}
		if server.interaction != nil {
			server.interaction.pressed(server, b)
			return
		}
		if server.pressIdle(b) {
			server.seat.PointerNotifyButton(t, b, wlr.ButtonPressed)
		}

	case wlr.ButtonReleased:
		delete(server.pressed, b)
		if (server.interaction == nil) || server.interaction.released(server, b) {
			server.seat.PointerNotifyButton(t, b, wlr.ButtonReleased)
		}
	}
}

func (server *Server) onCursorAxis(dev wlr.Pointer, t time.Time, source wlr.AxisSource, orient wlr.AxisOrientation, delta float64, deltaDiscrete int32, relativeDirection wlr.AxisRelativeDirection) {
	server.seat.PointerNotifyAxis(t, orient, delta, deltaDiscrete, source, relativeDirection)
}

func (server *Server) onCursorFrame() {
	// Clients get no other events during an interaction, and a frame
	// for every motion can fill up a client's socket during a resize.
	if server.interaction != nil {
		return
	}
	server.seat.PointerNotifyFrame()
}
