package extractor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// List of blocked CIDR ranges representing private, loopback, link-local, multicast,
// and reserved IP spaces across IPv4 and IPv6.
var blockedCIDRs = []string{
	// IPv4 loopback & unspecified
	"0.0.0.0/8",
	"127.0.0.0/8",
	"255.255.255.255/32",

	// IPv4 private networks (RFC 1918)
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",

	// IPv4 link-local & cloud metadata (RFC 3927)
	"169.254.0.0/16",

	// IPv4 Carrier-Grade NAT (RFC 6598)
	"100.64.0.0/10",

	// IPv4 documentation & benchmark (RFC 5737, RFC 2544)
	"192.0.0.0/24",
	"192.0.2.0/24",
	"198.18.0.0/15",
	"198.51.100.0/24",
	"203.0.113.0/24",

	// IPv4 multicast & reserved
	"224.0.0.0/4",
	"240.0.0.0/4",

	// IPv6 loopback, unspecified & unique local
	"::/128",
	"::1/128",
	"fc00::/7",

	// IPv6 link-local
	"fe80::/10",

	// IPv6 multicast
	"ff00::/8",

	// IPv6 documentation
	"2001:db8::/32",

	// IPv6 transition mechanisms that can embed arbitrary (including private)
	// IPv4 addresses: NAT64 (RFC 6052/8215), 6to4 (RFC 3056), Teredo (RFC 4380)
	"64:ff9b::/96",
	"64:ff9b:1::/48",
	"2002::/16",
	"2001::/32",

	// IPv6 discard-only (RFC 6666)
	"100::/64",
}

var parsedBlockedPrefixes []netip.Prefix

func init() {
	for _, cidr := range blockedCIDRs {
		prefix, err := netip.ParsePrefix(cidr)
		if err != nil {
			panic(fmt.Sprintf("invalid static blocked CIDR %q: %v", cidr, err))
		}
		parsedBlockedPrefixes = append(parsedBlockedPrefixes, prefix)
	}
}

// IsBlockedIP checks whether an IP address belongs to any loopback, private,
// link-local, multicast, or reserved range.
func IsBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}

	return IsBlockedAddr(addr)
}

// IsBlockedAddr checks whether a netip.Addr is restricted.
// Handles IPv4-mapped IPv6 addresses by unmapping first.
func IsBlockedAddr(addr netip.Addr) bool {
	// Unmap IPv4-in-IPv6 (e.g. ::ffff:127.0.0.1 -> 127.0.0.1)
	if addr.Is4In6() {
		addr = addr.Unmap()
	}

	if addr.IsLoopback() || addr.IsPrivate() || addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() || addr.IsMulticast() || addr.IsUnspecified() {
		return true
	}

	for _, prefix := range parsedBlockedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

// ValidateURL checks that a URL is a valid, absolute HTTP or HTTPS URL
// and does not point directly to a loopback, local hostname, or blocked IP literal.
func ValidateURL(rawURL string) (*url.URL, error) {
	if strings.TrimSpace(rawURL) == "" {
		return nil, errors.New("url cannot be empty")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return nil, fmt.Errorf("disallowed scheme %q: only http and https are permitted", parsed.Scheme)
	}

	host := parsed.Hostname()
	if host == "" {
		return nil, errors.New("url host is empty")
	}

	// Check for common local / loopback hostnames
	lowerHost := strings.ToLower(host)
	if lowerHost == "localhost" ||
		strings.HasSuffix(lowerHost, ".localhost") ||
		strings.HasSuffix(lowerHost, ".local") ||
		strings.HasSuffix(lowerHost, ".internal") {
		return nil, fmt.Errorf("access to local hostname %q is blocked", host)
	}

	// If hostname is an IP literal, validate immediately
	if ip := net.ParseIP(host); ip != nil {
		if IsBlockedIP(ip) {
			return nil, fmt.Errorf("access to restricted IP address %q is blocked", host)
		}
	}

	return parsed, nil
}

// SafeDialContext creates a net.DialContext function that resolves DNS and verifies
// all resolved IPs against the blocked ranges before connecting, mitigating DNS rebinding.
func SafeDialContext(dialTimeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, fmt.Errorf("invalid address %q: %w", addr, err)
		}

		// Resolve IP addresses using system / context resolver
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, fmt.Errorf("dns resolution failed for %q: %w", host, err)
		}
		if len(ips) == 0 {
			return nil, fmt.Errorf("no IP addresses resolved for %q", host)
		}

		// Security guard: verify that none of the resolved IPs are private or restricted
		var safeIPs []net.IP
		for _, ip := range ips {
			if IsBlockedIP(ip) {
				return nil, fmt.Errorf("host %q resolved to restricted IP %s (access blocked)", host, ip.String())
			}
			safeIPs = append(safeIPs, ip)
		}

		if len(safeIPs) == 0 {
			return nil, fmt.Errorf("no safe public IP addresses available for %q", host)
		}

		// Dial the vetted IPs directly (never re-resolving the hostname) to
		// prevent DNS rebinding, falling back across addresses so a host whose
		// first record is unreachable (e.g. IPv6 without a route) still works.
		dialer := &net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: 30 * time.Second,
		}

		var lastErr error
		for _, ip := range safeIPs {
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
			if ctx.Err() != nil {
				break
			}
		}
		return nil, lastErr
	}
}

// SafeCheckRedirect creates an http.Client.CheckRedirect function enforcing maximum
// hops and validating every redirect URL scheme and host against SSRF.
func SafeCheckRedirect(maxRedirects int) func(req *http.Request, via []*http.Request) error {
	if maxRedirects <= 0 {
		maxRedirects = 5
	}

	return func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return fmt.Errorf("stopped after %d redirects (redirect loop or limit exceeded)", maxRedirects)
		}

		if _, err := ValidateURL(req.URL.String()); err != nil {
			return fmt.Errorf("redirect blocked: %w", err)
		}

		return nil
	}
}
