package odysee

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
	"github.com/DggHQ/dggarchiver-uploader/notifications"
	"github.com/DggHQ/dggarchiver-uploader/platforms/lbry"
	"github.com/DggHQ/dggarchiver-uploader/util"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/eventials/go-tus"
)

const (
	maxFileSize int64 = 16000000000 // 16 gigs is max file size on odysee
)

var (
	ErrFileTooLarge              = errors.New("file too large")
	ErrUnableToCreateUploadToken = errors.New("unable to create upload token")
	ErrUnableToCreateStream      = errors.New("unable to create stream")
)

type publishResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type publish struct {
	Output   output
	URI      string
	ClaimID  string
	Outpoint string
}

type output struct {
	Type           string `json:"type"`
	Nout           int    `json:"nout"`
	ClaimID        string `json:"claim_id"`
	Name           string `json:"name"`
	NormalizedName string `json:"normalized_name"`
	PermanentURL   string `json:"permanent_url"`
	Channel        struct {
		Name string `json:"name"`
	} `json:"signing_channel"`
}

type odyseeOutputResponse struct {
	*RPC
	Result struct {
		TxID    string   `json:"txid"`
		Outputs []output `json:"outputs"`
	} `json:"result"`
}

type RPC struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	ID      int64  `json:"id"`
}

