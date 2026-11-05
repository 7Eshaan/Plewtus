package duckduckgo

import "testing"

func TestMapRegionDefaultsAndAliases(t *testing.T) {
	for _, tc := range []struct{ country, language, want string }{
		{"", "", "us-en"},
		{"gb", "en", "uk-en"},
		{"uk", "en", "uk-en"},
		{"fr", "", "fr-fr"},
		{"", "de", "us-de"},
	} {
		if got := mapRegion(tc.country, tc.language); got != tc.want {
			t.Errorf("mapRegion(%q, %q) = %q, want %q", tc.country, tc.language, got, tc.want)
		}
	}
}
