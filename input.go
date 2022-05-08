package main

import (
	"time"

	"deedles.dev/wlr"
)

func (server *Server) CursorMotion(dev wlr.InputDevice, t time.Time, dx, dy float64) {
	panic("Not implemented.")
}

func (server *Server) CursorMotionAbsolute(dev wlr.InputDevice, t time.Time, x, y float64) {
	panic("Not implemented.")
}

func (server *Server) CursorButton(dev wlr.InputDevice, t time.Time, b uint32, state wlr.ButtonState) {
	panic("Not implemented.")
}

func (server *Server) CursorAxis(dev wlr.InputDevice, t time.Time, source wlr.AxisSource, orient wlr.AxisOrientation, delta float64, deltaDiscrete int32) {
	panic("Not implemented.")
}

func (server *Server) CursorFrame() {
	panic("Not implemented.")
}
