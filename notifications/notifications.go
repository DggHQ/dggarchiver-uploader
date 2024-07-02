package notifications

import (
	"bytes"
	"strings"
	"text/template"
	"time"

	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
	"github.com/DggHQ/dggarchiver-uploader/util"
)

var (
	receive = strings.Join([]string{
		"Platform: {{ .Platform }}",
		"ID: {{ .VID }}",
		"Title: {{ .Title }}",
		"Duration: {{ duration .Duration }}",
		"Start Time: {{ .StartTime }}",
		"End Time: {{ endTime .StartTime .Duration }}",
	}, "\n")

	progress = strings.Join([]string{
		"Hosting: {{ .HostingPlatform }}",
		"Platform: {{ .Platform }}",
		"ID: {{ .VID }}",
		"Title: {{ .Title }}",
		"Progress: {{ .Progress }}%",
	}, "\n")

	insert = strings.Join([]string{
		"Hosting: {{ .HostingPlatform }}",
		"Platform: {{ .Platform }}",
		"ID: {{ .VID }}",
		"Title: {{ .Title }}",
		"URL: {{ .HostingURL }}",
	}, "\n")

	filtered = strings.Join([]string{
		"Platform: {{ .Platform }}",
		"ID: {{ .VID }}",
		"Title: {{ .Title }}",
		"Filter: {{ .Filter }}",
	}, "\n")

	funcMap = template.FuncMap{
		"duration": func(dur int) string {
			d := time.Duration(dur * int(time.Second))
			return d.String()
		},
		"endTime": func(startTime string, dur int) string {
			endTime, err := util.CalculateEndTime(startTime, dur)
			if err != nil {
				return ""
			}
			return endTime
		},
	}
)

var (
	receiveTemplate, _  = template.New("receive").Funcs(funcMap).Parse(receive)
	progressTemplate, _ = template.New("progress").Funcs(funcMap).Parse(progress)
	insertTemplate, _   = template.New("insert").Funcs(funcMap).Parse(insert)
	filteredTemplate, _ = template.New("filtered").Funcs(funcMap).Parse(filtered)
)

func GetReceiveMessage(vod *dggarchivermodel.VOD) string {
	var b bytes.Buffer

	_ = receiveTemplate.Execute(&b, vod)

	return b.String()
}

type p struct {
	*dggarchivermodel.VOD
	HostingPlatform string
	Progress        float64
}

func GetProgressMessage(vod *dggarchivermodel.VOD, hosting string, progress float64) string {
	var b bytes.Buffer

	_ = progressTemplate.Execute(&b, p{
		VOD:             vod,
		HostingPlatform: hosting,
		Progress:        progress,
	})

	return b.String()
}

func GetInsertMessage(vod *dggarchivermodel.UploadedVOD) string {
	var b bytes.Buffer

	_ = insertTemplate.Execute(&b, vod)

	return b.String()
}

type f struct {
	*dggarchivermodel.VOD
	Filter string
}

func GetFilteredMessage(vod *dggarchivermodel.VOD, filter string) string {
	var b bytes.Buffer

	_ = filteredTemplate.Execute(&b, f{
		VOD:    vod,
		Filter: filter,
	})

	return b.String()
}
