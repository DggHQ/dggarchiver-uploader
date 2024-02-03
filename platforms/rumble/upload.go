package rumble

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
	"github.com/DggHQ/dggarchiver-uploader/util"
	"github.com/PuerkitoBio/goquery"
	lua "github.com/yuin/gopher-lua"
)

var (
	ErrFileTooLarge         = errors.New("file too large")
	ErrNoUploadServer       = errors.New("no upload server found")
	APIVersion              = "1.3"
	maxSingleChunk    int64 = 5000000
	maxFileSize       int64 = 15000000000 // 15 gigs is max file size on rumble (3000 chunks of maxSingleChunk size)
	regularURLRegexp        = regexp.MustCompile(`(?U)https:\/\/rumble\.com\/v.*\.html`)
	embedURLRegexp          = regexp.MustCompile(`(?U)https:\/\/rumble\.com\/embed\/v.*\/`)
)

type uploadFormTemplate struct {
	Title          string
	Description    string
	ServerFileName string
	Tags           string
	Category       string
	Visibility     string
	Metadata       fileMetadata
	Thumbnail      string
}

type fileMetadata struct {
	Name      string `json:"name"`
	Modified  int64  `json:"modified"`
	Size      int64  `json:"size"`
	Type      string `json:"type"`
	TimeStart int64  `json:"time_start"`
	Speed     int64  `json:"speed"`
	NumChunks int64  `json:"num_chunks"`
	TimeEnd   int64  `json:"time_end"`
}

