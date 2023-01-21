package tunascript

import (
	"strings"
	"unicode"
)

const (
	// if newline is put in any of these it will break the lexer detection of
	// position so don't do that

	literalStrStringQuote     = "@"
	literalStrSeparator       = ","
	literalStrGroupOpen       = "("
	literalStrGroupClose      = ")"
	literalStrIdentifierStart = "$"
	literalStrOpIs            = "=="
	literalStrOpIsNot         = "!="
	literalStrOpLessThan      = "<"
	literalStrOpGreaterThan   = ">"
	literalStrOpLessThanIs    = "<="
	literalStrOpGreaterThanIs = ">="
	literalStrOpSet           = "="
	literalStrOpPlus          = "+"
	literalStrOpMinus         = "-"
	literalStrOpMultiply      = "*"
	literalStrOpDivide        = "/"
	literalStrOpIncset        = "+="
	literalStrOpDecset        = "-="
	literalStrOpInc           = "++"
	literalStrOpDec           = "--"
	literalStrOpAnd           = "&&"
	literalStrOpOr            = "||"
	literalStrOpNot           = "!"
)

type token struct {
	lexeme   string
	class    tokenClass
	pos      int
	line     int
	fullLine string
}

type tokenStream struct {
	tokens []token
	cur    int
}

// a type of token
type tokenClass struct {
	id    string
	human string
	lbp   int
}

// TODO: Do The Unmarshal Function Thing With The Operator Data Objects. Or
// Structs. Or Something Like That.

func (class tokenClass) String() string {
	return class.id
}

// Human returns human readable name of the string.
func (class tokenClass) Human() string {
	return class.human
}

func (ts *tokenStream) Next() token {
	n := ts.tokens[ts.cur]
	ts.cur++
	return n
}

func (ts *tokenStream) Peek() token {
	return ts.tokens[ts.cur]
}

func (ts tokenStream) Len() int {
	return len(ts.tokens)
}

func (ts tokenStream) Remaining() int {
	return len(ts.tokens) - ts.cur
}

var (
	literalStringQuote     = []rune(literalStrStringQuote)
	literalSeparator       = []rune(literalStrSeparator)
	literalGroupOpen       = []rune(literalStrGroupOpen)
	literalGroupClose      = []rune(literalStrGroupClose)
	literalIdentifierStart = []rune(literalStrIdentifierStart)
	literalOpIs            = []rune(literalStrOpIs)
	literalOpIsNot         = []rune(literalStrOpIsNot)
	literalOpLessThan      = []rune(literalStrOpLessThan)
	literalOpGreaterThan   = []rune(literalStrOpGreaterThan)
	literalOpLessThanIs    = []rune(literalStrOpLessThanIs)
	literalOpGreaterThanIs = []rune(literalStrOpGreaterThanIs)
	literalOpSet           = []rune(literalStrOpSet)
	literalOpPlus          = []rune(literalStrOpPlus)
	literalOpMinus         = []rune(literalStrOpMinus)
	literalOpMultiply      = []rune(literalStrOpMultiply)
	literalOpDivide        = []rune(literalStrOpDivide)
	literalOpIncset        = []rune(literalStrOpIncset)
	literalOpDecset        = []rune(literalStrOpDecset)
	literalOpInc           = []rune(literalStrOpInc)
	literalOpDec           = []rune(literalStrOpDec)
	literalOpAnd           = []rune(literalStrOpAnd)
	literalOpOr            = []rune(literalStrOpOr)
	literalOpNot           = []rune(literalStrOpNot)
)

