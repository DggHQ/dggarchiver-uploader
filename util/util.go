package util

import (
	"reflect"
	"strings"
	"time"

	config "github.com/DggHQ/dggarchiver-config/uploader"
)

func CalculateEndTime(startTime string, duration int) (string, error) {
	parsed, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return "", err
	}

	endTimeParsed := parsed.Add(time.Second * time.Duration(duration))
	return endTimeParsed.Format(time.RFC3339), nil
}

func GetEnabledPlatforms(cfg *config.Config) []string {
	enabledPlatforms := []string{}

	platformsValue := reflect.ValueOf(cfg.Platforms)
	platformsFields := reflect.VisibleFields(reflect.TypeOf(cfg.Platforms))
	for _, field := range platformsFields {
		if platformsValue.FieldByName(field.Name).FieldByName("Enabled").Bool() {
			enabledPlatforms = append(enabledPlatforms, strings.ToLower(field.Name))
		}
	}

	return enabledPlatforms
}
