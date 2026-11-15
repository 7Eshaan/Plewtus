package extractor

import (
	"bytes"
	"regexp"
	"strings"
	"unicode/utf8"
)

// metaCharset finds a charset declared in the first bytes of a document:
// <meta charset="..."> or <meta http-equiv="Content-Type" content="...; charset=...">.
var metaCharset = regexp.MustCompile(`(?i)<meta[^>]+charset\s*=\s*["']?\s*([a-z0-9_\-:.]+)`)

// singleByteWestern lists labels for ISO-8859-1 and Windows-1252, which the
// HTML standard decodes identically (as Windows-1252).
var singleByteWestern = map[string]bool{
	"iso-8859-1": true, "iso8859-1": true, "latin1": true, "l1": true, "iso_8859-1": true,
	"windows-1252": true, "cp1252": true, "x-cp1252": true, "us-ascii": true, "ascii": true,
}

// windows1252High maps bytes 0x80-0x9F to their Windows-1252 code points
// (0 marks the five undefined bytes, kept as U+FFFD).
var windows1252High = [32]rune{
	0x20AC, 0, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0, 0x017D, 0,
	0, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0, 0x017E, 0x0178,
}

// declaredCharset returns the lower-cased charset from the Content-Type
// header, or else from a <meta> tag in the first 2 KB of the page.
func declaredCharset(contentType string, body []byte) string {
	for _, part := range strings.Split(contentType, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if ok && strings.EqualFold(strings.TrimSpace(key), "charset") {
			return strings.ToLower(strings.Trim(strings.TrimSpace(value), `"'`))
		}
	}
	head := body
	if len(head) > 2048 {
		head = head[:2048]
	}
	if m := metaCharset.FindSubmatch(head); m != nil {
		return strings.ToLower(string(m[1]))
	}
	return ""
}

// toUTF8 converts pages declared as ISO-8859-1/Windows-1252 to UTF-8; the
// HTML parser assumes UTF-8, so such pages otherwise come out garbled.
// Pages that are already valid UTF-8, or use other charsets, are returned
// unchanged.
func toUTF8(body []byte, contentType string) []byte {
	if utf8.Valid(body) || !singleByteWestern[declaredCharset(contentType, body)] {
		return body
	}
	var buf bytes.Buffer
	buf.Grow(len(body) + len(body)/8)
	for _, b := range body {
		switch {
		case b < 0x80:
			buf.WriteByte(b)
		case b < 0xA0:
			if r := windows1252High[b-0x80]; r != 0 {
				buf.WriteRune(r)
			} else {
				buf.WriteRune(utf8.RuneError)
			}
		default:
			buf.WriteRune(rune(b))
		}
	}
	return buf.Bytes()
}
