package cmd

import (
	"strings"
	"testing"
)

// --- byteOffsetToLineCol ---

func TestByteOffsetToLineCol(t *testing.T) {
	cases := []struct {
		text     string
		offset   int
		wantLine int
		wantCol  int
	}{
		// Offset 0 is always line 1, col 1
		{"abc", 0, 1, 1},
		// Second char
		{"abc", 1, 1, 2},
		{"abc", 2, 1, 3},
		// After newline
		{"a\nbc", 2, 2, 1},
		{"a\nbc", 3, 2, 2},
		// Multiple newlines
		{"a\nb\nc", 4, 3, 1},
		// Offset at end of string
		{"abc", 3, 1, 4},
		// Empty string at offset 0
		{"", 0, 1, 1},
		// Newline at offset 0
		{"\nabc", 1, 2, 1},
	}
	for _, tc := range cases {
		line, col := byteOffsetToLineCol(tc.text, tc.offset)
		if line != tc.wantLine || col != tc.wantCol {
			t.Errorf("byteOffsetToLineCol(%q, %d) = (%d,%d), want (%d,%d)",
				tc.text, tc.offset, line, col, tc.wantLine, tc.wantCol)
		}
	}
}

func TestByteOffsetToLineCol_MultibytUTF8(t *testing.T) {
	// "こ" is 3 bytes (e3 81 93), "a" starts at byte offset 3
	text := "こa"
	line, col := byteOffsetToLineCol(text, 3)
	if line != 1 || col != 2 {
		t.Errorf("multibyte: got line=%d col=%d, want line=1 col=2", line, col)
	}
}

// --- blankNoscanLines ---

func TestBlankNoscanLines_NoMarker(t *testing.T) {
	text := "line one\nline two\nline three"
	got := blankNoscanLines(text)
	if got != text {
		t.Errorf("expected unchanged text, got %q", got)
	}
}

func TestBlankNoscanLines_HashMarker(t *testing.T) {
	text := "normal\nAPI_KEY=secret # noscan\nnormal"
	got := blankNoscanLines(text)
	if strings.Contains(got, "secret") {
		t.Error("noscan line should be blanked")
	}
}

func TestBlankNoscanLines_SlashMarker(t *testing.T) {
	text := "normal\nconst token = \"abc\" // noscan\nnormal"
	got := blankNoscanLines(text)
	lines := strings.Split(got, "\n")
	if strings.Contains(lines[1], "token") || strings.Contains(lines[1], "abc") {
		t.Error("// noscan line should be blanked")
	}
}

func TestBlankNoscanLines_CaseInsensitive(t *testing.T) {
	cases := []string{
		"KEY=val # NOSCAN",
		"KEY=val # NoScan",
		"KEY=val // NOSCAN",
		"KEY=val // NoScan",
	}
	for _, text := range cases {
		got := blankNoscanLines(text)
		if strings.Contains(got, "val") {
			t.Errorf("blankNoscanLines(%q) should blank line, got %q", text, got)
		}
	}
}

func TestBlankNoscanLines_PreservesByteLength(t *testing.T) {
	// Offset calculation depends on byte lengths being preserved across lines
	text := "first\nKEY=secret # noscan\nthird"
	got := blankNoscanLines(text)
	if len(got) != len(text) {
		t.Errorf("byte length changed: before=%d after=%d", len(text), len(got))
	}
}

func TestBlankNoscanLines_OnlyMatchedLinesAffected(t *testing.T) {
	text := "keep_this\nKEY=val # noscan\nkeep_this_too"
	got := blankNoscanLines(text)
	lines := strings.Split(got, "\n")
	if lines[0] != "keep_this" {
		t.Errorf("line before noscan line changed: %q", lines[0])
	}
	if lines[2] != "keep_this_too" {
		t.Errorf("line after noscan line changed: %q", lines[2])
	}
}

// --- redact ---

func TestRedact(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		// Short values: all masked
		{"", ""},
		{"a", "*"},
		{"abc", "***"},
		{"123456", "******"},
		// Long values: first 6 + stars + ellipsis
		{"1234567", "123456*…"},
		{"abcdefghij", "abcdef****…"},
		// Whitespace trimmed
		{"  abc  ", "***"},
		{"  abcdefg  ", "abcdef*…"},
	}
	for _, tc := range cases {
		got := redact(tc.in)
		if got != tc.want {
			t.Errorf("redact(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestRedact_MultibytRunes(t *testing.T) {
	// "こんにちは" is 5 runes — all should be masked (≤6 runes)
	got := redact("こんにちは")
	if strings.ContainsRune(got, 'こ') {
		t.Error("multibyte string should be fully masked when ≤6 runes")
	}
	if len(got) != 5 {
		t.Errorf("expected 5 stars for 5 runes, got %q", got)
	}

	// 7 runes: first 6 shown + tail masked
	got7 := redact("こんにちはAB")
	if !strings.HasPrefix(got7, "こんにちはA") {
		t.Errorf("first 6 runes should show: %q", got7)
	}
}

// --- looksLikeText ---

func TestLooksLikeText(t *testing.T) {
	cases := []struct {
		desc    string
		content []byte
		want    bool
	}{
		{"valid UTF-8 text", []byte("hello world\nsome text"), true},
		{"empty", []byte{}, true},
		{"ASCII only", []byte("abc123\n\t"), true},
		// Invalid UTF-8 bytes: binary heuristic fires when non-printable rate > 10%
		// 0x80 makes it invalid UTF-8; 0x01 counts as non-printable
		{
			"binary (invalid UTF-8 + non-printable)",
			func() []byte {
				b := make([]byte, 100)
				for i := range b {
					if i < 20 {
						b[i] = 0x01 // non-printable, triggers threshold
					} else if i < 25 {
						b[i] = 0x80 // invalid UTF-8 byte
					} else {
						b[i] = 'a'
					}
				}
				return b
			}(),
			false,
		},
		{"valid UTF-8 with multibyte", []byte("こんにちは世界"), true},
	}
	for _, tc := range cases {
		got := looksLikeText(tc.content)
		if got != tc.want {
			t.Errorf("looksLikeText(%s) = %v, want %v", tc.desc, got, tc.want)
		}
	}
}

func TestLooksLikeText_BinaryThreshold(t *testing.T) {
	// 0x01 is valid UTF-8, so we need at least one invalid-UTF-8 byte (0x80)
	// to get past the utf8.Valid fast path, then enough 0x01 bytes to exceed
	// the 10% non-printable threshold.
	b := make([]byte, 100)
	for i := range b {
		switch {
		case i < 20:
			b[i] = 0x01 // non-printable (c < 0x09), triggers threshold
		case i < 22:
			b[i] = 0x80 // invalid UTF-8 byte, forces utf8.Valid → false
		default:
			b[i] = 'a'
		}
	}
	// utf8.Valid → false (0x80), nonPrint/total = 20/100 = 20% > 10% → binary
	if looksLikeText(b) {
		t.Error("content with >10% non-printable bytes should be detected as binary")
	}
}

// --- stripANSI ---

func TestStripANSI(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"plain text", "plain text"},
		{"\033[31mred\033[0m", "red"},
		{"\033[1m\033[32mbold green\033[0m", "bold green"},
		{"no\033[33myellow\033[0mcolor", "noyellowcolor"},
	}
	for _, tc := range cases {
		got := stripANSI(tc.in)
		if got != tc.want {
			t.Errorf("stripANSI(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
