package main

import (
	"flag"
	"os"
	"strings"

	"deedles.dev/wlr"
)

func (server *Server) genMenuTextures() {
	panic("Not implemented.")
}

func main() {
	cage := flag.String("cage", "cage -d", "wrapper to use for caging windows")
	term := flag.String("term", "alacritty", "terminal to use when creating a new window")
	flag.Parse()

	server := Server{
		Cage: strings.Fields(*cage),
		Term: strings.Fields(*term),
	}

	wlr.InitLog(wlr.Debug, nil)

	server.display = wlr.CreateDisplay()
	server.backend = wlr.AutocreateBackend(server.display)
	server.renderer = wlr.AutocreateRenderer(server.backend)
	server.allocator = wlr.AutocreateAllocator(server.backend, server.renderer)
	server.renderer.InitWLDisplay(server.display)

	wlr.CreateCompositor(server.display, server.renderer)
	wlr.CreateDataDeviceManager(server.display)

	wlr.CreateExportDMABufV1(server.display)
	wlr.CreateScreencopyManagerV1(server.display)
	wlr.CreateDataControlManagerV1(server.display)
	wlr.CreatePrimarySelectionV1DeviceManager(server.display)

	wlr.CreateGammaControlManagerV1(server.display)

	server.newOutput = server.backend.OnNewOutput(server.NewOutput)

	server.outputLayout = wlr.CreateOutputLayout()
	wlr.CreateXDGOutputManagerV1(server.display, server.outputLayout)

	server.cursor = wlr.CreateCursor()
	server.cursor.AttachOutputLayout(server.outputLayout)
	server.cursorMgr = wlr.CreateXCursorManager("", 24)
	server.cursorMgr.Load(1)

	for _, c := range server.outputConfigs {
		server.cursorMgr.Load(float64(c.Scale))
	}

	server.cursorMotion = server.cursor.OnMotion(server.CursorMotion)
	server.cursorMotionAbsolute = server.cursor.OnMotionAbsolute(server.CursorMotionAbsolute)
	server.cursorButton = server.cursor.OnButton(server.CursorButton)
	server.cursorAxis = server.cursor.OnAxis(server.CursorAxis)
	server.cursorFrame = server.cursor.OnFrame(server.CursorFrame)

	server.newInput = server.backend.OnNewInput(server.NewInput)

	server.seat = wlr.CreateSeat(server.display, "seat0")
	server.requestCursor = server.seat.OnRequestSetCursor(server.RequestCursor)

	server.xdgShell = wlr.CreateXDGShell(server.display)
	server.newXDGSurface = server.xdgShell.OnNewSurface(server.NewXDGSurface)

	server.layerShell = wlr.CreateLayerShellV1(server.display)
	server.newLayerSurface = server.layerShell.OnNewSurface(server.NewLayerSurface)

	server.menu.X, server.menu.Y = -1, -1
	server.genMenuTextures()
	server.inputState = InputStateNone

	socket, err := server.display.AddSocketAuto()
	if err != nil {
		server.backend.Destroy()
		os.Exit(1)
	}

	err = server.backend.Start()
	if err != nil {
		server.backend.Destroy()
		server.display.Destroy()
		os.Exit(1)
	}

	os.Setenv("WAYLAND_DISPLAY", socket)
	wlr.Log(wlr.Info, "Running Wayland compositor on WAYLAND_DISPLAY=%v", socket)
	server.display.Run()

	server.display.DestroyClients()
	server.display.Destroy()
}
