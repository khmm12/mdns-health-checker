package mdns

import (
	"context"
	"errors"
	"time"

	"github.com/khmm12/mdns-health-checker/internal/ports"
)

type Probe struct {
	client *Client
}

func NewProbe(client *Client) *Probe {
	return &Probe{client: client}
}

func (p *Probe) Probe(ctx context.Context, host string, timeout time.Duration) (ports.HostState, error) {
	if err := p.client.sem.Acquire(ctx, 1); err != nil {
		return ports.HostUnknown, err
	}

	defer p.client.sem.Release(1)

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, _, err := p.client.conn.QueryAddr(ctx, host)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ports.HostDown, nil
		}

		return ports.HostUnknown, err
	}

	return ports.HostUp, nil
}
