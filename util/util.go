package util

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	log "github.com/DggHQ/dggarchiver-logger"
)

type odycdnThumbnailResponse struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	Type     string `json:"type"`
	Message  string `json:"message"`
}

func UploadThumbnail(path string) string {
	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)
	fw, err := writer.CreateFormFile("file-input", filepath.Base(path))
	if err != nil {
		log.Errorf("Wasn't able to create form file for the thumbnail: %s", err)
		return ""
	}
	fd, err := os.Open(path)
	if err != nil {
		log.Errorf("Wasn't able to open thumbnail: %s", err)
		return ""
	}
	defer fd.Close()
	_, err = io.Copy(fw, fd)
	if err != nil {
		log.Errorf("Wasn't able to copy thumbnail: %s", err)
		return ""
	}

	writer.Close()

	client := &http.Client{
		Timeout: time.Minute * 2,
	}
	req, err := http.NewRequest("POST", "https://thumbs.odycdn.com/upload", form)
	if err != nil {
		log.Errorf("Wasn't able to create the thumbs.odycdn.com request: %s", err)
		return ""
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := client.Do(req)
	if err != nil && response.StatusCode != 200 {
		log.Errorf("Wasn't able to upload thumbnail to thumbs.odycdn.com: %s", err)
		return ""
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Errorf("Wasn't able to read the thumbs.odycdn.com response: %s", err)
		return ""
	}

	parsed := &odycdnThumbnailResponse{}
	err = json.Unmarshal(body, parsed)
	if err != nil {
		log.Errorf("Wasn't able to unmarshal the thumbs.odycdn.com response: %s", err)
		return ""
	}

	if parsed.Type != "success" {
		log.Errorf("thumbs.odycdn.com hasn't returned 'success'")
		return ""
	}

	return parsed.URL
}

func CalculateEndTime(startTime string, duration int) string {
	parsed, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		log.Errorf("Wasn't able to parse the \"%s\" timestamp: %s", startTime, err)
	}

	endTimeParsed := parsed.Add(time.Second * time.Duration(duration))
	return endTimeParsed.Format(time.RFC3339)
}
