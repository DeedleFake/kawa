package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"os/signal"
	"runtime/pprof"
	"strconv"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"deedles.dev/wlr"
)

// box creates a new rectangle with a top-left corner at the given
// coordinates and the given width and height.
func box(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y, x+w, y+h)
}

// parseTransform parses an OutputTransform from a string.
func parseTransform(str string) (wlr.OutputTransform, error) {
	switch str {
	case "normal", "0":
		return wlr.OutputTransformNormal, nil
	case "90":
		return wlr.OutputTransform90, nil
	case "180":
		return wlr.OutputTransform180, nil
	case "270":
		return wlr.OutputTransform270, nil
	case "flipped":
		return wlr.OutputTransformFlipped, nil
	case "flipped-90":
		return wlr.OutputTransformFlipped90, nil
	case "flipped-180":
		return wlr.OutputTransformFlipped180, nil
	case "flipped-270":
		return wlr.OutputTransformFlipped270, nil
	default:
		return 0, fmt.Errorf("invalid transform: %q", str)
	}
}

// parseOutputConfigs parses an OutputConfig from a string.
func parseOutputConfigs(outputConfigs string) (configs []OutputConfig, err error) {
	if outputConfigs == "" {
		return
	}

	// TODO: Handle errors.
	for _, config := range strings.Split(outputConfigs, ",") {
		parts := strings.Split(config, ":")
		c := OutputConfig{Name: parts[0]}
		c.X, _ = strconv.Atoi(parts[1])
		c.Y, _ = strconv.Atoi(parts[2])
		if len(parts) >= 5 {
			c.Width, _ = strconv.Atoi(parts[3])
			c.Height, _ = strconv.Atoi(parts[4])
		}
		if len(parts) >= 6 {
			scale, _ := strconv.ParseFloat(parts[5], 32)
			c.Scale = float32(scale)
		}
		if len(parts) >= 7 {
			c.Transform, _ = parseTransform(parts[6])
		}

		configs = append(configs, c)
	}

	return configs, nil
}

// init initializes the boilerplate necessary to get wlroots up and
// running, as well as a few other pieces of initialization.
func (server *Server) init() error {
	server.newViews = make(map[int]NewView)

	server.display = wlr.CreateDisplay()
	//defer server.display.Destroy()
	//defer server.display.DestroyClients()

	server.backend = wlr.AutocreateBackend(server.display)
	//defer server.backend.Destroy()

	server.renderer = wlr.AutocreateRenderer(server.backend)
	server.allocator = wlr.AutocreateAllocator(server.backend, server.renderer)
	server.renderer.InitWLDisplay(server.display)

	server.compositor = wlr.CreateCompositor(server.display, server.renderer)
	wlr.CreateDataDeviceManager(server.display)

	wlr.CreateExportDMABufV1(server.display)
	wlr.CreateScreencopyManagerV1(server.display)
	wlr.CreateDataControlManagerV1(server.display)
	wlr.CreatePrimarySelectionV1DeviceManager(server.display)

	wlr.CreateGammaControlManagerV1(server.display)

	server.onNewOutputListener = server.backend.OnNewOutput(server.onNewOutput)

	server.outputLayout = wlr.CreateOutputLayout()
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

	server.xdgShell = wlr.CreateXDGShell(server.display)
	server.onNewXDGSurfaceListener = server.xdgShell.OnNewSurface(server.onNewXDGSurface)

	server.layerShell = wlr.CreateLayerShellV1(server.display)
	server.onNewLayerSurfaceListener = server.layerShell.OnNewSurface(server.onNewLayerSurface)

	server.xwayland = wlr.CreateXWayland(server.display, server.compositor, false)
	server.onNewXWaylandSurfaceListener = server.xwayland.OnNewSurface(server.onNewXWaylandSurface)

	server.decorationManager = wlr.CreateServerDecorationManager(server.display)
	server.decorationManager.SetDefaultMode(wlr.ServerDecorationManagerModeServer)
	server.onNewDecorationListener = server.decorationManager.OnNewDecoration(server.onNewDecoration)

	server.initMainMenu()

	server.startNormal()

	return nil
}

// run runs the server's main loop. It does not return unless there is
// an error, but it probably should eventually.
func (server *Server) run() error {
	socket, err := server.display.AddSocketAuto()
	if err != nil {
		return err
	}

	err = server.backend.Start()
	if err != nil {
		return err
	}

	os.Setenv("WAYLAND_DISPLAY", socket)
	wlr.Log(wlr.Info, "Running Wayland compositor on WAYLAND_DISPLAY=%v", socket)

	os.Setenv("DISPLAY", server.xwayland.Server().DisplayName())
	wlr.Log(wlr.Info, "Running XWayland on DISPLAY=%v", server.xwayland.Server().DisplayName())

	server.display.Run()

	return nil
}

// profileCPU writes profiling information to the file at path. This
// function intercepts SIGINT and will not return until that signal
// intercepted.
func profileCPU(path string) {
	defer wlr.Log(wlr.Debug, "CPU profile written to: %q", path)

	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = pprof.StartCPUProfile(f)
	if err != nil {
		panic(err)
	}
	defer pprof.StopCPUProfile()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer signal.Stop(c)
	<-c

	wlr.Log(wlr.Debug, "Writing CPU profile...")
}

func main() {
	wlr.InitLog(wlr.Debug, nil)

	term := flag.String("term", "alacritty", "terminal to use when creating a new window")
	bg := flag.String("bg", "", "background image")
	outputConfigs := flag.String("out", "", "output configs (name:x:y[:width:height][:scale][:transform])")
	cprof := flag.String("cprof", "", "cpu profile file")
	flag.Parse()

	if *cprof != "" {
		wlr.Log(wlr.Debug, "CPU profiling enabled. Send SIGINT to write profile to: %q", *cprof)
		go profileCPU(*cprof)
	}

	outputConfigsParsed, err := parseOutputConfigs(*outputConfigs)
	if err != nil {
		wlr.Log(wlr.Error, "parse output configs: %v", err)
		os.Exit(1)
	}

	server := Server{
		Term:          strings.Fields(*term),
		OutputConfigs: outputConfigsParsed,
	}

	err = server.init()
	if err != nil {
		wlr.Log(wlr.Error, "init server: %v", err)
		os.Exit(1)
	}

	if *bg != "" {
		server.loadBG(*bg)
	}

	err = server.run()
	if err != nil {
		wlr.Log(wlr.Error, "run server: %v", err)
		os.Exit(1)
	}
}
