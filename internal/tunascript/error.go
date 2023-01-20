package tunascript

import "fmt"

// file error.go contains errors generated from tunascript parsing and
// interpretation.

type SyntaxError struct {
	sourceLine string
	source     string

	// line that error occured on, 1-indexed.
	line int

	// position in line of error, 1-indexed.
	pos     int
	message string
}

func (se SyntaxError) Error() string {
	if se.line == 0 {
		return fmt.Sprintf("syntax error: %s", se.message)
	}

	return fmt.Sprintf("syntax error: around line %d, char %d: %s", se.line, se.pos, se.message)
}

// Source returns the exact text of the specific source code that caused the
// issue. If no particular source was the cause (such as for unexpected EOF
// errors), this will return an empty string.
func (se SyntaxError) Source() string {
	return se.source
}

// Line returns the line the error occured on. Lines are 1-indexed. This will
// return 0 if the line is not set.
func (se SyntaxError) Line() int {
	return se.line
}

// Position returns the character position that the error occured on. Character
// positions are 1-indexed. This will return 0 if the character position is not
// set.
func (se SyntaxError) Position() int {
	return se.pos
}

// FullMessage shows the complete message of the error string along with the
// offending line and a cursor to the problem position in a formatted way.
func (se SyntaxError) FullMessage() string {
	errMsg := se.Error()

	if se.line != 0 {
		errMsg = se.SourceLineWithCursor() + "\n" + errMsg
	}

	return errMsg
}

// SourceLineWithCursor returns the source offending code on one line and
// directly under it a cursor showing where the error occured.
//
// Returns a blank string if no source line was provided for the error (such as
// for unexpected EOF errors).
func (se SyntaxError) SourceLineWithCursor() string {
	if se.sourceLine == "" {
		return ""
	}

	cursorLine := ""
	// pos will be 1-indexed.
	for i := 0; i < se.pos-1; i++ {
		cursorLine += " "
	}

	return se.sourceLine + "\n" + cursorLine
}

func syntaxErrorFromLexeme(msg string, lexeme opTokenizedLexeme) SyntaxError {
	return SyntaxError{
		message:    msg,
		sourceLine: lexeme.fullLine,
		source:     lexeme.value,
		pos:        lexeme.pos,
		line:       lexeme.line,
	}
}
