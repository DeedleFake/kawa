package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"deedles.dev/wlr"
)

func (server *Server) genMenuTextures() {
	panic("Not implemented.")
}

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

func parseOutputConfigs(outputConfigs string) (configs []OutputConfig, err error) {
	if outputConfigs == "" {
		return
	}

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
			c.Scale, _ = strconv.Atoi(parts[5])
		}
		if len(parts) >= 7 {
			c.Transform, _ = parseTransform(parts[6])
		}

		configs = append(configs, c)
	}

	return configs, nil
}

func main() {
	wlr.InitLog(wlr.Debug, nil)

	cage := flag.String("cage", "cage -d", "wrapper to use for caging windows")
	term := flag.String("term", "alacritty", "terminal to use when creating a new window")
	outputConfigs := flag.String("out", "", "output configs (name:x:y[:width:height][:scale][:transform])")
	flag.Parse()

	outputConfigsParsed, err := parseOutputConfigs(*outputConfigs)
	if err != nil {
		wlr.Log(wlr.Error, "parse output configs: %v", err)
		os.Exit(1)
	}

	server := Server{
		Cage:          strings.Fields(*cage),
		Term:          strings.Fields(*term),
		OutputConfigs: outputConfigsParsed,
	}

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

	server.newOutput = server.backend.OnNewOutput(server.onNewOutput)

	server.outputLayout = wlr.CreateOutputLayout()
	wlr.CreateXDGOutputManagerV1(server.display, server.outputLayout)

	server.cursor = wlr.CreateCursor()
	server.cursor.AttachOutputLayout(server.outputLayout)
	server.cursorMgr = wlr.CreateXCursorManager("", 24)
	server.cursorMgr.Load(1)

	for _, c := range server.OutputConfigs {
		server.cursorMgr.Load(float64(c.Scale))
	}

	server.cursorMotion = server.cursor.OnMotion(server.onCursorMotion)
	server.cursorMotionAbsolute = server.cursor.OnMotionAbsolute(server.onCursorMotionAbsolute)
	server.cursorButton = server.cursor.OnButton(server.onCursorButton)
	server.cursorAxis = server.cursor.OnAxis(server.onCursorAxis)
	server.cursorFrame = server.cursor.OnFrame(server.onCursorFrame)

	server.newInput = server.backend.OnNewInput(server.onNewInput)

	server.seat = wlr.CreateSeat(server.display, "seat0")
	server.requestCursor = server.seat.OnRequestSetCursor(server.onRequestCursor)

	server.xdgShell = wlr.CreateXDGShell(server.display)
	server.newXDGSurface = server.xdgShell.OnNewSurface(server.onNewXDGSurface)

	server.layerShell = wlr.CreateLayerShellV1(server.display)
	server.newLayerSurface = server.layerShell.OnNewSurface(server.onNewLayerSurface)

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
