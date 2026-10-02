package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"deedles.dev/kawa/internal/bg"
	"deedles.dev/kawa/internal/kawa"
	"deedles.dev/kawa/internal/output"
	"deedles.dev/kawa/internal/xflag"
	"deedles.dev/wlr"
)

func main() {
	if addr, ok := os.LookupEnv("PPROF_ADDR"); ok {
		go func() { log.Println(http.ListenAndServe(addr, nil)) }()
	}

	wlr.InitLog(wlr.Debug, nil)

	var server kawa.Server
	terms := xflag.StringsFlag("terms", []string{"sakura", "alacritty"}, "preferentially ordered list of terminals for new windows to use")
	flag.StringVar(&server.Background, "bg", "", "background image")
	flag.TextVar(&server.BackgroundScale, "bgscale", bg.Stretch, "background image scaling method (stretch, center, fit, fill)")
	outputConfigs := flag.String("out", "", "output configs (name:x:y[:width:height][:scale][:transform])")
	flag.Parse()

	server.Terms = *terms
	server.OutputConfigs = output.Parse(*outputConfigs)

	err := server.Run(context.Background())
	if err != nil {
		wlr.Log(wlr.Error, "%v", err)
		os.Exit(1)
	}
}
