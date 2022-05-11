package main

import (
	"image"

	"deedles.dev/wlr"
)

const (
	MinWidth  = 100
	MinHeight = 100

	WindowBorder = 5
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

	outputs   []*Output
	inputs    []wlr.InputDevice
	pointers  []wlr.InputDevice
	keyboards []*Keyboard
	views     []*View
	newViews  map[int]image.Rectangle

	mainMenu *Menu

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

	inputMode InputMode
}

type Output struct {
	Output wlr.Output
	Layers [4][]LayerSurface
	Frame  wlr.Listener
}

type OutputConfig struct {
	Name          string
	X, Y          int
	Width, Height int
	Scale         float32
	Transform     wlr.OutputTransform
}

type Keyboard struct {
	Device    wlr.InputDevice
	Modifiers wlr.Listener
	Key       wlr.Listener
}

type LayerSurface struct {
	LayerSurface wlr.LayerSurfaceV1

	Destroy       wlr.Listener
	Map           wlr.Listener
	SurfaceCommit wlr.Listener
	OutputDestroy wlr.Listener

	Geo image.Rectangle
}
