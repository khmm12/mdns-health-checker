package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Exporter struct {
	reg     *prometheus.Registry
	metrics *metrics
}

func NewExporter() (*Exporter, error) {
	reg := prometheus.NewRegistry()

	metrics, err := newMetrics(reg)
	if err != nil {
		return nil, err
	}

	return &Exporter{
		reg:     reg,
		metrics: metrics,
	}, nil
}

func (e *Exporter) Handler() http.Handler {
	return promhttp.HandlerFor(e.reg, promhttp.HandlerOpts{})
}
