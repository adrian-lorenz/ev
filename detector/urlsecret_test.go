package detector

import "testing"

func TestDetectURLSecret(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTexts []string // only the secret value, not the key
	}{
		{"token param", "https://api.example.com?token=abc123def456", []string{"abc123def456"}},
		{"api_key param", "https://api.example.com?api_key=mysecretkey123", []string{"mysecretkey123"}},
		{"password param", "login?password=s3cr3tP@ss!", []string{"s3cr3tP@ss!"}},
		{"access_token param", "?access_token=eyJhbGciOiJIUzI1NiJ9", []string{"eyJhbGciOiJIUzI1NiJ9"}},
		{"client_secret param", "?client_secret=supersecretvalue", []string{"supersecretvalue"}},
		{"multiple params", "?token=abc123def&api_key=xyz789ghi000", []string{"abc123def", "xyz789ghi000"}},
		{"no match value too short", "?token=abc12", nil},
		{"no match no sensitive key", "?foo=bar123", nil},
		{"key name not suffix", "?mytoken=abc123def456", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectURLSecret(tt.input)
			if len(got) != len(tt.wantTexts) {
				t.Fatalf("got %d findings, want %d — %v", len(got), len(tt.wantTexts), got)
			}
			for i, f := range got {
				if f.Text != tt.wantTexts[i] {
					t.Errorf("[%d] text = %q, want %q", i, f.Text, tt.wantTexts[i])
				}
				if f.Confidence != 0.85 {
					t.Errorf("[%d] confidence = %v, want 0.85", i, f.Confidence)
				}
				// key name must NOT be part of the finding text
				if f.Type != PiiURLSecret {
					t.Errorf("[%d] type = %v, want URL_SECRET", i, f.Type)
				}
			}
		})
	}
}
