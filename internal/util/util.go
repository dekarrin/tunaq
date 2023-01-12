package util

import (
	"strings"
	"unicode"
)

// MakeTextList gives a nice list of things based on their display name.
//
// TODO: turn this into a generic function that accepts displayable OR ~string
func MakeTextList(items []string, articles bool) string {
	if len(items) < 1 {
		return ""
	}

	output := ""

	withArts := make([]string, len(items))
	for i := range items {
		art := ""
		item := items[i]
		if articles {
			art = ArticleFor(item, false)

			iRunes := []rune(item)
			leadingUpper := unicode.IsUpper(iRunes[0])
			allCaps := leadingUpper
			if leadingUpper && len(iRunes) > 1 {
				allCaps = unicode.IsUpper(iRunes[1])
			}

			if leadingUpper && !allCaps {
				// make the item lower case
				iRunes[0] = unicode.ToLower(iRunes[0])
				item = string(iRunes)
			}

			art += " "
		}
		withArts[i] = art + " " + item
	}

	if len(withArts) == 1 {
		output += withArts[0]
	} else if len(withArts) == 2 {
		output += withArts[0] + " and " + withArts[1]
	} else {
		// if its more than two, use an oxford comma
		withArts[len(withArts)-1] = "and " + withArts[len(withArts)-1]
		output += strings.Join(withArts, ", ")
	}

	return output
}

// ArticleFor returns the article for the given string. It will be capitalized
// the same as the string.
func ArticleFor(s string, definite bool) string {
	sRunes := []rune(s)

	if len(sRunes) < 1 {
		return ""
	}

	leadingUpper := unicode.IsUpper(sRunes[0])
	allCaps := leadingUpper
	if leadingUpper && len(sRunes) > 1 {
		allCaps = unicode.IsUpper(sRunes[1])
	}

	art := ""
	if definite {
		if allCaps {
			art = "THE"
		} else if leadingUpper {
			art = "The"
		} else {
			art = "the"
		}
	} else {
		if allCaps || leadingUpper {
			art = "A"
		} else {
			art = "a"
		}

		sUpperRunes := []rune(strings.ToUpper(s))
		first := sUpperRunes[0]
		if first == 'A' || first == 'E' || first == 'I' || first == 'O' || first == 'U' {
			if allCaps {
				art += "N"
			} else {
				art += "n"
			}
		}
	}

	return art
}
