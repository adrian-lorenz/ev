package detector

import (
	"regexp"
	"strings"
)

var phoneRE = regexp.MustCompile(
	`(?:(?:\+|00)(?:49|43|41)[\s()\-]*(?:\(0\)[\s()\-]*)?\d[\d\s()\-]{5,16}\d|0[1-9]\d[\d\s\-/]{5,13}\d)`,
)

func detectPhone(text string) []Finding {
	var out []Finding
	for _, loc := range phoneRE.FindAllStringIndex(text, -1) {
		raw := strings.TrimRight(text[loc[0]:loc[1]], " \t")
		digits := 0
		for _, c := range raw {
			if c >= '0' && c <= '9' {
				digits++
			}
		}
		if digits < 9 {
			continue
		}
		if loc[0] > 0 {
			prev := text[loc[0]-1]
			if prev >= '0' && prev <= '9' || prev == '+' {
				continue
			}
		}
		end := loc[0] + len(raw)
		if end < len(text) && text[end] >= '0' && text[end] <= '9' {
			continue
		}
		out = append(out, Finding{
			Type: PiiPhone, Start: loc[0], End: end,
			Text: raw, Confidence: 1.0,
		})
	}
	return out
}
