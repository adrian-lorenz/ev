package detector

import "regexp"

var personalIDRE = regexp.MustCompile(`\b[A-Z][A-Z0-9]{8}\b`)

func detectPersonalID(text string) []Finding {
	var out []Finding
	for _, loc := range personalIDRE.FindAllStringIndex(text, -1) {
		out = append(out, Finding{
			Type: PiiPersonalID, Start: loc[0], End: loc[1],
			Text: text[loc[0]:loc[1]], Confidence: 0.75,
		})
	}
	return out
}