var (
	tsSeparator       = tokenClass{"TS_SEPARATOR", "'" + literalStrSeparator + "'", 0}
	tsGroupOpen       = tokenClass{"TS_GROUP_OPEN", "'" + literalStrGroupOpen + "'", 100}
	tsGroupClose      = tokenClass{"TS_GROUP_CLOSE", "'" + literalStrGroupClose + "'", 0}
	tsIdentifier      = tokenClass{"TS_IDENTIFIER", "identifier", 0}
	tsEndOfText       = tokenClass{"TS_END_OF_TEXT", "end of text", 0}
	tsUndefined       = tokenClass{"TS_UNDEFINED", "undefined", 0}
	tsNumber          = tokenClass{"TS_NUMBER", "number", 0}
	tsBool            = tokenClass{"TS_BOOL", "boolean value", 0}
	tsUnquotedString  = tokenClass{"TS_UNQUOTED_STRING", "text value", 0}
	tsQuotedString    = tokenClass{"TS_QUOTED_STRING", literalStrStringQuote + "-text value", 0}
	tsOpIs            = tokenClass{"TS_OP_IS", "'" + literalStrOpIs + "'", 5}
	tsOpIsNot         = tokenClass{"TS_OP_IS_NOT", "'" + literalStrOpIsNot + "'", 5}
	tsOpLessThan      = tokenClass{"TS_OP_LESS_THAN", "'" + literalStrOpLessThan + "'", 5}
	tsOpGreaterThan   = tokenClass{"TS_OP_GREATER_THAN", "'" + literalStrOpGreaterThan + "'", 5}
	tsOpLessThanIs    = tokenClass{"TS_OP_LESS_THEN_IS", "'" + literalStrOpLessThanIs + "'", 5}
	tsOpGreaterThanIs = tokenClass{"TS_OP_GREATER_THAN_IS", "'" + literalStrOpGreaterThanIs + "'", 5}
	tsOpSet           = tokenClass{"TS_OP_SET", "'" + literalStrOpSet + "'", 0}
	tsOpPlus          = tokenClass{"TS_OP_PLUS", "'" + literalStrOpPlus + "'", 10}
	tsOpMinus         = tokenClass{"TS_OP_MINUS", "'" + literalStrOpMinus + "'", 10}
	tsOpMultiply      = tokenClass{"TS_OP_MULTIPLY", "'" + literalStrOpMultiply + "'", 20}
	tsOpDivide        = tokenClass{"TS_OP_DIVIDE", "'" + literalStrOpDivide + "'", 20}
	tsOpIncset        = tokenClass{"TS_OP_INCSET", "'" + literalStrOpIncset + "'", 90}
	tsOpDecset        = tokenClass{"TS_OP_DECSET", "'" + literalStrOpDecset + "'", 90}
	tsOpInc           = tokenClass{"TS_OP_INC", "'" + literalStrOpInc + "'", 150}
	tsOpDec           = tokenClass{"TS_OP_DEC", "'" + literalStrOpDec + "'", 150}
	tsOpAnd           = tokenClass{"TS_OP_AND", "'" + literalStrOpAnd + "'", 0}
	tsOpOr            = tokenClass{"TS_OP_OR", "'" + literalStrOpOr + "'", 0}
	tsOpNot           = tokenClass{"TS_OP_NOT", "'" + literalStrOpNot + "'", 0}
)

func LexOperationText(s string) (tokenStream, error) {
	sRunes := []rune(s)
	tokens, _, err := lexRunes(sRunes, false)
	return tokens, err
}

