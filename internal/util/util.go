package util

import "strings"

// MakeTextList gives a nice list of things based on their display name.
//
// TODO: turn this into a generic function that accepts displayable OR ~string
func MakeTextList(items []string) string {
	if len(items) < 1 {
		return ""
	}

	output := ""

	if len(items) == 1 {
		output += items[0]
	} else if len(items) == 2 {
		output += items[0] + " and " + items[1]
	} else {
		// if its more than two, use an oxford comma
		items[len(items)-1] = "and " + items[len(items)-1]
		output += strings.Join(items, ", ")
	}

	return output
}
