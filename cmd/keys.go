package cmd

import "strings"

func normalizeKeyInput(key string) string {
	return strings.ToUpper(strings.TrimSpace(key))
}
