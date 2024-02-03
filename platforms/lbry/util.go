package lbry

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var (
	ErrStatusNotOk = errors.New("server returned != 200")
	ErrNoSuccess   = errors.New("thumbs.odycdn.com hasn't returned 'success'")
)

type odycdnThumbnailResponse struct {
	URL      string `json:"url"`
	FileName string `json:"file_name"`
	Type     string `json:"type"`
	Message  string `json:"message"`
}

func uploadThumbnail(path string) (string, error) {
	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)
	fw, err := writer.CreateFormFile("file-input", filepath.Base(path))
	if err != nil {
		return "", err
	}
	fd, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer fd.Close()
	_, err = io.Copy(fw, fd)
	if err != nil {
		return "", err
	}

	writer.Close()

	client := &http.Client{
		Timeout: time.Minute * 2,
	}
	req, err := http.NewRequest("POST", "https://thumbs.odycdn.com/upload", form)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != 200 {
		return "", ErrStatusNotOk
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	parsed := &odycdnThumbnailResponse{}
	err = json.Unmarshal(body, parsed)
	if err != nil {
		return "", err
	}

	if parsed.Type != "success" {
		return "", ErrNoSuccess
	}

	return parsed.URL, nil
}
