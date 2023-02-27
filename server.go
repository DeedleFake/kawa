package main

import (
	"image"
	"os"
	"os/exec"
	"strings"

	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

var (
	mainMenuText = []string{
		"New",
		"Resize",
		"Tile",
		"Move",
		"Close",
		"Hide",
	}

	systemMenuText = []string{
		"Log Out",
	}
)

type Server struct {
	Terms         []string
	OutputConfigs []OutputConfig

	display wlr.Display

	allocator         wlr.Allocator
	backend           wlr.Backend
	compositor        wlr.Compositor
	cursor            wlr.Cursor
	outputLayout      wlr.OutputLayout
	renderer          wlr.Renderer
	seat              wlr.Seat
	cursorMgr         wlr.XCursorManager
	xdgShell          wlr.XDGShell
	layerShell        wlr.LayerShellV1
	xwayland          wlr.XWayland
	decorationManager wlr.XDGDecorationManagerV1

	outputs     []*Output
	inputs      []wlr.InputDevice
	pointers    []wlr.Pointer
	keyboards   []*Keyboard
	views       []*View
	tiled       []*View
	hidden      []*View
	newViews    map[int]*geom.Rect[float64]
	decorations []*Decoration

	bg      wlr.Texture
	bgScale scaleFunc

	mainMenu   *Menu
	systemMenu *Menu

	statusBar *StatusBar

	inputMode InputMode

	onNewOutputListener             wlr.Listener
	onNewInputListener              wlr.Listener
	onCursorMotionListener          wlr.Listener
	onCursorMotionAbsoluteListener  wlr.Listener
	onCursorButtonListener          wlr.Listener
	onCursorAxisListener            wlr.Listener
	onCursorFrameListener           wlr.Listener
	onRequestCursorListener         wlr.Listener
	onNewXDGSurfaceListener         wlr.Listener
	onNewXWaylandSurfaceListener    wlr.Listener
	onNewLayerSurfaceListener       wlr.Listener
	onNewToplevelDecorationListener wlr.Listener
}

func (server *Server) Release() {
	server.onNewOutputListener.Destroy()
	server.onNewInputListener.Destroy()
	server.onCursorMotionListener.Destroy()
	server.onCursorMotionAbsoluteListener.Destroy()
	server.onCursorButtonListener.Destroy()
	server.onCursorAxisListener.Destroy()
	server.onCursorFrameListener.Destroy()
	server.onRequestCursorListener.Destroy()
	server.onNewXDGSurfaceListener.Destroy()
	server.onNewXWaylandSurfaceListener.Destroy()
	server.onNewLayerSurfaceListener.Destroy()
	server.onNewToplevelDecorationListener.Destroy()
}

func (server *Server) Shutdown() {
	server.display.Terminate()
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

func (server *Server) exec(to *geom.Rect[float64]) {
	for _, term := range server.Terms {
		args := strings.Fields(term)
		cmd := exec.Command(args[0], args[1:]...) // TODO: Context support?
		err := cmd.Start()
		if err != nil {
			wlr.Log(wlr.Error, "start new with %q: %v", term, err)
			continue
		}

		server.newViews[cmd.Process.Pid] = to
		return
	}

	wlr.Log(wlr.Error, "no valid terminals found for new window")
}

func (server *Server) initUI() {
	server.initMainMenu()
	server.initSystemMenu()
}

func (server *Server) initMainMenu() {
	cbs := []func(){
		server.onMainMenuNew,
		server.onMainMenuResize,
		server.onMainMenuTile,
		server.onMainMenuMove,
		server.onMainMenuClose,
		server.onMainMenuHide,
	}

	items := make([]*MenuItem, 0, len(mainMenuText))
	for i, text := range mainMenuText {
		item := NewTextMenuItem(server.renderer, text)
		item.OnSelect = cbs[i]
		items = append(items, item)
	}

	server.mainMenu = NewMenu(items...)
}

func (server *Server) onMainMenuNew() {
	server.startNew()
}

func (server *Server) onMainMenuResize() {
	server.startSelectView(wlr.BtnRight, func(view *View) {
		server.startResize(view)
	})
}

func (server *Server) onMainMenuTile() {
	server.startSelectView(wlr.BtnRight, func(view *View) {
		server.toggleViewTiling(view)
		server.startNormal()
	})
}

func (server *Server) onMainMenuMove() {
	server.startSelectView(wlr.BtnRight, func(view *View) {
		server.startMove(view)
	})
}

func (server *Server) onMainMenuClose() {
	server.startSelectView(wlr.BtnRight, func(view *View) {
		server.closeView(view)
		server.startNormal()
	})
}

func (server *Server) onMainMenuHide() {
	server.startSelectView(wlr.BtnRight, func(view *View) {
		server.hideView(view)
		server.startNormal()
	})
}

func (server *Server) initSystemMenu() {
	cbs := []func(){
		server.onSystemMenuLogOut,
	}

	items := make([]*MenuItem, 0, len(systemMenuText))
	for i, text := range systemMenuText {
		item := NewTextMenuItem(server.renderer, text)
		item.OnSelect = cbs[i]
		items = append(items, item)
	}

	server.systemMenu = NewMenu(items...)
}

func (server *Server) onSystemMenuLogOut() {
	server.Shutdown()
}
