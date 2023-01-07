// Package input contains identifiers used in getting TunaQuest command input
// from CLI or other sources of input.
package input

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/chzyer/readline"
)

// DirectCommandReader implements command.Reader and reads commands from any
// generic input stream directly. It can be used generically with any io.Reader
// but does not sanitize the input of control and escape sequences.
//
// DirectCommandReader should not be used directly; instead, create one with
// [NewDirectReader].
type DirectCommandReader struct {
	r             *bufio.Reader
	blanksAllowed bool
}

// InteractiveCommandReader implements command.Reader and reads commands from
// stdin using a go implementation of the GNU Readline library. This keeps input
// clear of all typing and editing escape sequences and enables the use of
// command history. This should in general probably only be used when directly
// connecting to a TTY for input.
//
// InteractiveCommandReader should not be used directly; instead, create one
// with [NewInteractiveReader].
type InteractiveCommandReader struct {
	rl            *readline.Instance
	blanksAllowed bool
	prompt        string
}

// Create a new DirectCommandReader and initialize a buffered reader on the
// provided reader. The returned CommandReader must have Close() called on it
// before disposal to properly teardown readline resources.
func NewDirectReader(r io.Reader) *DirectCommandReader {
	return &DirectCommandReader{
		r: bufio.NewReader(r),
	}
}

// Create a new InteractiveCommandReader and initialize readline. The returned
// InteractiveCommandReader must have Close() called on it before disposal to
// properly teardown readline resources.
func NewInteractiveReader() (*InteractiveCommandReader, error) {
	rl, err := readline.NewEx(&readline.Config{
		Prompt: "> ",
	})
	if err != nil {
		return nil, fmt.Errorf("create readline config: %w", err)
	}

	return &InteractiveCommandReader{
		rl:     rl,
		prompt: "> ",
	}, nil
}

// Close cleans up resources associated with the DirectCommandReader.
func (dcr *DirectCommandReader) Close() error {
	// this function is here so DirectCommandReader implements
	// game.CommandReader. For now it doesn't really do anything as the
	// DirectCommandReader does not create resources but it may in the future
	// and callers should treat it as though it must have Close called on it.

	return nil
}

// Close cleans up readline resources and other resources associated with the
// InteractiveCommandReader.
func (icr *InteractiveCommandReader) Close() error {
	return icr.rl.Close()
}

// ReadCommand reads the next line from stdin. The returned string will only
// be empty if there is an error reading input, otherwise this function is
// blocked on until a line containing non-space characters is read.
//
// If at end of input, the returned string will be empty and error will be
// io.EOF. If any other error occurs, the returned string will be empty and
// error will be that error.
func (dcr *DirectCommandReader) ReadCommand() (string, error) {
	var line string
	var err error

	for line == "" {
		line, err = dcr.r.ReadString('\n')
		if err != nil && (err != io.EOF || line == "") {
			return "", err
		}

		line = strings.TrimSpace(line)

		if line == "" && dcr.blanksAllowed {
			return line, nil
		}
	}

	return line, nil
}

// ReadCommand reads the next command from stdin. The returned string will only
// be empty if there is an error, otherwise this function is blocked on until a
// line consisting of more than empty or whitespace-only input is read.
//
// If at end of input, the returned string will be empty and error will be
// io.EOF. If any other error occurs, the returned string will be empty and
// error will be that error.
func (icr *InteractiveCommandReader) ReadCommand() (string, error) {
	var line string
	var err error

	for line == "" {
		line, err = icr.rl.Readline()
		if err != nil && (err != io.EOF || line == "") {
			return "", err
		}

		line = strings.TrimSpace(line)

		if line == "" && icr.blanksAllowed {
			return line, nil
		}
	}

	return line, nil
}

// AllowBlank sets whether blank output is allowed. By default it is not.
func (dcr *DirectCommandReader) AllowBlank(allow bool) {
	dcr.blanksAllowed = allow
}

// AllowBlank sets whether blank output is allowed. By default it is not.
func (icr *InteractiveCommandReader) AllowBlank(allow bool) {
	icr.blanksAllowed = allow
}

// SetPrompt updates the prompt to the given text.
func (icr *InteractiveCommandReader) SetPrompt(p string) {
	icr.rl.SetPrompt(p)
}

// GetPrompt gets the current prompt.
func (icr *InteractiveCommandReader) GetPrompt() string {
	return icr.prompt
}
