package lbry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	config "github.com/DggHQ/dggarchiver-config/uploader"
	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
	"github.com/DggHQ/dggarchiver-uploader/monitoring"
	"github.com/DggHQ/dggarchiver-uploader/platforms/implementation"
	"github.com/DggHQ/dggarchiver-uploader/util"
	"github.com/prometheus/client_golang/prometheus"
	lua "github.com/yuin/gopher-lua"
)

const (
	platformName string = "lbry"
)

var (
	ErrNoOutputs = errors.New("no lbry outputs")
	ErrNoClaim   = errors.New("no file with claim")
)

func init() {
	implementation.Map[platformName] = New
}

type Platform struct {
	cfg     *config.Config
	monitor *monitoring.Monitor
}

func New(cfg *config.Config, monitor *monitoring.Monitor) (implementation.Platform, error) {
	return &Platform{
		cfg:     cfg,
		monitor: monitor,
	}, nil
}

func (p *Platform) Upload(_ context.Context, vod *dggarchivermodel.VOD, l *lua.LState) error {
	slogVodGroup := slog.Group("vod",
		slog.String("platform", vod.Platform),
		slog.String("id", vod.VID),
	)

	slog.Debug("uploading thumbnail", slog.String("platform", platformName), slogVodGroup)
	thumbnail, err := UploadThumbnail(vod.ThumbnailPath)
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

	// TODO: prob better to check the youtube api first, and use the calculation as fallback
	if vod.EndTime == "" {
		slog.Info("calculating endtime based on the duration", slog.String("platform", platformName), slogVodGroup)
		vod.EndTime, err = util.CalculateEndTime(vod.StartTime, vod.Duration)
		if err != nil {
			return err
		}
	}

	params := VideoParams{
		Name:         fmt.Sprintf("%s-r-%s%d", vod.VID, vod.Platform, rand.Intn(1000)),
		Title:        fmt.Sprintf("[%s:%s] %s", vod.Platform, vod.VID, vod.Title),
		BID:          "0.0001",
		FilePath:     vod.Path,
		ValidateFile: false,
		OptimizeFile: false,
		Author:       p.cfg.Platforms.LBRY.Author,
		Description:  fmt.Sprintf("%s\n%s", vod.StartTime, vod.EndTime),
		ThumbnailURL: thumbnail,
		Tags: []string{
			"destiny",
			"vod",
			"yee wins",
			"reupload",
			"mirror",
		},
		Languages: []string{
			"en",
		},
		Locations:         []string{},
		ChannelName:       p.cfg.Platforms.LBRY.ChannelName,
		WalletID:          "default_wallet",
		FundingAccountIDs: []string{},
		Preview:           false,
		Blocking:          true,
	}

	if vod.Visibility == 1 || vod.Visibility == 2 {
		params.Tags = append(params.Tags, "c:unlisted")
	}

	slog.Info("starting to upload", slog.String("platform", platformName), slogVodGroup)
	result, err := p.uploadVideo(params)
	if err != nil {
		return err
	}
	if result.Error.Code != 0 {
		return errors.New(result.Error.Message)
	}
	if len(result.Result.Outputs) == 0 {
		return ErrNoOutputs
	}

	claim := ""

	for _, v := range result.Result.Outputs {
		if v.ClaimID != "" {
			claim = v.ClaimID
			break
		}
	}

	var uploadProgress int
	uploadResult := false

	slog.Debug("waiting before checking progress",
		slog.String("platform", platformName),
		slog.Group("vod",
			slog.String("platform", vod.Platform),
			slog.String("id", vod.VID),
		),
		slog.Int("sleep", 15),
	)
	time.Sleep(15 * time.Second)

	for !uploadResult {
		progressResult, err := p.checkProgress(claim)
		if err != nil {
			return err
		}
		if progressResult.Error.Code != 0 {
			return errors.New(result.Error.Message)
		}
		if len(progressResult.Result.Items) != 1 {
			return ErrNoClaim
		}

		uploadProgress = progressResult.Result.Items[0].ReflectorProgress
		// 	Set Prometheus Gauge Value to the current upload progress value
		p.monitor.ChangeCurrentProgress(float64(uploadProgress), prometheus.Labels{
			"id":           vod.VID,
			"channel_name": p.cfg.Platforms.LBRY.ChannelName,
			"vod_title":    vod.Title,
		})
		uploadResult = progressResult.Result.Items[0].IsFullyReflected
		if uploadResult {
			break
		}
		slog.Info("VOD upload status",
			slog.String("platform", platformName),
			slogVodGroup,
			slog.String("claim", claim),
			slog.Int("progress", uploadProgress),
		)
		if p.cfg.Plugins.Enabled {
			util.LuaCallProgressFunction(l, uploadProgress)
		}
		time.Sleep(15 * time.Second)
	}

	if uploadResult {
		slog.Info("VOD uploaded", slog.String("platform", platformName), slogVodGroup)
		removalStatus, err := p.deleteFile(claim)
		if err != nil {
			return err
		}
		if removalStatus.Error.Code != 0 {
			slog.Warn("unable to delete VOD",
				slog.String("platform", platformName),
				slogVodGroup,
				slog.String("err", result.Error.Message),
			)
		}
		if !removalStatus.Result.Bool {
			cleanBlobsStatusResponse, err := p.cleanBlobs()
			if err != nil {
				return err
			}
			if cleanBlobsStatusResponse.Error.Code != 0 {
				slog.Warn("unable to clean VOD blobs",
					slog.String("platform", platformName),
					slogVodGroup,
					slog.String("err", result.Error.Message),
				)
			}
			if cleanBlobsStatusResponse.Result.Bool {
				slog.Debug("VOD blobs cleaned up", slog.String("platform", platformName), slogVodGroup)
			} else {
				slog.Warn("unable to clean VOD blobs",
					slog.String("platform", platformName),
					slogVodGroup,
					slog.String("err", result.Error.Message),
				)
			}
		} else {
			slog.Info("VOD cleaned up", slog.String("platform", platformName), slogVodGroup)
		}

		addInfo := map[string]string{
			"Claim": result.Result.Outputs[0].ClaimID,
		}
		addInfoBytes, _ := json.Marshal(addInfo)

		err = p.cfg.SQLite.DB.Create(&dggarchivermodel.UploadedVOD{
			HostingPlatform:       platformName,
			VOD:                   *vod,
			HostingAdditionalInfo: addInfoBytes,
			HostingChannel:        p.cfg.Platforms.LBRY.ChannelName,
			HostingName:           result.Result.Outputs[0].Name,
			HostingNormalizedName: result.Result.Outputs[0].NormalizedName,
			HostingURL:            result.Result.Outputs[0].PermanentURL,
		}).Error
		if err != nil {
			return err
		}
		if p.cfg.Plugins.Enabled {
			util.LuaCallInsertFunction(l, vod, err == nil)
		}
	} else {
		slog.Error("VOD upload failed", slog.String("platform", platformName), slogVodGroup)
		return nil
	}

	if p.cfg.Plugins.Enabled {
		util.LuaCallFinishFunction(l, vod, uploadResult)
	}
	return nil
}

