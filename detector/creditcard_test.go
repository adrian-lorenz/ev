package detector

import "testing"

func TestDetectCreditCard(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTexts []string
		wantConfs []float64
	}{
		{
			"visa spaces",
			"4111 1111 1111 1111",
			[]string{"4111 1111 1111 1111"}, []float64{1.0},
		},
		{
			"visa dashes",
			"4111-1111-1111-1111",
			[]string{"4111-1111-1111-1111"}, []float64{1.0},
		},
		{
			"amex formatted",
			"3782 822463 10005",
			[]string{"3782 822463 10005"}, []float64{1.0},
		},
		{
			"raw no spaces valid luhn",
			"4111111111111111",
			[]string{"4111111111111111"}, []float64{0.9},
		},
		{
			"formatted invalid luhn",
			"4111 1111 1111 1112",
			[]string{"4111 1111 1111 1112"}, []float64{0.6},
		},
		{
			"raw invalid luhn no match",
			"4111111111111112",
			nil, nil,
		},
		{
			"multiple cards",
			"4111 1111 1111 1111 and 5500 0000 0000 0004",
			[]string{"4111 1111 1111 1111", "5500 0000 0000 0004"}, []float64{1.0, 1.0},
		},
		{
			"no match short",
			"1234 5678",
			nil, nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectCreditCard(tt.input)
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

func TestLuhnValid(t *testing.T) {
	tests := []struct {
		digits string
		want   bool
	}{
		{"4111111111111111", true},
		{"4111111111111112", false},
		{"378282246310005", true},  // AmEx
		{"5500000000000004", true}, // Mastercard
	}
	for _, tt := range tests {
		if got := luhnValid(tt.digits); got != tt.want {
			t.Errorf("luhnValid(%q) = %v, want %v", tt.digits, got, tt.want)
		}
	}
}
