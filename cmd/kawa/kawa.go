package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// After the first signal, let another one kill kawa in case shutting
	// down gets stuck.
	context.AfterFunc(ctx, stop)

	err := server.Run(ctx)
	if err != nil {
		wlr.Log(wlr.Error, "%v", err)
		os.Exit(1)
	}
}
