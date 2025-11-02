package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	networkStatus     prometheus.Gauge
	networkHostsTotal prometheus.Gauge
	networkHostsUp    prometheus.Gauge
	networkHostsDown  prometheus.Gauge
	networkHostStatus *prometheus.GaugeVec
}

const (
	prefix = "mdns_"
)

func newMetrics(reg *prometheus.Registry) (*metrics, error) {
	m := &metrics{
		networkStatus: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prefix + "network_status",
			Help: "Status of the network (1: success, 0: failure)",
		}),
		networkHostsTotal: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prefix + "network_hosts_total",
			Help: "Total number of hosts on the network",
		}),
		networkHostsUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prefix + "network_hosts_up",
			Help: "Number of hosts up on the network",
		}),
		networkHostsDown: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: prefix + "network_hosts_down",
			Help: "Number of hosts down on the network",
		}),
		networkHostStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: prefix + "network_host_status",
			Help: "Status of a specific host (1: up, 0: down)",
		}, []string{"host"}),
	}

	err := register(reg,
		m.networkStatus,
		m.networkHostsTotal,
		m.networkHostsUp,
		m.networkHostsDown,
		m.networkHostStatus,
	)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func register(r *prometheus.Registry, cs ...prometheus.Collector) error {
	for i, c := range cs {
		if err := r.Register(c); err != nil {
			for _, c := range cs[:i] {
				r.Unregister(c)
			}

			return err
		}
	}

	return nil
}
