package detector

import "testing"

func TestDetectPhone(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"german mobile", "Ruf mich an: 0171 1234567", []string{"0171 1234567"}},
		{"german landline", "030 12345678", []string{"030 12345678"}},
		{"with country code +49", "+49 171 1234567", []string{"+49 171 1234567"}},
		{"austrian +43", "+43 1 58858-0", []string{"+43 1 58858-0"}},
		{"swiss +41", "+41 44 668 18 00", []string{"+41 44 668 18 00"}},
		{"double zero prefix", "0049 30 12345678", []string{"0049 30 12345678"}},
		{"multiple", "0171 1234567 und 030 98765432", []string{"0171 1234567", "030 98765432"}},
		{"no match year", "im Jahr 2024 passierte", nil},
		{"no match short", "0800", nil},
		{"no match all zeros", "000 000 000", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectPhone(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d findings, want %d — findings: %v", len(got), len(tt.want), got)
			}
			for i, f := range got {
				if f.Text != tt.want[i] {
					t.Errorf("[%d] text = %q, want %q", i, f.Text, tt.want[i])
				}
				if f.Confidence != 1.0 {
					t.Errorf("[%d] confidence = %v, want 1.0", i, f.Confidence)
				}
			}
		})
	}
}
