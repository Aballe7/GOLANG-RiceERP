package purchasing

// Pure unit tests for TIN validation.  No DB, no testutil — just the regex.
// These are the fastest tests in the suite; they run in microseconds.

import "testing"

func TestTINValidation(t *testing.T) {
	cases := []struct {
		tin   string
		valid bool
	}{
		// Valid formats
		{"123-456-789",       true},  // 9-digit dashed (minimum)
		{"123-456-789-000",   true},  // 12-digit dashed (with branch code)
		{"123-456-789-00000", true},  // 14-digit dashed (long branch code)
		{"123456789",         true},  // 9 digits no dashes
		{"1234567890",        true},  // 10 digits no dashes
		{"123456789012",      true},  // 12 digits no dashes (max)

		// Invalid formats
		{"123-456-78",        false}, // too short (8 digits)
		{"12-345-678",        false}, // wrong grouping
		{"abc-def-ghi",       false}, // non-numeric
		{"bad-tin",           false}, // random string
		{"12-34",             false}, // way too short
		{"",                  false}, // empty
		{"123 456 789",       false}, // spaces instead of dashes
	}

	for _, tc := range cases {
		tc := tc // capture range var
		t.Run(tc.tin, func(t *testing.T) {
			got := tinRegexp.MatchString(tc.tin)
			if got != tc.valid {
				t.Errorf("TIN %q: expected valid=%v, got %v", tc.tin, tc.valid, got)
			}
		})
	}
}
