package main

import (
	"image"

	"deedles.dev/kawa/geom"
	"deedles.dev/wlr"
)

type Output struct {
	Output wlr.Output
	Layers [4][]LayerSurface
	Child  Widget

	onFrameListener wlr.Listener
}

type OutputConfig struct {
	Name          string
	X, Y          int
	Width, Height int
	Scale         float32
	Transform     wlr.OutputTransform
}

func (server *Server) outputAt(p geom.Point[float64]) *Output {
	wout := server.outputLayout.OutputAt(p.X, p.Y)
	for _, out := range server.outputs {
		if out.Output == wout {
			return out
		}
	}
	return nil
}

func (server *Server) outputBounds(out *Output) geom.Rect[float64] {
	x, y := server.outputLayout.OutputCoords(out.Output)
	return geom.Rt(0, StatusBarHeight, float64(out.Output.Width()), float64(out.Output.Height())).Add(geom.Pt(x, y))
}

func (server *Server) onNewOutput(wout wlr.Output) {
	//box := NewBox(true)
	//box.Add(NewStatusBar())
	//box.Add(NewViewer())
	box := NewCenter(NewLabel(server.renderer, image.White, "This is a test."))

	out := Output{
		Output: wout,
		Child:  box,
	}
	out.onFrameListener = wout.OnFrame(func(wout wlr.Output) {
		server.onFrame(&out)
	})
	server.addOutput(&out)

	wout.InitRender(server.allocator, server.renderer)
	wout.Commit()
	wout.CreateGlobal()
}

func (server *Server) addOutput(out *Output) {
	server.outputs = append(server.outputs, out)

	for _, config := range server.OutputConfigs {
		if config.Name != out.Output.Name() {
			continue
		}

		server.configureOutput(out, &config)
		return
	}

	server.configureOutput(out, nil)

	if server.statusBar.Bounds().IsZero() {
		server.statusBar.MoveToOutput(server, out)
	}
}

func (server *Server) configureOutput(out *Output, config *OutputConfig) {
	server.layoutOutput(out, config)
	server.setOutputMode(out, config)

	if config == nil {
		return
	}

	if config.Scale != 0 {
		out.Output.SetScale(config.Scale)
	}

	if config.Transform != 0 {
		out.Output.SetTransform(config.Transform)
	}
}

func (server *Server) layoutOutput(out *Output, config *OutputConfig) {
	if (config == nil) || (config.X == -1) && (config.Y == -1) {
		server.outputLayout.AddAuto(out.Output)
		return
	}

	server.outputLayout.Add(out.Output, config.X, config.Y)
}

func (server *Server) setOutputMode(out *Output, config *OutputConfig) {
	var set bool
	defer func() {
		if !set {
			mode := out.Output.PreferredMode()
			if mode.Valid() {
				out.Output.SetMode(mode)
			}
		}
	}()

	modes := out.Output.Modes()
	if (config == nil) || (config.Width == 0) || (config.Height == 0) || (len(modes) == 0) {
		return
	}

	for _, mode := range modes {
		if (mode.Width() == int32(config.Width)) && (mode.Height() == int32(config.Height)) {
			out.Output.SetMode(mode)
			set = true
			return
		}
	}
}
