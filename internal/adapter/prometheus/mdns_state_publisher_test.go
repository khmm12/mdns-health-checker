package prometheus

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestMDNSStatePublisher_PublishMetricsForUpAndDownHosts(t *testing.T) {
	ctx := context.Background()
	exporter, publisher := newTestPublisher(t)

	err := publisher.Publish(ctx, []string{"host-up"}, []string{"host-down-1", "host-down-2"})
	require.NoError(t, err)

	requireMetric(t, 1.0, exporter.metrics.networkStatus)
	requireMetric(t, 3.0, exporter.metrics.networkHostsTotal)
	requireMetric(t, 1.0, exporter.metrics.networkHostsUp)
	requireMetric(t, 2.0, exporter.metrics.networkHostsDown)
	requireMetric(t, 1.0, exporter.metrics.networkHostStatus.WithLabelValues("host-up"))
	requireMetric(t, 0.0, exporter.metrics.networkHostStatus.WithLabelValues("host-down-1"))
	requireMetric(t, 0.0, exporter.metrics.networkHostStatus.WithLabelValues("host-down-2"))
}

func TestMDNSStatePublisher_PublishFailureWhenAllDown(t *testing.T) {
	ctx := context.Background()
	exporter, publisher := newTestPublisher(t)

	err := publisher.Publish(ctx, nil, []string{"host-down"})
	require.NoError(t, err)

	requireMetric(t, 0.0, exporter.metrics.networkStatus)
	requireMetric(t, 1.0, exporter.metrics.networkHostsTotal)
	requireMetric(t, 0.0, exporter.metrics.networkHostsUp)
	requireMetric(t, 1.0, exporter.metrics.networkHostsDown)
	requireMetric(t, 0.0, exporter.metrics.networkHostStatus.WithLabelValues("host-down"))
}

func TestMDNSStatePublisher_PublishNoHostsNoop(t *testing.T) {
	ctx := context.Background()
	exporter, publisher := newTestPublisher(t)

	err := publisher.Publish(ctx, nil, nil)
	require.NoError(t, err)

	requireMetric(t, 0.0, exporter.metrics.networkStatus)
	requireMetric(t, 0.0, exporter.metrics.networkHostsTotal)
	requireMetric(t, 0.0, exporter.metrics.networkHostsUp)
	requireMetric(t, 0.0, exporter.metrics.networkHostsDown)
}

func newTestPublisher(t *testing.T) (*Exporter, *MDNSStatePublisher) {
	t.Helper()

	exporter, err := NewExporter()
	require.NoError(t, err)

	publisher := NewMDNSStatePublisher(slog.New(slog.NewTextHandler(io.Discard, nil)), exporter)

	return exporter, publisher
}

func requireMetric(t *testing.T, expected float64, metric prometheus.Collector) {
	t.Helper()

	require.InDelta(t, expected, testutil.ToFloat64(metric), 0.001)
}