type odyseeStreamCreateParams struct {
	Name         string   `json:"name"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Locations    []string `json:"locations"`
	BID          string   `json:"bid"`
	Languages    []string `json:"languages"`
	ThumbnailURL string   `json:"thumbnail_url"`
	ReleaseTime  int64    `json:"release_time"`
	Blocking     bool     `json:"blocking"`
	Preview      bool     `json:"preview"`
	ChannelID    string   `json:"channel_id"`
	License      string   `json:"license"`
	Tags         []string `json:"tags"`
	FilePath     string   `json:"file_path"`
}

type odyseeStreamCreate struct {
	Params odyseeStreamCreateParams `json:"params"`
	*RPC
}

type odyseeStreamCreateResponse struct {
	Status  string `json:"status"`
	Payload struct {
		QueryID string `json:"query_id"`
	} `json:"payload"`
}

type odyseeUploadTokenResponse struct {
	Status  string `json:"status"`
	Payload struct {
		Token    string `json:"token"`
		Location string `json:"location"`
	} `json:"payload"`
}

func (p *Platform) Upload(ctx context.Context, vod *dggarchivermodel.VOD) error {
	var err error

	slogVodGroup := slog.Group("vod",
		slog.String("platform", vod.Platform),
		slog.String("id", vod.VID),
	)

	if vod.EndTime == "" {
		slog.Info("calculating endtime based on the duration", slog.String("platform", platformName), slogVodGroup)
		vod.EndTime, err = util.CalculateEndTime(vod.StartTime, vod.Duration)
		if err != nil {
			return err
		}
	}

	f, err := os.Open(vod.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	if fi.Size() > maxFileSize {
		return ErrFileTooLarge
	}

	slog.Debug("checking login status", slog.String("platform", platformName), slogVodGroup)
	loggedIn, err := p.IsLoggedIn(ctx)
	if err != nil {
		return err
	}

	if !loggedIn {
		slog.Debug("trying to log in", slog.String("platform", platformName), slogVodGroup)
		if err := p.Login(ctx); err != nil {
			return err
		}
	}

	tusClient, err := p.getUpload(ctx)
	if err != nil {
		return err
	}
	slog.Debug("created the upload client", slog.String("platform", platformName), slog.String("url", tusClient.Url), slog.Any("headers", tusClient.Header), slogVodGroup)

	tusUpload, err := tus.NewUploadFromFile(f)
	if err != nil {
		return err
	}
	slog.Debug("created upload", slog.String("platform", platformName), slogVodGroup)

	tusUploader, err := tusClient.CreateUpload(tusUpload)
	if err != nil {
		return err
	}
	slog.Debug("created uploader", slog.String("platform", platformName), slogVodGroup)

	progessChan := make(chan tus.Upload)
	tusUploader.NotifyUploadProgress(progessChan)
	go func() {
		for u := range progessChan {
			progress := u.Progress()

			slog.Info("progress",
				slog.Int64("offset", u.Offset()),
				slog.Int64("size", u.Size()),
				slog.Int64("percent", progress),
				slog.String("file", fi.Name()),
			)

			if p.cfg.Notifications.Condition("progress") {
				errs := p.cfg.Notifications.Sender.Send(notifications.GetProgressMessage(vod, platformName, float64(progress)), &types.Params{
					"title": "Uploading VOD...",
				})
				for _, err := range errs {
					if err != nil {
						slog.Warn("unable to send notification", slog.Any("vod", vod), slog.Any("err", err))
					}
				}
			}
		}
	}()

	slog.Info("starting to upload", slog.String("platform", platformName), slogVodGroup)
	err = tusUploader.Upload()
	if err != nil {
		return err
	}

	slog.Debug("uploading thumbnail", slog.String("platform", platformName), slogVodGroup)
	thumbnail, err := lbry.UploadThumbnail(vod.ThumbnailPath)
	if err != nil {
		slog.Warn("unable to upload thumbnail, skipping", slog.String("platform", platformName), slogVodGroup)
		thumbnail = vod.Thumbnail
	} else {
		slog.Debug("thumbnail uploaded",
			slog.String("platform", platformName),
			slogVodGroup,
			slog.String("thumbnail", thumbnail),
		)
	}

	slog.Info("VOD uploaded", slog.String("platform", platformName), slogVodGroup)

	queryURL, err := p.createQuery(ctx, vod, tusUploader.Url(), thumbnail)
	if err != nil {
		return err
	}
	slog.Debug("created stream query", slog.String("platform", platformName), slogVodGroup)

	pub, err := p.waitForConfirm(ctx, queryURL)
	if err != nil {
		return err
	}
	slog.Debug("stream confirmed", slog.String("platform", platformName), slogVodGroup)

	err = p.publish(ctx, pub)
	if err != nil {
		return err
	}
	slog.Info("stream published", slog.String("platform", platformName), slogVodGroup)

	uvod := &dggarchivermodel.UploadedVOD{
		HostingPlatform:       platformName,
		VOD:                   *vod,
		HostingChannel:        pub.Output.Channel.Name,
		HostingName:           pub.Output.Name,
		HostingNormalizedName: pub.Output.NormalizedName,
		HostingURL:            pub.Output.PermanentURL,
	}
	err = p.cfg.SQLite.DB.Create(uvod).Error
	if err != nil {
		return err
	}

	if p.cfg.Notifications.Condition("insert") {
		errs := p.cfg.Notifications.Sender.Send(notifications.GetInsertMessage(uvod), &types.Params{
			"title": fmt.Sprintf("Uploaded VOD to %s", platformName),
		})
		for _, err := range errs {
			if err != nil {
				slog.Warn("unable to send notification", slog.Any("vod", vod), slog.Any("err", err))
			}
		}
	}

	return nil
}

func (p *Platform) getUpload(ctx context.Context) (*tus.Client, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", backendAPIURLString+"/v1/asynqueries/uploads/", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Lbry-Auth-Token", p.authToken.Get())

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrStatusCode
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var uploadToken odyseeUploadTokenResponse
	err = json.Unmarshal(b, &uploadToken)
	if err != nil {
		return nil, err
	}

	if uploadToken.Status != "upload_token_created" {
		return nil, ErrUnableToCreateUploadToken
	}

	h := http.Header{}
	h.Set("Authorization", fmt.Sprintf("Bearer %s", uploadToken.Payload.Token))
	dc := tus.DefaultConfig()
	dc.Header = h
	tusClient, err := tus.NewClient(uploadToken.Payload.Location, dc)
	if err != nil {
		return nil, err
	}

	return tusClient, nil
}

func (p *Platform) createQuery(ctx context.Context, vod *dggarchivermodel.VOD, path, thumbnail string) (string, error) {
	stream := odyseeStreamCreate{
		RPC: &RPC{},
		Params: odyseeStreamCreateParams{
			Name:        fmt.Sprintf("%s-r-%s%d", vod.VID, vod.Platform, rand.Intn(1000)),
			Title:       fmt.Sprintf("[%s:%s] %s", vod.Platform, vod.VID, vod.Title),
			Description: fmt.Sprintf("%s\n%s", vod.StartTime, vod.EndTime),
			Locations:   []string{},
			BID:         "0.0001",
			Languages: []string{
				"en",
			},
			ThumbnailURL: thumbnail,
			ReleaseTime:  time.Now().Unix(),
			Blocking:     true,
			Preview:      false,
			ChannelID:    p.cfg.Platforms.Odysee.ChannelID,
			License:      "None",
			Tags:         vod.Tags,
			FilePath:     path,
		},
	}

	if vod.Visibility == 1 || vod.Visibility == 2 {
		stream.Params.Tags = append(stream.Params.Tags, "c:unlisted")
	}

	stream.JSONRPC = "2.0"
	stream.Method = "stream_create"
	stream.ID = time.Now().UnixMilli()

	bOut, err := json.Marshal(stream)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", backendAPIURLString+"/v1/asynqueries/", bytes.NewBuffer(bOut))
	if err != nil {
		return "", err
	}

	req.Header.Set("X-Lbry-Auth-Token", p.authToken.Get())

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != 201 {
		return "", ErrStatusCode
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var streamCreate odyseeStreamCreateResponse
	err = json.Unmarshal(b, &streamCreate)
	if err != nil {
		return "", err
	}

	if streamCreate.Status != "query_created" {
		return "", ErrUnableToCreateStream
	}

	return backendAPIURLString + "/v1/asynqueries/" + streamCreate.Payload.QueryID, nil
}

func (p *Platform) waitForConfirm(ctx context.Context, queryURL string) (publish, error) {
	var r odyseeOutputResponse

	for r.Result.TxID == "" {
		time.Sleep(10 * time.Second)

		req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
		if err != nil {
			return publish{}, err
		}

		req.Header.Set("X-Lbry-Auth-Token", p.authToken.Get())

		resp, err := p.client.Do(req)
		if err != nil {
			return publish{}, err
		}
		defer resp.Body.Close()

		if resp.StatusCode == 204 {
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return publish{}, ErrStatusCode
		}

		b, err := io.ReadAll(resp.Body)
		if err != nil {
			return publish{}, err
		}

		err = json.Unmarshal(b, &r)
		if err != nil {
			return publish{}, err
		}
	}

	o := slices.IndexFunc(r.Result.Outputs, func(out output) bool {
		return out.Type == "claim"
	})

	return publish{
		Output:   r.Result.Outputs[o],
		URI:      r.Result.Outputs[o].PermanentURL,
		ClaimID:  r.Result.Outputs[o].ClaimID,
		Outpoint: fmt.Sprintf("%s:0", r.Result.TxID),
	}, nil
}

func (p *Platform) publish(ctx context.Context, pub publish) error {
	val := url.Values{}
	val.Set("auth_token", p.authToken.Get())
	val.Set("uri", pub.URI)
	val.Set("claim_id", pub.ClaimID)
	val.Set("outpoint", pub.Outpoint)
	val.Set("channel_claim_id", p.cfg.Platforms.Odysee.ChannelID)

	req, err := http.NewRequestWithContext(ctx, "POST", mainAPIURLString+"/event/publish", strings.NewReader(val.Encode()))
	if err != nil {
		return err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != 202 {
		return ErrStatusCode
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var publishResponse publishResponse
	err = json.Unmarshal(b, &publishResponse)
	if err != nil {
		return err
	}

	if !publishResponse.Success {
		return errors.New(publishResponse.Error)
	}

	return nil
}
