package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"

	"deedles.dev/kawa/internal/drm"
	"deedles.dev/wlr"
	"golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

var monoFont *sfnt.Font

func init() {
	gomonoFont, err := opentype.Parse(gomono.TTF)
	if err != nil {
		panic(fmt.Errorf("parse font: %w", err))
	}

	monoFont = gomonoFont
}

func (server *Server) genMenuTextures() {
	ren := server.renderer

	gomono, err := opentype.NewFace(monoFont, &opentype.FaceOptions{
		Size: 24,
	})
	if err != nil {
		panic(fmt.Errorf("create font face: %w", err))
	}

	surf := image.NewNRGBA(image.Rect(0, 0, 128, 128))

	fdraw := font.Drawer{
		Dst:  surf,
		Src:  image.Black,
		Face: gomono,
	}

	text := []string{"New", "Resize", "Move", "Delete", "Hide"}

	for i, item := range text {
		draw.Copy(surf, image.ZP, image.Transparent, image.Transparent.Bounds(), draw.Src, nil)

		fdraw.Dot = fixed.P(0, 0)
		fdraw.DrawString(item)

		extents, _ := fdraw.BoundString(item)
		server.menu.InactiveTextures[i] = wlr.TextureFromPixels(
			ren,
			drm.FormatRGBA8888,
			uint32(surf.Stride),
			uint32((extents.Max.X-extents.Min.X)+2),
			uint32((extents.Max.Y-extents.Min.Y)+2),
			surf.Pix,
		)
	}

	fdraw.Src = image.White

	for i, item := range text {
		draw.Copy(surf, image.ZP, image.Transparent, image.Transparent.Bounds(), draw.Src, nil)

		fdraw.Dot = fixed.P(0, 0)
		fdraw.DrawString(item)

		extents, _ := fdraw.BoundString(item)
		server.menu.ActiveTextures[i] = wlr.TextureFromPixels(
			ren,
			drm.FormatRGBA8888,
			uint32(surf.Stride),
			uint32((extents.Max.X-extents.Min.X)+2),
			uint32((extents.Max.Y-extents.Min.Y)+2),
			surf.Pix,
		)
	}
}

func box(x, y, w, h int) image.Rectangle {
	return image.Rect(x, y, x+w, y+h)
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

func (server *Server) run() error {
	server.newViews = make(map[int]image.Rectangle)
	server.inputMode = &inputModeNormal{}

	server.display = wlr.CreateDisplay()
	defer server.display.Destroy()
	defer server.display.DestroyClients()

	server.backend = wlr.AutocreateBackend(server.display)
	defer server.backend.Destroy()

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

	server.genMenuTextures()

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
	server.display.Run()

	return nil
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

	err = server.run()
	if err != nil {
		wlr.Log(wlr.Error, "run server: %v", err)
		os.Exit(1)
	}
}
