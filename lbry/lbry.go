package lbry

import (
	"bytes"
	"encoding/json"
	"net/http"

	config "github.com/DggHQ/dggarchiver-config/uploader"
)

func UploadVideo(config config.Config, params LBRYVideoParams) LBRYVideoResponse {
	req := LBRYVideoUpload{
		Method: "publish",
		Params: params,
	}
	reqJson, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(config.Uploader.LBRY.URI, "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	result := &LBRYVideoResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	return *result
}

func CheckProgress(config config.Config, claim string) LBRYFileListResponse {
	req := LBRYFileList{
		Method: "file_list",
		Params: LBRYFileListParams{
			ClaimID: claim,
		},
	}
	reqJson, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(config.Uploader.LBRY.URI, "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	result := &LBRYFileListResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	return *result
}

func DeleteFile(config config.Config, claim string) LBRYFileDeleteResponse {
	req := LBRYFileList{
		Method: "file_delete",
		Params: LBRYFileListParams{
			ClaimID: claim,
		},
	}
	reqJson, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(config.Uploader.LBRY.URI, "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	result := &LBRYFileDeleteResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	return *result
}

func CleanBlobs(config config.Config) LBRYBlobCleanResponse {
	req := LBRYBlobClean{
		Method: "blob_clean",
	}
	reqJson, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(config.Uploader.LBRY.URI, "application/json", bytes.NewBuffer(reqJson))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	result := &LBRYBlobCleanResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	return *result
}
