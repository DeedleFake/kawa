package main

import (
	"time"

	"deedles.dev/wlr"
)

func (server *Server) onCursorMotion(dev wlr.InputDevice, t time.Time, dx, dy float64) {
	panic("Not implemented.")
}

func (server *Server) onCursorMotionAbsolute(dev wlr.InputDevice, t time.Time, x, y float64) {
	panic("Not implemented.")
}

func (server *Server) onCursorButton(dev wlr.InputDevice, t time.Time, b uint32, state wlr.ButtonState) {
	panic("Not implemented.")
}

func (server *Server) onCursorAxis(dev wlr.InputDevice, t time.Time, source wlr.AxisSource, orient wlr.AxisOrientation, delta float64, deltaDiscrete int32) {
	panic("Not implemented.")
}

func (server *Server) onCursorFrame() {
	panic("Not implemented.")
}

func (server *Server) onNewInput(input wlr.InputDevice) {
	panic("Not implemented.")
}

func (server *Server) onRequestCursor(client wlr.SeatClient, surface wlr.Surface, serial uint32, hotspotX, hotspotY int32) {
	panic("Not implemented.")
}
