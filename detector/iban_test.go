package detector

import "testing"

func TestDetectIBAN(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTexts []string
		wantConfs []float64
	}{
		{
			"german with spaces",
			"IBAN: DE89 3704 0044 0532 0130 00",
			[]string{"DE89 3704 0044 0532 0130 00"}, []float64{1.0},
		},
		{
			"german without spaces",
			"DE89370400440532013000",
			[]string{"DE89370400440532013000"}, []float64{1.0},
		},
		{
			"austrian",
			"AT61 1904 3002 3457 3201",
			[]string{"AT61 1904 3002 3457 3201"}, []float64{1.0},
		},
		{
			"swiss",
			"CH56 0483 5012 3456 7800 9",
			[]string{"CH56 0483 5012 3456 7800 9"}, []float64{1.0},
		},
		{
			"invalid checksum low confidence",
			"DE00370400440532013000",
			[]string{"DE00370400440532013000"}, []float64{0.6},
		},
		{
			"invalid country no match",
			"XX89370400440532013000",
			nil, nil,
		},
		{
			"wrong length no match",
			"DE89 3704 0044 0532 01",
			nil, nil,
		},
		{
			"multiple IBANs",
			"DE89370400440532013000 und AT61 1904 3002 3457 3201",
			[]string{"DE89370400440532013000", "AT61 1904 3002 3457 3201"}, []float64{1.0, 1.0},
		},
		{
			"no match plain text",
			"Kein IBAN hier",
			nil, nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectIBAN(tt.input)
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

func TestMod97(t *testing.T) {
	// Rearranged DE89... → "370400440532013000DE89" numerically → mod97 == 1
	if got := mod97("370400440532013000131489"); got != 1 {
		t.Errorf("mod97 = %d, want 1", got)
	}
}
