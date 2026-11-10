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
	"msockid": true,
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

	// Generic click tracking
	"spm":       true,
	"_openstat": true,
}

// referralParams usually carry a referrer and do not change the page, so they
// are ignored when comparing URLs. They stay in the displayed URL because some
// sites give them meaning (for example ?ref=<branch> on code hosts).
var referralParams = map[string]bool{
	"ref":      true,
	"ref_":     true,
	"referrer": true,
}

func isTrackingParam(name string) bool {
	name = strings.ToLower(name)
	return trackingParams[name] || strings.HasPrefix(name, "utm_")
}

var defaultIndexFiles = []string{
	"/index.html",
	"/index.htm",
	"/index.php",
	"/default.asp",
	"/default.aspx",
}

// NormalizeURL returns a clean display URL and a canonical comparison key.
//
// The display URL changes as little as possible so it still opens the same
// page: scheme and host are lowercased, default ports, fragments, and tracking
// parameters are removed, and everything else (path, encoding, parameter
// order) is left exactly as the engine reported it.
//
// The canonical key is aggressive, because it only decides whether two results
// are the same page: it ignores the scheme, a leading "www.", trailing slashes,
// default index files, referral parameters, and parameter order.
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

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		scheme = "https"
	}
	host := strings.ToLower(parsed.Host)
	if scheme == "http" {
		host = strings.TrimSuffix(host, ":80")
	} else {
		host = strings.TrimSuffix(host, ":443")
	}

	// Split the query once: displayed pairs keep their original text and order.
	var displayPairs []string
	keyValues := url.Values{}
	for _, pair := range strings.Split(parsed.RawQuery, "&") {
		if pair == "" {
			continue
		}
		rawName, rawValue, _ := strings.Cut(pair, "=")
		name, nameErr := url.QueryUnescape(rawName)
		if nameErr != nil {
			name = rawName
		}
		if isTrackingParam(name) {
			continue
		}
		displayPairs = append(displayPairs, pair)
		if referralParams[strings.ToLower(name)] {
			continue
		}
		value, valueErr := url.QueryUnescape(rawValue)
		if valueErr != nil {
			value = rawValue
		}
		keyValues.Add(name, value)
	}

	displayPath := parsed.EscapedPath()
	if displayPath == "" {
		displayPath = "/"
	}
	cleanURL = scheme + "://" + host + displayPath
	if len(displayPairs) > 0 {
		cleanURL += "?" + strings.Join(displayPairs, "&")
	}

	// Canonical key: cleaned path without index files or a trailing slash.
	keyPath := parsed.Path
	if keyPath == "" || keyPath == "/" {
		keyPath = "/"
	} else {
		keyPath = path.Clean(keyPath)
		lowPath := strings.ToLower(keyPath)
		for _, idx := range defaultIndexFiles {
			if strings.HasSuffix(lowPath, idx) {
				keyPath = keyPath[:len(keyPath)-len(idx)]
				break
			}
		}
		if keyPath == "" {
			keyPath = "/"
		}
		if !strings.HasPrefix(keyPath, "/") {
			keyPath = "/" + keyPath
		}
	}

	// Hosts are case-insensitive, but URL paths and query values may not be.
	canonicalKey = strings.TrimPrefix(host, "www.") + keyPath
	if len(keyValues) > 0 {
		canonicalKey += "?" + keyValues.Encode() // Encode sorts by key
	}

	return cleanURL, canonicalKey, nil
}
