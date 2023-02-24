package lbry

import (
	"gopkg.in/guregu/null.v4/zero"
)

// Contains the data structure to send a video upload request to the LBRY daemon
type LBRYVideoUpload struct {
	Method string          `json:"method"`
	Params LBRYVideoParams `json:"params"`
}

// Contains the data structure with params to add to the LBRYUpload structure
type LBRYVideoParams struct {
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
type LBRYVideoResponse struct {
	Result LBRYVideoResult `json:"result,omitempty"`
	Error  LBRYError
}

// Contains the data structure with the "result" field of the LBRY response
type LBRYVideoResult struct {
	Outputs []LBRYVideoOutputs
}

// Contains the data structure with the "result" field of the LBRY response
type LBRYVideoOutputs struct {
	ClaimID        string `json:"claim_id"`
	Name           string `json:"name"`
	NormalizedName string `json:"normalized_name"`
	PermanentURL   string `json:"permanent_url"`
}

// Contains the data structure with the response of the LBRY daemon
type LBRYError struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

// Contains the data structure to send a file list request to the LBRY daemon
type LBRYFileList struct {
	Method string             `json:"method"`
	Params LBRYFileListParams `json:"params"`
}

// Contains the data structure with params to add to the LBRYFileList structure
type LBRYFileListParams struct {
	ClaimID string `json:"claim_id"`
}

// Contains the data structure with the response of the LBRY daemon
type LBRYFileListResponse struct {
	Result LBRYFileListResult `json:"result,omitempty"`
	Error  LBRYError
}

// Contains the data structure with the "result" field of the LBRY response
type LBRYFileListResult struct {
	Items []LBRYFileListItems
}

// Contains the data structure with the "result" field of the LBRY response
type LBRYFileListItems struct {
	UploadingToReflector bool `json:"uploading_to_reflector"`
	IsFullyReflected     bool `json:"is_fully_reflected"`
	ReflectorProgress    int  `json:"reflector_progress"`
}

// Contains the data structure to send a file list request to the LBRY daemon
type LBRYFileDelete struct {
	Method string               `json:"method"`
	Params LBRYFileDeleteParams `json:"params"`
}

// Contains the data structure with params to add to the LBRYFileList structure
type LBRYFileDeleteParams struct {
	ClaimID string `json:"claim_id"`
}

// Contains the data structure with the response of the LBRY daemon
type LBRYFileDeleteResponse struct {
	Result zero.Bool `json:"result,omitempty"`
	Error  LBRYError
}

// Contains the data structure to send a file list request to the LBRY daemon
type LBRYBlobClean struct {
	Method string `json:"method"`
}

// Contains the data structure with the response of the LBRY daemon
type LBRYBlobCleanResponse struct {
	Result zero.Bool `json:"result,omitempty"`
	Error  LBRYError
}
