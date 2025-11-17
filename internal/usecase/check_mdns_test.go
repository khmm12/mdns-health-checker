package usecase

import (
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/khmm12/mdns-health-checker/internal/ports"
	portsm "github.com/khmm12/mdns-health-checker/internal/ports/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCheckMDNSUseCase_CountsUpAndDownState(t *testing.T) {
	ctx := t.Context()

	probe := portsm.NewMockMDNSProbe(t)
	publisher := portsm.NewMockMDNSStatePublisher(t)

	uc := newTestCheckMDNSUseCase(t, probe, publisher)

	probe.On("Probe", mock.Anything, "printer1.local", 10*time.Second).Return(ports.HostUp, nil)
	probe.On("Probe", mock.Anything, "printer2.local", 10*time.Second).Return(ports.HostDown, nil)

	publisher.On("Publish", mock.Anything, []string{"printer1.local"}, []string{"printer2.local"}).Return(nil)

	err := uc.Execute(ctx, CheckMDNSCommand{
		Hosts: []string{"printer1.local", "printer2.local"},
	})

	require.NoError(t, err)
}

func TestCheckMDNSUseCase_BubblesUpProbeError(t *testing.T) {
	ctx := t.Context()

	probe := portsm.NewMockMDNSProbe(t)
	publisher := portsm.NewMockMDNSStatePublisher(t)

	uc := newTestCheckMDNSUseCase(t, probe, publisher)

	probe.On("Probe", mock.Anything, "printer1.local", 10*time.Second).Return(ports.HostUnknown, errors.New("probe failed"))
	probe.On("Probe", mock.Anything, "printer2.local", 10*time.Second).Return(ports.HostUp, nil)

	err := uc.Execute(ctx, CheckMDNSCommand{
		Hosts: []string{"printer1.local", "printer2.local"},
	})

	require.ErrorContains(t, err, "failed to probe host")
	publisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)
}

func TestCheckMDNSUseCase_FailsOnUnknownState(t *testing.T) {
	ctx := t.Context()

	probe := portsm.NewMockMDNSProbe(t)
	publisher := portsm.NewMockMDNSStatePublisher(t)

	uc := newTestCheckMDNSUseCase(t, probe, publisher)

	probe.On("Probe", mock.Anything, "printer1.local", 10*time.Second).Return(ports.HostUnknown, nil)

	err := uc.Execute(ctx, CheckMDNSCommand{
		Hosts: []string{"printer1.local"},
	})

	require.ErrorContains(t, err, "unknown MDNS state for host")
	publisher.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)
}

func TestCheckMDNSUseCase_ReturnsErrorWhenPublishingFails(t *testing.T) {
	ctx := t.Context()

	probe := portsm.NewMockMDNSProbe(t)
	publisher := portsm.NewMockMDNSStatePublisher(t)

	uc := newTestCheckMDNSUseCase(t, probe, publisher)

	probe.On("Probe", mock.Anything, "printer1.local", 10*time.Second).Return(ports.HostUp, nil)
	probe.On("Probe", mock.Anything, "printer2.local", 10*time.Second).Return(ports.HostDown, nil)

	publisher.On("Publish", mock.Anything, []string{"printer1.local"}, []string{"printer2.local"}).Return(errors.New("publish failed"))

	err := uc.Execute(ctx, CheckMDNSCommand{
		Hosts: []string{"printer1.local", "printer2.local"},
	})

	require.ErrorContains(t, err, "failed to publish mdns check results")
}

func newTestCheckMDNSUseCase(t *testing.T, probe ports.MDNSProbe, publisher ports.MDNSStatePublisher) *CheckMDNSUseCase {
	t.Helper()

	return NewCheckMDNSUseCase(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		probe,
		publisher,
		10*time.Second,
	)
}
