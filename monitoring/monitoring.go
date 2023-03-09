package monitoring

import (
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Monitor struct {
	CurrentProgress *prometheus.GaugeVec
}

func (m *Monitor) Run() {
	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (m *Monitor) Init() {
	m.CurrentProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "dgghq",
			Subsystem: "archiver_upload",
			Name:      "upload_progress",
			Help:      "Current uploader progress value. Value ",
		},
		[]string{
			"id",
			"channel_name",
			"vod_title",
		},
	)
	prometheus.MustRegister(m.CurrentProgress)
}

func (m *Monitor) ChangeCurrentProgress(value float64, labels prometheus.Labels) {
	m.CurrentProgress.With(labels).Set(value)
}