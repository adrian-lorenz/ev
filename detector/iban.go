package detector

import (
	"regexp"
	"strings"
)

var ibanLengths = map[string]int{
	"AD": 24, "AE": 23, "AL": 28, "AT": 20, "AZ": 28,
	"BA": 20, "BE": 16, "BG": 22, "BH": 22, "BR": 29,
	"BY": 28, "CH": 21, "CR": 22, "CY": 28, "CZ": 24,
	"DE": 22, "DK": 18, "DO": 28, "EE": 20, "EG": 29,
	"ES": 24, "FI": 18, "FK": 18, "FO": 18, "FR": 27,
	"GB": 22, "GE": 22, "GI": 23, "GL": 18, "GR": 27,
	"GT": 28, "HR": 21, "HU": 28, "IE": 22, "IL": 23,
	"IQ": 23, "IS": 26, "IT": 27, "JO": 30, "KW": 30,
	"KZ": 20, "LB": 28, "LC": 32, "LI": 21, "LT": 20,
	"LU": 20, "LV": 21, "LY": 25, "MC": 27, "MD": 24,
	"ME": 22, "MK": 19, "MR": 27, "MT": 31, "MU": 30,
	"NI": 32, "NL": 18, "NO": 15, "PK": 24, "PL": 28,
	"PS": 29, "PT": 25, "QA": 29, "RO": 24, "RS": 22,
	"SA": 24, "SC": 31, "SD": 18, "SE": 24, "SI": 19,
	"SK": 24, "SM": 27, "ST": 25, "SV": 28, "TL": 23,
	"TN": 24, "TR": 26, "UA": 29, "VA": 22, "VG": 24,
	"XK": 20,
}

var ibanRE = regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9 ]{11,31}\b`)

func mod97(s string) int {
	rem := 0
	for _, ch := range s {
		if ch >= '0' && ch <= '9' {
			rem = (rem*10 + int(ch-'0')) % 97
		} else {
			rem = (rem*100 + int(ch-'A') + 10) % 97
		}
	}
	return rem
}

func detectIBAN(text string) []Finding {
	var out []Finding
	for _, loc := range ibanRE.FindAllStringIndex(text, -1) {
		raw := strings.TrimRight(text[loc[0]:loc[1]], " ")
		clean := strings.ToUpper(strings.ReplaceAll(raw, " ", ""))
		expected, ok := ibanLengths[clean[:2]]
		if !ok || len(clean) != expected {
			continue
		}
		rearranged := clean[4:] + clean[:4]
		confidence := 1.0
		if mod97(rearranged) != 1 {
			confidence = 0.6
		}
		out = append(out, Finding{
			Type: PiiIBAN, Start: loc[0], End: loc[0] + len(raw),
			Text: raw, Confidence: confidence,
		})
	}
	return out
}
