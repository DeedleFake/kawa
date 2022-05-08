package main

import (
	"deedles.dev/kawa/internal/util"
	"deedles.dev/wlr"
)

func (out *Output) onFrame(output wlr.Output) {
	panic("Not implemented.")
}

func (server *Server) onNewOutput(output wlr.Output) {
	output.InitRender(server.allocator, server.renderer)

	out := Output{
		Output: output,
		Server: server,
	}
	out.Frame = output.OnFrame(out.onFrame)
	server.outputs = append(server.outputs, out)

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