func (p *Platform) Upload(ctx context.Context, vod *dggarchivermodel.VOD, l *lua.LState) error {
	var err error

	slogVodGroup := slog.Group("vod",
		slog.String("platform", vod.Platform),
		slog.String("id", vod.ID),
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

	loggedIn, err := p.IsLoggedIn(ctx)
	if err != nil {
		return err
	}

	if !loggedIn {
		if err := p.Login(ctx); err != nil {
			return err
		}
	}

	slog.Debug("getting the upload url", slog.String("platform", platformName), slogVodGroup)
	u, err := p.getUploadURL(ctx)
	if err != nil {
		return err
	}

	slog.Info("starting to upload", slog.String("platform", platformName), slogVodGroup)
	var urls string
	if fi.Size() < maxSingleChunk {
		if urls, err = p.smallUpload(ctx, vod, f, fi, u); err != nil {
			return err
		}
	} else {
		if urls, err = p.bigUpload(ctx, vod, f, fi, u); err != nil {
			return err
		}
	}

	slog.Info("VOD uploaded", slog.String("platform", platformName), slogVodGroup)
	err = p.cfg.SQLite.DB.Create(&dggarchivermodel.UploadedVOD{
		HostingPlatform: platformName,
		VOD:             *vod,
		HostingChannel:  p.cfg.Platforms.Rumble.Login,
		HostingURL:      urls,
	}).Error
	if err != nil {
		return err
	}
	if p.cfg.Plugins.Enabled {
		util.LuaCallInsertFunction(l, vod, err == nil)
	}

	if p.cfg.Plugins.Enabled {
		util.LuaCallFinishFunction(l, vod, true)
	}

	return nil
}

func (p *Platform) getUploadURL(ctx context.Context) (*url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://rumble.com/upload.php", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrStatusCode
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	uploadServer, exists := doc.Find("input#upload_server").Attr("value")
	if !exists {
		return nil, ErrNoUploadServer
	}

	u, err := url.Parse(fmt.Sprintf("https://%s.rumble.com/upload.php?api=%s", uploadServer, APIVersion))
	if err != nil {
		return nil, err
	}

	return u, nil
}

func (p *Platform) smallUpload(ctx context.Context, vod *dggarchivermodel.VOD, f *os.File, fi os.FileInfo, u *url.URL) (string, error) {
	timeStart := time.Now()

	fileName, err := p.multipartUpload(ctx, f, fi, u)
	if err != nil {
		return "", err
	}

	timeEnd := time.Now()

	_, err = p.checkDuration(ctx, u, fileName)
	if err != nil {
		return "", err
	}

	thumb, err := p.checkThumbnails(ctx, u, fileName)
	if err != nil {
		return "", err
	}

	selectedThumb := ""
	for key := range thumb {
		selectedThumb = key
		break
	}

	speed := ((fi.Size()) / (timeEnd.UnixMilli() - timeStart.UnixMilli())) * 100

	info := uploadFormTemplate{
		Title:          fmt.Sprintf("[%s:%s] %s", vod.Platform, vod.ID, vod.Title),
		Description:    fmt.Sprintf("%s\n%s", vod.StartTime, vod.EndTime),
		ServerFileName: fileName,
		Tags:           "destiny,vod,yee wins,reupload,mirror",
		Category:       "15",
		Visibility:     "private",
		Thumbnail:      selectedThumb,
		Metadata: fileMetadata{
			Name:      fi.Name(),
			Modified:  fi.ModTime().UnixMilli(),
			Size:      fi.Size(),
			Type:      "video/mp4",
			Speed:     speed,
			NumChunks: 1,
			TimeStart: timeStart.UnixMilli(),
			TimeEnd:   timeEnd.UnixMilli(),
		},
	}

	res, err := p.sendUploadForm(ctx, u, info)
	if err != nil {
		return "", err
	}

	return res, err
}

func (p *Platform) bigUpload(ctx context.Context, vod *dggarchivermodel.VOD, f *os.File, fi os.FileInfo, u *url.URL) (string, error) {
	timeStart := time.Now()

	initialFileName := generatePutName(fi.Name(), timeStart)
	serverFileName, chunkQty, err := p.putUpload(ctx, f, fi, u, initialFileName)
	if err != nil {
		return "", err
	}

	timeEnd := time.Now()

	_, err = p.checkDuration(ctx, u, serverFileName)
	if err != nil {
		return "", err
	}

	thumb, err := p.checkThumbnails(ctx, u, serverFileName)
	if err != nil {
		return "", err
	}

	selectedThumb := ""
	for key := range thumb {
		selectedThumb = key
		break
	}

	speed := ((fi.Size()) / (timeEnd.UnixMilli() - timeStart.UnixMilli())) * 100

	processedTitle := ""
	for i, v := range strings.Fields(fmt.Sprintf("[%s:%s] %s", vod.Platform, vod.ID, vod.Title)) {
		var t string
		if i == 0 {
			t = v
		} else {
			t = fmt.Sprintf("%s %s", processedTitle, v)
		}
		if len(t) > 99 {
			break
		}
		processedTitle = t
	}

	info := uploadFormTemplate{
		Title:          processedTitle,
		Description:    fmt.Sprintf("%s\n%s", vod.StartTime, vod.EndTime),
		ServerFileName: serverFileName,
		Tags:           "destiny,vod,yee wins,reupload,mirror",
		Category:       "15",
		Visibility:     "private",
		Thumbnail:      selectedThumb,
		Metadata: fileMetadata{
			Name:      fi.Name(),
			Modified:  fi.ModTime().UnixMilli(),
			Size:      fi.Size(),
			Type:      "video/mp4",
			Speed:     speed,
			NumChunks: int64(chunkQty),
			TimeStart: timeStart.UnixMilli(),
			TimeEnd:   timeEnd.UnixMilli(),
		},
	}

	res, err := p.sendUploadForm(ctx, u, info)
	if err != nil {
		return "", err
	}

	return res, nil
}

func (p *Platform) putUpload(ctx context.Context, f *os.File, fi os.FileInfo, u *url.URL, fileName string) (string, int, error) {
	chunkQty := int(fi.Size() / maxSingleChunk)

	chunkNames := []string{}
	for i := 0; i < chunkQty; i++ {
		chunkNames = append(chunkNames, fmt.Sprintf("%d_%s", i, fileName))
	}

	r := bufio.NewReader(f)

	for _, v := range chunkNames {
		uWithChunk := *u
		qVals := uWithChunk.Query()
		qVals.Add("chunk", v)
		qVals.Add("chunkSz", fmt.Sprintf("%d", maxSingleChunk))
		qVals.Add("chunkQty", fmt.Sprintf("%d", chunkQty))
		uWithChunk.RawQuery = qVals.Encode()

		chunk := make([]byte, maxSingleChunk)
		_, err := r.Read(chunk)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return "", 0, err
		}

		req, err := http.NewRequestWithContext(ctx, "PUT", uWithChunk.String(), bytes.NewBuffer(chunk))
		if err != nil {
			return "", 0, err
		}

		resp, err := p.client.Do(req)
		if err != nil {
			return "", 0, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", 0, ErrStatusCode
		}
	}

	uMerge := *u
	qVals := uMerge.Query()
	qVals.Add("merge", fmt.Sprintf("%d", chunkQty-1))
	qVals.Add("chunk", fileName)
	qVals.Add("chunkSz", fmt.Sprintf("%d", maxSingleChunk))
	qVals.Add("chunkQty", fmt.Sprintf("%d", chunkQty))
	uMerge.RawQuery = qVals.Encode()

	req, err := http.NewRequestWithContext(ctx, "POST", uMerge.String(), nil)
	if err != nil {
		return "", 0, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", 0, ErrStatusCode
	}

	ret, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	return string(ret), chunkQty, nil
}

