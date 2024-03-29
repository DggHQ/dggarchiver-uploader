package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	config "github.com/DggHQ/dggarchiver-config/uploader"
	log "github.com/DggHQ/dggarchiver-logger"
	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
	lbry "github.com/DggHQ/dggarchiver-uploader/lbry"
	"github.com/DggHQ/dggarchiver-uploader/monitoring"
	"github.com/DggHQ/dggarchiver-uploader/util"
	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	luaLibs "github.com/vadv/gopher-lua-libs"
	lua "github.com/yuin/gopher-lua"
)

func init() {
	loc, err := time.LoadLocation("UTC")
	if err != nil {
		log.Fatalf("%s", err)
	}
	time.Local = loc
}

func main() {
	cfg := config.Config{}
	cfg.Load()

	if cfg.Uploader.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	monitor := monitoring.Monitor{}
	monitor.Init()
	go monitor.Run()

	L := lua.NewState()
	if cfg.Uploader.Plugins.Enabled {
		luaLibs.Preload(L)
		if err := L.DoFile(cfg.Uploader.Plugins.PathToPlugin); err != nil {
			log.Fatalf("Wasn't able to load the Lua script: %s", err)
		}
	}

	if _, err := cfg.NATS.NatsConnection.Subscribe(fmt.Sprintf("%s.upload", cfg.NATS.Topic), func(msg *nats.Msg) {
		vod := &dggarchivermodel.VOD{}
		err := json.Unmarshal(msg.Data, vod)
		if err != nil {
			log.Errorf("Wasn't able to unmarshal VOD, skipping: %s", err)
			return
		}
		log.Infof("Received a VOD: %s", vod)
		if cfg.Uploader.Plugins.Enabled {
			util.LuaCallReceiveFunction(L, vod)
		}

		log.Infof("Uploading the thumbnail for VOD %s...", vod.ID)
		thumbnail := util.UploadThumbnail(vod.ThumbnailPath)
		if thumbnail == "" {
			log.Errorf("Wasn't able to upload the thumbnail, falling back to the YouTube one")
			thumbnail = vod.Thumbnail
		} else {
			log.Infof("Thumbnail for VOD %s uploaded successfully: %s", vod.ID, thumbnail)
		}

		// TODO: prob better to check the youtube api first, and use the calculation as fallback
		if vod.EndTime == "" {
			log.Infof("VOD %s doesn't have EndTime, calculating it based on the duration...", vod.ID)
			vod.EndTime = util.CalculateEndTime(vod.StartTime, vod.Duration)
		}

		params := lbry.VideoParams{
			Name:         fmt.Sprintf("%s-r-%s%d", vod.ID, vod.Platform, rand.Intn(1000)),
			Title:        fmt.Sprintf("[%s:%s] %s", vod.Platform, vod.ID, vod.Title),
			BID:          "0.0001",
			FilePath:     vod.Path,
			ValidateFile: false,
			OptimizeFile: false,
			Author:       cfg.Uploader.LBRY.Author,
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
			ChannelName:       cfg.Uploader.LBRY.ChannelName,
			WalletID:          "default_wallet",
			FundingAccountIDs: []string{},
			Preview:           false,
			Blocking:          true,
		}

		result := lbry.UploadVideo(cfg, params)
		if result.Error.Code != 0 {
			log.Errorf("Wasn't able to upload VOD (LBRY error), skipping: %s", result.Error.Message)
			return
		}
		if len(result.Result.Outputs) == 0 {
			log.Errorf("Wasn't able to upload VOD (didn't get any outputs), skipping.")
			return
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

		log.Infof("Waiting for 15 seconds before starting to check progress...")
		time.Sleep(15 * time.Second)

		for !uploadResult {
			progressResult := lbry.CheckProgress(cfg, claim)
			if progressResult.Error.Code != 0 {
				log.Errorf("Wasn't able to upload VOD (LBRY error), skipping: %s", result.Error.Message)
				break
			}
			if len(progressResult.Result.Items) != 1 {
				log.Errorf("Wasn't able to upload VOD (couldn't find the file with the claim ID %s), skipping.", claim)
				break
			}

			uploadProgress = progressResult.Result.Items[0].ReflectorProgress
			// 	Set Prometheus Gauge Value to the current upload progress value
			monitor.ChangeCurrentProgress(float64(uploadProgress), prometheus.Labels{
				"id":           vod.ID,
				"channel_name": cfg.Uploader.LBRY.ChannelName,
				"vod_title":    vod.Title,
			})
			uploadResult = progressResult.Result.Items[0].IsFullyReflected
			if uploadResult {
				break
			}
			log.Infof("VOD %s (claim ID: %s) upload status: %d%%", vod.ID, claim, uploadProgress)
			if cfg.Uploader.Plugins.Enabled {
				util.LuaCallProgressFunction(L, uploadProgress)
			}
			time.Sleep(15 * time.Second)
		}

		if uploadResult {
			log.Infof("VOD %s uploaded successfully :)", vod.ID)
			removalStatus := lbry.DeleteFile(cfg, claim)
			if removalStatus.Error.Code != 0 {
				log.Errorf("Wasn't able to delete VOD (LBRY error), skipping: %s", result.Error.Message)
			}
			if !removalStatus.Result.Bool {
				log.Errorf("Wasn't able to delete VOD (LBRY daemon responded with 'false' or 'null' for claim ID %s).", claim)
				cleanBlobsStatusResponse := lbry.CleanBlobs(cfg)
				if cleanBlobsStatusResponse.Error.Code != 0 {
					log.Errorf("Wasn't able to cleanup blobs using LBRY (LBRY error): %s", result.Error.Message)
				}
				if cleanBlobsStatusResponse.Result.Bool {
					log.Infof("Blobs cleaned up successfully.")
				} else {
					log.Errorf("Wasn't able to cleanup blobs VOD using LBRY (LBRY daemon responded with 'false' or 'null'), skipping.")
				}
			} else {
				log.Infof("File %s deleted successfully.", vod.ID)
			}

			err := cfg.Uploader.SQLite.DB.Create(&dggarchivermodel.UploadedVOD{
				VOD:                *vod,
				Claim:              result.Result.Outputs[0].ClaimID,
				LBRYChannel:        cfg.Uploader.LBRY.ChannelName,
				LBRYName:           result.Result.Outputs[0].Name,
				LBRYNormalizedName: result.Result.Outputs[0].NormalizedName,
				LBRYPermanentURL:   result.Result.Outputs[0].PermanentURL,
			}).Error
			if err != nil {
				log.Errorf("Wasn't able to insert VOD into SQLite DB: %s", err)
				return
			}
			if cfg.Uploader.Plugins.Enabled {
				util.LuaCallInsertFunction(L, vod, err == nil)
			}
		} else {
			log.Errorf("VOD %s failed to upload :(", vod.ID)
		}

		if cfg.Uploader.Plugins.Enabled {
			util.LuaCallFinishFunction(L, vod, uploadResult)
		}
	}); err != nil {
		log.Fatalf("An error occured when subscribing to topic: %s", err)
	}

	log.Infof("Waiting for VODs...")
	var forever chan struct{}
	<-forever
}
