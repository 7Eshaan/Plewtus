package pipeline

import (
	"testing"
	"time"
)

func TestNormalizeDate(t *testing.T) {
	now := time.Date(2026, 11, 24, 10, 30, 0, 0, time.UTC)
	cases := map[string]string{
		"":                              "",
		"  ":                            "",
		"Sep 3, 2025":                   "2025-09-03", // Bing, Yahoo
		"Sep 3, 2025 · ":                "2025-09-03",
		"September 3, 2025":             "2025-09-03",
		"19 August 2026":                "2026-08-19", // Brave
		"19 Aug 2026":                   "2026-08-19",
		"2013-01-02":                    "2013-01-02", // Qwant
		"2026-10-01T00:00:00":           "2026-10-01", // Brave API page_age
		"2026-05-15T10:00:00Z":          "2026-05-15", // article:published_time
		"2026-05-15T23:30:00-07:00":     "2026-05-15", // the source's calendar day, not UTC
		"2026-05-15 08:00:00":           "2026-05-15",
		"Tue, 10 Nov 2026 08:00:00 GMT": "2026-11-10",
		"3 days ago":                    "2026-11-21",
		"1 day ago":                     "2026-11-23",
		"an hour ago":                   "2026-11-24",
		"12 hours ago":                  "2026-11-23",
		"2 weeks ago":                   "2026-11-10",
		"3 Weeks Ago":                   "2026-11-03",
		"1 month ago":                   "2026-10-24",
		"2 years ago":                   "2024-11-24",
		"yesterday":                     "2026-11-23",
		"Today":                         "2026-11-24",
		"Spring 2024":                   "Spring 2024", // not understood: kept as is
		"  Q3   2025 ":                  "Q3 2025",
	}
	for raw, want := range cases {
		if got := NormalizeDate(raw, now); got != want {
			t.Errorf("NormalizeDate(%q) = %q, want %q", raw, got, want)
		}
	}
}
