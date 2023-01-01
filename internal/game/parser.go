package game

import (
	"strings"

	"github.com/dekarrin/tunaq/internal/tqerrors"
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

// Command is a valid command received from a game prompt.
type Command struct {

	// Verb is the canonical name of the command being invoked, such as "MOVE",
	// "GET", "USE", or "QUIT". Some verbs may have shorthand forms which are
	// typed differently, for instance "GO" could be typed instead of "MOVE", or
	// "NORTH" instead of "MOVE NORTH", etc, and for all those cases they would
	// result in a Command with a verb of GO.
	Verb string

	// Instrument is what is doing the action, for instance in "GET WATER WITH
	// CUP", or alternatively "USE WATER WITH CUP", "CUP" would be identified as
	// the instrument. The exact meaning depends on the verb.
	Instrument string

	// Recipient is the thing receiving the action, for instance "GET CUP" or
	// "TALK TO MAN", the recipient would be "CUP" and "MAN" respectively. For
	// MOVE commands, this can also be a direction.
	Recipient string
}

// ParseCommand parses a command from the given text. If it cannot, a non-nil
// error is returned.
//
// If an empty string or a string composed only of whitespace is passed in, nil
// error is returned and a zero value for Command will be returned.
func ParseCommand(toParse string) (Command, error) {
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
		if len(tokens) > 1 && (tokens[1] == "TO" || tokens[1] == "THROUGH" || tokens[1] == "IN" || tokens[1] == "INTO") {
			tokens = append(tokens[0:1], tokens[2:]...)
		}

		// need the object; WHERE are we going?
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know where you want to go")
		}

		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know where you want to go")
		}

		parsedCmd.Recipient = tokens[1]
	case "TAKE":
		// need to know what we are taking
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know what you want to take")
		}
		parsedCmd.Recipient = tokens[1]
	case "DROP":
		// what are we dropping
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know what you want to drop")
		}
		parsedCmd.Recipient = tokens[1]
	case "USE":
		// what are we using
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know what you want to use")
		}
		parsedCmd.Recipient = tokens[1]
	case "TALK":
		// talk p much always takes a 'to', make shore we ignore that
		if len(tokens) > 1 && tokens[1] == "TO" {
			tokens = append(tokens[0:1], tokens[2:]...)
		}

		// who are we talking to
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("I don't know what or who you want to talk to")
		}
		parsedCmd.Recipient = tokens[1]
	case "LOOK":
		// check for 'at' and remove it
		if len(tokens) > 1 && tokens[1] == "AT" {
			tokens = append(tokens[0:1], tokens[2:]...)
		}

		// look has an optional recipient
		if len(tokens) > 1 {
			parsedCmd.Recipient = tokens[1]
		}
	case "DEBUG":
		if len(tokens) < 2 {
			return parsedCmd, tqerrors.Interpreterf("Debug what, exactly?")
		}

		if tokens[1] == "ROOM" {
			parsedCmd.Recipient = "ROOM"
		} else if tokens[1] == "NPC" {
			parsedCmd.Recipient = "NPC"
			if len(tokens) > 2 {
				parsedCmd.Instrument = tokens[2]
			}
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
