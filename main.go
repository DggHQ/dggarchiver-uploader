package main

import (
	"log/slog"

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

	p := platforms.New(cfg, monitor)
	p.Start()
}
