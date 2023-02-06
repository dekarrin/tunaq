// Package buffalo contains parsers and parser-generator constructs used as part
// of research into compiling techniques. It is the tunascript compilers pulled
// out after it turned from "small knowledge gaining side-side project" into
// full-blown compilers and translators research.
//
// It's based off of the buffalo fish and also bison because of course.
//
// This will probably never be as good as bison, so consider using that. This is
// for research and does not seek to replace existing toolchains in any
// practical fashion.
package buffalo

type SyntaxError interface {
	error

	// Source returns the exact text of the specific source code that caused the
	// issue. If no particular source was the cause (such as for unexpected EOF
	// errors), this will return an empty string.
	Source() string

	// Line returns the line the error occured on. Lines are 1-indexed. This will
	// return 0 if the line is not set.
	Line() int

	// Position returns the character position that the error occured on. Character
	// positions are 1-indexed. This will return 0 if the character position is not
	// set.
	Position() int

	// FullMessage shows the complete message of the error string along with the
	// offending line and a cursor to the problem position in a formatted way.
	FullMessage() string

	// SourceLineWithCursor returns the source offending code on one line and
	// directly under it a cursor showing where the error occured.
	//
	// Returns a blank string if no source line was provided for the error (such as
	// for unexpected EOF errors).
	SourceLineWithCursor() string
}
