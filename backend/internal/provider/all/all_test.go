package all

import (
	"testing"
	"time"
)

func TestNewRegistry(t *testing.T) {
	reg := NewRegistry(3*time.Second, "test-agent")
	names := reg.Names()

	expected := []string{"bing", "brave", "duckduckgo", "marginalia", "qwant", "yahoo"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d providers, got %d (%v)", len(expected), len(names), names)
	}

	for i, name := range expected {
		if names[i] != name {
			t.Errorf("expected provider [%d] to be %q, got %q", i, name, names[i])
		}
		p, ok := reg.Get(name)
		if !ok || p == nil {
			t.Errorf("failed to retrieve provider %q from registry", name)
		}
	}
}
