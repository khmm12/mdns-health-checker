package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/khmm12/mdns-health-checker/internal/adapter/httpsrv"
	"github.com/khmm12/mdns-health-checker/internal/adapter/mdns"
	"github.com/khmm12/mdns-health-checker/internal/adapter/prometheus"
	"github.com/khmm12/mdns-health-checker/internal/adapter/worker"
	"github.com/khmm12/mdns-health-checker/internal/common/logging"
	"github.com/khmm12/mdns-health-checker/internal/usecase"
)

type Probe struct {
	Interval         time.Duration `name:"interval" env:"PROBE_INTERVAL" default:"30s" help:"The interval between each full cycle of mDNS host checks (e.g., 1s, 5m, 1h)."`
	Timeout          time.Duration `name:"timeout" env:"PROBE_TIMEOUT" default:"10s" help:"The maximum duration to wait for an mDNS probe response from a single host (e.g., 1s, 5m, 1h)."`
	ProbeConcurrency int           `name:"concurrency" env:"PROBE_CONCURRENCY" default:"10" help:"The maximum number of mDNS probes to run concurrently."`
	UseIPv4          bool          `name:"ipv4" env:"PROBE_USE_IPV4" default:"true" help:"Enable mDNS probing over IPv4. Enabled by default."`
	IPv4Addr         string        `name:"ipv4.addr" env:"PROBE_IPV4_ADDR" default:"224.0.0.0:5353" help:"IPv4 address to bind to for mDNS probing."`
	UseIPv6          bool          `name:"ipv6" env:"PROBE_USE_IPV6" default:"true" help:"Enable mDNS probing over IPv6. Enabled by default."`
	IPv6Addr         string        `name:"ipv6.addr" env:"PROBE_IPV6_ADDR" default:"[FF02::]:5353" help:"IPv6 address to bind to for mDNS probing."`
	Hosts            []string      `name:"hosts" env:"PROBE_HOSTS" required:"" sep:"," help:"A comma-separated list of mDNS hostnames (e.g., 'mydevice.local,another.local') to check."`
}

type Metrics struct {
	Addr string `name:"addr" env:"METRICS_ADDR" default:"0.0.0.0:8080" help:"HTTP Address to bind Prometheus metrics"`
	Path string `name:"path" env:"METRICS_PATH" default:"/metrics" help:"Path to serve Prometheus metrics"`
}

type Serve struct {
	Probe    Probe   `embed:"" prefix:"probe."`
	Metrics  Metrics `embed:"" prefix:"metrics."`
	LogLevel string  `name:"log.level" env:"LOG_LEVEL" default:"info" help:"Log level (debug, info, warn, error, fatal)"`
}

func serve(cli *CLI) error {
	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	logLevel, err := parseLogLevel(cli.Serve.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to parse to log level: %w", err)
	}

	logger := slog.New(logging.NewEnhancedHandler(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: logLevel,
		}),
	)).With(logging.NewProgramAttr())

	mdnsClient, err := mdns.New(
		logger,
		cli.Serve.Probe.UseIPv4,
		cli.Serve.Probe.UseIPv6,
		cli.Serve.Probe.IPv4Addr,
		cli.Serve.Probe.IPv6Addr,
		cli.Serve.Probe.ProbeConcurrency,
	)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to create mdns checker", logging.Error(err))
		return err
	}

	defer func() {
		logger.InfoContext(ctx, "Closing mdns client")
		_ = mdnsClient.Close()
	}()

	exporter, err := prometheus.NewExporter()
	if err != nil {
		logger.ErrorContext(ctx, "Failed to create prometheus exporter", logging.Error(err))
		return err
	}

	mdnsProbe := mdns.NewProbe(mdnsClient)

	uc := usecase.NewCheckMDNSUseCase(
		logger,
		mdnsProbe,
		prometheus.NewMDNSStatePublisher(logger, exporter),
		cli.Serve.Probe.Timeout,
	)

	httpsrv := httpsrv.NewServer(cli.Serve.Metrics.Addr, httpsrv.ServerOptions{
		MetricsHandler: exporter.Handler().ServeHTTP,
	})

	worker := worker.NewWorker(
		logger,
		cli.Serve.Probe.Interval,
		newTask(logger, uc, cli.Serve.Probe.Hosts),
	)

	defer func() {
		logger.InfoContext(ctx, "Stopping...")
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()

		logger.InfoContext(ctx, "Stopping Worker...")
		serr := worker.Shutdown(shutdownCtx)
		if serr != nil {
			logger.ErrorContext(ctx, "Failed to stop Worker", logging.Error(serr))
		}

		logger.InfoContext(ctx, "Stopping HTTP Server...")
		serr = httpsrv.Shutdown(shutdownCtx)
		if serr != nil {
			logger.ErrorContext(ctx, "Failed to stop HTTP Server", logging.Error(serr))
		}

		logger.InfoContext(ctx, "Stopped")
	}()

	errCh := make(chan error)

	go func() {
		logger.InfoContext(ctx, "Start HTTP Server", slog.String("address", httpsrv.ListenAddr()))

		closed := make(chan struct{})
		defer close(closed)

		go func() {
			select {
			case <-closed:
				return
			case <-time.Tick(1 * time.Second):
				logger.InfoContext(ctx, "Listening at "+httpsrv.ListenAddr())
			}
		}()

		err := httpsrv.Start()
		if err != nil {
			logger.ErrorContext(ctx, "Failed to start HTTP Server", logging.Error(err))
			errCh <- err
		}
	}()

	go func() {
		logger.InfoContext(ctx, "Start Worker", slog.Duration("interval", cli.Serve.Probe.Interval))

		err := worker.Start()
		if err != nil {
			logger.ErrorContext(ctx, "Failed to start Worker", logging.Error(err))
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return nil
	}
}

