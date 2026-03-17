package cmd

import (
	"strings"
	"testing"
)

func TestParseDotEnv(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name:  "postgres connection string",
			input: "DATABASE_URL=postgres://user:pass@localhost:5432/mydb",
			want:  map[string]string{"DATABASE_URL": "postgres://user:pass@localhost:5432/mydb"},
		},
		{
			name:  "https url",
			input: "API_URL=https://api.example.com/v1",
			want:  map[string]string{"API_URL": "https://api.example.com/v1"},
		},
		{
			name:  "redis url",
			input: "REDIS_URL=redis://localhost:6379",
			want:  map[string]string{"REDIS_URL": "redis://localhost:6379"},
		},
		{
			name:  "quoted https url",
			input: `API_URL="https://api.example.com/v1"`,
			want:  map[string]string{"API_URL": "https://api.example.com/v1"},
		},
		{
			name:  "inline // comment after whitespace",
			input: "VALUE=hello world // this is a comment",
			want:  map[string]string{"VALUE": "hello world"},
		},
		{
			name:  "inline # comment",
			input: "VALUE=some value # inline comment",
			want:  map[string]string{"VALUE": "some value"},
		},
		{
			name:  "// at start of line is skipped",
			input: "// this is a comment\nKEY=val",
			want:  map[string]string{"KEY": "val"},
		},
		{
			name:  "plain value unchanged",
			input: "KEY=justvalue",
			want:  map[string]string{"KEY": "justvalue"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDotEnv(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d keys, want %d: %v", len(got), len(tt.want), got)
			}
			for k, wantV := range tt.want {
				if gotV, ok := got[k]; !ok {
					t.Errorf("missing key %q", k)
				} else if gotV != wantV {
					t.Errorf("key %q: got %q, want %q", k, gotV, wantV)
				}
			}
		})
	}
}

func TestStripInlineComment(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		// URLs must not be truncated
		{"https://example.com", "https://example.com"},
		{"postgres://user:pass@host/db", "postgres://user:pass@host/db"},
		{"redis://localhost:6379", "redis://localhost:6379"},
		// // as comment: only when preceded by whitespace or at start
		{"hello // comment", "hello"},
		{"// full line comment", ""},
		// # comment
		{"value # comment", "value"},
		// no comment
		{"plainvalue", "plainvalue"},
	}

	for _, tt := range tests {
		got := stripInlineComment(tt.in)
		if got != tt.want {
			t.Errorf("stripInlineComment(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
