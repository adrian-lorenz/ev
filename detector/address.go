package detector

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var (
	addressRE      *regexp.Regexp
	plzPrefilterRE = regexp.MustCompile(`\b\d{4,5}\b`)
)

var streetSuffixes = []string{
	"chaussee", "promenade", "boulevard", "avenue",
	"straße", "strasse", "gässchen", "gäßchen", "gässle", "gäßle", "gässli", "gaessli",
	"str", "weg", "gasse", "allee", "ring", "damm", "pfad",
	"steig", "stieg", "stiege", "steige", "steg", "zeile", "winkel",
	"bogen", "ufer", "kai", "quai", "lände",
	"platz", "markt",
	"berg", "höhe", "halde", "hang", "grund", "graben", "tal", "thal",
	"bach", "aue", "heide", "feld", "ried", "riet", "moos", "holz",
	"forst", "schlag", "leiten", "bühel", "bühl", "büel", "egg", "horst", "weid",
	"acker", "anger", "wiese", "matte", "rain", "trift", "kamp", "breite", "hecke",
	"hof", "garten", "gärtle", "gärtli", "park", "mühle",
	"brücke", "bruck", "brück", "tor", "hafen", "burg", "turm", "warte", "mauer",
	"grün", "runde", "siedlung", "stall", "schanze", "staffel", "stutz",
	"lehen", "rotte",
}

var streetPrepositions = []string{
	"An der", "An dem", "An den", "An die",
	"Auf der", "Auf dem", "Auf den", "Auf die",
	"Bei der", "Bei dem", "Bei den",
	"In der", "In dem", "In den", "In die",
	"Unter der", "Unter dem", "Unter den", "Unter die",
	"Hinter dem", "Hinter der", "Hinter den", "Hinter die",
	"Neben dem", "Neben der",
	"Vor dem", "Vor der", "Vor den", "Vor die",
	"Zu den", "Ob der", "Ob dem",
	"Im", "Am", "Beim", "Zur", "Zum",
}

func init() {
	sort.Slice(streetSuffixes, func(i, j int) bool { return len(streetSuffixes[i]) > len(streetSuffixes[j]) })
	sort.Slice(streetPrepositions, func(i, j int) bool { return len(streetPrepositions[i]) > len(streetPrepositions[j]) })

	sParts := make([]string, len(streetSuffixes))
	for i, s := range streetSuffixes {
		sParts[i] = regexp.QuoteMeta(s)
	}

	pParts := make([]string, len(streetPrepositions))
	for i, p := range streetPrepositions {
		esc := regexp.QuoteMeta(p)
		esc = strings.ReplaceAll(esc, `\ `, `\s+`)
		pParts[i] = esc
	}

	pattern := fmt.Sprintf(
		`(?i)(?:(?:%s)\s+)?(?:[A-ZÄÖÜ][a-zäöüß]+(?:[-][A-ZÄÖÜ]?[a-zäöüß]+)*)[-\s]*(?:%s)\.?\s+(?:\d+\s*[a-zA-Z]?(?:\s*/\s*\d+)?),?\s+(?:\d{5}|\d{4})\s+(?:[A-ZÄÖÜ][a-zäöüß]+(?:(?:\s+|-)[A-ZÄÖÜ]?[a-zäöüß]+){0,2})`,
		strings.Join(pParts, "|"),
		strings.Join(sParts, "|"),
	)
	addressRE = regexp.MustCompile(pattern)
}

func detectAddress(text string) []Finding {
	if !plzPrefilterRE.MatchString(text) {
		return nil
	}
	var out []Finding
	for _, loc := range addressRE.FindAllStringIndex(text, -1) {
		out = append(out, Finding{
			Type: PiiAddress, Start: loc[0], End: loc[1],
			Text: text[loc[0]:loc[1]], Confidence: 0.9,
		})
	}
	return out
}
