package main

import (
	"net"
	"strings"
)

func isUDP4AddrResolvable(val string) bool {
	if !isIP4Addr(val) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp4", val)

	return err == nil
}

func isUDP6AddrResolvable(val string) bool {
	if !isIP6Addr(val) {
		return false
	}

	_, err := net.ResolveUDPAddr("udp6", val)

	return err == nil
}

func isIP4Addr(val string) bool {
	if idx := strings.LastIndex(val, ":"); idx != -1 {
		val = val[0:idx]
	}

	ip := net.ParseIP(val)

	return ip != nil && ip.To4() != nil
}

func isIP6Addr(val string) bool {
	if idx := strings.LastIndex(val, ":"); idx != -1 {
		if idx != 0 && val[idx-1:idx] == "]" {
			val = val[1 : idx-1]
		}
	}

	ip := net.ParseIP(val)

	return ip != nil && ip.To4() == nil
}

func isTCPAddr(val string) bool {
	if !isIP4Addr(val) && !isIP6Addr(val) {
		return false
	}

	_, err := net.ResolveTCPAddr("tcp", val)

	return err == nil
}

func isLogLevel(val string) bool {
	return val == "debug" || val == "info" || val == "warn" || val == "error"
}
