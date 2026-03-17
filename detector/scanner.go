// Package detector scans text for leaked secrets and credentials.
package detector

import (
	"fmt"
	"sort"
	"strings"
)

// PiiType identifies the kind of finding.
type PiiType string

const (
	PiiSecret PiiType = "SECRET"
)

// piiPriority controls overlap resolution: higher value wins.
var piiPriority = map[PiiType]int{
	PiiSecret: 6,
}

// Finding is a single detection result.
type Finding struct {
	Type        PiiType
	Start       int
	End         int
	Text        string
	Confidence  float64
	Placeholder string
	RuleID      string // only set for SECRET findings
}

// Scanner runs enabled detectors over text.
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
