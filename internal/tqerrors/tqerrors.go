package tqerrors

import "fmt"

// InterpreterError is an error caused by attempting to interpret input. Either
// the input could not be understood or it specifies doing something that is
// impossible or not allowed at the current time.
//
// InterpreterError includes a human-readable message to show to an operator as
// well as a typical more technical "error message" style message.
type interpreterError struct {
	msg   string
	human string
	wrap  error
}

func (e *interpreterError) Error() string {
	return e.msg
}

// GameMessage shows the message that should be displayed in-game to describe
// the error.
func (e *interpreterError) GameMessage() string {
	return e.human
}

// Unwrap gives the error that the InterpreterError wraps, if it wraps one.
func (e *interpreterError) Unwrap() error {
	return e.wrap
}

// Interpreter returns a new InterpreterError that has both the message to show
// the player and the technical description of the error.
func Interpreter(game, technical string) error {
	if technical == "" {
		technical = fmt.Sprintf("got InterpreterError(%q)", game)
	}
	return &interpreterError{
		msg:   technical,
		human: game,
	}
}

// Interpreterf returns a new InterpreterError that has a message to show to
// the player and an automatically generated Error() description. The arguments
// given are the format string and the arguments to the format string.
func Interpreterf(gameFormat string, a ...interface{}) error {
	gameMessage := fmt.Sprintf(gameFormat, a...)
	return Interpreter(gameMessage, "")
}

// WrapInterpreter returns a new InterpreterError that has both the message to
// show the player and the technical description of the error, and that wraps
// the given error.
func WrapInterpreter(e error, game, technical string) error {
	if technical == "" {
		technical = fmt.Sprintf("got InterpreterError(%q)", game)
	}
	return &interpreterError{
		msg:   technical,
		human: game,
		wrap:  e,
	}
}

// WrapInterpreterf returns a new InterpreterError that has both the message to
// show the player and an automatically generated Error() description, and that
// wraps the given error. The arguments given are the error to wrap, then the
// format followed by its arguments.
func WrapInterpreterf(e error, gameFormat string, a ...interface{}) error {
	gameMessage := fmt.Sprintf(gameFormat, a...)
	return WrapInterpreter(e, gameMessage, "")
}

// GameMessage gets the message to display to the console for the given error.
// If it is one of the types defined in tqerrors, the special game message is
// returned (if it exists). Otherwise, err.Error() is returned.
func GameMessage(err error) string {
	if intErr, ok := err.(*interpreterError); ok {
		return intErr.GameMessage()
	}
	return err.Error()
}
