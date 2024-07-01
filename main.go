package main

import (
	"log/slog"
	"os"

	config "github.com/DggHQ/dggarchiver-config/uploader"
	"github.com/DggHQ/dggarchiver-uploader/monitoring"
	"github.com/DggHQ/dggarchiver-uploader/platforms"
	"github.com/DggHQ/dggarchiver-uploader/util"

	_ "github.com/DggHQ/dggarchiver-uploader/platforms/lbry"
	_ "github.com/DggHQ/dggarchiver-uploader/platforms/odysee"
	_ "github.com/DggHQ/dggarchiver-uploader/platforms/rumble"
)

func main() {
	cfg := config.New()

	monitor := monitoring.New()
	go monitor.Run()

	enabledPlatforms := util.GetEnabledPlatforms(cfg)
	slog.Info("running the uploader service", slog.Any("platforms", enabledPlatforms))

	p, err := platforms.New(cfg, monitor, enabledPlatforms)
	if err != nil {
		slog.Error("unable to initialize platforms", slog.Any("err", err))
		os.Exit(1)
	}
	p.Start()
}
