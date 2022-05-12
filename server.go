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
	tiled     []*View
	hidden    []*View
	bg        wlr.Texture

	mainMenu *Menu

	onNewOutputListener            wlr.Listener
	onNewInputListener             wlr.Listener
	onCursorMotionListener         wlr.Listener
	onCursorMotionAbsoluteListener wlr.Listener
	onCursorButtonListener         wlr.Listener
	onCursorAxisListener           wlr.Listener
	onCursorFrameListener          wlr.Listener
	onRequestCursorListener        wlr.Listener
	onNewXDGSurfaceListener        wlr.Listener
	onNewXWaylandSurfaceListener   wlr.Listener
	onNewLayerSurfaceListener      wlr.Listener

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
		server.startSelectView(wlr.BtnRight, func(view *View) {
			if !server.isTiled(view) {
				server.startResize(view)
			}
		})
	case 2: // Tile
		server.startSelectView(wlr.BtnRight, func(view *View) {
			defer server.startNormal()
			if server.isTiled(view) {
				server.untileView(view)
				return
			}
			server.tileView(view)
		})
	case 3: // Move
		server.startSelectView(wlr.BtnRight, func(view *View) {
			if !server.isTiled(view) {
				server.startMove(view)
			}
		})
	case 4: // Delete
		server.startSelectView(wlr.BtnRight, func(view *View) {
			server.closeView(view)
			server.startNormal()
		})
	case 5: // Hide
		server.startSelectView(wlr.BtnRight, func(view *View) {
			if !server.isTiled(view) {
				server.hideView(view)
				server.startNormal()
			}
		})
	default:
		server.unhideView(server.hidden[n-6])
	}
}
