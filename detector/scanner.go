// Package detector provides regex-based PII detection for DACH (DE/AT/CH) text.
// Supported types: EMAIL, PHONE, CREDIT_CARD, IBAN, PERSONAL_ID, SOCIAL_SECURITY,
// TAX_ID, VAT_ID, KVNR, LICENSE_PLATE, DRIVER_LICENSE, URL_SECRET, SECRET, ADDRESS.
package detector

import (
	"fmt"
	"sort"
	"strings"
)

// PiiType identifies the kind of PII.
type PiiType string

const (
	PiiEmail         PiiType = "EMAIL"
	PiiPhone         PiiType = "PHONE"
	PiiCreditCard    PiiType = "CREDIT_CARD"
	PiiIBAN          PiiType = "IBAN"
	PiiPersonalID    PiiType = "PERSONAL_ID"
	PiiSocialSec     PiiType = "SOCIAL_SECURITY"
	PiiTaxID         PiiType = "TAX_ID"
	PiiAddress       PiiType = "ADDRESS"
	PiiSecret        PiiType = "SECRET"
	PiiURLSecret     PiiType = "URL_SECRET"
	PiiKVNR          PiiType = "KVNR"
	PiiLicensePlate  PiiType = "LICENSE_PLATE"
	PiiVATID         PiiType = "VAT_ID"
	PiiDriverLicense PiiType = "DRIVER_LICENSE"
)

// piiPriority controls overlap resolution: higher value wins.
var piiPriority = map[PiiType]int{
	PiiSecret:        6,
	PiiURLSecret:     6,
	PiiIBAN:          5,
	PiiCreditCard:    5,
	PiiSocialSec:     5,
	PiiKVNR:          5,
	PiiPersonalID:    4,
	PiiTaxID:         4,
	PiiEmail:         4,
	PiiVATID:         4,
	PiiDriverLicense: 4,
	PiiPhone:         3,
	PiiLicensePlate:  3,
	PiiAddress:       2,
}

// Finding is a single PII detection result.
type Finding struct {
	Type        PiiType
	Start       int
	End         int
	Text        string
	Confidence  float64
	Placeholder string
	RuleID      string // only set for SECRET findings
}

// Scanner runs all enabled detectors and produces anonymised text.
type Scanner struct {
	enabledTypes map[PiiType]bool
	allTypes     bool
}

// NewScanner creates a Scanner. Pass an empty slice to enable all detectors.
func NewScanner(detectors []string) *Scanner {
	if len(detectors) == 0 {
		return &Scanner{allTypes: true}
	}
	m := make(map[PiiType]bool, len(detectors))
	for _, d := range detectors {
		m[PiiType(strings.ToUpper(d))] = true
	}
	return &Scanner{enabledTypes: m}
}

func (s *Scanner) isEnabled(t PiiType) bool {
	if s.allTypes {
		return true
	}
	return s.enabledTypes[t]
}

// Scan returns the anonymised text and the individual findings.
// Findings are sorted by start position (ascending) in the returned slice.
func (s *Scanner) Scan(text string) (string, []Finding) {
	return s.ScanWithWhitelist(text, nil)
}

// ScanWithWhitelist behaves like Scan, but suppresses findings whose matched
// text contains any whitelist token (case-insensitive).
func (s *Scanner) ScanWithWhitelist(text string, whitelist []string) (string, []Finding) {
	type entry struct {
		t  PiiType
		fn func(string) []Finding
	}
	detectors := []entry{
		{PiiSecret, detectSecrets},
		{PiiURLSecret, detectURLSecret},
		{PiiEmail, detectEmail},
		{PiiPhone, detectPhone},
		{PiiCreditCard, detectCreditCard},
		{PiiIBAN, detectIBAN},
		{PiiPersonalID, detectPersonalID},
		{PiiSocialSec, detectSocialSecurity},
		{PiiTaxID, detectTaxID},
		{PiiVATID, detectVATID},
		{PiiKVNR, detectKVNR},
		{PiiLicensePlate, detectLicensePlate},
		{PiiDriverLicense, detectDriverLicense},
		{PiiAddress, detectAddress},
	}

	var all []Finding
	for _, d := range detectors {
		if s.isEnabled(d.t) {
			all = append(all, d.fn(text)...)
		}
	}
	if len(all) == 0 {
		return text, nil
	}

	all = filterWhitelisted(all, whitelist)
	if len(all) == 0 {
		return text, nil
	}

	resolved := resolveOverlaps(all)

	// Assign placeholders; identical text → identical placeholder.
	counters := map[PiiType]int{}
	seenText := map[string]string{}
	for i := range resolved {
		f := &resolved[i]
		if ph, ok := seenText[f.Text]; ok {
			f.Placeholder = ph
		} else {
			counters[f.Type]++
			f.Placeholder = fmt.Sprintf("[%s_%d]", string(f.Type), counters[f.Type])
			seenText[f.Text] = f.Placeholder
		}
	}

	// Apply replacements right-to-left to preserve byte positions.
	byPos := make([]Finding, len(resolved))
	copy(byPos, resolved)
	sort.Slice(byPos, func(i, j int) bool { return byPos[i].Start > byPos[j].Start })

	result := []byte(text)
	for _, f := range byPos {
		result = append(result[:f.Start], append([]byte(f.Placeholder), result[f.End:]...)...)
	}

	// Return findings sorted ascending by start position.
	sort.Slice(resolved, func(i, j int) bool { return resolved[i].Start < resolved[j].Start })
	return string(result), resolved
}

func filterWhitelisted(findings []Finding, whitelist []string) []Finding {
	tokens := make([]string, 0, len(whitelist))
	for _, w := range whitelist {
		w = strings.TrimSpace(strings.ToLower(w))
		if w != "" {
			tokens = append(tokens, w)
		}
	}
	if len(tokens) == 0 {
		return findings
	}
	filtered := make([]Finding, 0, len(findings))
	for _, f := range findings {
		txt := strings.ToLower(f.Text)
		skip := false
		for _, t := range tokens {
			if strings.Contains(txt, t) {
				skip = true
				break
			}
		}
		if !skip {
			filtered = append(filtered, f)
		}
	}
	return filtered
}

// resolveOverlaps keeps the highest-priority non-overlapping finding at each position.
func resolveOverlaps(findings []Finding) []Finding {
	sort.SliceStable(findings, func(i, j int) bool {
		fi, fj := findings[i], findings[j]
		if fi.Start != fj.Start {
			return fi.Start < fj.Start
		}
		pi, pj := piiPriority[fi.Type], piiPriority[fj.Type]
		if pi != pj {
			return pi > pj
		}
		return (fi.End - fi.Start) > (fj.End - fj.Start)
	})
	var result []Finding
	lastEnd := -1
	for _, f := range findings {
		if f.Start >= lastEnd {
			result = append(result, f)
			lastEnd = f.End
		} else if len(result) > 0 {
			// Overlap: replace previous finding if current has higher priority or is longer at same priority.
			prev := result[len(result)-1]
			pi, pj := piiPriority[prev.Type], piiPriority[f.Type]
			if pj > pi || (pj == pi && (f.End-f.Start) > (prev.End-prev.Start)) {
				result[len(result)-1] = f
				lastEnd = f.End
			}
		}
	}
	return result
}
