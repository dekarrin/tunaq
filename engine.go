// Package tunaq contains a CLI-driven engine for getting commands and advancing
// the game state continuously until the user quits.
package tunaq

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/dekarrin/tunaq/internal/game"
)

// Engine contains the things needed to run a game from an interactive shell
// attached to an input stream and an output stream.
type Engine struct {
	state   game.State
	in      *bufio.Reader
	out     *bufio.Writer
	running bool
}

// New creates a new engine ready to operate on the given input and output
// streams. It will immediately open a buffered reader on the input stream and a
// buffered writer on the output stream.
//
// If nil is given for the input stream, a bufio.Reader is opened on stdin. If
// nil is given for the output stream, a bufio.Writer is opened on stdout.
func New(inputStream io.Reader, outputStream io.Writer, worldFilePath string) (*Engine, error) {
	if inputStream == nil {
		inputStream = os.Stdin
	}
	if outputStream == nil {
		outputStream = os.Stdout
	}

	// load world file
	world, start, err := game.LoadWorldDefFile(worldFilePath)
	if err != nil {
		return nil, err
	}

	state, err := game.New(world, start)
	if err != nil {
		return nil, fmt.Errorf("initializing CLI engine: %w", err)
	}

	eng := &Engine{
		in:      bufio.NewReader(inputStream),
		out:     bufio.NewWriter(outputStream),
		state:   state,
		running: false,
	}

	return eng, nil
}

// RunUntilQuit begins reading commands from the streams and applying them to the game until the
// QUIT command is received.
func (eng *Engine) RunUntilQuit() error {
	introMsg := "Welcome to GoQuest\n"
	introMsg += "==================\n"
	introMsg += "\n"
	introMsg += "You are in " + eng.state.CurrentRoom.Name + "\n"

	if _, err := eng.out.WriteString(introMsg); err != nil {
		return fmt.Errorf("could not write output: %w", err)
	}
	if err := eng.out.Flush(); err != nil {
		return fmt.Errorf("could not flush output: %w", err)
	}

	eng.running = true
	// so we dont have to remember to do this on every returned error condition
	defer func() {
		eng.running = false
	}()

	for eng.running {
		cmd, err := game.GetCommand(eng.in, eng.out)
		if err != nil {
			return fmt.Errorf("get user command: %w", err)
		}

		// special check: actual game will not use the QUIT command, only a
		// runner can do that. so check if that's what we got
		if cmd.Verb == "QUIT" {
			eng.running = false
			break
		}

		err = eng.state.Advance(cmd, eng.out)
		if err != nil {
			if _, err := eng.out.WriteString(err.Error() + "\n"); err != nil {
				return fmt.Errorf("could not write output: %w", err)
			}
			if err := eng.out.Flush(); err != nil {
				return fmt.Errorf("could not flush output: %w", err)
			}
		}
	}

	if _, err := eng.out.WriteString("Goodbye\n"); err != nil {
		return fmt.Errorf("could not write output: %w", err)
	}
	if err := eng.out.Flush(); err != nil {
		return fmt.Errorf("could not flush output: %w", err)
	}

	return nil
}
