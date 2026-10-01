package main

import (
	"encoding/binary"
	"fmt"
	"image"
	"os"
	"os/exec"
	"strings"
	"syscall"

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

	allocator            wlr.Allocator
	backend              wlr.Backend
	compositor           wlr.Compositor
	cursor               wlr.Cursor
	outputLayout         wlr.OutputLayout
	renderer             wlr.Renderer
	seat                 wlr.Seat
	cursorMgr            wlr.XCursorManager
	xdgShell             wlr.XDGShell
	layerShell           wlr.LayerShellV1
	xwayland             wlr.Xwayland
	decorationManager    wlr.ServerDecorationManager
	xdgDecorationManager wlr.XDGDecorationManagerV1

	outputs []*Output
	//inputs    []wlr.InputDevice
	pointers  []wlr.Pointer
	keyboards []*Keyboard
	views     []*View
	tiled     []*View
	hidden    []*View
	newViews  map[int]*geom.Rect[float64]

	// exited carries the pids of programs started from New, as they
	// exit, from the goroutines that wait on them to the event loop.
	exited struct {
		r, w int
		src  wlr.EventSource
	}

	bg      wlr.Texture
	bgScale scaleFunc

	mainMenu   *Menu
	systemMenu *Menu

	statusBar *StatusBar

	inputMode InputMode
	// pressed holds the pointer buttons that are currently down.
	pressed map[wlr.CursorButton]struct{}

	onNewOutputListener             wlr.Listener
	onNewInputListener              wlr.Listener
	onCursorMotionListener          wlr.Listener
	onCursorMotionAbsoluteListener  wlr.Listener
	onCursorButtonListener          wlr.Listener
	onCursorAxisListener            wlr.Listener
	onCursorFrameListener           wlr.Listener
	onRequestCursorListener         wlr.Listener
	onNewXDGToplevelListener        wlr.Listener
	onNewXDGSurfaceListener         wlr.Listener
	onNewXwaylandSurfaceListener    wlr.Listener
	onNewLayerSurfaceListener       wlr.Listener
	onNewDecorationListener         wlr.Listener
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
	server.onNewXDGToplevelListener.Destroy()
	server.onNewXDGSurfaceListener.Destroy()
	server.onNewXwaylandSurfaceListener.Destroy()
	server.onNewLayerSurfaceListener.Destroy()
	server.onNewDecorationListener.Destroy()
	server.onNewToplevelDecorationListener.Destroy()
	server.exited.src.Remove()
	syscall.Close(server.exited.r)
	syscall.Close(server.exited.w)
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

		pid := cmd.Process.Pid
		go func() {
			cmd.Wait()

			// Only the event loop can touch the server, so pass the pid
			// along to it. A write this small to a pipe is atomic.
			var b [4]byte
			binary.NativeEndian.PutUint32(b[:], uint32(pid))
			syscall.Write(server.exited.w, b[:])
		}()

		server.newViews[pid] = to
		return
	}

	wlr.Log(wlr.Error, "no valid terminals found for new window")
}

func (server *Server) initExited() error {
	var fds [2]int
	err := syscall.Pipe2(fds[:], syscall.O_CLOEXEC)
	if err != nil {
		return fmt.Errorf("create pipe: %w", err)
	}
	err = syscall.SetNonblock(fds[0], true)
	if err != nil {
		return fmt.Errorf("set pipe non-blocking: %w", err)
	}

	server.exited.r, server.exited.w = fds[0], fds[1]
	server.exited.src = server.display.EventLoop().AddFd(uintptr(fds[0]), wlr.EventReadable, server.onExited)
	return nil
}

// onExited forgets the New box of each program that has exited. One
// that never opened a window would otherwise leave its box up forever.
// A window that a program's own child opens after it exits gets placed
// like any other.
func (server *Server) onExited(fd uintptr, mask wlr.EventMask) {
	var buf [4 * 64]byte
	for {
		n, _ := syscall.Read(int(fd), buf[:])
		if n <= 0 {
			return
		}
		for b := buf[:n]; len(b) >= 4; b = b[4:] {
			delete(server.newViews, int(binary.NativeEndian.Uint32(b)))
		}
	}
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

	items := func(yield func(*MenuItem) bool) {
		for i, text := range mainMenuText {
			item := NewTextMenuItem(server.renderer, text)
			item.OnSelect = cbs[i]
			if !yield(item) {
				return
			}
		}
	}

	server.mainMenu = NewMenuFromSeq(items, len(mainMenuText))
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

	items := func(yield func(*MenuItem) bool) {
		for i, text := range systemMenuText {
			item := NewTextMenuItem(server.renderer, text)
			item.OnSelect = cbs[i]
			if !yield(item) {
				return
			}
		}
	}

	server.systemMenu = NewMenuFromSeq(items, len(systemMenuText))
}

func (server *Server) onSystemMenuLogOut() {
	server.Shutdown()
}
