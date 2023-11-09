package lbry

import (
	"bytes"
	"encoding/json"
	"net/http"

	config "github.com/DggHQ/dggarchiver-config/uploader"
)

func UploadVideo(config config.Config, params VideoParams) VideoResponse {
	req := VideoUpload{
		Method: "publish",
		Params: params,
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(config.Uploader.LBRY.URI, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	result := &VideoResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	return *result
}

func CheckProgress(config config.Config, claim string) FileListResponse {
	req := FileList{
		Method: "file_list",
		Params: FileListParams{
			ClaimID: claim,
		},
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(config.Uploader.LBRY.URI, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	result := &FileListResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	return *result
}

func DeleteFile(config config.Config, claim string) FileDeleteResponse {
	req := FileList{
		Method: "file_delete",
		Params: FileListParams{
			ClaimID: claim,
		},
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(config.Uploader.LBRY.URI, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	result := &FileDeleteResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	return *result
}

func CleanBlobs(config config.Config) BlobCleanResponse {
	req := BlobClean{
		Method: "blob_clean",
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(config.Uploader.LBRY.URI, "application/json", bytes.NewBuffer(reqJSON))
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	result := &BlobCleanResponse{}

	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		panic(err)
	}

	return *result
}
