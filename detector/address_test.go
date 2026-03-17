package detector

import "testing"

func TestDetectAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantN   int
		wantMin string // substring that must appear in first match
	}{
		{
			"german address",
			"Hauptstraße 12, 10115 Berlin",
			1, "Hauptstraße 12",
		},
		{
			"austrian address",
			"Mariahilfer Straße 100, 1070 Wien",
			1, "Mariahilfer Straße 100",
		},
		{
			"swiss address",
			"Bahnhofstrasse 21, 8001 Zürich",
			1, "Bahnhofstrasse 21",
		},
		{
			"weg suffix",
			"Birkenweg 5, 80331 München",
			1, "Birkenweg 5",
		},
		{
			"platz suffix",
			"Marienplatz 3, 80331 München",
			1, "Marienplatz 3",
		},
		{
			"house number with letter",
			"Musterstraße 12a, 10115 Berlin",
			1, "Musterstraße 12a",
		},
		{
			"multiple addresses",
			"Hauptstraße 12, 10115 Berlin; Bahnhofstrasse 21, 8001 Zürich",
			2, "",
		},
		{
			"no match no postal code",
			"Die Straße ist lang und die Nacht ist finster",
			0, "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectAddress(tt.input)
			if len(got) != tt.wantN {
				t.Fatalf("got %d findings, want %d — %v", len(got), tt.wantN, got)
			}
			if tt.wantMin != "" && tt.wantN > 0 {
				if got[0].Type != PiiAddress {
					t.Errorf("type = %v, want ADDRESS", got[0].Type)
				}
				if got[0].Confidence != 0.9 {
					t.Errorf("confidence = %v, want 0.9", got[0].Confidence)
				}
				found := false
				for _, f := range got {
					if contains(f.Text, tt.wantMin) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("no finding contains %q — got %v", tt.wantMin, got)
				}
			}
		})
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := range len(s) - len(sub) + 1 {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
