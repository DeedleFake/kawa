package main

import (
	"image"
	"os"
	"os/exec"

	"deedles.dev/kawa/geom"
	"deedles.dev/wlr"
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
)

type Server struct {
	Term          []string
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
	decorationManager wlr.ServerDecorationManager

	outputs     []*Output
	inputs      []wlr.InputDevice
	pointers    []wlr.InputDevice
	keyboards   []*Keyboard
	views       []*View
	tiled       []*View
	hidden      []*View
	newViews    map[int]NewView
	popups      []*Popup
	decorations []*Decoration
	bg          wlr.Texture

	mainMenu     *Menu
	mainMenuPrev *MenuItem

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
	onNewDecorationListener        wlr.Listener

	inputMode InputMode
}

func (server *Server) cursorCoords() geom.Point[float64] {
	return geom.Pt(server.cursor.X(), server.cursor.Y())
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
		item := NewMenuItem(
			CreateTextTexture(server.renderer, image.White, text),
			CreateTextTexture(server.renderer, image.Black, text),
		)
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
