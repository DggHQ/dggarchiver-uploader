package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"time"

	config "github.com/DggHQ/dggarchiver-config/uploader"
	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
	"github.com/DggHQ/dggarchiver-uploader/monitoring"
	"github.com/DggHQ/dggarchiver-uploader/platforms/implementation"
	"github.com/DggHQ/dggarchiver-uploader/util"
	"github.com/nats-io/nats.go"
	luaLibs "github.com/vadv/gopher-lua-libs"
	lua "github.com/yuin/gopher-lua"
)

type Platforms struct {
	enabledPlatforms []string
	monitor          *monitoring.Monitor
	cfg              *config.Config
}

func New(cfg *config.Config, monitor *monitoring.Monitor) *Platforms {
	enabledPlatforms := []string{}

	platformsValue := reflect.ValueOf(cfg.Platforms)
	platformsFields := reflect.VisibleFields(reflect.TypeOf(cfg.Platforms))
	for _, field := range platformsFields {
		if platformsValue.FieldByName(field.Name).FieldByName("Enabled").Bool() {
			enabledPlatforms = append(enabledPlatforms, strings.ToLower(field.Name))
		}
	}

	return &Platforms{
		enabledPlatforms: enabledPlatforms,
		monitor:          monitor,
		cfg:              cfg,
	}
}

func (p *Platforms) Start() {
	l := lua.NewState()
	if p.cfg.Plugins.Enabled {
		luaLibs.Preload(l)
		if err := l.DoFile(p.cfg.Plugins.PathToPlugin); err != nil {
			slog.Error("unable to load lua script", slog.Any("err", err))
			os.Exit(1)
		}
	}

	if _, err := p.cfg.NATS.NatsConnection.Subscribe(fmt.Sprintf("%s.upload", p.cfg.NATS.Topic), func(msg *nats.Msg) {
		vod := &dggarchivermodel.VOD{}
		err := json.Unmarshal(msg.Data, vod)
		if err != nil {
			slog.Error("unable to unmarshal VOD", slog.Any("err", err))
			return
		}

		slog.Info("received a vod", slog.Any("vod", vod))
		if p.cfg.Plugins.Enabled {
			util.LuaCallReceiveFunction(l, vod)
		}

		ctx := context.Background()

		for _, v := range p.enabledPlatforms {
			imp, err := implementation.Map[v](p.cfg, p.monitor)
			if err != nil {
				slog.Error("unable to create a platform", slog.Any("err", err))
			}
			if err := imp.Upload(ctx, vod, l); err != nil {
				slog.Error("upload error", slog.Any("err", err))
			}

			time.Sleep(time.Second * 1)
		}
	}); err != nil {
		slog.Error("unable to subscribe to NATS topic", slog.Any("err", err))
		os.Exit(1)
	}

	var forever chan struct{}
	<-forever
}
