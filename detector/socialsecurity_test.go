package detector

import "testing"

func TestDetectSocialSecurity(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"formatted", "SVN: 65 180675 B 003", []string{"65 180675 B 003"}},
		{"compact no spaces", "12345678X123", []string{"12345678X123"}},
		{"partial spaces", "12 345678X123", []string{"12 345678X123"}},
		{"with spaces", "12 345678 X 123", []string{"12 345678 X 123"}},
		{"multiple", "65 180675 B 003 und 12345678X123", []string{"65 180675 B 003", "12345678X123"}},
		{"no match lowercase letter", "12345678x123", nil},
		{"no match all digits", "12345678123", nil},
		{"no match too short", "1234567X12", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectSocialSecurity(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d findings, want %d — %v", len(got), len(tt.want), got)
			}
			for i, f := range got {
				if f.Text != tt.want[i] {
					t.Errorf("[%d] text = %q, want %q", i, f.Text, tt.want[i])
				}
				if f.Confidence != 0.9 {
					t.Errorf("[%d] confidence = %v, want 0.9", i, f.Confidence)
				}
			}
		})
	}
}
