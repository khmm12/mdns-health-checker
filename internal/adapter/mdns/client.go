package mdns

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/pion/mdns/v2"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/sync/semaphore"
)

type Client struct {
	logger      *slog.Logger
	conn        *mdns.Conn
	concurrency int
	sem         *semaphore.Weighted
}

func New(logger *slog.Logger, useIPv4, useIPv6 bool, ipv4Addr, ipv6Addr string, concurrency int) (*Client, error) {
	if concurrency <= 0 {
		return nil, fmt.Errorf("mdns: probe concurrency must be greater than zero")
	}

	conn, err := buildServer(useIPv4, useIPv6, ipv4Addr, ipv6Addr)
	if err != nil {
		return nil, err
	}

	return &Client{
		logger:      logger,
		conn:        conn,
		concurrency: concurrency,
		sem:         semaphore.NewWeighted(int64(concurrency)),
	}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func buildServer(useIPv4, useIPv6 bool, ipv4Addr, ipv6Addr string) (*mdns.Conn, error) {
	var err error

	var packetConnV4 *ipv4.PacketConn

	if useIPv4 {
		packetConnV4, err = buildV4Conn(ipv4Addr)
		if err != nil {
			return nil, err
		}
	}

	var packetConnV6 *ipv6.PacketConn
	if useIPv6 {
		packetConnV6, err = buildV6Conn(ipv6Addr)
		if err != nil {
			return nil, err
		}
	}

	server, err := mdns.Server(packetConnV4, packetConnV6, &mdns.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to init mdns server: %w", err)
	}

	return server, nil
}

func buildV4Conn(addr string) (*ipv4.PacketConn, error) {
	addr4, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve IPv4 address: %w", err)
	}

	l4, err := net.ListenUDP("udp4", addr4)
	if err != nil {
		return nil, fmt.Errorf("failed to bind UDP IPv4 listener: %w", err)
	}

	return ipv4.NewPacketConn(l4), nil
}

func buildV6Conn(addr string) (*ipv6.PacketConn, error) {
	addr6, err := net.ResolveUDPAddr("udp6", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve IPv6 address: %w", err)
	}

	l6, err := net.ListenUDP("udp6", addr6)
	if err != nil {
		return nil, fmt.Errorf("failed to bind UDP IPv6 listener: %w", err)
	}

	return ipv6.NewPacketConn(l6), nil
}
