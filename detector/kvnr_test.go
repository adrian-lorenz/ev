package detector

import "testing"

func TestDetectKVNR(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTexts []string
		wantConfs []float64
	}{
		{
			"valid checksum",
			"KVNR: T123456780",
			[]string{"T123456780"}, []float64{0.95},
		},
		{
			"invalid checksum low confidence",
			"T123456789",
			[]string{"T123456789"}, []float64{0.6},
		},
		{
			"multiple",
			"T123456780 und A000000002",
			[]string{"T123456780", "A000000002"}, []float64{0.95, 0.95},
		},
		{
			"no match lowercase",
			"t123456780",
			nil, nil,
		},
		{
			"no match too short",
			"T12345678",
			nil, nil,
		},
		{
			"no match all digits",
			"1234567890",
			nil, nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectKVNR(tt.input)
			if len(got) != len(tt.wantTexts) {
				t.Fatalf("got %d findings, want %d — %v", len(got), len(tt.wantTexts), got)
			}
			for i, f := range got {
				if f.Text != tt.wantTexts[i] {
					t.Errorf("[%d] text = %q, want %q", i, f.Text, tt.wantTexts[i])
				}
				if f.Confidence != tt.wantConfs[i] {
					t.Errorf("[%d] confidence = %v, want %v", i, f.Confidence, tt.wantConfs[i])
				}
			}
		})
	}
}

func TestKVNRChecksumValid(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{"T123456780", true},
		{"T123456789", false},
		{"A000000002", true},
	}
	for _, tt := range tests {
		if got := kvnrChecksumValid(tt.raw); got != tt.want {
			t.Errorf("kvnrChecksumValid(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}
