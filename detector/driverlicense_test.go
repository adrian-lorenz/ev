package detector

import "testing"

func TestDetectDriverLicense(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			"with führerschein keyword",
			"Führerscheinnummer: B951204XY",
			[]string{"B951204XY"},
		},
		{
			"with fahrerlaubnis keyword",
			"Fahrerlaubnis B951204XY wurde ausgestellt",
			[]string{"B951204XY"},
		},
		{
			"with driver license keyword",
			"Driver license: B951204XY",
			[]string{"B951204XY"},
		},
		{
			"with driving licence keyword",
			"Driving licence B951204XY",
			[]string{"B951204XY"},
		},
		{
			"with FS-Nr keyword",
			"FS-Nr. B951204XY",
			[]string{"B951204XY"},
		},
		{
			"multiple licenses",
			"Führerschein B951204XY und Führerschein M123456AB",
			[]string{"B951204XY", "M123456AB"},
		},
		{
			"no match without context",
			"B951204XY",
			nil,
		},
		{
			"no match context out of window",
			"Führerschein " + repeatStr("x", 250) + " B951204XY",
			nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectDriverLicense(tt.input)
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

func repeatStr(s string, n int) string {
	result := make([]byte, len(s)*n)
	for i := range n {
		copy(result[i*len(s):], s)
	}
	return string(result)
}
