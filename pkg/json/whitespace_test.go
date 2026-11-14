package json_test

import (
	"testing"

	"myjson/pkg/json"
)

func TestIsBlank(t *testing.T) {
	for _, tc := range []struct {
		name, source string
		want         bool
	}{
		{"empty", "", true},
		{"JSON whitespace", " \t\r\n", true},
		{"text", " \t{}\r\n", false},
		{"non-breaking space", " \u00a0\n", false},
		{"ideographic space", "\u3000", false},
		{"vertical tab", "\v", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := json.IsBlank(tc.source); got != tc.want {
				t.Errorf("IsBlank(%q) = %v, want %v", tc.source, got, tc.want)
			}
		})
	}
}