// returns the created tokenstream, num runes consumed, and error.
func lexRunes(sRunes []rune, endAtMatchingParen bool) (tokenStream, int, error) {
	var tokens []token

	curLine := 1
	curLinePos := 1

	var curToken token
	var sb strings.Builder

	var escaping bool

	type lexMode int

	const (
		lexDefault lexMode = iota
		lexIdent
		lexString
	)

	mode := lexDefault

	// track our paren-depth in case endAtMatching is set
	parenDepth := 0
	runesConsumed := len(sRunes)

	var currentfullLine = readFullLine(sRunes)
	flushCurrentPendingToken := func() {
		if sb.Len() > 0 {
			curToken.lexeme = sb.String()
			curToken.fullLine = currentfullLine
			sb.Reset()

			// is the cur token literally one of the bool values?
			vUp := strings.ToUpper(curToken.lexeme)
			if patBool.MatchString(vUp) {
				curToken.class = tsBool
			}
			if patNum.MatchString(vUp) {
				curToken.class = tsNumber
			}

			tokens = append(tokens, curToken)
			curToken = token{}
		}
	}

	for i := 0; i < len(sRunes); i++ {
		ch := sRunes[i]

		// if it's a newline for any reason, get the next line for the current
		// one
		if ch == '\n' {
			//
			currentfullLine = readFullLine(sRunes[i+1:])
		}

		switch mode {
		case lexIdent:
			if ('A' <= ch && ch <= 'Z') || ('a' <= ch && ch <= 'z') || ('0' <= ch && ch <= '9') || ch == '_' {
				sb.WriteRune(ch)
			} else {
				curToken.lexeme = sb.String()
				sb.Reset()
				curToken.fullLine = currentfullLine
				tokens = append(tokens, curToken)
				curToken = token{}
				mode = lexDefault
				i-- // re-lex in normal mode
			}
		case lexString:
			if !escaping && startMatches(sRunes[i:], literalStringQuote) {
				writeRuneSlice(sb, literalStringQuote)
				flushCurrentPendingToken()
				mode = lexDefault
				sb.Reset()
				i += len(literalStringQuote) - 1
			} else if !escaping && ch == '\\' {
				// preserve ALL escape sequences not directly linked to
				// operators as further parse passes may need to interpret
				// them
				escaping = true
				sb.WriteRune('\\')
			} else {
				escaping = false
				sb.WriteRune(ch)
			}
		case lexDefault:
			if !escaping && startMatches(sRunes[i:], literalStringQuote) {
				flushCurrentPendingToken()

				// we are entering a string, set type and current position
				// (value set on a deferred basis once string is complete)
				curToken.pos = curLinePos
				curToken.line = curLine
				curToken.class = tsQuotedString
				mode = lexString
				writeRuneSlice(sb, literalStringQuote)
				i += len(literalStringQuote) - 1
			} else if !escaping && startMatches(sRunes[i:], literalIdentifierStart) {
				flushCurrentPendingToken()

				// we are entering an identifier, set type and current position
				// (value set on a deferred basis once identifier is complete)
				curToken.pos = curLinePos
				curToken.line = curLine
				curToken.class = tsIdentifier
				mode = lexIdent
				writeRuneSlice(sb, literalIdentifierStart)
				i += len(literalIdentifierStart) - 1
			} else if !escaping && ch == '\\' {
				escaping = true
			} else if startMatches(sRunes[i:], literalSeparator) {
				if escaping {
					writeRuneSlice(sb, literalSeparator)
					escaping = false
				} else {
					flushCurrentPendingToken()
					curToken = token{pos: curLinePos, line: curLine, class: tsSeparator, lexeme: literalStrSeparator}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = token{}
				}
				i += len(literalSeparator) - 1
			} else if ch == '+' {
				if escaping {
					sb.WriteRune('+')
					escaping = false
				} else {
					flushCurrentPendingToken()
					if i+1 < len(sRunes) && sRunes[i+1] == '+' {
						// it is double-plus
						curToken = token{pos: curLinePos, line: curLine, class: tsOpInc, lexeme: "++"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
						i++
					} else if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is inc-by
						curToken = token{pos: curLinePos, line: curLine, class: tsOpIncset, lexeme: "-="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
						i++
					} else {
						// it is a plus
						curToken = token{pos: curLinePos, line: curLine, class: tsOpPlus, lexeme: "+"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
					}
				}
			} else if ch == '-' {
				if escaping {
					sb.WriteRune('-')
					escaping = false
				} else {
					flushCurrentPendingToken()
					if i+1 < len(sRunes) && sRunes[i+1] == '-' {
						// it is double-minus
						curToken = token{pos: curLinePos, line: curLine, class: tsOpDec, lexeme: "--"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
						i++
					} else if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is dec-by
						curToken = token{pos: curLinePos, line: curLine, class: tsOpDecset, lexeme: "-="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
						i++
					} else {
						// it is a minus
						curToken = token{pos: curLinePos, line: curLine, class: tsOpMinus, lexeme: "-"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
					}
				}
			} else if ch == '/' {
				if escaping {
					sb.WriteRune('/')
					escaping = false
				} else {
					flushCurrentPendingToken()
					curToken = token{pos: curLinePos, line: curLine, class: tsOpDivide, lexeme: "/"}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = token{}
				}
			} else if ch == '*' {
				if escaping {
					sb.WriteRune('*')
					escaping = false
				} else {
					flushCurrentPendingToken()
					curToken = token{pos: curLinePos, line: curLine, class: tsOpMultiply, lexeme: "*"}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = token{}
				}
			} else if ch == '!' {
				if escaping {
					sb.WriteRune('!')
					escaping = false
				} else {
					flushCurrentPendingToken()

					if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is not-equal
						curToken = token{pos: curLinePos, line: curLine, class: tsOpIsNot, lexeme: "!="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
						i++
					} else {
						// it is a negation
						curToken = token{pos: curLinePos, line: curLine, class: tsOpNot, lexeme: "!"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
					}
				}
			} else if ch == '<' {
				if escaping {
					sb.WriteRune('<')
					escaping = false
				} else {
					flushCurrentPendingToken()

					if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is lt/eq
						curToken = token{pos: curLinePos, line: curLine, class: tsOpLessThanIs, lexeme: "<="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
						i++
					} else {
						// it is less-than
						curToken = token{pos: curLinePos, line: curLine, class: tsOpLessThan, lexeme: "<"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
					}
				}
			} else if ch == '>' {
				if escaping {
					sb.WriteRune('>')
					escaping = false
				} else {
					flushCurrentPendingToken()

					if i+1 < len(sRunes) && sRunes[i+1] == '=' {
						// it is gt/eq
						curToken = token{pos: curLinePos, line: curLine, class: tsOpGreaterThanIs, lexeme: ">="}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
						i++
					} else {
						// it is greater-than
						curToken = token{pos: curLinePos, line: curLine, class: tsOpGreaterThan, lexeme: ">"}
						curToken.fullLine = currentfullLine
						tokens = append(tokens, curToken)
						curToken = token{}
					}
				}
			} else if startMatches(sRunes[i:], literalGroupOpen) {
				if escaping {
					writeRuneSlice(sb, literalGroupOpen)
					escaping = false
				} else {
					flushCurrentPendingToken()
					curToken = token{pos: curLinePos, line: curLine, class: tsGroupOpen, lexeme: literalStrGroupOpen}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = token{}
					parenDepth++
				}
				i += len(literalGroupOpen) - 1
			} else if startMatches(sRunes[i:], literalGroupClose) {
				i += len(literalGroupClose) - 1
				if escaping {
					writeRuneSlice(sb, literalGroupClose)
					escaping = false
				} else {
					// if we are not the parent this is an error.
					flushCurrentPendingToken()
					curToken = token{pos: curLinePos, line: curLine, class: tsGroupClose, lexeme: literalStrGroupClose}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = token{}
					parenDepth--
					if endAtMatchingParen && parenDepth == 0 {
						runesConsumed = i
						break
					}
				}
				// added to i at top
			} else if ch == '&' {
				if escaping {
					sb.WriteRune('&')
					escaping = false
				} else if i+1 < len(sRunes) && sRunes[i+1] == '&' {
					flushCurrentPendingToken()
					curToken = token{pos: curLinePos, line: curLine, class: tsOpAnd, lexeme: "&&"}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = token{}
					i++
				} else {
					sb.WriteRune('&')
				}
			} else if ch == '|' {
				if escaping {
					sb.WriteRune('|')
					escaping = false
				} else if i+1 < len(sRunes) && sRunes[i+1] == '|' {
					flushCurrentPendingToken()
					curToken = token{pos: curLinePos, line: curLine, class: tsOpOr, lexeme: "||"}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = token{}
					i++
				} else {
					sb.WriteRune('|')
				}
			} else if startMatches(sRunes[i:], literalOpIs) {
				if escaping {
					writeRuneSlice(sb, literalOpIs)
					escaping = false
				} else {
					flushCurrentPendingToken()

					curToken = token{pos: curLinePos, line: curLine, class: tsOpIs, lexeme: literalStrOpIs}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = token{}
				}
				i += len(literalOpIs) - 1
			} else if startMatches(sRunes[i:], literalOpSet) {
				if escaping {
					writeRuneSlice(sb, literalOpSet)
					escaping = false
				} else {
					flushCurrentPendingToken()

					curToken = token{pos: curLinePos, line: curLine, class: tsOpIs, lexeme: literalStrOpSet}
					curToken.fullLine = currentfullLine
					tokens = append(tokens, curToken)
					curToken = token{}
				}
				i += len(literalOpSet) - 1
			} else {

				// do not include whitespace unless it is escaped
				if escaping || !unicode.IsSpace(ch) {
					// is this the first non empty char? set the props for an unquoted string,
					// the default.
					if sb.Len() == 0 {
						curToken.line = curLine
						curToken.pos = curLinePos
						curToken.class = tsUnquotedString
					}
					sb.WriteRune(ch)
					escaping = false
				}
			}
		}

		curLinePos++
		if ch == '\n' {
			curLine++
			curLinePos = 1
		}
	}

	// do we have leftover parsing string? this is a lexing error, immediately
	// end
	if mode == lexString {
		return tokenStream{}, 0, SyntaxError{
			message: "unterminated '" + literalStrStringQuote + "'-string; missing a second '" + literalStrStringQuote + "'",
		}
	}

	// do we have leftover unparsed text? add it to the tokens list
	flushCurrentPendingToken()

	// add special EOT token
	tokens = append(tokens, token{
		pos:   curLinePos,
		line:  curLine,
		class: tsEndOfText,
	})

	return tokenStream{tokens: tokens}, runesConsumed, nil
}

func readFullLine(sRunes []rune) string {
	var lineBuilder strings.Builder
	for i := 0; i < len(sRunes) && sRunes[i] != '\n'; i++ {
		lineBuilder.WriteRune(sRunes[i])
	}
	return lineBuilder.String()
}

func startMatches(s []rune, check []rune) bool {
	if len(check) > len(s) {
		return false
	}

	for i := range check {
		if s[i] != check[i] {
			return false
		}
	}

	return true
}
