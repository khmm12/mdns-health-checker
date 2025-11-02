package prometheus

import (
	"context"
	"log/slog"
)

type MDNSStatePublisher struct {
	logger   *slog.Logger
	exporter *Exporter
}

func NewMDNSStatePublisher(logger *slog.Logger, exporter *Exporter) *MDNSStatePublisher {
	return &MDNSStatePublisher{
		logger:   logger,
		exporter: exporter,
	}
}

func (p *MDNSStatePublisher) Publish(ctx context.Context, up, down []string) error {
	p.logger.DebugContext(ctx, "Publishing mdns check results",
		slog.Group("publish",
			slog.Int("up_hosts", len(up)),
			slog.Int("down_hosts", len(down)),
		))

	total := len(up) + len(down)
	if total == 0 {
		p.logger.DebugContext(ctx, "No hosts found for mdns check")
		return nil
	}

	var status float64
	if len(up) > 0 {
		status = 1.0
	}

	m := p.exporter.metrics

	m.networkStatus.Set(status)
	m.networkHostsTotal.Set(float64(len(up) + len(down)))
	m.networkHostsUp.Set(float64(len(up)))
	m.networkHostsDown.Set(float64(len(down)))

	for _, host := range up {
		m.networkHostStatus.WithLabelValues(host).Set(1.0)
	}

	for _, host := range down {
		m.networkHostStatus.WithLabelValues(host).Set(0.0)
	}

	return nil
}
