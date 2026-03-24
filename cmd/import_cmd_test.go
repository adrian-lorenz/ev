package cmd

import (
	"os"
	"path/filepath"
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

func TestParseDotEnv_CRLF(t *testing.T) {
	input := "KEY1=value1\r\nKEY2=value2\r\n# comment\r\nKEY3=value3"
	got, err := parseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
		"KEY3": "value3",
	}
	for k, wantV := range want {
		if gotV, ok := got[k]; !ok {
			t.Errorf("missing key %q", k)
		} else if gotV != wantV {
			t.Errorf("key %q: got %q, want %q", k, gotV, wantV)
		}
	}
}

func TestParseDotEnv_ExportPrefix(t *testing.T) {
	input := "export DB_HOST=localhost\nexport API_KEY=secret"
	got, err := parseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["DB_HOST"] != "localhost" || got["API_KEY"] != "secret" {
		t.Errorf("export prefix not stripped: %v", got)
	}
}

func TestParseDotEnv_EmptyValue(t *testing.T) {
	input := "EMPTY=\nALSO_EMPTY=\"\""
	got, err := parseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v, ok := got["EMPTY"]; !ok || v != "" {
		t.Errorf("EMPTY: got %q, want %q", v, "")
	}
	if v, ok := got["ALSO_EMPTY"]; !ok || v != "" {
		t.Errorf("ALSO_EMPTY: got %q, want %q", v, "")
	}
}

func TestParseMakefileTargets_BasicTargets(t *testing.T) {
	content := "build:\n\tgo build .\n\ntest:\n\tgo test ./...\n\nclean:\n\trm -rf dist\n"
	path := filepath.Join(t.TempDir(), "Makefile")
	os.WriteFile(path, []byte(content), 0600)

	targets := parseMakefileTargets(path, 5)
	if len(targets) == 0 {
		t.Fatal("expected targets, got none")
	}
	has := func(name string) bool {
		for _, t := range targets {
			if t == name {
				return true
			}
		}
		return false
	}
	for _, want := range []string{"build", "test", "clean"} {
		if !has(want) {
			t.Errorf("target %q not found in %v", want, targets)
		}
	}
}

func TestParseMakefileTargets_PhonyFirst(t *testing.T) {
	content := ".PHONY: build test\nfile.o: file.c\n\tcc -c file.c\nbuild:\n\tgo build .\ntest:\n\tgo test ./...\n"
	path := filepath.Join(t.TempDir(), "Makefile")
	os.WriteFile(path, []byte(content), 0600)

	targets := parseMakefileTargets(path, 5)

	idx := map[string]int{}
	for i, name := range targets {
		idx[name] = i
	}
	if buildIdx, ok := idx["build"]; ok {
		if fileIdx, ok2 := idx["file.o"]; ok2 && buildIdx > fileIdx {
			t.Errorf("phony target 'build' should come before 'file.o', got order: %v", targets)
		}
	}
}

func TestParseMakefileTargets_SkipDollarTargets(t *testing.T) {
	content := "$(BINARY): main.go\n\tgo build .\nbuild:\n\tgo build .\n"
	path := filepath.Join(t.TempDir(), "Makefile")
	os.WriteFile(path, []byte(content), 0600)

	targets := parseMakefileTargets(path, 5)
	for _, name := range targets {
		if strings.Contains(name, "$") {
			t.Errorf("variable target %q should be skipped", name)
		}
	}
}

func TestParseMakefileTargets_MaxTargets(t *testing.T) {
	var sb strings.Builder
	for i := 0; i < 20; i++ {
		sb.WriteString(strings.Repeat("t", i+1) + ":\n\techo ok\n")
	}
	path := filepath.Join(t.TempDir(), "Makefile")
	os.WriteFile(path, []byte(sb.String()), 0600)

	targets := parseMakefileTargets(path, 3)
	if len(targets) > 3 {
		t.Errorf("expected at most 3 targets, got %d: %v", len(targets), targets)
	}
}

func TestParseMakefileTargets_EmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Makefile")
	os.WriteFile(path, []byte(""), 0600)

	targets := parseMakefileTargets(path, 5)
	if len(targets) != 0 {
		t.Errorf("expected no targets from empty file, got %v", targets)
	}
}

func TestParseMakefileTargets_MissingFile(t *testing.T) {
	targets := parseMakefileTargets("/nonexistent/Makefile", 5)
	if targets != nil {
		t.Errorf("missing file should return nil, got %v", targets)
	}
}