type taskUC interface {
	Execute(ctx context.Context, cmd usecase.CheckMDNSCommand) error
}

type task struct {
	logger *slog.Logger
	uc     taskUC
	hosts  []string
}

func newTask(logger *slog.Logger, uc taskUC, hosts []string) *task {
	return &task{
		logger: logger,
		uc:     uc,
		hosts:  hosts,
	}
}

func (t *task) Execute(ctx context.Context) error {
	now := time.Now()

	t.logger.InfoContext(ctx, "Run MDNS check")

	err := t.uc.Execute(ctx, usecase.CheckMDNSCommand{
		Hosts: t.hosts,
	})

	if err != nil {
		t.logger.ErrorContext(ctx, "Failed to execute mdns check", logging.Error(err), slog.Duration("duration", time.Since(now)))
	} else {
		t.logger.InfoContext(ctx, "Finished mdns check", slog.Duration("duration", time.Since(now)))
	}

	return nil
}

func (c *CLI) Validate() error {
	var errs []error

	s := &c.Serve
	p := &s.Probe

	if p.Interval <= 0 {
		errs = append(errs, fmt.Errorf("--probe.interval: must be greater than zero"))
	}

	if p.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("--probe.timeout: must be greater than zero"))
	}

	if p.Interval <= p.Timeout {
		errs = append(errs, fmt.Errorf("--probe.interval: must be greater than --probe.timeout"))
	}

	if p.ProbeConcurrency <= 0 {
		errs = append(errs, fmt.Errorf("--probe.concurrency: must be greater than zero"))
	}

	if !p.UseIPv4 && !p.UseIPv6 {
		errs = append(errs, errors.New("at least one of --probe.ipv4 or --probe.ipv6 must be enabled"))
	}

	if !isIP4Addr(p.IPv4Addr) {
		errs = append(errs, fmt.Errorf("--probe.ipv4: must be a valid UDP IPv4 address e.g. 224.0.0.0:5353"))
	}

	if !isUDP4AddrResolvable(p.IPv4Addr) {
		errs = append(errs, fmt.Errorf("--probe.ipv4: must be resolvable"))
	}

	if !isIP6Addr(p.IPv6Addr) {
		errs = append(errs, fmt.Errorf("--probe.ipv6: must be a valid UDP IPv6 address e.g. [FF02::]:5353"))
	}

	if !isUDP6AddrResolvable(p.IPv6Addr) {
		errs = append(errs, fmt.Errorf("--probe.ipv6: must be resolvable"))
	}

	if !isTCPAddr(s.Metrics.Addr) {
		errs = append(errs, fmt.Errorf("--metrics.addr: must be a valid tcp listening address (e.g. 0.0.0.0:8080)"))
	}

	if !isLogLevel(s.LogLevel) {
		errs = append(errs, fmt.Errorf("--log.level: must be one of debug, info, warn, error"))
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

func parseLogLevel(levelStr string) (slog.Level, error) {
	switch strings.ToLower(levelStr) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.Level(-1), fmt.Errorf("invalid log level: %s", levelStr)
	}
}
