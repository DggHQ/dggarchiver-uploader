package token

import (
	"io"
	"os"
)

type Token struct {
	t        string
	f        *os.File
	filePath string
}

func New() (*Token, error) {
	t := &Token{
		filePath: "odyseetoken.txt",
	}

	var err error
	t.f, err = os.Open(t.filePath)
	if err != nil {
		t.f, err = os.Create(t.filePath)
		if err != nil {
			return nil, err
		}
	}

	return t, nil
}

func (t *Token) Get() string {
	return t.t
}

func (t *Token) Load() error {
	b, err := io.ReadAll(t.f)
	if err != nil {
		return err
	}

	t.t = string(b)

	return nil
}

func (t *Token) Save(token string) error {
	t.t = token

	err := os.WriteFile(t.filePath, []byte(token), 0o644)
	if err != nil {
		return err
	}

	return nil
}
