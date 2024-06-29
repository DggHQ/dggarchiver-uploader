package lbry

import (
	"gopkg.in/guregu/null.v4/zero"
)

// Contains the data structure to send a video upload request to the LBRY daemon
type VideoUpload struct {
	Method string      `json:"method"`
	Params VideoParams `json:"params"`
}

// Contains the data structure with params to add to the LBRYUpload structure
type VideoParams struct {
	Name              string   `json:"name"`
	Title             string   `json:"title"`
	BID               string   `json:"bid"`
	FilePath          string   `json:"file_path"`
	ValidateFile      bool     `json:"validate_file"`
	OptimizeFile      bool     `json:"optimize_file"`
	Author            string   `json:"author"`
	Description       string   `json:"description"`
	ThumbnailURL      string   `json:"thumbnail_url"`
	Tags              []string `json:"tags"`
	Languages         []string `json:"languages"`
	Locations         []string `json:"locations"`
	ChannelName       string   `json:"channel_name"`
	WalletID          string   `json:"wallet_id"`
	FundingAccountIDs []string `json:"funding_account_ids"`
	Preview           bool     `json:"preview"`
	Blocking          bool     `json:"blocking"`
}

// Contains the data structure with the response of the LBRY daemon
type VideoResponse struct {
	Result VideoResult `json:"result,omitempty"`
	Error  Error
}

// Contains the data structure with the "result" field of the LBRY response
type VideoResult struct {
	Outputs []VideoOutputs
}

// Contains the data structure with the "result" field of the LBRY response
type VideoOutputs struct {
	ClaimID        string `json:"claim_id"`
	Name           string `json:"name"`
	NormalizedName string `json:"normalized_name"`
	PermanentURL   string `json:"permanent_url"`
}

// Contains the data structure with the response of the LBRY daemon
type Error struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// Contains the data structure to send a file list request to the LBRY daemon
type FileList struct {
	Method string         `json:"method"`
	Params FileListParams `json:"params"`
}

// Contains the data structure with params to add to the LBRYFileList structure
type FileListParams struct {
	ClaimID string `json:"claim_id"`
}

// Contains the data structure with the response of the LBRY daemon
type FileListResponse struct {
	Result FileListResult `json:"result,omitempty"`
	Error  Error
}

// Contains the data structure with the "result" field of the LBRY response
type FileListResult struct {
	Items []FileListItems
}

// Contains the data structure with the "result" field of the LBRY response
type FileListItems struct {
	UploadingToReflector bool `json:"uploading_to_reflector"`
	IsFullyReflected     bool `json:"is_fully_reflected"`
	ReflectorProgress    int  `json:"reflector_progress"`
}

// Contains the data structure to send a file list request to the LBRY daemon
type FileDelete struct {
	Method string           `json:"method"`
	Params FileDeleteParams `json:"params"`
}

// Contains the data structure with params to add to the LBRYFileList structure
type FileDeleteParams struct {
	ClaimID string `json:"claim_id"`
}

// Contains the data structure with the response of the LBRY daemon
type FileDeleteResponse struct {
	Result zero.Bool `json:"result,omitempty"`
	Error  Error
}

// Contains the data structure to send a file list request to the LBRY daemon
type BlobClean struct {
	Method string `json:"method"`
}

// Contains the data structure with the response of the LBRY daemon
type BlobCleanResponse struct {
	Result zero.Bool `json:"result,omitempty"`
	Error  Error
}
