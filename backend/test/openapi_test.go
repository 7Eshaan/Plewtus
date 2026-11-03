package test

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOpenAPISpecification_ValidJSON(t *testing.T) {
	data, err := os.ReadFile("../docs/openapi.json")
	if err != nil {
		t.Fatalf("failed to read docs/openapi.json: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("docs/openapi.json contains invalid JSON: %v", err)
	}

	if parsed["openapi"] != "3.1.0" {
		t.Fatalf("expected openapi 3.1.0, got %v", parsed["openapi"])
	}

	paths := parsed["paths"].(map[string]any)
	if paths["/search"] == nil {
		t.Fatal("expected /search path defined in OpenAPI spec")
	}
	if paths["/health"] == nil {
		t.Fatal("expected /health path defined in OpenAPI spec")
	}
}
