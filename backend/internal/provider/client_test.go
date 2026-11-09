package provider

import (
	"testing"
	"time"
)

// Regression: Yahoo needs HTTP/1.1; WithHTTP1Only must disable h2 ALPN.
func TestWithHTTP1OnlyDisablesHTTP2(t *testing.T) {
	c := NewHTTPClient(time.Second, "ua", WithHTTP1Only())
	if c.transport.ForceAttemptHTTP2 || c.transport.TLSNextProto == nil || len(c.transport.TLSNextProto) != 0 {
		t.Fatal("HTTP/2 must be disabled")
	}
	if got := c.transport.TLSClientConfig.NextProtos; len(got) != 1 || got[0] != "http/1.1" {
		t.Fatalf("ALPN must offer only http/1.1, got %v", got)
	}
	if def := NewHTTPClient(time.Second, "ua"); def.transport.TLSClientConfig != nil && len(def.transport.TLSClientConfig.NextProtos) == 1 {
		t.Fatal("default clients must keep HTTP/2")
	}
}
