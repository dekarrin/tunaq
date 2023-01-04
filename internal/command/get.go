package command

import (
	"bufio"
	"fmt"

	"github.com/dekarrin/tunaq/internal/tqerrors"
)

// Reader is a type that can be used for getting command input.
// TODO: make this return parsed Commands, not 'lines'. Maybe. might not be
// feasible bc parse error handling would then need to be able to output from
// within the Reader.
type Reader interface {
	// ReadCommand reads a single user command. It will block until one is
	// ready. If there is an error or output is at end (EOF), the returned
	// string will be empty, otherwise it will always be non-empty.
	//
	// When error is io.EOF, string will always be empty. If EOF was encountered
	// on a call but some input was received, the input will be returned and
	// error will be nil, and the next call to ReadCommand will return "",
	// io.EOF.
	ReadCommand() (string, error)

	// Close performs any operations required to clean the resources created by
	// the Reader. It should be called at least once when the Reader is no
	// longer needed.
	Close() error
}

// Get obtains a single command from input by reading from the provided Reader.
// It reads a line of input and attempts to parse it as a valid command,
// returning that command if it is successful. If it is not, error output is
// printed to the ostream and the input is read until a valid command is
// encountered.
//
// Note that this function does not check if the command is executable, only
// that a Command can be parsed from the user input.
//
// TODO: abstract this and the entire command parsing structure to new package,
// cmd.
func Get(cmdStream Reader, ostream *bufio.Writer) (Command, error) {
	var cmd Command
	gotValidCommand := false

	if _, err := ostream.WriteString("Enter command\n"); err != nil {
		return cmd, fmt.Errorf("could not write output: %w", err)
	}
	if err := ostream.Flush(); err != nil {
		return cmd, fmt.Errorf("could not flush output: %w", err)
	}

	for !gotValidCommand {
		// IO to get input:
		input, err := cmdStream.ReadCommand()
		if err != nil {
			return cmd, fmt.Errorf("could not get input: %w", err)
		}

		// now attempt to parse the input
		cmd, err = ParseCommand(input)
		if err != nil {
			consoleMessage := tqerrors.GameMessage(err)
			errMsg := fmt.Sprintf("%v\nTry HELP for valid commands\n", consoleMessage)
			// IO to report error and prompt user to try again
			if _, err := ostream.WriteString(errMsg); err != nil {
				return cmd, fmt.Errorf("could not write output: %w", err)
			}
			if err := ostream.Flush(); err != nil {
				return cmd, fmt.Errorf("could not flush output: %w", err)
			}
		} else if cmd.Verb != "" {
			gotValidCommand = true
		}
	}

	return cmd, nil
}
