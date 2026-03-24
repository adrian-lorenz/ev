package cmd

import "testing"

func TestQuoteEnvValue(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Simple values need no quoting
		{"simple", "simple"},
		{"UPPER123", "UPPER123"},
		{"with-dash", "with-dash"},
		// Empty string must be quoted
		{"", `""`},
		// Spaces trigger quoting
		{"hello world", `"hello world"`},
		{"leading space", `"leading space"`},
		// Tabs trigger quoting
		{"a\tb", "\"a\tb\""},
		// Newline triggers quoting
		{"a\nb", "\"a\nb\""},
		// Hash triggers quoting (would be treated as comment)
		{"val#extra", `"val#extra"`},
		// Dollar triggers quoting (would be expanded by shells)
		{"$HOME", `"$HOME"`},
		// Backtick triggers quoting
		{"`date`", "\"`date`\""},
		// Exclamation triggers quoting
		{"hello!", `"hello!"`},
		// Double quote is always escaped, triggers quoting if space present
		{`say "hi"`, `"say \"hi\""`},
		// Backslash is always escaped
		{`back\slash`, `back\\slash`},
		// Backslash + double quote: both escaped, no quoting trigger → no outer quotes
		{`a\"b`, `a\\\"b`},
		// Only backslash escaping, no quoting triggers
		{`a\b`, `a\\b`},
	}
	for _, tc := range cases {
		got := quoteEnvValue(tc.in)
		if got != tc.want {
			t.Errorf("quoteEnvValue(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestQuoteEnvValue_BackslashBeforeQuote verifies the tricky case where a
// backslash immediately precedes a double quote.
func TestQuoteEnvValue_BackslashBeforeQuote(t *testing.T) {
	// input: abc\"def  (backslash + double-quote in value)
	in := `abc\"def`
	got := quoteEnvValue(in)
	// backslash → \\, then " → \"  → abc\\\"def
	// no quoting trigger (no space etc.) → abc\\\"def
	want := `abc\\\"def`
	if got != want {
		t.Errorf("quoteEnvValue(%q) = %q, want %q", in, got, want)
	}
}