func (p *Platform) uploadVideo(params VideoParams) (VideoResponse, error) {
	req := VideoUpload{
		Method: "publish",
		Params: params,
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return VideoResponse{}, err
	}

	resp, err := http.Post(p.cfg.Platforms.LBRY.URI, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return VideoResponse{}, err
	}

	defer resp.Body.Close()

	result := VideoResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (p *Platform) checkProgress(claim string) (FileListResponse, error) {
	req := FileList{
		Method: "file_list",
		Params: FileListParams{
			ClaimID: claim,
		},
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return FileListResponse{}, err
	}

	resp, err := http.Post(p.cfg.Platforms.LBRY.URI, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return FileListResponse{}, err
	}

	defer resp.Body.Close()

	result := FileListResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (p *Platform) deleteFile(claim string) (FileDeleteResponse, error) {
	req := FileList{
		Method: "file_delete",
		Params: FileListParams{
			ClaimID: claim,
		},
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return FileDeleteResponse{}, err
	}

	resp, err := http.Post(p.cfg.Platforms.LBRY.URI, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return FileDeleteResponse{}, err
	}

	defer resp.Body.Close()

	result := FileDeleteResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (p *Platform) cleanBlobs() (BlobCleanResponse, error) {
	req := BlobClean{
		Method: "blob_clean",
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return BlobCleanResponse{}, err
	}

	resp, err := http.Post(p.cfg.Platforms.LBRY.URI, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		return BlobCleanResponse{}, err
	}

	defer resp.Body.Close()

	result := BlobCleanResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, nil
}
