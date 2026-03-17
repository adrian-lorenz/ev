package detector

import "testing"

func TestDetectVATID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"plain", "DE123456789", []string{"DE123456789"}},
		{"spaced", "DE 123 456 789", []string{"DE 123 456 789"}},
		{"partially spaced", "DE123 456 789", []string{"DE123 456 789"}},
		{"multiple", "DE123456789 und DE987654321", []string{"DE123456789", "DE987654321"}},
		{"no match lowercase prefix", "de123456789", nil},
		{"no match wrong country", "FR12345678901", nil},
		{"no match too few digits", "DE12345678", nil},
		{"no match too many digits", "DE1234567890", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectVATID(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d findings, want %d — %v", len(got), len(tt.want), got)
			}
			for i, f := range got {
				if f.Text != tt.want[i] {
					t.Errorf("[%d] text = %q, want %q", i, f.Text, tt.want[i])
				}
				if f.Confidence != 0.85 {
					t.Errorf("[%d] confidence = %v, want 0.85", i, f.Confidence)
				}
			}
		})
	}
}
