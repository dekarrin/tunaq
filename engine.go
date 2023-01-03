// Package tunaq contains a CLI-driven engine for getting commands and advancing
// the game state continuously until the user quits.
package tunaq

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/game"
	"github.com/dekarrin/tunaq/internal/input"
	"github.com/dekarrin/tunaq/internal/tqerrors"
)

// Engine contains the things needed to run a game from an interactive shell
// attached to an input stream and an output stream.
type Engine struct {
	state   game.State
	in      game.CommandReader
	out     *bufio.Writer
	running bool
}

const consoleOutputWidth = 80

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

	state, err := game.New(world, start, consoleOutputWidth)
	if err != nil {
		return nil, fmt.Errorf("initializing game engine: %w", err)
	}

	eng := &Engine{
		out:     bufio.NewWriter(outputStream),
		state:   state,
		running: false,
	}

	if inputStream == os.Stdin && outputStream == os.Stdout {
		eng.in, err = input.NewInteractiveReader()
		if err != nil {
			return nil, fmt.Errorf("initializing interactive-mode input reader: %w", err)
		}
	} else {
		eng.in = input.NewDirectReader(inputStream)
	}

	return eng, nil
}

// Close closes all resources associated with the Engine, including any
// readline-related resources created for interactive mode.
func (eng *Engine) Close() error {
	// TODO: make it so Close called on running engine actually shuts it down.
	// requirements: need to tell CommandReader that it is time to stop reading
	// immediately and go EOF.
	if eng.running {
		return fmt.Errorf("cannot close a running game engine")
	}

	err := eng.in.Close()
	if err != nil {
		return fmt.Errorf("close command reader: %w", err)
	}

	return nil
}

// RunUntilQuit begins reading commands from the streams and applying them to
// the game until the QUIT command is received.
func (eng *Engine) RunUntilQuit() error {
	introMsg := "Welcome to TunaQuest Engine\n"
	introMsg += "===========================\n"
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
			consoleMessage := tqerrors.GameMessage(err)
			consoleMessage = rosed.Edit(consoleMessage).Wrap(consoleOutputWidth).String()
			if _, err := eng.out.WriteString(consoleMessage + "\n"); err != nil {
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
