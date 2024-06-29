package odysee

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	config "github.com/DggHQ/dggarchiver-config/uploader"
	"github.com/DggHQ/dggarchiver-uploader/monitoring"
	"github.com/DggHQ/dggarchiver-uploader/platforms/implementation"
)

const (
	platformName        string = "odysee"
	mainAPIURLString    string = "https://api.odysee.com"
	backendAPIURLString string = "https://api.na-backend.odysee.com/api"
	appID               string = "odyseecom692EAWhtoqDuAfQ6KHMXxFxt8tkhmt7sfprEMHWKjy5hf6PwZcHDV542V"
)

var ErrStatusCode = errors.New("server returned not OK")

func init() {
	implementation.Map[platformName] = New
}

type odyseeUserMe struct {
	*MainAPIResponse
	Data struct {
		PrimaryEmail string `json:"primary_email"`
	} `json:"data"`
}

type odyseeUserNew struct {
	*MainAPIResponse
	Data struct {
		AuthToken string `json:"auth_token"`
	} `json:"data"`
}

type MainAPIResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

type Platform struct {
	cfg     *config.Config
	client  http.Client
	monitor *monitoring.Monitor

	authToken string
}

func New(cfg *config.Config, monitor *monitoring.Monitor) (implementation.Platform, error) {
	return &Platform{
		cfg:     cfg,
		monitor: monitor,
		client:  http.Client{},

		authToken: "",
	}, nil
}

func (p *Platform) IsLoggedIn(ctx context.Context) (bool, error) {
	val := url.Values{}
	val.Set("auth_token", p.authToken)

	req, err := http.NewRequestWithContext(ctx, "POST", mainAPIURLString+"/user/me", strings.NewReader(val.Encode()))
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

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, err
	}

	var check odyseeUserMe
	err = json.Unmarshal(b, &check)
	if err != nil {
		return false, err
	}

	return check.Data.PrimaryEmail == p.cfg.Platforms.Odysee.Email, nil
}

func (p *Platform) Login(ctx context.Context) error {
	val1 := url.Values{}
	val1.Set("auth_token", "")
	val1.Set("language", "en")
	val1.Set("app_id", appID)

	req, err := http.NewRequestWithContext(ctx, "POST", mainAPIURLString+"/user/new", strings.NewReader(val1.Encode()))
	if err != nil {
		return err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return errors.Join(ErrStatusCode, fmt.Errorf("%d %s", resp.StatusCode, string(b)))
	}

	var newAuthToken odyseeUserNew
	err = json.Unmarshal(b, &newAuthToken)
	if err != nil {
		return err
	}

	if !newAuthToken.Success {
		return errors.New(newAuthToken.Error)
	}

	val2 := url.Values{}
	val2.Set("auth_token", newAuthToken.Data.AuthToken)
	val2.Set("email", p.cfg.Platforms.Odysee.Email)
	val2.Set("password", p.cfg.Platforms.Odysee.Password)

	req, err = http.NewRequestWithContext(ctx, "POST", mainAPIURLString+"/user/signin", strings.NewReader(val2.Encode()))
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
		return errors.Join(ErrStatusCode, fmt.Errorf("%d %s", resp.StatusCode, string(b)))
	}

	p.authToken = newAuthToken.Data.AuthToken

	return nil
}
