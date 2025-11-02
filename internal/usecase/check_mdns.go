package usecase

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/khmm12/mdns-health-checker/internal/ports"
)

type CheckMDNSUseCase struct {
	logger    *slog.Logger
	publisher ports.MDNSStatePublisher
	probe     ports.MDNSProbe
	timeout   time.Duration
}

func NewCheckMDNSUseCase(logger *slog.Logger, probe ports.MDNSProbe, publisher ports.MDNSStatePublisher, timeout time.Duration) *CheckMDNSUseCase {
	return &CheckMDNSUseCase{
		logger:    logger,
		publisher: publisher,
		probe:     probe,
		timeout:   timeout,
	}
}

type CheckMDNSCommand struct {
	Hosts []string
}

func (u *CheckMDNSUseCase) Execute(ctx context.Context, cmd CheckMDNSCommand) error {
	var (
		mu   sync.Mutex
		up   = make([]string, 0, len(cmd.Hosts))
		down = make([]string, 0, len(cmd.Hosts))
	)

	g, gctx := errgroup.WithContext(ctx)

	for _, host := range cmd.Hosts {
		g.Go(func() error {
			state, err := u.probe.Probe(gctx, host, u.timeout)
			if err != nil {
				return fmt.Errorf("failed to probe host %s: %w", host, err)
			}

			mu.Lock()
			defer mu.Unlock()

			switch state {
			case ports.HostUp:
				up = append(up, host)
			case ports.HostDown:
				down = append(down, host)
			case ports.HostUnknown:
				return fmt.Errorf("unknown MDNS state for host %s: %d", host, state)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	err := u.publisher.Publish(ctx, up, down)
	if err != nil {
		return fmt.Errorf("failed to publish mdns check results: %w", err)
	}

	return nil
}
