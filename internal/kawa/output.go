package kawa

import (
	"slices"

	"deedles.dev/kawa/internal/output"
	"deedles.dev/wlr"
	"deedles.dev/ximage/geom"
)

type Output struct {
	Output wlr.Output
	Layers [4][]*LayerSurface

	// usable is the part of the output, relative to it, that is left
	// for windows once the status bar and the exclusive zones of layer
	// surfaces are taken out.
	usable geom.Rect[int]

	onFrameListener   wlr.Listener
	onDestroyListener wlr.Listener
}

var outputTransforms = [...]wlr.OutputTransform{
	output.Normal:     wlr.OutputTransformNormal,
	output.Rotate90:   wlr.OutputTransform90,
	output.Rotate180:  wlr.OutputTransform180,
	output.Rotate270:  wlr.OutputTransform270,
	output.Flipped:    wlr.OutputTransformFlipped,
	output.Flipped90:  wlr.OutputTransformFlipped90,
	output.Flipped180: wlr.OutputTransformFlipped180,
	output.Flipped270: wlr.OutputTransformFlipped270,
}

func (server *Server) outputAt(p geom.Point[float64]) *Output {
	return server.outputFor(server.outputLayout.OutputAt(p.X, p.Y))
}

func (server *Server) outputFor(wout wlr.Output) *Output {
	for _, out := range server.outputs {
		if out.Output == wout {
			return out
		}
	}
	return nil
}

func (server *Server) outputBounds(out *Output) geom.Rect[float64] {
	x, y := server.outputLayout.OutputCoords(out.Output)
	return geom.Rt(0, 0, float64(out.Output.Width()), float64(out.Output.Height())).Add(geom.Pt(x, y))
}

// outputVisibleBounds returns the part of the output that the status
// bar doesn't cover.
func (server *Server) outputVisibleBounds(out *Output) geom.Rect[float64] {
	b := server.outputBounds(out)
	if out == server.statusBar.Output() {
		return b.Pad(StatusBarHeight, 0, 0, 0)
	}
	return b
}

func (server *Server) outputUsableBounds(out *Output) geom.Rect[float64] {
	return geom.RConv[float64](out.usable).Add(server.outputBounds(out).Min)
}

func (server *Server) statusBarBounds() geom.Rect[float64] {
	b := server.outputBounds(server.statusBar.Output())
	b.Max.Y = b.Min.Y + StatusBarHeight
	return b
}

func (server *Server) onNewOutput(wout wlr.Output) {
	out := Output{
		Output: wout,
	}
	out.onFrameListener = wout.OnFrame(func(wout wlr.Output) {
		server.onFrame(&out)
	})
	out.onDestroyListener = wout.OnDestroy(func(wout wlr.Output) {
		server.onDestroyOutput(&out)
	})

	wout.InitRender(server.allocator, server.renderer)
	server.addOutput(&out)
	wout.CreateGlobal(server.display)

	switch {
	case server.statusBar == nil:
		server.statusBar = NewStatusBar(&out)
	case server.statusBar.Output() == nil:
		server.statusBar.SetOutput(&out)
	}
	server.arrangeLayers(&out)
}

func (server *Server) onDestroyOutput(out *Output) {
	out.closeLayerSurfaces()
	out.onFrameListener.Destroy()
	out.onDestroyListener.Destroy()

	i := slices.Index(server.outputs, out)
	if i >= 0 {
		server.outputs = slices.Delete(server.outputs, i, i+1)
	}

	if server.statusBar.Output() == out {
		var next *Output
		if len(server.outputs) > 0 {
			next = server.outputs[0]
		}
		server.statusBar.SetOutput(next)
		if next != nil {
			server.arrangeLayers(next)
		}
	}
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
}

func (server *Server) configureOutput(out *Output, config *output.Config) {
	state := wlr.NewOutputState()
	defer state.Finish()
	state.SetEnabled(true)

	server.setOutputMode(state, out, config)

	if config != nil {
		if config.Scale != 0 {
			state.SetScale(config.Scale)
		}
		if config.Transform != output.Normal {
			state.SetTransform(outputTransforms[config.Transform])
		}
	}

	out.Output.CommitState(state)
	server.layoutOutput(out, config)
}

func (server *Server) layoutOutput(out *Output, config *output.Config) {
	if (config == nil) || (config.X == -1) && (config.Y == -1) {
		server.outputLayout.AddAuto(out.Output)
		return
	}

	server.outputLayout.Add(out.Output, config.X, config.Y)
}

func (server *Server) setOutputMode(state wlr.OutputState, out *Output, config *output.Config) {
	if config != nil && config.Width != 0 && config.Height != 0 {
		for mode := range out.Output.Modes() {
			if (mode.Width() == int32(config.Width)) && (mode.Height() == int32(config.Height)) {
				state.SetMode(mode)
				return
			}
		}
	}

	mode := out.Output.PreferredMode()
	if mode.Valid() {
		state.SetMode(mode)
	}
}
