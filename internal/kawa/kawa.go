package kawa

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"syscall"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

// init initializes the boilerplate necessary to get wlroots up and
// running, as well as a few other pieces of initialization.
func (server *Server) init() error {
	server.newViews = make(map[int]*geom.Rect[float64])
	server.pressed = make(map[wlr.CursorButton]struct{})

	server.display = wlr.CreateDisplay()

	var err error
	server.exited, err = newLoopPipe(server.display.EventLoop(), server.onExited)
	if err != nil {
		return err
	}
	server.canceled, err = newLoopPipe(server.display.EventLoop(), server.onCanceled)
	if err != nil {
		return err
	}

	server.backend = wlr.AutocreateBackend(server.display.EventLoop())
	if !server.backend.Valid() {
		return errors.New("failed to create backend")
	}

	server.renderer = wlr.AutocreateRenderer(server.backend)
	server.renderer.InitWLSHM(server.display)

	server.allocator = wlr.AutocreateAllocator(server.backend, server.renderer)
	if !server.allocator.Valid() {
		return errors.New("failed to create allocator")
	}

	server.compositor = wlr.CreateCompositor(server.display, 5, server.renderer)

	wlr.CreateDRM(server.display, server.renderer)
	wlr.CreateDataDeviceManager(server.display)
	wlr.CreateLinuxDMABufV1WithRenderer(server.display, 1, server.renderer)
	wlr.CreateExportDMABufV1(server.display)
	wlr.CreateScreencopyManagerV1(server.display)
	wlr.CreateDataControlManagerV1(server.display)
	wlr.CreatePrimarySelectionV1DeviceManager(server.display)
	wlr.CreateSubcompositor(server.display)
	// Clients like GTK 4 that get no presentation feedback time frames
	// from the output's refresh rate, which nested outputs don't have.
	wlr.CreatePresentation(server.display, server.backend, 2)

	wlr.CreateGammaControlManagerV1(server.display)

	server.onNewOutputListener = server.backend.OnNewOutput(server.onNewOutput)

	server.outputLayout = wlr.CreateOutputLayout(server.display)
	wlr.CreateXDGOutputManagerV1(server.display, server.outputLayout)

	server.cursor = wlr.CreateCursor()
	server.cursor.AttachOutputLayout(server.outputLayout)
	server.cursorMgr = wlr.CreateXCursorManager("", 24)
	server.cursorMgr.Load(1)

	for _, c := range server.OutputConfigs {
		server.cursorMgr.Load(float64(c.Scale))
	}

	server.onCursorMotionListener = server.cursor.OnMotion(server.onCursorMotion)
	server.onCursorMotionAbsoluteListener = server.cursor.OnMotionAbsolute(server.onCursorMotionAbsolute)
	server.onCursorButtonListener = server.cursor.OnButton(server.onCursorButton)
	server.onCursorAxisListener = server.cursor.OnAxis(server.onCursorAxis)
	server.onCursorFrameListener = server.cursor.OnFrame(server.onCursorFrame)

	server.onNewInputListener = server.backend.OnNewInput(server.onNewInput)

	server.seat = wlr.CreateSeat(server.display, "seat0")
	server.onRequestCursorListener = server.seat.OnRequestSetCursor(server.onRequestCursor)
	// Clients, Xwayland included, can only ask for the selection. It
	// doesn't change unless it's set here.
	server.onSetSelectionListener = server.seat.OnRequestSetSelection(server.seat.SetSelection)
	server.onSetPrimarySelectionListener = server.seat.OnRequestSetPrimarySelection(server.seat.SetPrimarySelection)

	server.xdgShell = wlr.CreateXDGShell(server.display, 3)
	server.onNewXDGToplevelListener = server.xdgShell.OnNewToplevel(server.onNewXDGToplevel)
	server.onNewXDGSurfaceListener = server.xdgShell.OnNewSurface(server.onNewXDGSurface)

	server.layerShell = wlr.CreateLayerShellV1(server.display, 4)
	server.onNewLayerSurfaceListener = server.layerShell.OnNewSurface(server.onNewLayerSurface)

	server.decorationManager = wlr.CreateServerDecorationManager(server.display)
	server.decorationManager.SetDefaultMode(wlr.ServerDecorationManagerModeServer)
	server.onNewDecorationListener = server.decorationManager.OnNewDecoration(server.onNewDecoration)

	server.xdgDecorationManager = wlr.CreateXDGDecorationManagerV1(server.display)
	server.onNewToplevelDecorationListener = server.xdgDecorationManager.OnNewToplevelDecoration(server.onNewToplevelDecoration)

	server.initActivation()

	server.initUI()

	server.startNormal()

	return nil
}

// Run starts the compositor and runs its event loop until the user
// logs out or ctx is done. It sets WAYLAND_DISPLAY and, if Xwayland
// started, DISPLAY in the process environment. Run may only be called
// once.
//
// It is invalid to change the struct from outside after this is called.
func (server *Server) Run(ctx context.Context) error {
	// wlroots leaves per-thread state between calls, such as the EGL
	// context that a GLES2 render pass keeps current from begin to
	// submit, so every call has to come from the same thread.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	err := server.init()
	if err != nil {
		return fmt.Errorf("init server: %w", err)
	}

	if server.Background != "" {
		server.loadBG(server.Background)
	}

	defer server.release()

	// wl_display_terminate isn't safe to call from another thread, so
	// wake the event loop and let it terminate the display itself.
	w := server.canceled.w
	wrote := make(chan struct{})
	stopCancel := context.AfterFunc(ctx, func() {
		defer close(wrote)
		syscall.Write(w, []byte{0})
	})
	defer func() {
		// release closes the pipe, so the write has to be over first.
		if !stopCancel() {
			<-wrote
		}
	}()

	server.xwayland = wlr.CreateXwayland(server.display, server.compositor, false)
	server.onNewXwaylandSurfaceListener = server.xwayland.OnNewSurface(server.onNewXwaylandSurface)
	// wlroots holds on to the seat until Xwayland is ready. Without it,
	// X clients get no selections.
	server.xwayland.SetSeat(server.seat)

	socket, err := server.display.AddSocketAuto()
	if err != nil {
		return fmt.Errorf("run server: %w", err)
	}

	err = server.backend.Start()
	if err != nil {
		return fmt.Errorf("run server: %w", err)
	}

	os.Setenv("WAYLAND_DISPLAY", socket)
	wlr.Log(wlr.Info, "Running Wayland compositor on WAYLAND_DISPLAY=%v", socket)

	if server.xwayland.Valid() {
		os.Setenv("DISPLAY", server.xwayland.Server().DisplayName())
		wlr.Log(wlr.Info, "Running Xwayland on DISPLAY=%v", server.xwayland.Server().DisplayName())
	}

	server.display.Run()

	return nil
}

// onCanceled stops the event loop once the context given to Run is
// done.
func (server *Server) onCanceled(fd uintptr, mask wlr.EventMask) {
	var buf [1]byte
	syscall.Read(int(fd), buf[:])
	server.shutdown()
}
