package main

import "github.com/swaywm/go-wlroots/wlroots"

type Server struct {
	Cage string
	Term string

	display wlroots.Display

	backend      wlroots.Backend
	cursor       wlroots.Cursor
	outputLayout wlroots.outputLayout
	renderer     wlroots.Renderer
	seat         wlroots.Seat
	cursorMgr    wlroots.XCursorManager
	xdgShell     wlroots.XDGShell
	layerShell   wlroots.LayerShellV1
}
