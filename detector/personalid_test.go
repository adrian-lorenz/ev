package detector

import "testing"

func TestDetectPersonalID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"personalausweis", "Ausweis: C22990047", []string{"C22990047"}},
		{"reisepass", "Pass: C12345678", []string{"C12345678"}},
		{"alphanumeric", "L01X00T47", []string{"L01X00T47"}},
		{"multiple", "C22990047 und L01X00T47", []string{"C22990047", "L01X00T47"}},
		{"no match too short", "C2299004", nil},
		{"no match too long not at boundary", "C229900471", nil},
		{"no match lowercase", "c22990047", nil},
		{"no match digits only", "123456789", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectPersonalID(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d findings, want %d — %v", len(got), len(tt.want), got)
			}
			for i, f := range got {
				if f.Text != tt.want[i] {
					t.Errorf("[%d] text = %q, want %q", i, f.Text, tt.want[i])
				}
				if f.Confidence != 0.75 {
					t.Errorf("[%d] confidence = %v, want 0.75", i, f.Confidence)
				}
			}
		})
	}
}
