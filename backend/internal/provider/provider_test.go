package provider

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"plewtus/internal/model"
)

func TestReadBodyRejectsOversizedResponse(t *testing.T) {
	resp := &http.Response{Body: io.NopCloser(strings.NewReader(strings.Repeat("x", (4<<20)+1)))}
	if _, err := ReadBody(resp); err == nil {
		t.Fatal("expected oversized search response to be rejected")
	}
}

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }
func (m *mockProvider) Search(ctx context.Context, req model.ProviderRequest) (*model.ProviderResponse, error) {
	return &model.ProviderResponse{Engine: m.name}, nil
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()

	p1 := &mockProvider{name: "duckduckgo"}
	p2 := &mockProvider{name: "bing"}

	reg.Register(p1)
	reg.Register(p2)

	names := reg.Names()
	if len(names) != 2 || names[0] != "bing" || names[1] != "duckduckgo" {
		t.Errorf("unexpected names: %v", names)
	}

	found, ok := reg.Get("duckduckgo")
	if !ok || found.Name() != "duckduckgo" {
		t.Errorf("expected to find duckduckgo provider")
	}

	_, ok = reg.Get("nonexistent")
	if ok {
		t.Errorf("expected nonexistent provider to return false")
	}

	all := reg.All()
	if len(all) != 2 {
		t.Errorf("expected 2 providers in All(), got %d", len(all))
	}
}
