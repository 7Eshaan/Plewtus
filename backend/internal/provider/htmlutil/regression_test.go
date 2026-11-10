package htmlutil

import (
	"net/url"
	"testing"
)

// Regression: DuckDuckGo redirect targets were percent-decoded twice, which
// turned "+" into spaces and broke "%"-escaped URLs.
func TestUnwrapURL_DuckDuckGoTargetIsDecodedOnce(t *testing.T) {
	for _, target := range []string{
		"https://en.wikipedia.org/wiki/C++",
		"https://example.com/a?q=100%25+off&x=a+b",
		"https://example.com/path%20with%20space",
	} {
		wrapped := "//duckduckgo.com/l/?uddg=" + url.QueryEscape(target) + "&rut=abc"
		if got := UnwrapURL(wrapped); got != target {
			t.Errorf("UnwrapURL(%q) = %q, want %q", wrapped, got, target)
		}
	}
}

func TestUnwrapURL_RelativeBingRedirect(t *testing.T) {
	if got := UnwrapURL("/ck/a?!&&p=x&u=a1aHR0cHM6Ly9nby5kZXYv&ntb=1"); got != "https://go.dev/" {
		t.Errorf("relative Bing redirect not unwrapped: %q", got)
	}
}
