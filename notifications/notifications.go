package notifications

import (
	"bytes"
	"text/template"

	dggarchivermodel "github.com/DggHQ/dggarchiver-model"
)

const (
	receive string = `Platform: '{{ .Platform }}'
		ID: '{{ .VID }}'
		Title: '{{ .Title }}'
		Duration: {{ .Duration }}
		Start Time: '{{ .StartTime }}'
	`
	progress string = `Hosting: '{{ .HostingPlatform }}'
		Platform: '{{ .Platform }}'
		ID: '{{ .VID }}'
		Title: '{{ .Title }}'

		Progress: {{ .Progress }}
	`
	insert string = `Hosting: '{{ .HostingPlatform }}'
		Platform: '{{ .Platform }}'
		ID: '{{ .VID }}'
		Title: '{{ .Title }}'
		URL: '{{ .HostingURL }}'
	`
	filtered string = `Platform: '{{ .Platform }}'
		ID: '{{ .VID }}'
		Title: '{{ .Title }}'

		Filter: '{{ .Filter }}'
	`
)

var (
	receiveTemplate, _  = template.New("receive").Parse(receive)
	progressTemplate, _ = template.New("progress").Parse(progress)
	insertTemplate, _   = template.New("insert").Parse(insert)
	filteredTemplate, _ = template.New("filtered").Parse(filtered)
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
