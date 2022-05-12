package main

import (
	"image"
	"os"
	"os/exec"

	"deedles.dev/wlr"
)

const (
	MinWidth  = 128
	MinHeight = 24

	WindowBorder = 5
)

type Server struct {
	Term          []string
	OutputConfigs []OutputConfig

	display wlr.Display

	allocator    wlr.Allocator
	backend      wlr.Backend
	compositor   wlr.Compositor
	cursor       wlr.Cursor
	outputLayout wlr.OutputLayout
	renderer     wlr.Renderer
	seat         wlr.Seat
	cursorMgr    wlr.XCursorManager
	xdgShell     wlr.XDGShell
	layerShell   wlr.LayerShellV1
	xwayland     wlr.XWayland

	outputs   []*Output
	inputs    []wlr.InputDevice
	pointers  []wlr.InputDevice
	keyboards []*Keyboard
	views     []*View
	popups    []*Popup
	newViews  map[int]NewView
	hidden    []*View
	bg        wlr.Texture

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

type NewView struct {
	To        *image.Rectangle
	OnStarted func(*View)
}

func (server *Server) loadBG(path string) {
	file, err := os.Open(path)
	if err != nil {
		wlr.Log(wlr.Error, "load %q as background: %v", path, err)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		wlr.Log(wlr.Error, "decode %q as background: %v", path, err)
		return
	}

	if server.bg.Valid() {
		server.bg.Destroy()
	}
	server.bg = wlr.TextureFromImage(server.renderer, img)
	wlr.Log(wlr.Info, "loaded %q as background", path)
}

func (server *Server) exec(to *image.Rectangle) {
	cmd := exec.Command(server.Term[0], server.Term[1:]...) // TODO: Context support?
	err := cmd.Start()
	if err != nil {
		wlr.Log(wlr.Error, "start new: %v", err)
		return
	}

	server.newViews[cmd.Process.Pid] = NewView{
		To: to,
		OnStarted: func(view *View) {
			server.startBorderResizeFrom(view, wlr.EdgeNone, *to)
		},
	}
}

func (server *Server) selectMainMenu(n int) {
	if n < 0 {
		return
	}

	switch n {
	case 0: // New
		server.startNew()
	case 1: // Resize
		server.startSelectView(wlr.BtnRight, server.startResize)
	case 2: // Move
		server.startSelectView(wlr.BtnRight, server.startMove)
	case 3: // Delete
		server.startSelectView(wlr.BtnRight, func(view *View) {
			server.closeView(view)
			server.startNormal()
		})
	case 4: // Hide
		server.startSelectView(wlr.BtnRight, func(view *View) {
			server.hideView(view)
			server.startNormal()
		})
	default:
		server.unhideView(server.hidden[n-5])
	}
}
