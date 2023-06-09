// Package syntax creates abstract TunaScript and TunaQuestExpansion language
// constructs from parse trees passed to it from generated frontends.
package syntax

import "strings"

func spaceIndentNewlines(str string, amount int) string {
	if strings.Contains(str, "\n") {
		// need to pad every newline
		pad := " "
		for len(pad) < amount {
			pad += " "
		}
		str = strings.ReplaceAll(str, "\n", "\n"+pad)
	}
	return str
}
