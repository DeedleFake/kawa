package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"strings"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"deedles.dev/kawa/internal/util"
	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

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
	server.newViews = make(map[int]*geom.Rect[float64])

	server.display = wlr.CreateDisplay()

	server.backend = wlr.AutocreateBackend(server.display)
	if !server.backend.Valid() {
		return errors.New("failed to create backend")
	}

	server.renderer = wlr.AutocreateRenderer(server.backend)
	server.renderer.InitWLSHM(server.display)

	server.allocator = wlr.AutocreateAllocator(server.backend, server.renderer)
	if !server.allocator.Valid() {
		return errors.New("failed to create allocator")
	}

	server.compositor = wlr.CreateCompositor(server.display, server.renderer)

	wlr.CreateDRM(server.display, server.renderer)
	wlr.CreateDataDeviceManager(server.display)
	wlr.CreateLinuxDMABufV1(server.display, server.renderer)
	wlr.CreateExportDMABufV1(server.display)
	wlr.CreateScreencopyManagerV1(server.display)
	wlr.CreateDataControlManagerV1(server.display)
	wlr.CreatePrimarySelectionV1DeviceManager(server.display)
	wlr.CreateSubcompositor(server.display)

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

	server.xdgShell = wlr.CreateXDGShell(server.display, 3)
	server.onNewXDGSurfaceListener = server.xdgShell.OnNewSurface(server.onNewXDGSurface)

	server.layerShell = wlr.CreateLayerShellV1(server.display)
	server.onNewLayerSurfaceListener = server.layerShell.OnNewSurface(server.onNewLayerSurface)

	server.decorationManager = wlr.CreateServerDecorationManager(server.display)
	server.decorationManager.SetDefaultMode(wlr.ServerDecorationManagerModeServer)
	server.onNewDecorationListener = server.decorationManager.OnNewDecoration(server.onNewDecoration)

	server.xdgDecorationManager = wlr.CreateXDGDecorationManagerV1(server.display)
	server.onNewToplevelDecorationListener = server.xdgDecorationManager.OnNewToplevelDecoration(server.onNewToplevelDecoration)

	server.initUI()

	server.startNormal()

	return nil
}

// run runs the server's main loop.
func (server *Server) run() error {
	defer server.Release()

	server.xwayland = wlr.CreateXwayland(server.display, server.compositor, false)
	server.onNewXwaylandSurfaceListener = server.xwayland.OnNewSurface(server.onNewXwaylandSurface)

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

	if server.xwayland.Valid() {
		os.Setenv("DISPLAY", server.xwayland.Server().DisplayName())
		wlr.Log(wlr.Info, "Running Xwayland on DISPLAY=%v", server.xwayland.Server().DisplayName())
	}

	server.display.Run()

	return nil
}

func main() {
	if addr, ok := os.LookupEnv("PPROF_ADDR"); ok {
		go func() { log.Println(http.ListenAndServe(addr, nil)) }()
	}

	wlr.InitLog(wlr.Debug, nil)

	terms := util.StringsFlag("terms", []string{"sakura", "alacritty"}, "preferentially ordered list of terminals for new windows to use")
	bg := flag.String("bg", "", "background image")
	bgScale := flag.String("bgscale", "stretch", "background image scaling method (stretch, center, fit, fill)")
	outputConfigs := flag.String("out", "", "output configs (name:x:y[:width:height][:scale][:transform])")
	flag.Parse()

	outputConfigsParsed, err := parseOutputConfigs(*outputConfigs)
	if err != nil {
		wlr.Log(wlr.Error, "parse output configs: %v", err)
		os.Exit(1)
	}

	server := Server{
		Terms:         *terms,
		OutputConfigs: outputConfigsParsed,
	}

	err = server.init()
	if err != nil {
		wlr.Log(wlr.Error, "init server: %v", err)
		os.Exit(1)
	}

	if *bg != "" {
		server.loadBG(*bg)
		switch *bgScale {
		case "stretch":
			server.bgScale = scaleStretch
		case "center":
			server.bgScale = scaleCenter
		case "fit":
			server.bgScale = scaleFit
		case "fill":
			server.bgScale = scaleFill
		default:
			wlr.Log(wlr.Error, "unknown scaling method: %q", *bgScale)
		}
	}

	err = server.run()
	if err != nil {
		wlr.Log(wlr.Error, "run server: %v", err)
		os.Exit(1)
	}
}
