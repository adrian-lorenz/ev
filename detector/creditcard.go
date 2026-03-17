package detector

import (
	"regexp"
	"strings"
)

var ccRE = regexp.MustCompile(
	`\d{4}[ \-]\d{4}[ \-]\d{4}[ \-]\d{4}|\d{4}[ \-]\d{6}[ \-]\d{5}|\d{4}[ \-]\d{6}[ \-]\d{4}|\d{13,19}`,
)

func luhnValid(digits string) bool {
	rev := []rune(digits)
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	total := 0
	for i, ch := range rev {
		n := int(ch - '0')
		if i%2 == 1 {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		total += n
	}
	return total%10 == 0
}

func isDigit(b byte) bool { return b >= '0' && b <= '9' }

func detectCreditCard(text string) []Finding {
	var out []Finding
	for _, loc := range ccRE.FindAllStringIndex(text, -1) {
		if loc[0] > 0 && isDigit(text[loc[0]-1]) {
			continue
		}
		if loc[1] < len(text) && isDigit(text[loc[1]]) {
			continue
		}
		raw := text[loc[0]:loc[1]]
		isFormatted := strings.ContainsAny(raw, " -")
		digits := strings.Map(func(r rune) rune {
			if r >= '0' && r <= '9' {
				return r
			}
			return -1
		}, raw)
		luhnOK := luhnValid(digits)
		var confidence float64
		if isFormatted {
			if luhnOK {
				confidence = 1.0
			} else {
				confidence = 0.6
			}
		} else {
			if !luhnOK {
				continue
			}
			confidence = 0.9
		}
		out = append(out, Finding{
			Type: PiiCreditCard, Start: loc[0], End: loc[1],
			Text: raw, Confidence: confidence,
		})
	}
	return out
}
