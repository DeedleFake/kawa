package main

import (
	"deedles.dev/wlr"
)

const (
	MinWidth  = 100
	MinHeight = 100
)

type Server struct {
	Cage          []string
	Term          []string
	OutputConfigs []OutputConfig

	display wlr.Display

	allocator    wlr.Allocator
	backend      wlr.Backend
	cursor       wlr.Cursor
	outputLayout wlr.OutputLayout
	renderer     wlr.Renderer
	seat         wlr.Seat
	cursorMgr    wlr.XCursorManager
	xdgShell     wlr.XDGShell
	layerShell   wlr.LayerShellV1

	outputs   []Output
	inputs    []wlr.InputDevice
	pointers  []wlr.InputDevice
	keyboards []Keyboard
	views     []View
	newViews  []NewView

	newOutput            wlr.Listener
	newInput             wlr.Listener
	cursorMotion         wlr.Listener
	cursorMotionAbsolute wlr.Listener
	cursorButton         wlr.Listener
	cursorAxis           wlr.Listener
	cursorFrame          wlr.Listener
	requestCursor        wlr.Listener

	newXDGSurface   wlr.Listener
	newLayerSurface wlr.Listener

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

type Output struct {
	Server *Server
	Output wlr.Output
	Layers [][4]LayerSurface
	Frame  wlr.Listener
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
	Area       ViewArea
	XDGSurface wlr.XDGSurface
	Server     *Server
	Map        wlr.Listener
	Destroy    wlr.Listener
}

type NewView struct {
	PID int
	Box wlr.Box
}

type Keyboard struct {
	Server    *Server
	Device    wlr.InputDevice
	Modifiers wlr.Listener
	Key       wlr.Listener
}

type LayerSurface struct {
	LayerSurface wlr.LayerSurfaceV1
	Server       *Server

	Destroy       wlr.Listener
	Map           wlr.Listener
	SurfaceCommit wlr.Listener
	OutputDestroy wlr.Listener

	Geo *wlr.Box
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

type ViewArea int

const (
	BorderTopLeft ViewArea = iota
	BorderTop
	BorderTopRight
	BorderLeft
	Surface
	BorderRight
	BorderBottomLeft
	BorderBottom
	BorderBottomRight
)
