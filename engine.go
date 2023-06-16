// Package tunaq contains a CLI-driven engine for getting commands and advancing
// the game state continuously until the user quits.
package tunaq

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/dekarrin/rosed"
	"github.com/dekarrin/tunaq/internal/command"
	"github.com/dekarrin/tunaq/internal/game"
	"github.com/dekarrin/tunaq/internal/input"
	"github.com/dekarrin/tunaq/internal/tqerrors"
	"github.com/dekarrin/tunaq/internal/tqw"
)

// Engine contains the things needed to run a game from an interactive shell
// attached to an input stream and an output stream.
type Engine struct {
	state   *game.State
	term    *terminalDevice
	running bool
}

const consoleOutputWidth = 80

// New creates a new engine ready to operate on the given input and output
// streams. It will immediately open a buffered reader on the input stream and a
// buffered writer on the output stream.
//
// If nil is given for the input stream, a bufio.Reader is opened on stdin. If
// nil is given for the output stream, a bufio.Writer is opened on stdout.
func New(inputStream io.Reader, outputStream io.Writer, worldFilePath string, forceDirectInput bool) (*Engine, error) {
	if inputStream == nil {
		inputStream = os.Stdin
	}
	if outputStream == nil {
		outputStream = os.Stdout
	}

	// load world file
	worldData, err := tqw.LoadResourceBundle(worldFilePath)
	if err != nil {
		return nil, err
	}

	// terminal for IO output.
	term := &terminalDevice{
		width:       consoleOutputWidth,
		forceDirect: forceDirectInput,
		out:         bufio.NewWriter(outputStream),
	}

	term.useReadline = !forceDirectInput && inputStream == os.Stdin && outputStream == os.Stdout

	if term.useReadline {
		term.in, err = input.NewInteractiveReader()
		if err != nil {
			return nil, fmt.Errorf("initializing interactive-mode input reader: %w", err)
		}
	} else {
		term.in = input.NewDirectReader(inputStream)
	}

	// create engine
	eng := &Engine{
		term:    term,
		running: false,
	}

	state, err := game.New(worldData.Rooms, worldData.Start, worldData.Flags, eng.term)
	if err != nil {
		return nil, fmt.Errorf("initializing game engine: %w", err)
	}
	eng.state = state

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

	err := eng.term.in.Close()
	if err != nil {
		return fmt.Errorf("close command reader: %w", err)
	}

	return nil
}

// RunUntilQuit begins reading commands from the streams and applying them to
// the game until the QUIT command is received.
//
// startCommands, if non nil, is commands to run as soon as it starts.
func (eng *Engine) RunUntilQuit(startCommands []string) error {
	introMsg := "Welcome to TunaQuest Engine\n"
	if eng.term.forceDirect {
		introMsg += "(direct input mode)\n"
	}
	introMsg += "===========================\n"
	introMsg += "\n"
	introMsg += "You are in " + eng.state.CurrentRoom.Name + "\n"

	if _, err := eng.term.out.WriteString(introMsg); err != nil {
		return fmt.Errorf("could not write output: %w", err)
	}
	if err := eng.term.out.Flush(); err != nil {
		return fmt.Errorf("could not flush output: %w", err)
	}

	eng.running = true
	// so we dont have to remember to do this on every returned error condition
	defer func() {
		eng.running = false
	}()

	startCmdIdx := 0

	for eng.running {
		var cmd command.Command
		var err error

		if startCmdIdx+1 <= len(startCommands) {
			cmd, err = command.Parse(startCommands[startCmdIdx])
			if err != nil {
				consoleMessage := tqerrors.GameMessage(err)
				consoleMessage = rosed.Edit(consoleMessage).Wrap(consoleOutputWidth).String()
				if _, err := eng.term.out.WriteString("\n" + consoleMessage + "\n\n"); err != nil {
					return fmt.Errorf("could not write output: %w", err)
				}
				if err := eng.term.out.Flush(); err != nil {
					return fmt.Errorf("could not flush output: %w", err)
				}
			}
			startCmdIdx++
		} else {
			cmd, err = command.Get(eng.term.in, eng.term.out)
			if err != nil {
				return fmt.Errorf("get user command: %w", err)
			}
		}

		// special check: actual game will not use the QUIT command, only a
		// runner can do that. so check if that's what we got
		if cmd.Verb == "QUIT" {
			eng.running = false
			break
		}

		err = eng.state.Advance(cmd)
		if err != nil {
			consoleMessage := tqerrors.GameMessage(err)
			consoleMessage = rosed.Edit(consoleMessage).Wrap(consoleOutputWidth).String()
			if _, err := eng.term.out.WriteString("\n" + consoleMessage + "\n\n"); err != nil {
				return fmt.Errorf("could not write output: %w", err)
			}
			if err := eng.term.out.Flush(); err != nil {
				return fmt.Errorf("could not flush output: %w", err)
			}
		}
	}

	if _, err := eng.term.out.WriteString("\nGoodbye\n"); err != nil {
		return fmt.Errorf("could not write output: %w", err)
	}
	if err := eng.term.out.Flush(); err != nil {
		return fmt.Errorf("could not flush output: %w", err)
	}

	return nil
}

type terminalDevice struct {
	width       int
	out         *bufio.Writer
	in          command.Reader
	forceDirect bool
	useReadline bool
}

func (td *terminalDevice) Width() int {
	return td.width
}

func (td *terminalDevice) SetWidth(w int) {
	td.width = w
}

func (td *terminalDevice) Output(s string, a ...interface{}) error {
	s = fmt.Sprintf(s, a...)
	if _, err := td.out.WriteString(s); err != nil {
		return fmt.Errorf("could not write output: %w", err)
	}
	if err := td.out.Flush(); err != nil {
		return fmt.Errorf("could not flush output: %w", err)
	}
	return nil
}

func (td *terminalDevice) Input(prompt string) (string, error) {
	var oldPrompt string
	var icr *input.InteractiveCommandReader
	if td.useReadline {
		icr = td.in.(*input.InteractiveCommandReader)
		oldPrompt = icr.GetPrompt()
		icr.SetPrompt(prompt)
	} else {
		if prompt != "" {
			if err := td.Output(prompt); err != nil {
				return "", err
			}
		}
	}
	td.in.AllowBlank(true)
	readInput, err := td.in.ReadCommand()
	td.in.AllowBlank(false)
	if td.useReadline {
		icr = td.in.(*input.InteractiveCommandReader)
		icr.SetPrompt(oldPrompt)
	}
	return readInput, err
}

func (td *terminalDevice) InputInt(prompt string) (int, error) {
	var intVal int
	var valSet bool

	for !valSet {
		inputVal, err := td.Input(prompt)
		if err != nil {
			return 0, err
		}
		intVal, err = strconv.Atoi(inputVal)
		if err != nil {
			msg := "Please enter a number\n"
			if strings.Contains(inputVal, ".") {
				msg = "Please enter a number without a decimal dot\n"
			}
			err := td.Output(msg)
			if err != nil {
				return 0, err
			}
		} else {
			valSet = true
		}
	}
	return intVal, nil
}
