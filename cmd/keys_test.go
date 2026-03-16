package cmd

import "testing"

func TestNormalizeKeyInput(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "db_password", want: "DB_PASSWORD"},
		{in: "  api.key  ", want: "API.KEY"},
		{in: "Mixed-Name", want: "MIXED-NAME"},
	}

	for _, tt := range tests {
		if got := normalizeKeyInput(tt.in); got != tt.want {
			t.Fatalf("normalizeKeyInput(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
