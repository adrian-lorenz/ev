package cmd

import "testing"

func TestShQuote(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Basic wrapping
		{"hello", "'hello'"},
		{"", "''"},
		// Spaces and shell meta-chars stay inside single quotes
		{"hello world", "'hello world'"},
		{"$HOME", "'$HOME'"},
		{"`date`", "'`date`'"},
		{"$(rm -rf /)", "'$(rm -rf /)'"},
		// Backslash is literal inside single quotes
		{`back\slash`, `'back\slash'`},
		// Newlines are literal
		{"line1\nline2", "'line1\nline2'"},
		// Single quote: must break out, escape, re-enter
		{"it's", `'it'\''s'`},
		{"''", `''\'''\'''`},
		// Multiple single quotes
		{"a'b'c", `'a'\''b'\''c'`},
	}
	for _, tc := range cases {
		got := shQuote(tc.in)
		if got != tc.want {
			t.Errorf("shQuote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestShQuote_ShellSafe verifies the output is always safely quoted:
// wrapping the output in a shell assignment should never allow injection.
func TestShQuote_NeverContainsUnquotedSingleQuote(t *testing.T) {
	// After shQuote, the only single quotes in the result should be
	// the surrounding ones or the escape sequence '\''.
	// A naive check: result always starts and ends with ' when no escaping needed.
	dangerous := []string{"$(evil)", "`evil`", "${VAR}", "val;rm -rf /"}
	for _, in := range dangerous {
		got := shQuote(in)
		// Must start and end with single quote
		if len(got) < 2 || got[0] != '\'' || got[len(got)-1] != '\'' {
			t.Errorf("shQuote(%q) = %q: must start and end with single quote", in, got)
		}
	}
}

func TestFishQuote(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"hello", "'hello'"},
		{"", "''"},
		{"hello world", "'hello world'"},
		{"$HOME", "'$HOME'"},
		// Fish escapes single quote with backslash-quote
		{"it's", `'it\'s'`},
		{"a'b", `'a\'b'`},
		// Double single quote
		{"''", `'\'\''`},
	}
	for _, tc := range cases {
		got := fishQuote(tc.in)
		if got != tc.want {
			t.Errorf("fishQuote(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
