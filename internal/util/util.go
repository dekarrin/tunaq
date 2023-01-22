package util

import (
	"sort"
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
// the same as the string. If definite is true, the returned value will be "the"
// capitalized as described; otherwise, it will be "a"/"an" capitalized as
// described.
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

// OrderedKeys returns the keys of m, ordered a particular way. The order is
// guaranteed to be the same on every run.
//
// As of this writing, the order is alphabetical, but this function does not
// guarantee this will always be the case.
func OrderedKeys[V any](m map[string]V) []string {
	var keys []string
	var idx int

	keys = make([]string, len(m))
	idx = 0

	for k := range m {
		keys[idx] = k
		idx++
	}

	sort.Strings(keys)

	return keys
}

// EqualNilness returns whether the two values are either both nil or both
// non-nil.
func EqualNilness(o1 any, o2 any) bool {
	if o1 == nil {
		return o2 == nil
	} else {
		return o2 != nil
	}
}

// CustomComparable is an interface for items that may be checked against
// arbitrary other objects. In practice most will attempt to typecast to their
// own type and immediately return false if the argument is not the same, but in
// theory this allows for comparison to multiple types of things.
type CustomComparable interface {
	Equal(other any) bool
}

// EqualSlices checks that the two slices contain the same items in the same
// order. Equality of items is checked by items in the slices are equal by
// calling the custom Equal function on each element. In particular, Equal is
// called on elements of sl1 with elements of sl2 passed in as the argument.
func EqualSlices[T CustomComparable](sl1 []T, sl2 []T) bool {
	if len(sl1) != len(sl2) {
		return false
	}

	for i := range sl1 {
		if !sl1[i].Equal(sl2[i]) {
			return false
		}
	}

	return true
}
