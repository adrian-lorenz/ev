package detector

import "regexp"

var emailRE = regexp.MustCompile(`(?i)\b[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}\b`)

func detectEmail(text string) []Finding {
	var out []Finding
	for _, loc := range emailRE.FindAllStringIndex(text, -1) {
		out = append(out, Finding{
			Type: PiiEmail, Start: loc[0], End: loc[1],
			Text: text[loc[0]:loc[1]], Confidence: 1.0,
		})
	}
	return out
}
