package app

import (
	"reflect"
	"testing"

	"plewtus/internal/config"
)

func TestFallbackProviders(t *testing.T) {
	cfg := &config.Config{DefaultProviders: []string{"duckduckgo", "bing", "yahoo"}}
	if got, want := FallbackProviders(cfg), []string{"duckduckgo", "bing", "yahoo", "marginalia"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("without a Brave key: got %v, want %v", got, want)
	}
	cfg.BraveAPIKey = "key"
	if got, want := FallbackProviders(cfg), []string{"duckduckgo", "bing", "yahoo", "marginalia", "brave"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("with a Brave key: got %v, want %v", got, want)
	}
}
