package command

import (
	"strings"

	"github.com/dekarrin/tunaq/internal/tqerrors"
)

var (
	// ReservedWords is the list of all token sequences that a symbol
	// cannot have anywhere or it will cause issues in parsing.
	ReservedWords = []string{
		"TO",
		"THROUGH",
		"INTO",
		"FROM",
		"ON",
		"IN",
		"WITH",
		"AT",
	}
)

var (
	// VerbAliases maps shorthand verbs (which must be the first words in a
	// command) to their canonical forms. They are all uppercase.
	VerbAliases map[string]string = map[string]string{
		"NORTH":    "GO NORTH",
		"SOUTH":    "GO SOUTH",
		"EAST":     "GO EAST",
		"WEST":     "GO WEST",
		"UP":       "GO UP",
		"DOWN":     "GO DOWN",
		"MOVE":     "GO",
		"BYE":      "QUIT",
		"SPEAK":    "TALK",
		"COMBINE":  "USE",
		"PUT":      "DROP",
		"PUT DOWN": "DROP",
		"GET":      "TAKE",
		"PICK":     "TAKE",
		"PICK UP":  "TAKE",
		"DESCRIBE": "LOOK",
		"DESC":     "LOOK",
		"?":        "HELP",
		"/?":       "HELP",
		"/H":       "HELP",
		"-H":       "HELP",
		"H":        "HELP",
		"INVEN":    "INVENTORY",
		"I":        "INVENTORY",
	}
)

// FindFirstReserved takes the input, tokenizes it, and then checks whether it
// contains one of the reserved sequences. It can be used to check whether a
// symbol definition should be rejected by callers and marked as valid/invalid.
//
// Returns the first reserved word encountered. If none are encountered it will
// return the empty string.
func FindFirstReserved(s string) string {
	normalizedCase := strings.ToUpper(s)
	tokens := strings.Fields(normalizedCase)

	for i := range ReservedWords {
		for j := 0; j < len(tokens); j++ {
			if tokens[j] == ReservedWords[i] {
				return ReservedWords[i]
			}
		}
	}

	return ""
}

