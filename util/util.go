package util

import (
	"time"
)

func CalculateEndTime(startTime string, duration int) (string, error) {
	parsed, err := time.Parse(time.RFC3339, startTime)
	if err != nil {
		return "", err
	}

	endTimeParsed := parsed.Add(time.Second * time.Duration(duration))
	return endTimeParsed.Format(time.RFC3339), nil
}
