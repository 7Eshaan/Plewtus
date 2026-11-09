package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"plewtus/internal/model"
)

func postSearch(t *testing.T, ts *httptest.Server, body string) *http.Response {
	t.Helper()
	resp, err := http.Post(ts.URL+"/search", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

// Regression: unknown providers are a client error, not a 500.
func TestRegression_UnknownProviderIs400(t *testing.T) {
	ts := httptest.NewServer(newTestServer(&mockSearcher{err: model.InvalidRequestf("unknown provider(s) google")}).Handler())
	defer ts.Close()
	if resp := postSearch(t, ts, `{"query":"go","providers":["google"]}`); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

// Regression: paginating without a limit uses the default limit (10) and works.
func TestRegression_OffsetOnlyRequest(t *testing.T) {
	ts := httptest.NewServer(newTestServer().Handler())
	defer ts.Close()
	if resp := postSearch(t, ts, `{"query":"go","offset":10}`); resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

// Regression: malformed requests do not consume the caller's daily quota.
func TestRegression_InvalidRequestsDoNotConsumeQuota(t *testing.T) {
	ts := httptest.NewServer(newTestServer().Handler())
	defer ts.Close()
	for i := 0; i < 3; i++ {
		resp := postSearch(t, ts, `{bad json`)
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", resp.StatusCode)
		}
		if resp.Header.Get("X-Quota-Remaining") != "" {
			t.Fatal("rejected requests must not report (or consume) quota")
		}
	}
	resp := postSearch(t, ts, `{"query":"go"}`)
	if got := resp.Header.Get("X-Quota-Remaining"); got != "99" {
		t.Fatalf("expected first valid request to leave 99 of 100, got %q", got)
	}
}
