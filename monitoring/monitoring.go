package monitoring

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Monitor struct {
	CurrentProgress *prometheus.GaugeVec
}

func New() *Monitor {
	m := &Monitor{}

	m.CurrentProgress = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "dgghq",
			Subsystem: "archiver",
			Name:      "upload_ratio",
			Help:      "Current uploader progress value. Value from 0-100",
		},
		[]string{
			"id",
			"channel_name",
			"vod_title",
		},
	)
	prometheus.MustRegister(m.CurrentProgress)

	return m
}

func (m *Monitor) Run() {
	http.Handle("/metrics", promhttp.Handler())
	slog.Error("prometheus http error", slog.Any("err", http.ListenAndServe(":8080", nil)))
	os.Exit(1)
}

func (m *Monitor) ChangeCurrentProgress(value float64, labels prometheus.Labels) {
	m.CurrentProgress.With(labels).Set(value)
}
