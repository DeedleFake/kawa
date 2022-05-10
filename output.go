package main

import "deedles.dev/wlr"

func (server *Server) outputAt(x, y float64) *Output {
	wout := server.outputLayout.OutputAt(x, y)
	for _, out := range server.outputs {
		if out.Output == wout {
			return out
		}
	}
	return nil
}

func (server *Server) onNewOutput(wout wlr.Output) {
	wout.InitRender(server.allocator, server.renderer)

	out := Output{
		Output: wout,
	}
	out.Frame = wout.OnFrame(func(wout wlr.Output) {
		server.onFrame(&out)
	})
	server.addOutput(&out)

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

	server.setOutputMode(out, nil)
}

func (server *Server) configureOutput(out *Output, config *OutputConfig) {
	server.layoutOutput(out, config)
	server.setOutputMode(out, config)

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
