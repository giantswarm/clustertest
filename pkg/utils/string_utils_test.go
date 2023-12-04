package utils

import (
	"strings"
	"testing"
)

func TestGenerateRandomName(t *testing.T) {
	expectedLength := 20

	testcases := []struct {
		prefix string
	}{
		{"t"},
		{"foo"},
		{""},
	}
	for _, tc := range testcases {
		out := GenerateRandomName(tc.prefix)

		if len(out) != expectedLength {
			t.Errorf("Expected length of random name to be %d characters - %s", expectedLength, out)
		}

		if tc.prefix == "" {
			if strings.HasPrefix(out, "-") {
				t.Errorf("Not expecting name to begin with '-' - %s", out)
			}
		} else if !strings.HasPrefix(out, tc.prefix+"-") {
			t.Errorf("Expected name to begin with '%s' - %s", tc.prefix, out)
		}
	}
}
