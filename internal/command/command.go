// Package command defines game command data types and handles parsing of
// commands from input sources.
package command

// Command is a valid command received from a game input source.
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
