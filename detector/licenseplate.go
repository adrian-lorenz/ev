package detector

import (
	"regexp"
	"strings"
)

var lpHyphenRE = regexp.MustCompile(`\b([A-ZÄÖÜ]{1,3})-([A-Z]{1,2}) ?([1-9][0-9]{0,3}[EH]?)\b`)
var lpSpaceRE = regexp.MustCompile(`\b([A-ZÄÖÜ]{1,3}) ([A-Z]{1,2}) ([1-9][0-9]{0,3}[EH]?)\b`)

func validPlate(district, letters, digits string) bool {
	base := strings.TrimRight(digits, "EH")
	total := len([]rune(district)) + len(letters) + len(base)
	return total >= 4 && total <= 8
}

func detectLicensePlate(text string) []Finding {
	var out []Finding
	seen := map[[2]int]bool{}
	for _, m := range lpHyphenRE.FindAllStringSubmatchIndex(text, -1) {
		if validPlate(text[m[2]:m[3]], text[m[4]:m[5]], text[m[6]:m[7]]) {
			key := [2]int{m[0], m[1]}
			seen[key] = true
			out = append(out, Finding{
				Type: PiiLicensePlate, Start: m[0], End: m[1],
				Text: text[m[0]:m[1]], Confidence: 0.75,
			})
		}
	}
	for _, m := range lpSpaceRE.FindAllStringSubmatchIndex(text, -1) {
		if validPlate(text[m[2]:m[3]], text[m[4]:m[5]], text[m[6]:m[7]]) {
			if !seen[[2]int{m[0], m[1]}] {
				out = append(out, Finding{
					Type: PiiLicensePlate, Start: m[0], End: m[1],
					Text: text[m[0]:m[1]], Confidence: 0.65,
				})
			}
		}
	}
	return out
}
