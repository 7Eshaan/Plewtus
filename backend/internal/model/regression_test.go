package model

import (
	"encoding/json"
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
