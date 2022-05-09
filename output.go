package main

import (
	"image"
	"time"

	"deedles.dev/kawa/internal/util"
	"deedles.dev/wlr"
)

func (out *Output) onFrame(output wlr.Output) {
	server := out.Server

	_, err := output.AttachRender()
	if err != nil {
		wlr.Log(wlr.Error, "output attach render: %v", err)
		return
	}

	server.renderer.Begin(output, output.Width(), output.Height())
	server.renderer.Clear(ColorBackground)

	out.renderLayer(out.Layers[wlr.LayerShellV1LayerBackground])
	out.renderLayer(out.Layers[wlr.LayerShellV1LayerBottom])

	now := time.Now()
	for _, view := range server.views {
		if !view.XDGSurface.Mapped() {
			continue
		}
		view := view

		out.renderViewBorder(
			view,
			view.X,
			view.Y,
			view.XDGSurface.Surface().Current().Width(),
			view.XDGSurface.Surface().Current().Height(),
			false,
		)
		view.XDGSurface.ForEachSurface(func(surface wlr.Surface, sx, sy int) {
			view.renderSurface(
				surface,
				sx,
				sy,
				output,
				now,
			)
		})
	}

	//view := server.interactive.View
	switch server.inputState {
	case InputStateBorderDrag:
		panic("Not implemented.")
	case InputStateMove:
		panic("Not implemented.")
	case InputStateNewEnd, InputStateResizeEnd:
		panic("Not implemented.")
	}

	out.renderLayer(out.Layers[wlr.LayerShellV1LayerTop])

	if (server.menu.X != -1) && (server.menu.Y != -1) {
		out.renderMenu()
	}

	out.renderLayer(out.Layers[wlr.LayerShellV1LayerOverlay])

	output.RenderSoftwareCursors(image.ZR)
	server.renderer.End()
	output.Commit()
}

func (server *Server) onNewOutput(output wlr.Output) {
	output.InitRender(server.allocator, server.renderer)

	out := Output{
		Output: output,
		Server: server,
	}
	out.Frame = output.OnFrame(out.onFrame)
	server.outputs = append(server.outputs, &out)

	config, ok := util.FindFunc(server.OutputConfigs, func(c OutputConfig) bool { return c.Name == output.Name() })
	if !ok {
		if (config.X == -1) && (config.Y == -1) {
			server.outputLayout.AddAuto(output)
		} else {
			server.outputLayout.Add(output, config.X, config.Y)
		}

		var modeset bool
		if (config.Width != 0) && (config.Height != 0) && (len(output.Modes()) != 0) {
			for _, mode := range output.Modes() {
				if (mode.Width() == int32(config.Width)) && (mode.Height() == int32(config.Height)) {
					output.SetMode(mode)
					modeset = true
					break
				}
			}
		}
		if !modeset {
			mode := output.PreferredMode()
			if mode.Valid() {
				output.SetMode(mode)
			}
		}

		if config.Scale != 0 {
			output.SetScale(float32(config.Scale))
		}

		if config.Transform != 0 {
			output.SetTransform(config.Transform)
		}

		output.Enable(true)
	} else {
		mode := output.PreferredMode()
		if mode.Valid() {
			output.SetMode(mode)
		}
		output.Enable(true)
		server.outputLayout.AddAuto(output)
	}

	output.Commit()
	output.CreateGlobal()
}

func (out *Output) renderLayer(layers []LayerSurface) {
	for _, surface := range layers {
		sv1 := surface.LayerSurface
		sv1.Surface().ForEachSurface(out.renderLayerSurface)
	}
}

func (out *Output) renderLayerSurface(surface wlr.Surface, sx, sy int) {
	panic("Not implemented.")
}

func (out *Output) renderViewBorder(view *View, x, y, w, h int, selection bool) {
	panic("Not implemented.")
}

func (out *Output) renderSurface(surface wlr.Surface, sx, sy int, view *View, renderer wlr.Renderer, t time.Time) {
	panic("Not implemented.")
}

func (out *Output) renderMenu() {
	panic("Not implemented.")
}

func (view *View) renderSurface(surface wlr.Surface, sx, sy int, output wlr.Output, t time.Time) {
	panic("Not implemented.")
}
