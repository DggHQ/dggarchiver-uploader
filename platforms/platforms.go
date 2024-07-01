package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	config "github.com/DggHQ/dggarchiver-config/uploader"
	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
	"github.com/DggHQ/dggarchiver-uploader/monitoring"
	"github.com/DggHQ/dggarchiver-uploader/notifications"
	"github.com/DggHQ/dggarchiver-uploader/platforms/implementation"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/nats-io/nats.go"
)

type Platforms struct {
	enabledPlatforms []string
	monitor          *monitoring.Monitor
	cfg              *config.Config
	filters          []*regexp.Regexp
	filtersBehaviour string
}

func New(cfg *config.Config, monitor *monitoring.Monitor) (*Platforms, error) {
	p := Platforms{
		enabledPlatforms: []string{},
		monitor:          monitor,
		cfg:              cfg,
		filters:          []*regexp.Regexp{},
		filtersBehaviour: cfg.Filters.Behaviour,
	}

	platformsValue := reflect.ValueOf(cfg.Platforms)
	platformsFields := reflect.VisibleFields(reflect.TypeOf(cfg.Platforms))
	for _, field := range platformsFields {
		if platformsValue.FieldByName(field.Name).FieldByName("Enabled").Bool() {
			p.enabledPlatforms = append(p.enabledPlatforms, strings.ToLower(field.Name))
		}
	}

	for _, f := range cfg.Filters.List {
		exp, err := regexp.Compile(f)
		if err != nil {
			return nil, err
		}
		p.filters = append(p.filters, exp)
	}

	return &p, nil
}

func (p *Platforms) Start() {
	if _, err := p.cfg.NATS.NatsConnection.Subscribe(fmt.Sprintf("%s.upload", p.cfg.NATS.Topic), func(msg *nats.Msg) {
		vod := &dggarchivermodel.VOD{}
		err := json.Unmarshal(msg.Data, vod)
		if err != nil {
			slog.Error("unable to unmarshal VOD", slog.Any("err", err))
			return
		}

	filterLoop:
		for _, f := range p.filters {
			if f.MatchString(vod.Title) {
				slog.Info("vod filtered", slog.Any("vod", vod))
				if p.cfg.Notifications.Condition("filter") {
					errs := p.cfg.Notifications.Sender.Send(notifications.GetFilteredMessage(vod, f.String()), &types.Params{
						"title": "Filtered VOD",
					})
					for _, err := range errs {
						if err != nil {
							slog.Warn("unable to send notification", slog.Any("vod", vod), slog.Any("err", err))
						}
					}
				}
				switch p.filtersBehaviour {
				case "private":
					vod.Visibility = 2
					break filterLoop
				case "unlist":
					vod.Visibility = 1
					break filterLoop
				default:
					return
				}
			}
		}

		slog.Info("received a vod", slog.Any("vod", vod))
		if p.cfg.Notifications.Condition("receive") {
			errs := p.cfg.Notifications.Sender.Send(notifications.GetReceiveMessage(vod), &types.Params{
				"title": "Received VOD",
			})
			for _, err := range errs {
				if err != nil {
					slog.Warn("unable to send notification", slog.Any("vod", vod), slog.Any("err", err))
				}
			}
		}

		ctx := context.Background()

		for _, v := range p.enabledPlatforms {
			imp, err := implementation.Map[v](p.cfg, p.monitor)
			if err != nil {
				slog.Error("unable to create a platform", slog.Any("err", err))
				continue
			}
			if err := imp.Upload(ctx, vod); err != nil {
				slog.Error("upload error", slog.Any("err", err))
				continue
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