func (p *Platform) multipartUpload(ctx context.Context, f *os.File, fi os.FileInfo, u *url.URL) (string, error) {
	var b bytes.Buffer
	mpw := multipart.NewWriter(&b)

	part, err := mpw.CreateFormFile("Filedata", fi.Name())
	if err != nil {
		return "", err
	}

	if _, err = io.Copy(part, f); err != nil {
		return "", err
	}

	mpw.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), &b)
	if err != nil {
		return "", err
	}
	req.Header.Add("content-type", mpw.FormDataContentType())

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ErrStatusCode
	}

	ret, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ret), nil
}

func (p *Platform) checkDuration(ctx context.Context, u *url.URL, fileName string) (float64, error) {
	uWithFileName := *u
	qVals := uWithFileName.Query()
	qVals.Add("duration", fileName)
	uWithFileName.RawQuery = qVals.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", uWithFileName.String(), nil)
	if err != nil {
		return 0, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, ErrStatusCode
	}

	ret, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	dur, err := strconv.ParseFloat(string(ret), 64)
	if err != nil {
		return 0, err
	}

	return dur, nil
}

func (p *Platform) checkThumbnails(ctx context.Context, u *url.URL, fileName string) (map[string]string, error) {
	uWithFileName := *u
	qVals := uWithFileName.Query()
	qVals.Add("thumbnails", fileName)
	uWithFileName.RawQuery = qVals.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", uWithFileName.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, ErrStatusCode
	}

	ret, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	thumbnails := make(map[string]string)
	err = json.Unmarshal(ret, &thumbnails)
	if err != nil {
		return nil, err
	}

	return thumbnails, nil
}

func (p *Platform) sendUploadForm(ctx context.Context, u *url.URL, info uploadFormTemplate) (string, error) {
	uWithForm := *u
	qVals := uWithForm.Query()
	qVals.Add("form", "1")
	uWithForm.RawQuery = qVals.Encode()

	meta, err := json.Marshal(info.Metadata)
	if err != nil {
		return "", err
	}

	data := url.Values{}
	data.Set("title", info.Title)
	data.Set("description", info.Description)
	data.Set("video[]", info.ServerFileName)
	data.Set("featured", "0")
	data.Set("rights", "1")
	data.Set("terms", "1")
	data.Set("facebookUpload", "")
	data.Set("vimeoUpload", "")
	data.Set("infoWho", "")
	data.Set("infoWhen", "")
	data.Set("infoWhere", "")
	data.Set("infoExtUser", "")
	data.Set("tags", info.Tags)
	data.Set("channelId", "0")
	data.Set("channelId", "0")
	data.Set("sideChannelId", info.Category)
	data.Set("mediaChannelId", "")
	data.Set("visibility", info.Visibility)
	data.Set("file_meta", string(meta))
	data.Set("thumb", info.Thumbnail)

	// formData := fmt.Sprintf(`title=%s&description=%s&video[]=%s&featured=0&rights=1&terms=1&facebookUpload=&vimeoUpload=&infoWho=&infoWhen=&infoWhere=&infoExtUser=&tags=%s&channelId=0
	// &sideChannelId=%s&mediaChannelId=&visibility=%s&file_meta=%s&thumb=%s`, info.Title, info.Description, info.ServerFileName, info.Tags, info.Category, info.Visibility, string(meta), info.Thumbnail)
	// formData = url.QueryEscape(formData)

	req, err := http.NewRequestWithContext(ctx, "POST", uWithForm.String(), strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Add("content-type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ErrStatusCode
	}

	ret, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	regularURL := regularURLRegexp.FindString(string(ret))
	embedURL := embedURLRegexp.FindString(string(ret))

	return fmt.Sprintf("%s %s", regularURL, embedURL), nil
}

func generatePutName(file string, timeStart time.Time) string {
	r := int64(100000*rand.Float64() + 100000)
	ext := filepath.Ext(file)
	return fmt.Sprintf("%d-%d%s", timeStart.UnixMilli(), r, ext)
}
