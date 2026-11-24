package model

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// Regression: paginating without a limit must not fail validation.
func TestRegression_OffsetWithoutLimit(t *testing.T) {
	req := SearchRequest{Query: "golang", Offset: 45}
	req.Normalize(50, 50, []string{"bing"})
	if err := req.Validate(); err != nil {
		t.Fatalf("offset-only request should be valid, got %v (limit=%d)", err, req.Limit)
	}
	if req.Limit != 5 {
		t.Fatalf("expected limit shrunk to 5, got %d", req.Limit)
	}

	explicit := SearchRequest{Query: "golang", Offset: 45, Limit: 10}
	explicit.Normalize(10, 50, []string{"bing"})
	if explicit.Validate() == nil {
		t.Fatal("an explicit limit beyond the window must still be rejected")
	}
}

func TestRegression_MaxLimitCappedToWindow(t *testing.T) {
	req := SearchRequest{Query: "golang", Limit: 80}
	req.Normalize(10, 100, nil)
	if req.Limit != MaxResultWindow {
		t.Fatalf("expected limit capped at %d, got %d", MaxResultWindow, req.Limit)
	}
}

// Regression: cloning (used for every cached/single-flight response) must keep
// empty arrays as [] instead of null.
func TestRegression_CloneKeepsEmptyArrays(t *testing.T) {
	data, err := json.Marshal(NewSearchResponse("q").Clone())
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"results":[]`, `"queried":[]`, `"successful":[]`, `"failed":[]`} {
		if !strings.Contains(string(data), field) {
			t.Errorf("expected %s in %s", field, data)
		}
	}
}

// Domain filters are bounded and must be plain host names, since each one
// becomes a site: operator in the query sent to every engine.
func TestRegression_DomainFiltersAreValidated(t *testing.T) {
	valid := func(inc, exc []string) error {
		req := &SearchRequest{Query: "go", IncludeDomains: inc, ExcludeDomains: exc}
		req.Normalize(10, 50, []string{"bing"})
		return req.Validate()
	}
	if err := valid([]string{"https://www.go.dev/doc", "*.example.com", "bücher.de", "my-site.co.uk:8080"}, nil); err != nil {
		t.Fatalf("ordinary domains should be accepted: %v", err)
	}
	many := make([]string, MaxDomainFilters+1)
	for i := range many {
		many[i] = fmt.Sprintf("d%d.com", i)
	}
	if err := valid(many, nil); err == nil {
		t.Fatal("more than MaxDomainFilters include domains must be rejected")
	}
	if err := valid(nil, many); err == nil {
		t.Fatal("more than MaxDomainFilters exclude domains must be rejected")
	}
	for _, bad := range []string{"example.com OR site:evil.com", `ex"ample.com`, "a(b).com", strings.Repeat("a", 254) + ".com"} {
		if err := valid([]string{bad}, nil); err == nil {
			t.Errorf("domain %q should be rejected", bad)
		}
	}
}
