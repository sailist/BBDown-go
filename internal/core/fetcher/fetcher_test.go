package fetcher

import (
	"testing"
)

func TestNewFetcher_ReturnsErrorForAllPrefixes(t *testing.T) {
	cases := []struct {
		id      string
		useIntl bool
	}{
		{"cheese123", false},
		{"ep123", false},
		{"ep123", true},
		{"mid123", false},
		{"listBizId123", false},
		{"seriesBizId123", false},
		{"favId123", false},
		{"av123", false},
		{"BV123", false},
	}

	for _, tc := range cases {
		f, err := NewFetcher(tc.id, tc.useIntl)
		if err == nil {
			t.Errorf("NewFetcher(%q, %v) expected error, got fetcher %v", tc.id, tc.useIntl, f)
		}
	}
}

func TestNewFetcher_ReturnsErrorForEmptyOrInvalid(t *testing.T) {
	cases := []struct {
		id      string
		useIntl bool
	}{
		{"", false},
		{"unknown", false},
		{"random_prefix", false},
	}

	for _, tc := range cases {
		f, err := NewFetcher(tc.id, tc.useIntl)
		if err == nil {
			t.Errorf("NewFetcher(%q, %v) expected error, got fetcher %v", tc.id, tc.useIntl, f)
		}
	}
}
