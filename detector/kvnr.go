package detector

import (
	"fmt"
	"regexp"
)

var kvnrRE = regexp.MustCompile(`\b[A-Z][0-9]{9}\b`)

// kvnrChecksumValid implements the modified Luhn check per § 290 SGB V.
func kvnrChecksumValid(raw string) bool {
	letterValue := int(raw[0]-'A') + 1
	digitsStr := fmt.Sprintf("%02d", letterValue) + raw[1:]
	weights := []int{1, 2, 1, 2, 1, 2, 1, 2, 1, 2}
	total := 0
	for i, ch := range digitsStr[:10] {
		p := int(ch-'0') * weights[i]
		total += p/10 + p%10
	}
	return total%10 == int(raw[9]-'0')
}

func detectKVNR(text string) []Finding {
	var out []Finding
	for _, loc := range kvnrRE.FindAllStringIndex(text, -1) {
		raw := text[loc[0]:loc[1]]
		confidence := 0.6
		if kvnrChecksumValid(raw) {
			confidence = 0.95
		}
		out = append(out, Finding{
			Type: PiiKVNR, Start: loc[0], End: loc[1],
			Text: raw, Confidence: confidence,
		})
	}
	return out
}
