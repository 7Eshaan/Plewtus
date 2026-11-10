package server

import (
	"encoding/json"
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

// Regression: unknown routes and methods answer with the JSON error contract.
func TestRegression_UnknownRouteAndMethodReturnJSON(t *testing.T) {
	ts := httptest.NewServer(newTestServer().Handler())
	defer ts.Close()
	for path, want := range map[string]int{"/nope": http.StatusNotFound, "/search": http.StatusMethodNotAllowed} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		var body model.ErrorResponse
		decodeErr := json.NewDecoder(resp.Body).Decode(&body)
		resp.Body.Close()
		if resp.StatusCode != want || decodeErr != nil || body.Code != want || body.Error == "" {
			t.Errorf("GET %s: status=%d body=%+v err=%v", path, resp.StatusCode, body, decodeErr)
		}
		if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("GET %s: content type %q", path, ct)
		}
	}
}

// Regression: an overlong query is rejected instead of being sent upstream.
func TestRegression_OverlongQueryIs400(t *testing.T) {
	ts := httptest.NewServer(newTestServer().Handler())
	defer ts.Close()
	body := `{"query":"` + strings.Repeat("a ", model.MaxQueryChars) + `"}`
	if resp := postSearch(t, ts, body); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}
