package extractor

import (
	"net"
	"net/http"
	"net/url"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		blocked bool
	}{
		// IPv4 Loopback & Unspecified
		{"ipv4 loopback 127.0.0.1", "127.0.0.1", true},
		{"ipv4 loopback 127.1.2.3", "127.1.2.3", true},
		{"ipv4 unspecified 0.0.0.0", "0.0.0.0", true},
		{"ipv4 broadcast 255.255.255.255", "255.255.255.255", true},

		// IPv4 Private (RFC 1918)
		{"ipv4 private 10.0.0.1", "10.0.0.1", true},
		{"ipv4 private 10.255.255.254", "10.255.255.254", true},
		{"ipv4 private 172.16.0.1", "172.16.0.1", true},
		{"ipv4 private 172.31.255.254", "172.31.255.254", true},
		{"ipv4 private 192.168.0.1", "192.168.0.1", true},
		{"ipv4 private 192.168.1.100", "192.168.1.100", true},

		// IPv4 Link-local (Cloud metadata)
		{"aws metadata 169.254.169.254", "169.254.169.254", true},
		{"ipv4 link-local 169.254.1.1", "169.254.1.1", true},

		// Carrier-Grade NAT (RFC 6598)
		{"cgnat 100.64.0.1", "100.64.0.1", true},
		{"cgnat 100.127.255.254", "100.127.255.254", true},

		// IPv6 Loopback & Private
		{"ipv6 loopback ::1", "::1", true},
		{"ipv6 unspecified ::", "::", true},
		{"ipv6 unique local fc00::1", "fc00::1", true},
		{"ipv6 unique local fd00::1", "fd00::1", true},
		{"ipv6 link-local fe80::1", "fe80::1", true},

		// IPv4-mapped IPv6
		{"ipv4-mapped ipv6 loopback", "::ffff:127.0.0.1", true},
		{"ipv4-mapped ipv6 metadata", "::ffff:169.254.169.254", true},
		{"ipv4-mapped ipv6 private", "::ffff:192.168.1.1", true},

		// Safe Public IPs
		{"google dns 8.8.8.8", "8.8.8.8", false},
		{"cloudflare dns 1.1.1.1", "1.1.1.1", false},
		{"quad9 9.9.9.9", "9.9.9.9", false},
		{"example.com 93.184.216.34", "93.184.216.34", false},
		{"cloudflare ipv6 2606:4700:4700::1111", "2606:4700:4700::1111", false},
		{"google ipv6 2001:4860:4860::8888", "2001:4860:4860::8888", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			parsed := net.ParseIP(tc.ip)
			if parsed == nil {
				t.Fatalf("failed to parse IP %q", tc.ip)
			}
			got := IsBlockedIP(parsed)
			if got != tc.blocked {
				t.Errorf("IsBlockedIP(%q) = %v, want %v", tc.ip, got, tc.blocked)
			}
		})
	}
}

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		// Valid URLs
		{"valid https", "https://example.com/article?id=123", false},
		{"valid http", "http://example.com/path", false},
		{"valid public ip", "https://93.184.216.34/path", false},

		// Disallowed schemes
		{"file scheme", "file:///etc/passwd", true},
		{"ftp scheme", "ftp://ftp.example.com/file", true},
		{"gopher scheme", "gopher://gopher.example.com", true},
		{"data scheme", "data:text/html,<h1>test</h1>", true},
		{"javascript scheme", "javascript:alert(1)", true},
		{"empty scheme", "//example.com/path", true},

		// Blocked hostnames & IPs
		{"localhost", "http://localhost/admin", true},
		{"localhost uppercase", "http://LOCALHOST:8080/metrics", true},
		{"sub.localhost", "http://app.localhost/", true},
		{"dot local", "http://mydevice.local/", true},
		{"dot internal", "http://service.internal/", true},
		{"ip loopback", "http://127.0.0.1:8080/debug", true},
		{"ip private 10.x", "http://10.0.0.1/", true},
		{"ip private 192.168.x", "http://192.168.1.1:8443", true},
		{"cloud metadata", "http://169.254.169.254/latest/meta-data/", true},

		// Malformed
		{"empty url", "", true},
		{"whitespace only", "   ", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateURL(tc.url)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateURL(%q) err = %v, wantErr = %v", tc.url, err, tc.wantErr)
			}
		})
	}
}

func TestSafeCheckRedirect(t *testing.T) {
	checkRedirect := SafeCheckRedirect(3)

	// Hop 1: safe redirect
	req1, _ := http.NewRequest("GET", "https://example.com/next", nil)
	via1 := []*http.Request{req1}
	if err := checkRedirect(req1, via1); err != nil {
		t.Errorf("unexpected error on safe redirect: %v", err)
	}

	// Malicious redirect to cloud metadata
	reqBad, _ := http.NewRequest("GET", "http://169.254.169.254/secret", nil)
	viaBad := []*http.Request{req1}
	if err := checkRedirect(reqBad, viaBad); err == nil {
		t.Errorf("expected error when redirecting to 169.254.169.254, got nil")
	}

	// Malicious redirect to file scheme
	uFile, _ := url.Parse("file:///etc/passwd")
	reqFile := &http.Request{URL: uFile}
	if err := checkRedirect(reqFile, via1); err == nil {
		t.Errorf("expected error when redirecting to file://, got nil")
	}

	// Exceeded hops
	reqHops, _ := http.NewRequest("GET", "https://example.com/hop4", nil)
	viaHops := []*http.Request{req1, req1, req1}
	if err := checkRedirect(reqHops, viaHops); err == nil {
		t.Errorf("expected error when exceeding 3 redirects, got nil")
	}
}
