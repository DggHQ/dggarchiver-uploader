package platforms

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"sync"

	config "github.com/DggHQ/dggarchiver-config/uploader"
	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
	"github.com/DggHQ/dggarchiver-uploader/monitoring"
	"github.com/DggHQ/dggarchiver-uploader/notifications"
	"github.com/DggHQ/dggarchiver-uploader/platforms/implementation"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/nats-io/nats.go"
)

type ReceivedVOD struct {
	*dggarchivermodel.VOD
	HostingPlatforms []string `json:"hosting_platforms"`
	Cleanup          bool     `json:"cleanup"`
}

type Platforms struct {
	enabledPlatforms []string
	monitor          *monitoring.Monitor
	cfg              *config.Config
	filters          map[*regexp.Regexp]string
}

func New(cfg *config.Config, monitor *monitoring.Monitor, enabledPlatforms []string) (*Platforms, error) {
	p := Platforms{
		enabledPlatforms: enabledPlatforms,
		monitor:          monitor,
		cfg:              cfg,
		filters:          make(map[*regexp.Regexp]string),
	}

	for f, b := range cfg.Filters {
		exp, err := regexp.Compile(f)
		if err != nil {
			return nil, err
		}
		p.filters[exp] = b
	}

	return &p, nil
}

func (p *Platforms) Start() {
	if _, err := p.cfg.NATS.NatsConnection.Subscribe(fmt.Sprintf("%s.upload", p.cfg.NATS.Topic), func(msg *nats.Msg) {
		rvod := ReceivedVOD{
			HostingPlatforms: []string{},
			Cleanup:          true,
		}
		err := json.Unmarshal(msg.Data, &rvod)
		if err != nil {
			slog.Error("unable to unmarshal VOD", slog.Any("err", err))
			return
		}
		vod := rvod.VOD

	filterLoop:
		for f, b := range p.filters {
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
				switch b {
				case "private":
					vod.Visibility = 2
					break filterLoop
				case "unlist":
					vod.Visibility = 1
					break filterLoop
				default:
					vod.Visibility = -1
					break filterLoop
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

		if vod.Visibility != -1 {
			platforms := []implementation.Platform{}

			for _, v := range p.enabledPlatforms {
				if len(rvod.HostingPlatforms) != 0 && !slices.Contains(rvod.HostingPlatforms, v) {
					slog.Debug("skipping platform", slog.String("platform", v))
					continue
				}

				imp, err := implementation.Map[v](p.cfg, p.monitor)
				if err != nil {
					slog.Error("unable to create a platform", slog.Any("err", err))
					continue
				}

				platforms = append(platforms, imp)
			}

			if p.cfg.ParallelUploads {
				platformsNormal := []implementation.Platform{}
				platformsParallel := []implementation.Platform{}

				for _, v := range platforms {
					if v.IsParallelable() {
						platformsParallel = append(platformsParallel, v)
						continue
					}

					platformsNormal = append(platformsNormal, v)
				}

				for _, v := range platformsNormal {
					ctx := context.Background()

					if err := v.Upload(ctx, vod); err != nil {
						slog.Error("upload error", slog.Any("err", err))
						continue
					}
				}

				if len(platformsParallel) > 0 {
					var wg sync.WaitGroup

					for _, v := range platformsParallel {
						wg.Add(1)
						go func() {
							defer wg.Done()

							ctx := context.Background()
							if err := v.Upload(ctx, vod); err != nil {
								slog.Error("upload error", slog.Any("err", err))
							}
						}()
					}

					wg.Wait()
				}
			} else {
				for _, v := range platforms {
					ctx := context.Background()

					if err := v.Upload(ctx, vod); err != nil {
						slog.Error("upload error", slog.Any("err", err))
						continue
					}
				}
			}
		}

		if !rvod.Cleanup {
			return
		}

		if err = p.cfg.NATS.NatsConnection.Publish(fmt.Sprintf("%s.cleanup", p.cfg.NATS.Topic), msg.Data); err != nil {
			slog.Error("unable to publish message",
				slog.String("id", vod.VID),
				slog.Any("err", err),
			)
			return
		}
	}); err != nil {
		slog.Error("unable to subscribe to NATS topic", slog.Any("err", err))
		os.Exit(1)
	}

	var forever chan struct{}
	<-forever
}
