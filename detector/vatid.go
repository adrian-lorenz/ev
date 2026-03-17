package detector

import (
	"regexp"
	"strings"
)

var vatIDRE = regexp.MustCompile(`\bDE ?[0-9]{3} ?[0-9]{3} ?[0-9]{3}\b`)

func detectVATID(text string) []Finding {
	var out []Finding
	for _, loc := range vatIDRE.FindAllStringIndex(text, -1) {
		raw := text[loc[0]:loc[1]]
		if len(strings.ReplaceAll(raw, " ", "")) != 11 {
			continue
		}
		out = append(out, Finding{
			Type: PiiVATID, Start: loc[0], End: loc[1],
			Text: raw, Confidence: 0.85,
		})
	}
	return out
}
