package pkglock

import "testing"

// TestNormalize pins the exact output of the ION-20 §2 normalization rule.
// The rule is frozen at lockfile schema v1: changing it silently invalidates
// every digest already stored in a compendium.lock, so each row here is a
// contract, not a snapshot.
//
// Rows fall into three groups:
//   - normalization must happen (trailing space/tab, CRLF, EOF newlines)
//   - already-clean input must pass through untouched
//   - normalization must NOT go further (mid-line space and indentation are
//     content, not whitespace to strip) — these guard against someone turning
//     this into semantic canonicalization, which §2 explicitly rules out.
func TestNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"already clean", "foo\n", "foo\n"},
		{"no newline at EOF", "foo", "foo\n"},
		{"extra blank lines", "foo\n\n\n", "foo\n"},
		{"trailing spaces", "foo   \n", "foo\n"},
		{"trailing tab", "foo\t\n", "foo\n"},
		{"CRLF", "foo\r\n", "foo\n"},
		{"empty", "", "\n"},
		{"mid-line space kept", "a:  b\n", "a:  b\n"},
		{"indentation kept", "  foo\n", "  foo\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(normalize([]byte(tt.in))); got != tt.want {
				t.Errorf("normalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
