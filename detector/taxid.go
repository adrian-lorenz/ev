package detector

import (
	"regexp"
	"strings"
)

var taxIDRE = regexp.MustCompile(`\b[1-9]\d(?: ?\d{3}){3}\b`)

// taxIDCheckDigit computes the expected 11th digit per § 139b AO.
// Returns (digit, true) on success or (0, false) if structurally invalid.
func taxIDCheckDigit(digits string) (int, bool) {
	if digits[0] == '0' {
		return 0, false
	}
	product := 10
	for _, d := range digits[:10] {
		total := (product + int(d-'0')) % 10
		if total == 0 {
			total = 10
		}
		product = (total * 2) % 11
	}
	check := 11 - product
	if check == 10 {
		return 0, false
	}
	if check == 11 {
		check = 0
	}
	return check, true
}

func detectTaxID(text string) []Finding {
	var out []Finding
	for _, loc := range taxIDRE.FindAllStringIndex(text, -1) {
		raw := text[loc[0]:loc[1]]
		digits := strings.ReplaceAll(raw, " ", "")
		if len(digits) != 11 {
			continue
		}
		expected, ok := taxIDCheckDigit(digits)
		if !ok {
			continue
		}
		confidence := 1.0
		if int(digits[10]-'0') != expected {
			confidence = 0.6
		}
		out = append(out, Finding{
			Type: PiiTaxID, Start: loc[0], End: loc[1],
			Text: raw, Confidence: confidence,
		})
	}
	return out
}