// ParseCommand parses a command from the given text. If it cannot, a non-nil
// error is returned.
//
// If an empty string or a string composed only of whitespace is passed in, nil
// error is returned and a zero value for Command will be returned.
func parseCommand(toParse string) (Command, error) {
	var parsedCmd Command

	// make entire input upper case to make matching easy
	normalizedCase := strings.ToUpper(toParse)

	// now tokenize our string, collapsing all whitespace
	originalTokens := strings.Fields(normalizedCase)

	// expand verb aliases up to 2 words long
	tokens := ExpandAliases(originalTokens, 2)

	// some simple sanity checking, make sure we at least have a command
	if len(tokens) < 1 {
		return parsedCmd, nil
	}

	// set verb as the first word here, we'll update it to synonyms as needed
	parsedCmd.Verb = tokens[0]

	// next, do simple matching on our main keywords based on the first word
	switch parsedCmd.Verb {
	case "HELP":
		// help takes an optional argument
		if len(tokens) > 1 {
			parsedCmd.Recipient = tokens[1]
		}
	case "EXITS":
		// ensure there are no additional args glub
		if len(tokens) > 1 {
			errMsg := "You can't %s *something*; type %s by itself to show exits"
			return parsedCmd, tqerrors.Interpreterf(errMsg, originalTokens[0], originalTokens[0])
		}
	case "GO":
		// make shore we ignore prepositions
		// TODO: ensure that world def parser never allows one of these to be the
		// start of a room alias.
		if len(tokens) > 1 && (tokens[1] == "TO" || tokens[1] == "THROUGH" || tokens[1] == "IN" || tokens[1] == "INTO") {
			tokens = append(tokens[0:1], tokens[2:]...)
		}

		// need the object; WHERE are we going?
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know where you want to go")
		}

		// otherwise, its the rest of the tokens
		parsedCmd.Recipient = strings.Join(tokens[1:], " ")
	case "TAKE":
		// need to know what we are taking
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know what you want to take")
		}

		// get from clause
		fromIdx := len(tokens)
		for i := range tokens[1:] {
			if tokens[i] == "FROM" {
				if i+1 >= len(tokens) {
					return parsedCmd, tqerrors.Interpreterf("I don't know where you want to take it from")
				}
				fromIdx = i
				parsedCmd.Instrument = strings.Join(tokens[i+1:], " ")
			}
		}

		// and the object is the rest of the tokens
		parsedCmd.Recipient = strings.Join(tokens[1:fromIdx], " ")
	case "DROP":
		// what are we dropping
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know what you want to drop")
		}

		onIdx := len(tokens)
		for i := range tokens[1:] {
			if tokens[i] == "ON" || tokens[i] == "IN" {
				if i+1 >= len(tokens) {
					return parsedCmd, tqerrors.Interpreterf("I don't know where you want to put it")
				}
				onIdx = i
				parsedCmd.Instrument = strings.Join(tokens[i+1:], " ")
			}
		}

		// and the object is the rest of the tokens
		parsedCmd.Recipient = strings.Join(tokens[1:onIdx], " ")
	case "USE":
		// what are we using
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know what you want to use")
		}

		withIdx := len(tokens)
		for i := range tokens[1:] {
			if tokens[i] == "WITH" {
				if i+1 >= len(tokens) {
					return parsedCmd, tqerrors.Interpreterf("I don't know where you want to use it with")
				}
				withIdx = i
				parsedCmd.Instrument = strings.Join(tokens[i+1:], " ")
			}
		}

		parsedCmd.Recipient = strings.Join(tokens[1:withIdx], " ")
	case "TALK":
		// talk p much always takes a 'to', make shore we ignore that
		if len(tokens) > 1 && (tokens[1] == "TO" || tokens[1] == "WITH") {
			tokens = append(tokens[0:1], tokens[2:]...)
		}

		// who are we talking to
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know what or who you want to talk to")
		}

		parsedCmd.Recipient = strings.Join(tokens[1:], " ")
	case "LOOK":
		// check for 'at' and remove it
		if len(tokens) > 1 && (tokens[1] == "AT" || tokens[1] == "IN") {
			tokens = append(tokens[0:1], tokens[2:]...)
		}

		// look has an optional recipient
		if len(tokens) > 1 {
			parsedCmd.Recipient = strings.Join(tokens[1:], " ")
		}
	case "DEBUG":
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("Debug what, exactly?")
		}

		if tokens[1] == "ROOM" {
			parsedCmd.Recipient = "ROOM"
			if len(tokens) > 2 {
				parsedCmd.Instrument = strings.Join(tokens[2:], " ")
			}
		} else if tokens[1] == "NPC" {
			parsedCmd.Recipient = "NPC"
			if len(tokens) > 2 {
				parsedCmd.Instrument = strings.Join(tokens[2:], " ")
			}
		} else if tokens[1] == "EXEC" {
			parsedCmd.Recipient = "EXEC"
			if len(tokens) < 3 {
				return parsedCmd, tqerrors.Interpreterf("I don't know what you want me to EXEC.")
			}
			casedTokens := strings.Fields(toParse)

			// we need to respect case for our arg
			parsedCmd.Instrument = strings.TrimSpace(strings.Join(casedTokens[2:], " "))
		} else if tokens[1] == "EXPAND" {
			parsedCmd.Recipient = "EXPAND"
			if len(tokens) < 3 {
				return parsedCmd, tqerrors.Interpreterf("I don't know what you want me to EXPAND.")
			}
			casedTokens := strings.Fields(toParse)

			// we need to respect case for our arg
			parsedCmd.Instrument = strings.TrimSpace(strings.Join(casedTokens[2:], " "))
		} else if tokens[1] == "FLAGS" {
			parsedCmd.Recipient = "FLAGS"
		} else {
			return parsedCmd, tqerrors.Interpreterf("%q is not a valid thing to be debugged", tokens[1])
		}
	case "INVENTORY":
		// ensure there are no additional args glub
		if len(tokens) > 1 {
			errMsg := "You can't %s *something*; type %s by itself to show inventory"
			return parsedCmd, tqerrors.Interpreterf(errMsg, originalTokens[0], originalTokens[0])
		}
	case "QUIT":
		// quit takes no additional args, make sure this is true
		if len(tokens) > 1 {
			errMsg := "You can't %s *something*; type %s by itself to quit"
			return parsedCmd, tqerrors.Interpreterf(errMsg, originalTokens[0], originalTokens[0])
		}
	default:
		return parsedCmd, tqerrors.Interpreterf("I don't know what you mean by %q", originalTokens[0])
	}

	return parsedCmd, nil
}

// HELP to show commands
// GO place
// TAKE thing
// DROP thing
// USE thing
// TALK to thing
// QUIT the game
// LOOK at the current scene or direction

// ExpandAliases takes a slice of tokens of user input and runs alias expansion
// on it. It expects all strings in the given slice to be upper case; failure to
// ensure this may cause the expansion to not work properly. The returned slice
// contains the same tokens but with aliases expanded.
//
// The unexpanded tokens slice is not modified during this operation.
//
// Aliases up to aliasLimit words long are supported. If it is less than 0, it
// is assumed to be 0. Passing 0 means the given tokens will be returned
// unchanged.
//
// Aliases will not be multi-expanded; that is, expansion is not applied to the
// results of an expansion; if the caller needs it, they will need to call
// ExpandAliases again on its output.
func ExpandAliases(tokens []string, aliasLimit int) []string {
	expandedTokens := append([]string{}, tokens...)
	if aliasLimit < 1 {
		return expandedTokens
	}

	// only modify verb up to minimum of limit and number of tokens
	if aliasLimit > len(tokens) {
		aliasLimit = len(tokens)
	}

	for curLimit := 1; curLimit <= aliasLimit; curLimit++ {
		checkStr := strings.Join(tokens[:curLimit], " ")
		expansion, ok := VerbAliases[checkStr]
		if ok {
			replacementTokens := strings.Fields(expansion)

			// luckily, we know we are operating from start of tokens passed in so we can just trash
			// all those in the checkStr and replace with the replacementTokens slice
			expandedTokens = append(replacementTokens, tokens[curLimit:]...)

			// we gaurantee only one single substitution, so we can immediately exit
			return expandedTokens
		}
	}

	return expandedTokens
}
