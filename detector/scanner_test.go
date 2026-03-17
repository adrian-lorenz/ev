package detector

import (
	"strings"
	"testing"
)

const (
	awsKey  = "AKIAIOSFODNN7EXAMPLE"
	awsKey2 = "AKIAI44QH8DHBEXAMPLE"
)

func TestScannerSingleSecret(t *testing.T) {
	s := NewScanner(nil)
	anon, findings := s.Scan("key=" + awsKey)
	if !strings.Contains(anon, "[SECRET_1]") {
		t.Errorf("expected [SECRET_1] in %q", anon)
	}
	if strings.Contains(anon, awsKey) {
		t.Errorf("secret should be redacted in %q", anon)
	}
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}
}

func TestScannerNoMatch(t *testing.T) {
	s := NewScanner(nil)
	text := "Kein Secret hier."
	anon, findings := s.Scan(text)
	if anon != text {
		t.Errorf("text should be unchanged, got %q", anon)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestScannerDeduplication(t *testing.T) {
	s := NewScanner(nil)
	// Same AWS key twice → same placeholder both times
	text := awsKey + " and " + awsKey
	anon, findings := s.Scan(text)
	count := strings.Count(anon, "[SECRET_1]")
	if count != 2 {
		t.Errorf("expected [SECRET_1] twice, got %d times in %q", count, anon)
	}
	seen := map[string]bool{}
	for _, f := range findings {
		seen[f.Placeholder] = true
	}
	if len(seen) != 1 {
		t.Errorf("expected 1 unique placeholder, got %d: %v", len(seen), findings)
	}
}

func TestScannerFindingsSortedByStart(t *testing.T) {
	s := NewScanner(nil)
	_, findings := s.Scan(awsKey + " later " + awsKey2)
	for i := 1; i < len(findings); i++ {
		if findings[i].Start < findings[i-1].Start {
			t.Errorf("findings not sorted: [%d].Start=%d < [%d].Start=%d",
				i, findings[i].Start, i-1, findings[i-1].Start)
		}
	}
}

func TestScannerRestoreRoundtrip(t *testing.T) {
	s := NewScanner(nil)
	text := "Use key=" + awsKey + " to authenticate."
	anon, findings := s.Scan(text)
	restore := map[string]string{}
	for _, f := range findings {
		restore[f.Placeholder] = f.Text
	}
	restored := anon
	for ph, orig := range restore {
		restored = strings.ReplaceAll(restored, ph, orig)
	}
	if restored != text {
		t.Errorf("roundtrip failed:\n  got  %q\n  want %q", restored, text)
	}
}

func TestScannerSelectiveDetectors(t *testing.T) {
	// Only enable SECRET — second key should also be redacted
	s := NewScanner([]string{"SECRET"})
	text := awsKey + " and " + awsKey2
	anon, findings := s.Scan(text)
	if !strings.Contains(anon, "[SECRET_1]") {
		t.Errorf("expected [SECRET_1] in %q", anon)
	}
	for _, f := range findings {
		if f.Type != PiiSecret {
			t.Errorf("expected only SECRET findings, got %v", f.Type)
		}
	}
}

func TestScannerPlaceholderFormat(t *testing.T) {
	s := NewScanner(nil)
	_, findings := s.Scan("key=" + awsKey)
	if len(findings) == 0 {
		t.Fatal("expected a finding")
	}
	f := findings[0]
	if f.Placeholder != "[SECRET_1]" {
		t.Errorf("placeholder = %q, want [SECRET_1]", f.Placeholder)
	}
	if f.Type != PiiSecret {
		t.Errorf("type = %v, want SECRET", f.Type)
	}
}

func TestScannerSecretDetected(t *testing.T) {
	s := NewScanner(nil)
	anon, findings := s.Scan("key=" + awsKey)
	if !strings.Contains(anon, "[SECRET_1]") {
		t.Errorf("expected [SECRET_1] in %q", anon)
	}
	if len(findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	if findings[0].Type != PiiSecret {
		t.Errorf("type = %v, want SECRET", findings[0].Type)
	}
}

func TestScannerWhitelistSkipsMatch(t *testing.T) {
	s := NewScanner(nil)
	text := awsKey + " and " + awsKey2
	anon, findings := s.ScanWithWhitelist(text, []string{awsKey})
	if strings.Contains(anon, awsKey2) {
		t.Fatalf("non-whitelisted key must be masked: %q", anon)
	}
	if !strings.Contains(anon, awsKey) {
		t.Fatalf("whitelisted key must remain: %q", anon)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(findings), findings)
	}
}

func TestScannerCountersPerType(t *testing.T) {
	s := NewScanner(nil)
	// Two different AWS keys → SECRET_1 and SECRET_2
	anon, _ := s.Scan(awsKey + " and " + awsKey2)
	if !strings.Contains(anon, "[SECRET_1]") {
		t.Errorf("expected [SECRET_1] in %q", anon)
	}
	if !strings.Contains(anon, "[SECRET_2]") {
		t.Errorf("expected [SECRET_2] in %q", anon)
	}
}

func TestOverlapPriorityResolution(t *testing.T) {
	// Two overlapping findings: longer match wins at equal priority.
	short := Finding{Type: PiiSecret, Start: 0, End: 30, Confidence: 0.9}
	long := Finding{Type: PiiSecret, Start: 10, End: 70, Confidence: 1.0}
	result := resolveOverlaps([]Finding{short, long})
	if len(result) != 1 {
		t.Fatalf("expected 1 finding after resolution, got %d: %v", len(result), result)
	}
}
