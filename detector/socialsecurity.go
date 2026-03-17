package detector

import "regexp"

var svnRE = regexp.MustCompile(`\b\d{2} ?\d{6} ?[A-Z] ?\d{3}\b`)

func detectSocialSecurity(text string) []Finding {
	var out []Finding
	for _, loc := range svnRE.FindAllStringIndex(text, -1) {
		out = append(out, Finding{
			Type: PiiSocialSec, Start: loc[0], End: loc[1],
			Text: text[loc[0]:loc[1]], Confidence: 0.9,
		})
	}
	return out
}
