package main

import (
	"log/slog"
	"os"

	config "github.com/DggHQ/dggarchiver-config/uploader"
	"github.com/DggHQ/dggarchiver-uploader/monitoring"
	"github.com/DggHQ/dggarchiver-uploader/platforms"

	_ "github.com/DggHQ/dggarchiver-uploader/platforms/lbry"
	_ "github.com/DggHQ/dggarchiver-uploader/platforms/rumble"
)

func main() {
	cfg := config.New()

	monitor := monitoring.New()
	go monitor.Run()

	slog.Info("running the uploader service")

	p, err := platforms.New(cfg, monitor)
	if err != nil {
		slog.Error("unable to initialize platforms", slog.Any("err", err))
		os.Exit(1)
	}
	p.Start()
}
