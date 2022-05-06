package main

import (
	"flag"
	"strings"

	"deedles.dev/wlr"
)

func main() {
	cage := flag.String("cage", "cage -d", "wrapper to use for caging windows")
	term := flag.String("term", "alacritty", "terminal to use when creating a new window")
	flag.Parse()

	server := Server{
		Cage: strings.Fields(*cage),
		Term: strings.Fields(*term),
	}

	wlr.LogInit(wlr.Debug, nil)

	server.display = wlr.CreateDisplay()
	server.backend = wlr.AutocreateBackend(server.display)
	server.renderer = wlr.AutocreateRenderer(server.backend)
	server.renderer.InitWLDisplay(server.display)

	wlr.CreateCompositor(server.display, server.renderer)
}
