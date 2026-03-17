package detector

import (
	"regexp"
	"strings"
)

var driverLicenseRE = regexp.MustCompile(`\b[A-Z]{1,3}[0-9]{6}[A-Z0-9]{2}\b`)
var driverContextRE = regexp.MustCompile(`(?i)f[uü]hrerschein|fahrerlaubnis|fs[-\s]?nr|driver\s+licen[sc]e|driving\s+licen[sc]e`)

const driverContextWindow = 200

func detectDriverLicense(text string) []Finding {
	var out []Finding
	lower := strings.ToLower(text)
	for _, loc := range driverLicenseRE.FindAllStringIndex(text, -1) {
		ws := max(0, loc[0]-driverContextWindow)
		we := min(len(text), loc[1]+driverContextWindow)
		if driverContextRE.MatchString(lower[ws:we]) {
			out = append(out, Finding{
				Type: PiiDriverLicense, Start: loc[0], End: loc[1],
				Text: text[loc[0]:loc[1]], Confidence: 0.75,
			})
		}
	}
	return out
}
