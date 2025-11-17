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

	innerCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, _, err := p.client.conn.QueryAddr(innerCtx, host)
	if err != nil {
		// If the parent context was canceled due to the deadline error, early return the error as-is.
		// Helps to distinguish between the parent context being canceled with timeout and the query timing out.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return ports.HostUnknown, ctx.Err()
		}

		// If the query failed due to a timeout, consider the host as down.
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(innerCtx.Err(), context.DeadlineExceeded) {
			return ports.HostDown, nil
		}

		// If the query failed for any other reason, return an error.
		return ports.HostUnknown, err
	}

	return ports.HostUp, nil
}
