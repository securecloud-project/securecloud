package checks

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"strings"
)

const maxTargetLength = 253

var blockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("::/128"),
	netip.MustParsePrefix("::1/128"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("ff00::/8"),
}

// NormalizeTarget accepts a DNS hostname or public IP only. URLs, ports,
// credentials, paths, and local names are deliberately rejected.
func NormalizeTarget(raw string) (string, error) {
	target := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(raw)), ".")
	if target == "" || len(target) > maxTargetLength {
		return "", errors.New("target must be between 1 and 253 characters")
	}
	if strings.ContainsAny(target, "/?#@") || strings.Contains(target, "://") {
		return "", errors.New("target must be a hostname without scheme, port, path, or credentials")
	}
	if strings.EqualFold(target, "localhost") || strings.HasSuffix(target, ".localhost") {
		return "", errors.New("local targets are not allowed")
	}
	if ip, err := netip.ParseAddr(target); err == nil {
		if !isPublicIP(ip) {
			return "", errors.New("private or reserved targets are not allowed")
		}
		return ip.String(), nil
	}
	if strings.Contains(target, ":") {
		return "", errors.New("target ports are not allowed")
	}
	labels := strings.Split(target, ".")
	if len(labels) < 2 {
		return "", errors.New("target must be a fully qualified hostname")
	}
	for _, label := range labels {
		if label == "" || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", errors.New("target contains an invalid DNS label")
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return "", errors.New("target contains an invalid DNS label")
			}
		}
	}
	return target, nil
}

func isPublicIP(ip netip.Addr) bool {
	if ip.Is4In6() {
		ip = ip.Unmap()
	}
	if !ip.IsValid() || !ip.IsGlobalUnicast() {
		return false
	}
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(ip) {
			return false
		}
	}
	return true
}

type resolver interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
}

// SecureDialer resolves a host, validates every returned address, then dials a
// validated IP directly. This prevents proxy use and closes DNS rebinding paths
// where validation and connection would otherwise perform separate lookups.
type SecureDialer struct {
	Resolver resolver
	Dialer   *net.Dialer
}

func (d SecureDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("split address: %w", err)
	}
	if _, err := NormalizeTarget(host); err != nil {
		return nil, err
	}
	lookup := d.Resolver
	if lookup == nil {
		lookup = net.DefaultResolver
	}
	addresses, err := lookup.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("resolve target: %w", err)
	}
	if len(addresses) == 0 {
		return nil, errors.New("target has no IP addresses")
	}
	for _, candidate := range addresses {
		parsed, ok := netip.AddrFromSlice(candidate.IP)
		if !ok || !isPublicIP(parsed) {
			return nil, fmt.Errorf("target resolved to blocked address %s", candidate.IP)
		}
	}
	dialer := d.Dialer
	if dialer == nil {
		dialer = &net.Dialer{}
	}
	var lastErr error
	for _, candidate := range addresses {
		connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
		if err == nil {
			return connection, nil
		}
		lastErr = err
	}
	return nil, fmt.Errorf("connect to target: %w", lastErr)
}
