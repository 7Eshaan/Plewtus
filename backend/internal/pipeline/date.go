package pipeline

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// dateLayouts are the absolute date formats engines and pages use, tried in
// order after ISO forms. Day-first and month-first layouts cannot both match
// one string, since the month is written as a name.
var dateLayouts = []string{
	"Jan 2, 2006",
	"January 2, 2006",
	"Jan. 2, 2006",
	"2 Jan 2006",
	"2 January 2006",
	"02 Jan 2006",
	"Mon, 02 Jan 2006 15:04:05 MST", // RFC 1123
	"Mon, 2 Jan 2006 15:04:05 -0700",
	"2006/01/02",
	"20060102",
}

// isoLayouts are tried on the leading part of ISO-8601 timestamps.
var isoLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02T15:04",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// relativeDate matches "3 days ago", "an hour ago", "1 wk ago".
var relativeDate = regexp.MustCompile(`^(\d+|a|an|one)\s+(sec|second|min|minute|hr|hour|day|wk|week|mo|month|yr|year)s?\.?\s+ago$`)

// NormalizeDate turns a publication date as engines and pages report it into
// ISO-8601 calendar form (YYYY-MM-DD), so results from every engine read the
// same. Relative dates ("3 days ago", "yesterday") are resolved against now
// (in UTC). A date that cannot be understood is returned unchanged (trimmed),
// so no information is lost.
func NormalizeDate(raw string, now time.Time) string {
	s := strings.Join(strings.Fields(raw), " ")
	s = strings.TrimRight(s, " ·-–—|")
	if s == "" {
		return ""
	}
	now = now.UTC()
	lower := strings.ToLower(s)

	switch lower {
	case "today", "just now", "now":
		return now.Format(time.DateOnly)
	case "yesterday":
		return now.AddDate(0, 0, -1).Format(time.DateOnly)
	}
	if m := relativeDate.FindStringSubmatch(lower); m != nil {
		n := 1
		if v, err := strconv.Atoi(m[1]); err == nil {
			n = v
		}
		var t time.Time
		switch m[2] {
		case "sec", "second":
			t = now.Add(-time.Duration(n) * time.Second)
		case "min", "minute":
			t = now.Add(-time.Duration(n) * time.Minute)
		case "hr", "hour":
			t = now.Add(-time.Duration(n) * time.Hour)
		case "day":
			t = now.AddDate(0, 0, -n)
		case "wk", "week":
			t = now.AddDate(0, 0, -7*n)
		case "mo", "month":
			t = now.AddDate(0, -n, 0)
		default: // year
			t = now.AddDate(-n, 0, 0)
		}
		return t.Format(time.DateOnly)
	}

	for _, layout := range isoLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			// Keep the calendar date the source wrote, not the UTC date.
			return t.Format(time.DateOnly)
		}
	}
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format(time.DateOnly)
		}
	}
	return s
}
