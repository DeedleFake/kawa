package main

import "deedles.dev/wlr"

type Server struct {
	Cage string
	Term string

	display wlr.Display

	backend      wlr.Backend
	cursor       wlr.Cursor
	outputLayout wlr.OutputLayout
	renderer     wlr.Renderer
	seat         wlr.Seat
	cursorMgr    wlr.XCursorManager
	xdgShell     wlr.XDGShell
	layerShell   wlr.LayerShellV1

	outputs       []wlr.Output
	outputConfigs []OutputConfig
	inputs        []wlr.InputDevice
	pointers      []wlr.InputDevice
	keyboards     []wlr.Keyboard
	views         []View
	newViews      []NewView

	newOutput            func(wlr.Output)
	newInput             func(wlr.InputDevice)
	cursorMotion         func(wlr.EventPointerMotion)
	cursorMotionAbsolute func(wlr.EventPointerMotionAbsolute)
	cursorButton         func(wlr.EventPointerButton)
	cursorAxis           func(wlr.EventPointerAxis)
	cursorFrame          func(wlr.EventPointerFrame)
	requestCursor        func(wlr.SeatPointerRequestSetCursorEvent) // What the heck? Is that correct?

	menu struct {
		X, Y             int
		Width, Height    int
		ActiveTextures   [5]wlr.Texture
		InactiveTextures [5]wlr.Texture
		Selected         int
	}

	interactive struct {
		SX, SY int
		View   View
	}

	inputState InputState
}

type OutputConfig struct {
	Name          string
	X, Y          int
	Width, Height int
	Scale         int
	Transform     wlr.OutputTransform
}

type View struct {
	X, Y       int
	XDGSurface *wlr.XDGSurface
	Server     *Server
	Map        func()
	Destroy    func()
}

type NewView struct {
	PID int
	Box wlr.Box
}

type InputState uint

const (
	InputStateNone InputState = iota
	InputStateMenu
	InputStateNewStart
	InputStateNewEnd
	InputStateMoveSelect
	InputStateMove
	InputStateResizeSelect
	InputStateResizeStart
	InputStateResizeEnd
	InputStateBorderDragTop
	InputStateBorderDragRight
	InputStateBorderDragBottom
	InputStateBorderDragLeft
	InputStateDeleteSelect
	InputStateHideSelect
)
