# mdns-health-checker

Periodic mDNS probe runner that verifies the reachability of a list of `.local` hosts and exports their status as Prometheus metrics. It is intended to be deployed alongside devices that only advertise themselves over mDNS (e.g. home lab gear, printers, smart-home hubs) so their availability can be monitored just like any other service.

## :sparkles: Highlights

- Polls every configured host on a fixed cadence with independent timeouts.
- Exposes a Prometheus scrape endpoint.

## :gear: How It Works

1. `mdns-health-checker` starts an mDNS client bound to the requested multicast addresses.
2. A worker kicks off probe batches on the requested interval (the first run happens immediately after start-up).
3. Each host is queried.
4. Results are published to the Prometheus exporter, updating per-host and aggregate gauges.

## :rocket: Getting Started

### :white_check_mark: Prerequisites

- Go 1.25 or newer (managed automatically via [`mise`](https://mise.jdx.dev/) if you have it installed).
- Access to the multicast interfaces you want to probe (default: `224.0.0.0:5353` for IPv4 and `[FF02::]:5353` for IPv6).

### :hammer_and_wrench: Build

```sh
go build -o bin/mdns-health-checker ./cmd/mdns-health-checker
```

### :arrow_forward: Run

`--probe.hosts` (or `PROBE_HOSTS`) is required; pass a comma-separated list of mDNS hostnames without spaces.

```sh
go run ./cmd/mdns-health-checker --probe.hosts=printer.local,lab-switch.local
```

Alternatively, configure the same settings with environment variables:

```sh
PROBE_HOSTS=printer.local,lab-switch.local ./bin/mdns-health-checker
```

### :whale: Docker

A multi-architecture container image is published to the GitHub Container Registry (`ghcr.io/khmm12/mdns-health-checker`).

Run the container with environment variables for configuration. mDNS requires access to the local multicast interface, so prefer `--network host` (Linux) or the equivalent bridge setup that forwards multicast packets.

```sh
docker run --rm \
  --name mdns-health-checker \
  --network host \
  -e PROBE_HOSTS=printer.local,lab-switch.local \
  -e PROBE_INTERVAL=1m \
  -e PROBE_TIMEOUT=30s \
  ghcr.io/khmm12/mdns-health-checker:latest
```

Using Docker Compose:

```yaml
services:
  mdns-health-checker:
    image: ghcr.io/khmm12/mdns-health-checker:latest
    restart: unless-stopped
    network_mode: host
    environment:
      PROBE_HOSTS: printer.local,lab-switch.local
      PROBE_INTERVAL: 5m
      PROBE_TIMEOUT: 30s
```

Expose the metrics endpoint to Prometheus by pointing the scrape job to `http://<host>:8080/metrics`.

### :wrench: Configuration

All options can be supplied via CLI flags (shown below) or their corresponding environment variables.

| Flag                  | Environment         | Default          | Description                                                         |
| --------------------- | ------------------- | ---------------- | ------------------------------------------------------------------- |
| `--probe.interval`    | `PROBE_INTERVAL`    | `30s`            | Delay between probe cycles; must be greater than `--probe.timeout`. |
| `--probe.timeout`     | `PROBE_TIMEOUT`     | `10s`            | Maximum time to wait for a single host response.                    |
| `--probe.concurrency` | `PROBE_CONCURRENCY` | `10`             | Maximum simultaneous probes; controls the semaphore weight.         |
| `--probe.ipv4`        | `PROBE_USE_IPV4`    | `true`           | Enable IPv4 mDNS probing.                                           |
| `--probe.ipv4.addr`   | `PROBE_IPV4_ADDR`   | `224.0.0.0:5353` | UDP address to bind for IPv4 probes.                                |
| `--probe.ipv6`        | `PROBE_USE_IPV6`    | `true`           | Enable IPv6 mDNS probing.                                           |
| `--probe.ipv6.addr`   | `PROBE_IPV6_ADDR`   | `[FF02::]:5353`  | UDP address to bind for IPv6 probes.                                |
| `--probe.hosts`       | `PROBE_HOSTS`       | _(required)_     | Comma-separated list of mDNS hostnames to check.                    |
| `--metrics.addr`      | `METRICS_ADDR`      | `0.0.0.0:8080`   | TCP address for the HTTP server (metrics).                          |
| `--metrics.path`      | `METRICS_PATH`      | `/metrics`       | HTTP path exposing Prometheus metrics.                              |
| `--log.level`         | `LOG_LEVEL`         | `info`           | Log verbosity: `debug`, `info`, `warn`, `error`.                    |

Run `mdns-health-checker --help` to see usage text.

## :bar_chart: Observability

- **Health check**: `GET /health` returns `200 OK` with body `OK`.
- **Metrics** (all prefixed with `mdns_`):
  - `mdns_network_status`: `1` when at least one host is up, otherwise `0`.
  - `mdns_network_hosts_total`: count of hosts probed.
  - `mdns_network_hosts_up`: count of hosts that responded within the timeout.
  - `mdns_network_hosts_down`: count of hosts that timed out.
  - `mdns_network_host_status{host="<name>"}`: per-host gauge (`1` up, `0` down).

Scrape `http://<addr>/metrics` from Prometheus. Each scrape reflects the most recent probe cycle.

## :test_tube: Development

- Align local tool versions with `mise install`.
- Format and lint with `golangci-lint run` (see `.golangci.yml` for enabled checks and formatters).
- Run the binary under `go run ./cmd/mdns-health-checker` during development; the worker runs immediately, which is helpful for quick feedback.

## :bulb: Limitations & Tips

- The process must bind to the multicast addresses you choose.
- A host that never responded is considered `down` until the next successful probe; there is no exponential backoff.
- All metrics are gauges; if you need historical trends, rely on Prometheus recording rules or alerts.
