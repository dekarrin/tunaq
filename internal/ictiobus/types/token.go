package types

// Token is a lexeme read from text combined with the token class it is as well
// as additional supplementary information gathered during lexing to inform
// error reporting.
type Token interface {
	// Class returns the TokenClass of the Token.
	Class() TokenClass

	// Lexeme returns the text that was lexed as the TokenClass of the Token, as
	// it appears in the source text.
	Lexeme() string

	// LinePos returns the 1-indexed character-of-line that the token appears
	// on in the source text.
	LinePos() int

	// Line returns the 1-indexed line number of the line that the token appears
	// on in the source text.
	Line() int

	// FullLine returns the full of text of the line in source that the token
	// appears on, including both anything that came before the token as well as
	// after it on the line.
	FullLine() string

	// String is the string representation.
	String() string
}
