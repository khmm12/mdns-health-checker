package ports

import (
	"context"
	"time"
)

type HostState int

const (
	HostUnknown HostState = iota
	HostUp
	HostDown
)

type MDNSProbe interface {
	Probe(ctx context.Context, host string, timeout time.Duration) (HostState, error)
}
