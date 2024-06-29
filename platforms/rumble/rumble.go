package rumble

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	config "github.com/DggHQ/dggarchiver-config/uploader"
	"github.com/DggHQ/dggarchiver-uploader/monitoring"
	"github.com/DggHQ/dggarchiver-uploader/platforms/implementation"
	"github.com/DggHQ/dggarchiver-uploader/platforms/rumble/cookies"
	"github.com/DggHQ/dggarchiver-uploader/platforms/rumble/md5"

	"github.com/PuerkitoBio/goquery"
)

const (
	platformName string = "rumble"
)

var (
	ErrStatusCode = errors.New("server returned not OK")
	checkRegex    = regexp.MustCompile(`\{username:.*,logged_in:(true|false)\}`)
)

func init() {
	implementation.Map[platformName] = New
}

type salts struct {
	Salts []string `json:"salts"`
}

type saltResp struct {
	Data salts `json:"data"`
}

type loginCheck struct {
	LoggedIn bool `json:"logged_in"`
}

type Platform struct {
	cfg     *config.Config
	client  http.Client
	monitor *monitoring.Monitor
	cookies *cookies.Cookies
}

func New(cfg *config.Config, monitor *monitoring.Monitor) (implementation.Platform, error) {
	c, err := cookies.New("")
	if err != nil {
		return nil, err
	}

	_ = c.Load()

	return &Platform{
		cfg:     cfg,
		monitor: monitor,
		client: http.Client{
			Jar: c.GetJar(),
		},
		cookies: c,
	}, nil
}

func (p *Platform) IsLoggedIn(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://rumble.com/browse", nil)
	if err != nil {
		return false, err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return false, err
	}

	someRumbleScript := doc.Find("body script").Text()
	loginString := checkRegex.FindString(someRumbleScript)
	loginString = strings.ReplaceAll(loginString, "username", "\"username\"")
	loginString = strings.ReplaceAll(loginString, "logged_in", "\"logged_in\"")

	var check loginCheck
	err = json.Unmarshal([]byte(loginString), &check)
	if err != nil {
		return false, err
	}

	return check.LoggedIn, nil
}

func (p *Platform) Login(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://rumble.com/service.php?name=user.get_salts", strings.NewReader(fmt.Sprintf("username=%s", p.cfg.Platforms.Rumble.Login)))
	if err != nil {
		return err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrStatusCode
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var salts saltResp
	err = json.Unmarshal(b, &salts)
	if err != nil {
		return err
	}

	hashes := []string{}
	hashes = append(hashes, md5.Hash(fmt.Sprintf("%s%s", md5.HashStretch(p.cfg.Platforms.Rumble.Password, salts.Data.Salts[0], 128), salts.Data.Salts[1])))
	hashes = append(hashes, md5.HashStretch(p.cfg.Platforms.Rumble.Password, salts.Data.Salts[2], 128))
	hashes = append(hashes, salts.Data.Salts[1])

	joinedHashes := strings.Join(hashes, ",")

	req, err = http.NewRequestWithContext(ctx, "POST", "https://rumble.com/service.php?name=user.login", strings.NewReader(fmt.Sprintf("username=%s&password_hashes=%s", p.cfg.Platforms.Rumble.Login, joinedHashes)))
	if err != nil {
		return err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	resp, err = p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ErrStatusCode
	}

	return p.cookies.Save(resp.Request.URL)
}
