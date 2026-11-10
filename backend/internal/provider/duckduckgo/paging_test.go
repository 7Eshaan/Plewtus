package duckduckgo

import (
	"os"
	"testing"
)

func TestParseNextPageForm(t *testing.T) {
	for _, fixture := range []string{"duckduckgo_html.html", "duckduckgo_lite.html"} {
		body, err := os.ReadFile("../../../testdata/fixtures/" + fixture)
		if err != nil {
			t.Fatal(err)
		}
		form, ok := ParseNextPageForm(body)
		if !ok {
			t.Fatalf("%s: expected a next-page form", fixture)
		}
		if form.Get("vqd") == "" || form.Get("s") != "10" || form.Get("dc") != "11" {
			t.Errorf("%s: unexpected next-page fields: %v", fixture, form)
		}
	}
	if _, ok := ParseNextPageForm([]byte("<html><body><form><input type=hidden name=q value=x></form></body></html>")); ok {
		t.Error("a page without a Next form must not report one")
	}
}
