package cookies

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
)

type Cookies struct {
	jar      *cookiejar.Jar
	cookies  map[string][]*http.Cookie
	filePath string
	f        *os.File
}

func New(path string) (*Cookies, error) {
	var (
		c = &Cookies{
			cookies: map[string][]*http.Cookie{},
		}
		err error
	)

	c.filePath = path
	if c.filePath == "" {
		c.filePath = "cookiejar.json"
	}

	c.f, err = os.Open(c.filePath)
	if err != nil {
		c.f, err = os.Create(c.filePath)
		if err != nil {
			return nil, err
		}
	}

	c.jar, err = cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Cookies) GetJar() *cookiejar.Jar {
	return c.jar
}

func (c *Cookies) Load() error {
	b, err := io.ReadAll(c.f)
	if err != nil {
		return err
	}

	_ = json.Unmarshal(b, &c.cookies)
	for k, v := range c.cookies {
		u, _ := url.Parse(k)
		c.jar.SetCookies(u, v)
	}

	return nil
}

func (c *Cookies) Save(u *url.URL) error {
	urlCookies := c.jar.Cookies(u)
	c.cookies[u.String()] = urlCookies

	b, err := json.Marshal(&c.cookies)
	if err != nil {
		return err
	}

	err = os.WriteFile(c.filePath, b, 0o644)
	if err != nil {
		return err
	}

	return nil
}
