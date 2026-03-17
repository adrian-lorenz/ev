package detector

import "testing"

func TestDetectTaxID(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTexts []string
		wantConfs []float64
	}{
		{
			"valid with spaces",
			"IdNr: 86 095 742 719",
			[]string{"86 095 742 719"}, []float64{1.0},
		},
		{
			"valid raw",
			"86095742719",
			[]string{"86095742719"}, []float64{1.0},
		},
		{
			"invalid checksum low confidence",
			"86 095 742 710",
			[]string{"86 095 742 710"}, []float64{0.6},
		},
		{
			"multiple",
			"86095742719 und 47 036 892 816",
			[]string{"86095742719", "47 036 892 816"}, []float64{1.0, 1.0},
		},
		{
			"no match starts with zero",
			"01 234 567 890",
			nil, nil,
		},
		{
			"no match too short",
			"1234567890",
			nil, nil,
		},
		{
			"no match too long",
			"123456789012",
			nil, nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectTaxID(tt.input)
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

func TestTaxIDCheckDigit(t *testing.T) {
	tests := []struct {
		digits    string
		wantDigit int
		wantOK    bool
	}{
		{"86095742719", 9, true},
		{"47036892816", 6, true},
		{"01234567890", 0, false}, // starts with 0
	}
	for _, tt := range tests {
		d, ok := taxIDCheckDigit(tt.digits)
		if ok != tt.wantOK {
			t.Errorf("taxIDCheckDigit(%q) ok = %v, want %v", tt.digits, ok, tt.wantOK)
			continue
		}
		if ok && d != tt.wantDigit {
			t.Errorf("taxIDCheckDigit(%q) = %d, want %d", tt.digits, d, tt.wantDigit)
		}
	}
}
