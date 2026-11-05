package pipeline

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// trackingParams contains common tracking, analytics, and affiliate query parameter names to strip.
var trackingParams = map[string]bool{
	// Google Analytics & UTM
	"utm_source":       true,
	"utm_medium":       true,
	"utm_campaign":     true,
	"utm_term":         true,
	"utm_content":      true,
	"utm_id":           true,
	"utm_name":         true,
	"utm_reader":       true,
	"utm_viz_id":       true,
	"utm_pubreferrer":  true,
	"utm_swu":          true,
	"utm_source_alias": true,

	// Google Ads / Click IDs
	"gclid":      true,
	"gclsrc":     true,
	"dclid":      true,
	"gad_source": true,
	"gbraid":     true,
	"wbraid":     true,

	// Facebook / Meta
	"fbclid":          true,
	"fb_action_ids":   true,
	"fb_action_types": true,
	"fb_source":       true,
	"fref":            true,

	// Twitter / X
	"twclid": true,

	// Microsoft / Bing
	"msclkid": true,
	"cvid":    true,

	// Yahoo
	"guccounter":        true,
	"guce_referrer":     true,
	"guce_referrer_sig": true,

	// Hubspot / Marketo
	"_hsenc":        true,
	"_hsmi":         true,
	"hsctatracking": true,
	"mkt_tok":       true,

	// Mailchimp
	"mc_cid": true,
	"mc_eid": true,

	// Yandex
	"yclid":  true,
	"ymclid": true,

	// TikTok & Instagram
	"ttclid": true,
	"igshid": true,

	// Generic referral & click tracking
	"ref":       true,
	"ref_":      true,
	"referrer":  true,
	"spm":       true,
	"_openstat": true,
}

var defaultIndexFiles = []string{
	"/index.html",
	"/index.htm",
	"/index.php",
	"/default.asp",
	"/default.aspx",
}

// NormalizeURL canonicalizes a raw URL by stripping tracking parameters, normalizing scheme/host,
// removing fragments, and producing both a clean URL and a normalized comparison key for deduplication.
func NormalizeURL(rawURL string) (cleanURL string, canonicalKey string, err error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", "", fmt.Errorf("empty url")
	}

	// Handle protocol-relative URLs
	if strings.HasPrefix(rawURL, "//") {
		rawURL = "https:" + rawURL
	} else if !strings.Contains(rawURL, "://") {
		rawURL = "https://" + rawURL
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid url %q: %w", rawURL, err)
	}
	if parsed.Hostname() == "" {
		return "", "", fmt.Errorf("invalid url %q: host is empty", rawURL)
	}
	if parsed.User != nil {
		return "", "", fmt.Errorf("invalid url %q: embedded credentials are not allowed", rawURL)
	}

	// 1. Normalize Scheme
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		scheme = "https"
	}
	parsed.Scheme = scheme

	// 2. Normalize Host (lowercased, strip default port)
	host := strings.ToLower(parsed.Host)
	if scheme == "http" && strings.HasSuffix(host, ":80") {
		host = strings.TrimSuffix(host, ":80")
	} else if scheme == "https" && strings.HasSuffix(host, ":443") {
		host = strings.TrimSuffix(host, ":443")
	}
	parsed.Host = host

	// 3. Strip Fragments (client-side anchors)
	parsed.Fragment = ""
	parsed.RawFragment = ""

	// 4. Normalize Path (clean slashes, strip index.html)
	uPath := parsed.Path
	if uPath == "" || uPath == "/" {
		uPath = "/"
	} else {
		uPath = path.Clean(uPath)
		// Check for default index files
		lowPath := strings.ToLower(uPath)
		for _, idx := range defaultIndexFiles {
			if strings.HasSuffix(lowPath, idx) {
				uPath = uPath[:len(uPath)-len(idx)]
				if uPath == "" {
					uPath = "/"
				}
				break
			}
		}
	}
	parsed.Path = uPath

	// 5. Filter Query Parameters (strip analytics/tracking parameters)
	q := parsed.Query()
	for k := range q {
		lowK := strings.ToLower(k)
		if trackingParams[lowK] || strings.HasPrefix(lowK, "utm_") {
			q.Del(k)
		}
	}

	if len(q) == 0 {
		parsed.RawQuery = ""
	} else {
		// url.Values.Encode() sorts keys deterministically
		parsed.RawQuery = q.Encode()
	}

	cleanURL = parsed.String()

	// 6. Build Canonical Comparison Key (used to detect duplicate pages)
	// Strip www. prefix for canonical key comparison
	canonHost := strings.TrimPrefix(host, "www.")

	canonPath := uPath
	if canonPath != "/" {
		canonPath = strings.TrimSuffix(canonPath, "/")
	}
	if canonPath != "" && !strings.HasPrefix(canonPath, "/") {
		canonPath = "/" + canonPath
	}

	// Hosts are case-insensitive, but URL paths and query values may not be.
	canonicalKey = canonHost + canonPath
	if parsed.RawQuery != "" {
		canonicalKey += "?" + parsed.RawQuery
	}

	return cleanURL, canonicalKey, nil
}
