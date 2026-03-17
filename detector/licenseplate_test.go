package detector

import "testing"

func TestDetectLicensePlate(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTexts []string
		wantConfs []float64
	}{
		{
			"standard hyphen",
			"B-AB 1234",
			[]string{"B-AB 1234"}, []float64{0.75},
		},
		{
			"three letter district",
			"MÜN-XY 99",
			[]string{"MÜN-XY 99"}, []float64{0.75},
		},
		{
			"e-vehicle suffix",
			"HH-AB 1234E",
			[]string{"HH-AB 1234E"}, []float64{0.75},
		},
		{
			"historic suffix",
			"K-CD 567H",
			[]string{"K-CD 567H"}, []float64{0.75},
		},
		{
			"space format lower confidence",
			"B AB 1234",
			[]string{"B AB 1234"}, []float64{0.65},
		},
		{
			"single letter district hyphen",
			"M-A 12",
			[]string{"M-A 12"}, []float64{0.75},
		},
		{
			"multiple",
			"B-AB 1234 und M-XY 999",
			[]string{"B-AB 1234", "M-XY 999"}, []float64{0.75, 0.75},
		},
		{
			"no match number starts with zero",
			"B-AB 0123",
			nil, nil,
		},
		{
			"no match too long district",
			"ABCD-XY 123",
			nil, nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectLicensePlate(tt.input)
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
