package detector

import "regexp"

var urlSecretRE = regexp.MustCompile(
	`(?i)\b(?:token|api[_\-]?key|apikey|api[_\-]?token|access[_\-]?token|auth[_\-]?token|auth|secret|password|passwd|pwd|client[_\-]?secret|private[_\-]?key)=([^&\s"'<>\[\]{}]{6,})`,
)

func detectURLSecret(text string) []Finding {
	var out []Finding
	for _, m := range urlSecretRE.FindAllStringSubmatchIndex(text, -1) {
		out = append(out, Finding{
			Type: PiiURLSecret, Start: m[2], End: m[3],
			Text: text[m[2]:m[3]], Confidence: 0.85,
		})
	}
	return out
}
