package detector

import "testing"

func TestDetectEmail(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string // expected matched texts
	}{
		{"basic", "hans.mueller@example.de", []string{"hans.mueller@example.de"}},
		{"with plus", "user+tag@domain.com is my mail", []string{"user+tag@domain.com"}},
		{"subdomain", "info@mail.company.co.at", []string{"info@mail.company.co.at"}},
		{"in sentence", "Schreib mir an foo@bar.com bitte", []string{"foo@bar.com"}},
		{"multiple", "a@b.de and c@d.com", []string{"a@b.de", "c@d.com"}},
		{"no match", "Kein E-Mail hier", nil},
		{"no match bare domain", "example.com", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectEmail(tt.input)
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
				if f.Type != PiiEmail {
					t.Errorf("[%d] type = %v, want %v", i, f.Type, PiiEmail)
				}
			}
		})
	}
}
